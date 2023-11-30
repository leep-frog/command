package cache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/stubs"
	"github.com/leep-frog/command/internal/testutil"
)

func TestCacheInitFunctions(t *testing.T) {
	for _, test := range []struct {
		name    string
		osOps   []osOp
		env     map[string]string
		f       func() (*Cache, error)
		want    *Cache
		wantErr error
	}{
		// FromDir tests
		{
			name: "FromDir fails if filepathAbs err",
			osOps: []osOp{
				&absOp{
					err:  fmt.Errorf("oops"),
					want: ptr("inputDir"),
				},
			},
			f:       func() (*Cache, error) { return FromDir("inputDir") },
			wantErr: fmt.Errorf("failed to get absolute path for cache directory: oops"),
		},
		{
			name: "FromDir fails if empty dir err",
			osOps: []osOp{
				&absOp{
					resp: "",
					want: ptr("inputDir"),
				},
			},
			f:       func() (*Cache, error) { return FromDir("inputDir") },
			wantErr: fmt.Errorf("invalid directory (inputDir) for cache: cache directory cannot be empty"),
		},
		{
			name: "FromDir fails if osStat err",
			osOps: []osOp{
				&absOp{
					resp: "full/path/inputDir",
					want: ptr("inputDir"),
				},
				&statOp{
					err:  fmt.Errorf("stat oops"),
					want: ptr("full/path/inputDir"),
				},
			},
			f:       func() (*Cache, error) { return FromDir("inputDir") },
			wantErr: fmt.Errorf("invalid directory (inputDir) for cache: failed to get info for cache: stat oops"),
		},
		{
			name: "FromDir fails if osMkdirAll err",
			osOps: []osOp{
				&absOp{
					resp: "full/path/inputDir",
					want: ptr("inputDir"),
				},
				&statOp{
					err:  fs.ErrNotExist,
					want: ptr("full/path/inputDir"),
				},
				&mkdirAllOp{
					err:  fmt.Errorf("mkdir all oops"),
					want: ptr("full/path/inputDir"),
				},
			},
			f:       func() (*Cache, error) { return FromDir("inputDir") },
			wantErr: fmt.Errorf("invalid directory (inputDir) for cache: cache directory does not exist and could not be created: mkdir all oops"),
		},
		{
			name: "FromDir succeeds with non-existant directory",
			osOps: []osOp{
				&absOp{
					resp: "full/path/inputDir",
					want: ptr("inputDir"),
				},
				&statOp{
					err:  fs.ErrNotExist,
					want: ptr("full/path/inputDir"),
				},
				&mkdirAllOp{
					want: ptr("full/path/inputDir"),
				},
			},
			f: func() (*Cache, error) { return FromDir("inputDir") },
			want: &Cache{
				Dir: "full/path/inputDir",
			},
		},
		{
			name: "FromDir fails if not a directory",
			osOps: []osOp{
				&absOp{
					resp: "full/path/inputDir",
					want: ptr("inputDir"),
				},
				&statOp{
					fi:   fakeFileType,
					want: ptr("full/path/inputDir"),
				},
			},
			f:       func() (*Cache, error) { return FromDir("inputDir") },
			wantErr: fmt.Errorf("invalid directory (inputDir) for cache: cache directory must point to a directory, not a file"),
		},
		{
			name: "FromDir succeeds with existing directory",
			osOps: []osOp{
				&absOp{
					resp: "full/path/inputDir",
					want: ptr("inputDir"),
				},
				&statOp{
					fi:   fakeDirType,
					want: ptr("full/path/inputDir"),
				},
			},
			f: func() (*Cache, error) { return FromDir("inputDir") },
			want: &Cache{
				Dir: "full/path/inputDir",
			},
		},
		// FromEnvVar tests
		{
			name:    "FromEnvVar fails if missing env var",
			f:       func() (*Cache, error) { return FromEnvVar("ENV_VAR") },
			wantErr: fmt.Errorf(`environment variable "ENV_VAR" is not set or is empty`),
		},
		{
			name: "FromEnvVar fails if env var is empty",
			env: map[string]string{
				"ENV_VAR": "",
			},
			f:       func() (*Cache, error) { return FromEnvVar("ENV_VAR") },
			wantErr: fmt.Errorf(`environment variable "ENV_VAR" is not set or is empty`),
		},
		{
			name: "FromEnvVar succeeds",
			env: map[string]string{
				"ENV_VAR": "inputDir",
			},
			osOps: []osOp{
				&absOp{
					resp: "full/path/inputDir",
					want: ptr("inputDir"),
				},
				&statOp{
					fi:   fakeDirType,
					want: ptr("full/path/inputDir"),
				},
			},
			f: func() (*Cache, error) { return FromEnvVar("ENV_VAR") },
			want: &Cache{
				Dir: "full/path/inputDir",
			},
		},
		{
			name: "FromEnvVarOrDir succeeds when using env var",
			env: map[string]string{
				"ENV_VAR": "envVarDir",
			},
			osOps: []osOp{
				&absOp{
					resp: "full/path/envVarDir",
					want: ptr("envVarDir"),
				},
				&statOp{
					fi:   fakeDirType,
					want: ptr("full/path/envVarDir"),
				},
			},
			f: func() (*Cache, error) { return FromEnvVarOrDir("ENV_VAR", "default/dir") },
			want: &Cache{
				Dir: "full/path/envVarDir",
			},
		},
		{
			name: "FromEnvVarOrDir succeeds when using default dir",
			osOps: []osOp{
				&absOp{
					resp: "full/path/default/dir",
					want: ptr("default/dir"),
				},
				&statOp{
					fi:   fakeDirType,
					want: ptr("full/path/default/dir"),
				},
			},
			f: func() (*Cache, error) { return FromEnvVarOrDir("ENV_VAR", "default/dir") },
			want: &Cache{
				Dir: "full/path/default/dir",
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			stubs.StubEnv(t, test.env)
			for _, osOp := range test.osOps {
				osOp.setup(t)
			}
			c, err := test.f()
			testutil.CmpError(t, "", test.wantErr, err)
			if diff := cmp.Diff(test.want, c, cmpopts.IgnoreUnexported(Cache{})); diff != "" {
				t.Errorf("Cache init produced incorrect cache (-want, +got):\n%s", diff)
			}
			for _, osOp := range test.osOps {
				osOp.verify(t)
			}
		})
	}
}

func TestPut(t *testing.T) {
	for _, test := range []struct {
		name       string
		key        string
		data       string
		want       string
		wantGetErr error
		wantGetOk  bool
		wantErr    error
	}{
		{
			name:       "put fails on empty",
			wantErr:    fmt.Errorf("failed to get file for key: invalid key format"),
			wantGetErr: fmt.Errorf("failed to get file for key: invalid key format"),
		}, {
			name:       "put fails on invalid key",
			key:        "abc-$",
			wantErr:    fmt.Errorf("failed to get file for key: invalid key format"),
			wantGetErr: fmt.Errorf("failed to get file for key: invalid key format"),
		}, {
			name:      "put succeeds",
			key:       "abc",
			data:      "def",
			want:      "def",
			wantGetOk: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := NewTestCache(t)

			prefix := fmt.Sprintf("Put(%s, %s)", test.key, test.data)
			err := c.Put(test.key, test.data)
			testutil.CmpError(t, prefix, test.wantErr, err)

			stored, ok, err := c.Get(test.key)
			testutil.CmpError(t, prefix, test.wantGetErr, err)
			if ok != test.wantGetOk {
				t.Errorf("PutStruct(%s, %v) returned ok=%v; want %v", test.key, test.data, ok, test.wantGetOk)
			}
			if diff := cmp.Diff(test.want, stored); diff != "" {
				t.Errorf("PutStruct(%s, %v) produced diff:\n%s", test.key, test.data, diff)
			}
		})
	}
}

func TestGet(t *testing.T) {
	for _, test := range []struct {
		name    string
		key     string
		want    string
		wantOk  bool
		wantErr error
	}{
		{
			name:    "get fails on empty",
			wantErr: fmt.Errorf("failed to get file for key: invalid key format"),
		}, {
			name:    "get fails on invalid key",
			key:     "abc-$",
			wantErr: fmt.Errorf("failed to get file for key: invalid key format"),
		}, {
			name: "returns empty string on missing key",
			key:  "xyz",
		}, {
			name:   "returns value on valid key",
			key:    "abc",
			want:   "123\n456\n",
			wantOk: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Temporarily change cache dir
			c := NewTestCache(t)
			Put(t, c, "abc", "123\n456\n")

			prefix := fmt.Sprintf("Get(%s)", test.key)
			resp, ok, err := c.Get(test.key)
			testutil.CmpError(t, prefix, test.wantErr, err)

			if ok != test.wantOk {
				t.Errorf("Get(%s) returned ok=%v; want %v", test.key, ok, test.wantOk)
			}

			if diff := cmp.Diff(test.want, resp); diff != "" {
				t.Fatalf("Get(%s) produced diff :\n%s", test.key, diff)
			}
		})
	}
}

func Put(t *testing.T, c *Cache, key, data string) {
	t.Helper()
	if err := c.Put(key, data); err != nil {
		t.Fatalf("Put(%s, %s) failed: %v", key, data, err)
	}
}

func PutStruct(t *testing.T, c *Cache, key string, data interface{}) {
	t.Helper()
	if err := c.PutStruct(key, data); err != nil {
		t.Fatalf("PutStruct(%s, %v) failed: %v", key, data, err)
	}
}

func Get(t *testing.T, c *Cache, key string) (string, bool) {
	t.Helper()
	s, b, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get(%s) returned error: %v", key, err)
	}
	return s, b
}

func TestDeleteOld(t *testing.T) {
	c := NewTestCache(t)
	key := "qwerty"

	t.Run("Delete works when file doesn't exist", func(t *testing.T) {
		if err := c.Delete(key); err != nil {
			t.Errorf("Delete(%s) returned error (%v); want nil", key, err)
		}
	})

	t.Run("Delete fails if invalid key", func(t *testing.T) {
		wantErr := fmt.Errorf("failed to get file for key: invalid key format")
		if diff := cmp.Diff(wantErr.Error(), c.Delete(".?.").Error()); diff != "" {
			t.Errorf("Delete('.?.') returned wrong error (-want, +got):\n%s", diff)
		}
	})

	Put(t, c, key, "uiop")
	if got, _ := Get(t, c, key); got != "uiop" {
		t.Fatalf("Get(%s) returned %s; want %s", key, got, "uiop")
	}

	t.Run("Delete fails if osRemove error", func(t *testing.T) {
		op := &removeOp{
			err:  fmt.Errorf("remove oops"),
			want: ptr(filepath.Join(c.Dir, key)),
		}
		op.setup(t)
		wantErr := fmt.Errorf("failed to delete file: remove oops")
		gotErr := c.Delete(key)
		if gotErr == nil {
			t.Fatalf("Delete(...) returned no error when should have returned (%v)", wantErr)
		}
		if diff := cmp.Diff(wantErr.Error(), gotErr.Error()); diff != "" {
			t.Errorf("Delete(...) returned wrong error (-want, +got):\n%s", diff)
		}
		op.verify(t)
	})

	t.Run("Delete works", func(t *testing.T) {
		if err := c.Delete(key); err != nil {
			t.Errorf("Delete(%s) returned error (%v); want nil", key, err)
		}
	})
	if _, ok := Get(t, c, key); ok {
		t.Fatalf("Get(%s) returned ok=true; want false", key)
	}
}

type testStruct struct {
	A int64
	B string
	C map[string]bool
}

type put struct {
	key  string
	data string
}

func fullCache(t *testing.T, c *Cache) map[string]string {
	t.Helper()
	keys, err := c.List()
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}
	m := map[string]string{}
	for _, k := range keys {
		s, _ := Get(t, c, k)
		m[k] = s
	}
	return m
}

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name          string
		c             *Cache
		puts          []*put
		etc           *commandtest.ExecuteTestCase
		osOps         []osOp
		skipFileCheck bool
		mkdirAllErr   error
		want          map[string]string
		wantC         *Cache
	}{
		{
			name: "Requires branching arg",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				WantErr:    fmt.Errorf("Branching argument must be one of [delete get list put setdir]"),
				WantStderr: "Branching argument must be one of [delete get list put setdir]\n",
			},
		},
		// Get tests
		{
			name: "Gets fails if unknown dir and mkdir error",
			c: &Cache{
				Dir: filepath.Join("bob", "lob", "law"),
			},
			skipFileCheck: true,
			mkdirAllErr:   fmt.Errorf("oops"),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"get", "here"},
				WantStderr: fmt.Sprintf("failed to get file for key: failed to get cache directory: cache directory does not exist and could not be created: oops\n"),
				WantErr:    fmt.Errorf("failed to get file for key: failed to get cache directory: cache directory does not exist and could not be created: oops"),
			},
		},
		{
			name: "Get requires key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"get"},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Get requires valid key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"get", ".?,"},
				WantErr:    fmt.Errorf(`validation for "KEY" failed: [MatchesRegex] value ".?," doesn't match regex %q`, keyRegex),
				WantStderr: fmt.Sprintf("validation for \"KEY\" failed: [MatchesRegex] value \".?,\" doesn't match regex %q\n", keyRegex),
			},
		},
		{
			name: "Get missing key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"get", "uh"},
				WantStderr: "key not found\n",
			},
		},
		{
			name: "Gets present key key",
			c:    NewTestCache(t),
			puts: []*put{
				{
					key:  "here",
					data: "hello\nthere",
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"get", "here"},
				WantStdout: "hello\nthere\n",
			},
			want: map[string]string{
				"here": "hello\nthere",
			},
		},
		// Put tests
		{
			name: "Put fails if unknown dir",
			c: &Cache{
				Dir: filepath.Join("bob", "lob", "law"),
			},
			skipFileCheck: true,
			mkdirAllErr:   fmt.Errorf("whoops"),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"put", "things", "here"},
				WantStderr: fmt.Sprintf("failed to get file for key: failed to get cache directory: cache directory does not exist and could not be created: whoops\n"),
				WantErr:    fmt.Errorf("failed to get file for key: failed to get cache directory: cache directory does not exist and could not be created: whoops"),
			},
		},
		{
			name: "Put requires key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"put"},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Put requires valid key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"put", ".?,"},
				WantErr:    fmt.Errorf(`validation for "KEY" failed: [MatchesRegex] value ".?," doesn't match regex %q`, keyRegex),
				WantStderr: fmt.Sprintf("validation for \"KEY\" failed: [MatchesRegex] value \".?,\" doesn't match regex %q\n", keyRegex),
			},
		},
		{
			name: "Put requires data",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"put", "things"},
				WantErr:    fmt.Errorf(`Argument "DATA" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"DATA\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Put works",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"put", "things", "better than", "you found them"},
			},
			want: map[string]string{
				"things": "better than you found them",
			},
		},
		{
			name: "Put overrides",
			c:    NewTestCache(t),
			puts: []*put{
				{"this", "that"},
				{"things", "worse"},
				{"hello", "there"},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"put", "things", "better than", "you found them"},
			},
			want: map[string]string{
				"this":   "that",
				"things": "better than you found them",
				"hello":  "there",
			},
		},
		// List tests
		{
			name: "list fails if unknown dir",
			c: &Cache{
				Dir: filepath.Join("bob", "lob", "law"),
			},
			skipFileCheck: true,
			mkdirAllErr:   fmt.Errorf("argh"),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"list"},
				WantStderr: fmt.Sprintf("cache directory does not exist and could not be created: argh\n"),
				WantErr:    fmt.Errorf("cache directory does not exist and could not be created: argh"),
			},
		},
		{
			name: "List works with no data",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"list"},
			},
		},
		{
			name: "List works with data",
			c:    NewTestCache(t),
			puts: []*put{
				{"this", "that"},
				{"things", "better than you found them"},
				{"hello", "there"},
			},
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"list"},
				WantStdout: "hello\nthings\nthis\n",
			},
			want: map[string]string{
				"this":   "that",
				"things": "better than you found them",
				"hello":  "there",
			},
		},
		// Delete tests
		{
			name: "Delete requires key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"delete"},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Delete requires valid key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"delete", ".?,"},
				WantErr:    fmt.Errorf(`validation for "KEY" failed: [MatchesRegex] value ".?," doesn't match regex %q`, keyRegex),
				WantStderr: fmt.Sprintf("validation for \"KEY\" failed: [MatchesRegex] value \".?,\" doesn't match regex %q\n", keyRegex),
			},
		},
		{
			name: "Delete non-existant key",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"delete", "uh"},
			},
		},
		{
			name: "Delete key",
			c:    NewTestCache(t),
			puts: []*put{
				{"this", "that"},
				{"things", "worse than you found them"},
				{"hello", "there"},
			},
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"delete", "things"},
			},
			want: map[string]string{
				"this":  "that",
				"hello": "there",
			},
		},
		// setdir tests
		{
			name: "setdir requires an argument",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"setdir"},
				WantErr:    fmt.Errorf(`Argument "DIR" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"DIR\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "setdir requires an existing file",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"setdir", "uh"},
				WantErr:    fmt.Errorf("validation for \"DIR\" failed: [FileExists] file %q does not exist", testutil.FilepathAbs(t, "uh")),
				WantStderr: fmt.Sprintf("validation for \"DIR\" failed: [FileExists] file %q does not exist\n", testutil.FilepathAbs(t, "uh")),
			},
		},
		{
			name: "setdir doesn't allow files",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"setdir", "cache.go"},
				WantErr:    fmt.Errorf("validation for \"DIR\" failed: [IsDir] argument %q is a file", testutil.FilepathAbs(t, "cache.go")),
				WantStderr: fmt.Sprintf("validation for \"DIR\" failed: [IsDir] argument %q is a file\n", testutil.FilepathAbs(t, "cache.go")),
			},
		},
		{
			name: "setdir works",
			c:    NewTestCache(t),
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"setdir", "testing"},
			},
			wantC: &Cache{
				Dir: testutil.FilepathAbs(t, "testing"),
			},
			want: map[string]string{
				"empty.txt": "nothing to see here",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, osOp := range test.osOps {
				osOp.setup(t)
			}
			testutil.StubValue(t, &osMkdirAll, func(string, fs.FileMode) error {
				return test.mkdirAllErr
			})
			for _, p := range test.puts {
				Put(t, test.c, p.key, p.data)
			}
			if test.etc == nil {
				test.etc = &commandtest.ExecuteTestCase{}
			}
			test.etc.OS = &commandtest.FakeOS{}
			test.etc.Node = test.c.Node()
			test.etc.SkipDataCheck = true
			commandertest.ExecuteTest(t, test.etc)
			commandertest.ChangeTest(t, test.wantC, test.c, cmpopts.IgnoreUnexported(Cache{}))

			if !test.skipFileCheck {
				if diff := cmp.Diff(test.want, fullCache(t, test.c), cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Execute(%v) resulted in incorrect cache (-want, +got):\n%s", test.etc.Args, diff)
				}
			}
			for _, osOp := range test.osOps {
				osOp.verify(t)
			}
		})
	}
}

func TestCompletion(t *testing.T) {
	puts := []*put{
		{"this", "that"},
		{"things", "worse than you found them"},
		{"hello", "there"},
	}
	for _, test := range []struct {
		name string
		puts []*put
		ctc  *commandtest.CompleteTestCase
	}{
		{
			name: "completes branches",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd ",
				Want: &commondels.Autocompletion{
					Suggestions: []string{"delete", "get", "list", "put", "setdir"},
				},
			},
		},
		{
			name: "completes for get",
			puts: puts,
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd get ",
				Want: &commondels.Autocompletion{
					Suggestions: []string{"hello", "things", "this"},
				},
			},
		},
		{
			name: "completes for put",
			puts: puts,
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd put t",
				Want: &commondels.Autocompletion{
					Suggestions: []string{"things", "this"},
				},
			},
		},
		{
			name: "completes for delete",
			puts: puts,
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd put thin",
				Want: &commondels.Autocompletion{
					Suggestions: []string{"things"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := NewTestCache(t)
			for _, p := range test.puts {
				Put(t, c, p.key, p.data)
			}
			if test.ctc == nil {
				test.ctc = &commandtest.CompleteTestCase{}
			}
			test.ctc.Node = c.Node()
			test.ctc.SkipDataCheck = true
			commandertest.CompleteTest(t, test.ctc)
		})
	}
}

func TestGetStruct(t *testing.T) {
	key := "key"
	c := NewTestCache(t)
	val := &testStruct{
		A: 1,
		B: "two",
		C: map[string]bool{
			"three": true,
		},
	}
	if err := c.PutStruct(key, val); err != nil {
		t.Fatalf("failed to put struct: %v", err)
	}

	for _, test := range []struct {
		name    string
		c       *Cache
		key     string
		obj     interface{}
		osOps   []osOp
		wantOK  bool
		wantErr error
		wantObj interface{}
	}{
		{
			name:    "fails if invalid key",
			key:     "abc!",
			wantErr: fmt.Errorf("failed to get file for key: invalid key format"),
		},
		{
			name:    "fails if empty cache dir",
			c:       &Cache{},
			key:     "abc-key",
			wantErr: fmt.Errorf("failed to get file for key: failed to get cache directory: cache directory cannot be empty"),
		},
		{
			name: "fails if stat err",
			c: &Cache{
				Dir: "some/dir",
			},
			osOps: []osOp{
				&statOp{
					want: ptr("some/dir"),
					err:  fmt.Errorf("stat oops"),
				},
			},
			key:     "abc-key",
			wantErr: fmt.Errorf("failed to get file for key: failed to get cache directory: failed to get info for cache: stat oops"),
		},
		{
			name: "fails if stat returns file type",
			c: &Cache{
				Dir: "some/dir",
			},
			osOps: []osOp{
				&statOp{
					want: ptr("some/dir"),
					fi:   fakeFileType,
				},
			},
			key:     "abc-key",
			wantErr: fmt.Errorf("failed to get file for key: failed to get cache directory: cache directory must point to a directory, not a file"),
		},
		{
			name: "fails if not exist err and mkdirAllErr",
			c: &Cache{
				Dir: "some/dir",
			},
			osOps: []osOp{
				&statOp{
					want: ptr("some/dir"),
					err:  fs.ErrNotExist,
				},
				&mkdirAllOp{
					want: ptr("some/dir"),
					err:  fmt.Errorf("mkdirAll oops"),
				},
			},
			key:     "abc-key",
			wantErr: fmt.Errorf("failed to get file for key: failed to get cache directory: cache directory does not exist and could not be created: mkdirAll oops"),
		},
		{
			name: "fails if read file err",
			c: &Cache{
				Dir: "some/dir",
			},
			osOps: []osOp{
				&statOp{
					want: ptr("some/dir"),
					fi:   fakeDirType,
				},
				&readFileOp{
					want: ptr(filepath.Join("some/dir", "abc-key")),
					err:  fmt.Errorf("readFile oops"),
				},
			},
			key:     "abc-key",
			wantErr: fmt.Errorf("failed to read file: readFile oops"),
		},
		{
			name: "succeeds if read file err is not exist",
			c: &Cache{
				Dir: "some/dir",
			},
			osOps: []osOp{
				&statOp{
					want: ptr("some/dir"),
					fi:   fakeDirType,
				},
				&readFileOp{
					want: ptr(filepath.Join("some/dir", "abc-key")),
					err:  fs.ErrNotExist,
				},
			},
			key: "abc-key",
		},
		{
			name: "fails if invalid json is returned",
			c: &Cache{
				Dir: "some/dir",
			},
			osOps: []osOp{
				&statOp{
					want: ptr("some/dir"),
					fi:   fakeDirType,
				},
				&readFileOp{
					want:     ptr(filepath.Join("some/dir", "abc-key")),
					contents: "}{",
				},
			},
			key:     "abc-key",
			wantOK:  true,
			wantErr: fmt.Errorf("failed to unmarshal cache data: invalid character '}' looking for beginning of value"),
		},
		{
			name: "succeeds if valid json",
			c: &Cache{
				Dir: "some/dir",
			},
			obj: &testStruct{},
			wantObj: &testStruct{
				A: 4,
				C: map[string]bool{
					"heyo": true,
					"ohno": false,
				},
			},
			osOps: []osOp{
				&statOp{
					want: ptr("some/dir"),
					fi:   fakeDirType,
				},
				&readFileOp{
					want: ptr(filepath.Join("some/dir", "abc-key")),
					contents: strings.Join([]string{
						`{`,
						`  "A": 4,`,
						`  "C": {`,
						`    "heyo": true,`,
						`    "ohno": false`,
						`  }`,
						`}`,
					}, "\n"),
				},
			},
			key:    "abc-key",
			wantOK: true,
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, osOp := range test.osOps {
				osOp.setup(t)
			}
			prefix := fmt.Sprintf("GetStruct(%s)", test.key)

			ok, err := test.c.GetStruct(test.key, test.obj)
			if ok != test.wantOK {
				t.Errorf("%s returned ok=%v; want %v", prefix, ok, test.wantOK)
			}
			testutil.CmpError(t, prefix, test.wantErr, err)

			if diff := cmp.Diff(test.wantObj, test.obj, cmp.AllowUnexported(Cache{})); diff != "" {
				t.Errorf("%s returned object diff (-want, +got):\n%s", prefix, diff)
			}

			for _, osOp := range test.osOps {
				osOp.verify(t)
			}
		})
	}
}

func TestPutStruct(t *testing.T) {
	var listA []interface{}
	listA = append(listA, &listA)
	for _, test := range []struct {
		name         string
		key          string
		data         interface{}
		osOps        []osOp
		stubCacheDir string
		wantGet      string
		wantGetErr   error
		wantGetOk    bool
		wantErr      error
	}{
		{
			name:       "put fails on empty",
			wantErr:    fmt.Errorf("failed to get file for key: invalid key format"),
			wantGetErr: fmt.Errorf("failed to get file for key: invalid key format"),
		},
		{
			name:       "put fails on invalid key",
			key:        "abc-$",
			wantErr:    fmt.Errorf("failed to get file for key: invalid key format"),
			wantGetErr: fmt.Errorf("failed to get file for key: invalid key format"),
		},
		{
			name:    "put fails on marshal error",
			key:     "abc",
			data:    listA,
			wantErr: fmt.Errorf("failed to marshal struct to json: json: unsupported value: encountered a cycle via []interface {}"),
		},
		{
			name: "put fails on osWriteFile error",
			key:  "abc",
			data: &testStruct{
				A: 1,
				B: "two",
				C: map[string]bool{
					"three": true,
				},
			},
			stubCacheDir: "fake-dir",
			osOps: []osOp{
				&statOp{
					fi:        fakeDirType,
					want:      ptr("fake-dir"),
					allowReal: true,
				},
				&writeFileOp{
					err:  fmt.Errorf("writeFile oops"),
					want: ptr(filepath.Join("fake-dir", "abc")),
					wantContents: ptr(strings.Join([]string{
						"{",
						`  "A": 1,`,
						`  "B": "two",`,
						`  "C": {`,
						`    "three": true`,
						"  }",
						"}",
					}, "\n")),
				},
			},
			wantErr: fmt.Errorf("failed to write file: writeFile oops"),
		},
		{
			name: "put succeeds",
			key:  "abc",
			data: &testStruct{
				A: 1,
				B: "two",
				C: map[string]bool{
					"three": true,
				},
			},
			wantGetOk: true,
			wantGet: strings.Join([]string{
				"{",
				`  "A": 1,`,
				`  "B": "two",`,
				`  "C": {`,
				`    "three": true`,
				"  }",
				"}",
			}, "\n"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, osOp := range test.osOps {
				osOp.setup(t)
			}
			c := NewTestCache(t)
			if test.stubCacheDir != "" {
				c = &Cache{Dir: test.stubCacheDir}
			}

			prefix := fmt.Sprintf("PutStruct(%s, %v)", test.key, test.data)

			err := c.PutStruct(test.key, test.data)
			testutil.CmpError(t, prefix, test.wantErr, err)

			stored, ok, err := c.Get(test.key)
			testutil.CmpError(t, prefix, test.wantGetErr, err)
			if ok != test.wantGetOk {
				t.Errorf("PutStruct(%s, %v) returned ok=%v; want %v", test.key, test.data, ok, test.wantGetOk)
			}
			if diff := cmp.Diff(test.wantGet, stored); diff != "" {
				t.Errorf("PutStruct(%s, %v) produced diff:\n%s", test.key, test.data, diff)
			}
			for _, osOp := range test.osOps {
				osOp.verify(t)
			}
		})
	}
}

func TestNewTestCacheWithData(t *testing.T) {
	cmpGet := func(t *testing.T, want, got interface{}, wantOK, ok bool, wantErr, err error) {
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Cache contained incorrect data (-want, +got):\n%s", diff)
		}
		if ok != wantOK {
			t.Errorf("Cache returned incorrect ok: got %v, want %v", ok, wantOK)
		}
		testutil.CmpError(t, "Cache", wantErr, err)
	}

	for _, test := range []struct {
		name string
		data map[string]interface{}
		t    func(t *testing.T, c *Cache)
	}{
		{
			name: "handles nil data",
		},
		{
			name: "handles empty data",
			data: map[string]interface{}{},
		},
		{
			name: "handles missing key",
			data: map[string]interface{}{},
			t: func(t *testing.T, c *Cache) {
				k, ok, err := c.Get("v1")
				cmpGet(t, "", k, false, ok, nil, err)
			},
		},
		{
			name: "inserts string",
			data: map[string]interface{}{
				"v1": "k1",
			},
			t: func(t *testing.T, c *Cache) {
				k, ok, err := c.Get("v1")
				cmpGet(t, "k1", k, true, ok, nil, err)
			},
		},
		{
			name: "inserts multiple strings",
			data: map[string]interface{}{
				"v1": "k1",
				"v2": "k2",
			},
			t: func(t *testing.T, c *Cache) {
				k, ok, err := c.Get("v1")
				cmpGet(t, "k1", k, true, ok, nil, err)
				k, ok, err = c.Get("v2")
				cmpGet(t, "k2", k, true, ok, nil, err)
			},
		},
		{
			name: "inserts interface",
			data: map[string]interface{}{
				"v1": &testStruct{
					A: 123,
					B: "456",
					C: map[string]bool{
						"78": true,
						"9":  false,
					},
				},
			},
			t: func(t *testing.T, c *Cache) {
				ts := &testStruct{}
				ok, err := c.GetStruct("v1", ts)
				wantTS := &testStruct{
					A: 123,
					B: "456",
					C: map[string]bool{
						"78": true,
						"9":  false,
					},
				}
				cmpGet(t, wantTS, ts, true, ok, nil, err)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := NewTestCacheWithData(t, test.data)
			if test.t != nil {
				test.t(t, c)
			}
		})
	}
}

func TestNewShell(t *testing.T) {
	dir, err := os.MkdirTemp("", "leep-cd-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	fos := &commandtest.FakeOS{}

	for _, test := range []struct {
		name         string
		etc          *commandtest.ExecuteTestCase
		mkdirTemp    string
		mkdirTempErr error
		mkdirAllErr  error
		wantCache    *Cache
	}{
		{
			name: "returns existing shell cache",
			etc: &commandtest.ExecuteTestCase{
				Env: map[string]string{
					ShellOSEnvVar: dir,
				},
				WantData: &commondels.Data{Values: map[string]interface{}{
					ShellDataKey: &Cache{Dir: dir},
				}},
			},
			wantCache: &Cache{
				Dir: dir,
			},
		},
		{
			name:        "returns error if existing shell cache doesn't point to a directory and fail to create",
			mkdirAllErr: fmt.Errorf("whoops"),
			etc: &commandtest.ExecuteTestCase{
				Env: map[string]string{
					ShellOSEnvVar: filepath.Join(dir, "bleh", "eh"),
				},
				WantStderr: fmt.Sprintf("failed to create shell-level cache: invalid directory (%s) for cache: cache directory does not exist and could not be created: whoops\n", filepath.Join(dir, "bleh", "eh")),
				WantErr:    fmt.Errorf("failed to create shell-level cache: invalid directory (%s) for cache: cache directory does not exist and could not be created: whoops", filepath.Join(dir, "bleh", "eh")),
			},
		},
		{
			name:         "Error if fails to create temp dir",
			mkdirTempErr: fmt.Errorf("oops"),
			etc: &commandtest.ExecuteTestCase{
				WantErr:    fmt.Errorf("failed to create temporary directory: oops"),
				WantStderr: "failed to create temporary directory: oops\n",
			},
		},
		{
			name:      "Creates dir and sets env",
			mkdirTemp: dir,
			wantCache: &Cache{
				Dir: dir,
			},
			etc: &commandtest.ExecuteTestCase{
				WantExecuteData: &commondels.ExecuteData{
					Executable: []string{
						fos.SetEnvVar(ShellOSEnvVar, dir),
					},
				},
				WantData: &commondels.Data{Values: map[string]interface{}{
					ShellDataKey: &Cache{Dir: dir},
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testutil.StubValue(t, &osMkdirTemp, func(string, string) (string, error) {
				return dir, test.mkdirTempErr
			})
			testutil.StubValue(t, &osMkdirAll, func(string, fs.FileMode) error {
				return test.mkdirAllErr
			})

			var c *Cache
			d := &commondels.Data{}
			test.etc.Node = commander.SerialNodes(
				ShellProcessor(),
				commander.SuperSimpleProcessor(func(i *commondels.Input, data *commondels.Data) error {
					c = ShellFromData(data)
					d = data
					return nil
				}),
			)
			test.etc.SkipDataCheck = true
			test.etc.OS = fos
			commandertest.ExecuteTest(t, test.etc)

			if diff := cmp.Diff(test.etc.WantData, d, cmp.AllowUnexported(Cache{}, commondels.Data{}), cmpopts.IgnoreFields(commondels.Data{}, "OS")); diff != "" {
				t.Errorf("ShellProcessor() produced incorrect data (-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.wantCache, c, cmp.AllowUnexported(Cache{})); diff != "" {
				t.Errorf("NewShell() returned incorrect cache (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	c := &Cache{}
	if c.Setup() != nil {
		t.Errorf("Cache returned unexpected setup: %v", c.Setup())
	}
	if c.Name() != "cash" {
		t.Errorf("Cache.Name() returned unexpected name: expected 'cash'; got %q", c.Name())
	}
}

type osOp interface {
	setup(*testing.T)
	verify(*testing.T)
}

type mkdirAllOp struct {
	err  error
	want *string
	got  *string
}

func (ma *mkdirAllOp) setup(t *testing.T) {
	testutil.StubValue(t, &osMkdirAll, func(s string, m fs.FileMode) error {
		if ma.got != nil {
			t.Fatalf("osMkdirAll called multiple times")
		}
		ma.got = &s
		return ma.err
	})
}

func (ma *mkdirAllOp) verify(t *testing.T) {
	if diff := cmp.Diff(ma.want, ma.got); diff != "" {
		t.Fatalf("osMkdirAll called with incorrect arguments (-want, +got):\n%s", diff)
	}
}

type statOp struct {
	err       error
	fi        *fakeFileInfo
	want      *string
	got       *string
	stubAt    int
	stubCount int
	allowReal bool
}

func (so *statOp) setup(t *testing.T) {
	testutil.StubValue(t, &osStat, func(s string) (fs.FileInfo, error) {
		defer func() { so.stubCount++ }()
		if so.stubCount != so.stubAt {
			if so.allowReal {
				return os.Stat(s)
			}
			t.Fatalf("osStat called multiple times")
		}

		so.got = &s
		return so.fi, so.err
	})
}

func (so *statOp) verify(t *testing.T) {
	if diff := cmp.Diff(so.want, so.got); diff != "" {
		t.Fatalf("osStat called with incorrect arguments (-want, +got):\n%s", diff)
	}
}

type readFileOp struct {
	err      error
	contents string
	want     *string
	got      *string
}

func (rfo *readFileOp) setup(t *testing.T) {
	testutil.StubValue(t, &osReadFile, func(s string) ([]byte, error) {
		if rfo.got != nil {
			t.Fatalf("osReadFile called multiple times")
		}
		rfo.got = &s
		return []byte(rfo.contents), rfo.err
	})
}

func (rfo *readFileOp) verify(t *testing.T) {
	if diff := cmp.Diff(rfo.want, rfo.got); diff != "" {
		t.Fatalf("osReadFile called with incorrect arguments (-want, +got):\n%s", diff)
	}
}

type writeFileOp struct {
	err          error
	want         *string
	got          *string
	wantContents *string
	gotContents  *string
}

func (wfo *writeFileOp) setup(t *testing.T) {
	testutil.StubValue(t, &osWriteFile, func(s string, data []byte, fm fs.FileMode) error {
		if wfo.got != nil {
			t.Fatalf("osReadFile called multiple times")
		}
		wfo.got = &s
		dataS := string(data)
		wfo.gotContents = &dataS
		return wfo.err
	})
}

func (wfo *writeFileOp) verify(t *testing.T) {
	if diff := cmp.Diff(wfo.want, wfo.got); diff != "" {
		t.Fatalf("osWriteFile called with incorrect filename (-want, +got):\n%s", diff)
	}
	if diff := cmp.Diff(wfo.wantContents, wfo.gotContents); diff != "" {
		t.Fatalf("osWriteFile called with incorrect contents (-want, +got):\n%s", diff)
	}
}

type removeOp struct {
	err  error
	want *string
	got  *string
}

func (ro *removeOp) setup(t *testing.T) {
	testutil.StubValue(t, &osRemove, func(s string) error {
		if ro.got != nil {
			t.Fatalf("osRemove called multiple times")
		}
		ro.got = &s
		return ro.err
	})
}

func (ro *removeOp) verify(t *testing.T) {
	if diff := cmp.Diff(ro.want, ro.got); diff != "" {
		t.Fatalf("osRemove called with incorrect arguments (-want, +got):\n%s", diff)
	}
}

type absOp struct {
	err  error
	resp string
	want *string
	got  *string
}

func (ao *absOp) setup(t *testing.T) {
	testutil.StubValue(t, &filepathAbs, func(s string) (string, error) {
		if ao.got != nil {
			t.Fatalf("filepathAbs called multiple times")
		}
		ao.got = &s
		return ao.resp, ao.err
	})
}

func (ao *absOp) verify(t *testing.T) {
	if diff := cmp.Diff(ao.want, ao.got); diff != "" {
		t.Fatalf("filepathAbs called with incorrect arguments (-want, +got):\n%s", diff)
	}
}

func ptr[T any](t T) *T { return &t }

type fakeFileInfo struct {
	isDir bool
}

func (*fakeFileInfo) Name() string       { return "" }
func (*fakeFileInfo) Size() int64        { return 0 }
func (*fakeFileInfo) Mode() os.FileMode  { return 0 }
func (*fakeFileInfo) ModTime() time.Time { return time.Now() }
func (ffi *fakeFileInfo) IsDir() bool    { return ffi.isDir }
func (*fakeFileInfo) Sys() interface{}   { return nil }

var (
	fakeFileType = &fakeFileInfo{}
	fakeDirType  = &fakeFileInfo{true}
)
