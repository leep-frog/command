package cache

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
)

func TestPut(t *testing.T) {
	for _, test := range []struct {
		name       string
		key        string
		data       string
		want       string
		wantGetErr string
		wantGetOk  bool
		wantErr    string
	}{
		{
			name:       "put fails on empty",
			wantErr:    "invalid key format",
			wantGetErr: "failed to get file for key: invalid key format",
		}, {
			name:       "put fails on invalid key",
			key:        "abc-$",
			wantErr:    "invalid key format",
			wantGetErr: "failed to get file for key: invalid key format",
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

			err := c.Put(test.key, test.data)
			if test.wantErr == "" && err != nil {
				t.Errorf("Put(%s, %s) returned err %v; want nil", test.key, test.data, err)
			}
			if test.wantErr != "" && err == nil {
				t.Errorf("Put(%s, %s) returned nil; want err %q", test.key, test.data, test.wantErr)
			}
			if test.wantErr != "" && err != nil && !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("Put(%s, %s) returned err %q; want %q", test.key, test.data, err.Error(), test.wantErr)
			}

			stored, ok, err := c.Get(test.key)
			if (err == nil) != (test.wantErr == "") {
				t.Fatalf("PutStruct(%s, %v) returned get error (%v); want %v", test.key, test.data, err, test.wantErr)
			}
			if err != nil {
				if diff := cmp.Diff(test.wantGetErr, err.Error()); diff != "" {
					t.Errorf("PutStruct(%s, %v) returned wrong get error (-want, +got):\n%s", test.key, test.data, diff)
				}
			}
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
		wantErr string
	}{
		{
			name:    "get fails on empty",
			wantErr: "invalid key format",
		}, {
			name:    "get fails on invalid key",
			key:     "abc-$",
			wantErr: "invalid key format",
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

			resp, ok, err := c.Get(test.key)
			if test.wantErr == "" && err != nil {
				t.Errorf("Get(%s) returned err: %v; want nil", test.key, err)
			}
			if test.wantErr != "" && err == nil {
				t.Errorf("Get(%s) returned nil; want err %q", test.key, test.wantErr)
			}
			if test.wantErr != "" && err != nil && !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("Get(%s) returned err: %q; want %q", test.key, err.Error(), test.wantErr)
			}

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

func TestDelete(t *testing.T) {
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
	t.Run("Delete works when file doesn't exist", func(t *testing.T) {
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
		etc           *command.ExecuteTestCase
		skipFileCheck bool
		want          map[string]string
		wantC         *Cache
	}{
		{
			name: "Requires branching arg",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				WantErr:    fmt.Errorf("Branching argument must be one of [delete get list put setdir]"),
				WantStderr: "Branching argument must be one of [delete get list put setdir]\n",
			},
		},
		// Get tests
		{
			name: "Gets fails if unknown dir",
			c: &Cache{
				Dir: filepath.Join("bob", "lob", "law"),
			},
			skipFileCheck: true,
			etc: &command.ExecuteTestCase{
				Args:       []string{"get", "here"},
				WantStderr: "failed to get file for key: failed to get cache directory: cache directory does not exist\n",
				WantErr:    fmt.Errorf("failed to get file for key: failed to get cache directory: cache directory does not exist"),
			},
		},
		{
			name: "Get requires key",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"get"},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Get requires valid key",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"get", ".?,"},
				WantErr:    fmt.Errorf(`validation for "KEY" failed: [MatchesRegex] value ".?," doesn't match regex %q`, keyRegex),
				WantStderr: fmt.Sprintf("validation for \"KEY\" failed: [MatchesRegex] value \".?,\" doesn't match regex %q\n", keyRegex),
			},
		},
		{
			name: "Get missing key",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
				Args:       []string{"put", "things", "here"},
				WantStderr: "failed to get file for key: failed to get cache directory: cache directory does not exist\n",
				WantErr:    fmt.Errorf("failed to get file for key: failed to get cache directory: cache directory does not exist"),
			},
		},
		{
			name: "Put requires key",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"put"},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Put requires valid key",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"put", ".?,"},
				WantErr:    fmt.Errorf(`validation for "KEY" failed: [MatchesRegex] value ".?," doesn't match regex %q`, keyRegex),
				WantStderr: fmt.Sprintf("validation for \"KEY\" failed: [MatchesRegex] value \".?,\" doesn't match regex %q\n", keyRegex),
			},
		},
		{
			name: "Put requires data",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"put", "things"},
				WantErr:    fmt.Errorf(`Argument "DATA" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"DATA\" requires at least 1 argument, got 0\n",
				//WantStderr: []string{"key not found"},
			},
		},
		{
			name: "Put works",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
				Args:       []string{"list"},
				WantStderr: "cache directory does not exist\n",
				WantErr:    fmt.Errorf("cache directory does not exist"),
			},
		},
		{
			name: "List works with no data",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
				Args:       []string{"delete"},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Delete requires valid key",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"delete", ".?,"},
				WantErr:    fmt.Errorf(`validation for "KEY" failed: [MatchesRegex] value ".?," doesn't match regex %q`, keyRegex),
				WantStderr: fmt.Sprintf("validation for \"KEY\" failed: [MatchesRegex] value \".?,\" doesn't match regex %q\n", keyRegex),
			},
		},
		{
			name: "Delete non-existant key",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
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
			etc: &command.ExecuteTestCase{
				Args:       []string{"setdir"},
				WantErr:    fmt.Errorf(`Argument "DIR" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"DIR\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "setdir requires an existing file",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"setdir", "uh"},
				WantErr:    fmt.Errorf("validation for \"DIR\" failed: [FileExists] file %q does not exist", command.FilepathAbs(t, "uh")),
				WantStderr: fmt.Sprintf("validation for \"DIR\" failed: [FileExists] file %q does not exist\n", command.FilepathAbs(t, "uh")),
			},
		},
		{
			name: "setdir doesn't allow files",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args:       []string{"setdir", "cache.go"},
				WantErr:    fmt.Errorf("validation for \"DIR\" failed: [IsDir] argument %q is a file", command.FilepathAbs(t, "cache.go")),
				WantStderr: fmt.Sprintf("validation for \"DIR\" failed: [IsDir] argument %q is a file\n", command.FilepathAbs(t, "cache.go")),
			},
		},
		{
			name: "setdir works",
			c:    NewTestCache(t),
			etc: &command.ExecuteTestCase{
				Args: []string{"setdir", "testing"},
			},
			wantC: &Cache{
				Dir: command.FilepathAbs(t, "testing"),
			},
			want: map[string]string{
				"empty.txt": "nothing to see here",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, p := range test.puts {
				Put(t, test.c, p.key, p.data)
			}
			if test.etc == nil {
				test.etc = &command.ExecuteTestCase{}
			}
			test.etc.Node = test.c.Node()
			test.etc.SkipDataCheck = true
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, test.wantC, test.c, cmpopts.IgnoreUnexported(Cache{}))

			if !test.skipFileCheck {
				if diff := cmp.Diff(test.want, fullCache(t, test.c), cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Execute(%v) resulted in incorrect cache (-want, +got):\n%s", test.etc.Args, diff)
				}
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
		ctc  *command.CompleteTestCase
	}{
		{
			name: "completes branches",
			ctc: &command.CompleteTestCase{
				Args: "cmd ",
				Want: []string{"delete", "get", "list", "put", "setdir"},
			},
		},
		{
			name: "completes for get",
			puts: puts,
			ctc: &command.CompleteTestCase{
				Args: "cmd get ",
				Want: []string{"hello", "things", "this"},
			},
		},
		{
			name: "completes for put",
			puts: puts,
			ctc: &command.CompleteTestCase{
				Args: "cmd put t",
				Want: []string{"things", "this"},
			},
		},
		{
			name: "completes for delete",
			puts: puts,
			ctc: &command.CompleteTestCase{
				Args: "cmd put thin",
				Want: []string{"things"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := NewTestCache(t)
			for _, p := range test.puts {
				Put(t, c, p.key, p.data)
			}
			if test.ctc == nil {
				test.ctc = &command.CompleteTestCase{}
			}
			test.ctc.Node = c.Node()
			test.ctc.SkipDataCheck = true
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestPutStruct(t *testing.T) {
	for _, test := range []struct {
		name       string
		key        string
		data       interface{}
		wantGet    string
		wantGetErr string
		wantGetOk  bool
		wantErr    string
	}{
		{
			name:       "put fails on empty",
			wantErr:    "invalid key format",
			wantGetErr: "failed to get file for key: invalid key format",
		}, {
			name:       "put fails on invalid key",
			key:        "abc-$",
			wantErr:    "invalid key format",
			wantGetErr: "failed to get file for key: invalid key format",
		}, {
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
			c := NewTestCache(t)

			err := c.PutStruct(test.key, test.data)
			if test.wantErr == "" && err != nil {
				t.Errorf("PutStruct(%s, %v) returned err %v; want nil", test.key, test.data, err)
			}
			if test.wantErr != "" && err == nil {
				t.Errorf("PutStruct(%s, %v) returned nil; want err %q", test.key, test.data, test.wantErr)
			}
			if test.wantErr != "" && err != nil && !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("PutStruct(%s, %v) returned err %q; want %q", test.key, test.data, err.Error(), test.wantErr)
			}

			stored, ok, err := c.Get(test.key)
			if (err == nil) != (test.wantErr == "") {
				t.Fatalf("PutStruct(%s, %v) returned get error (%v); want %v", test.key, test.data, err, test.wantErr)
			}
			if err != nil {
				if diff := cmp.Diff(test.wantGetErr, err.Error()); diff != "" {
					t.Errorf("PutStruct(%s, %v) returned wrong get error (-want, +got):\n%s", test.key, test.data, diff)
				}
			}
			if ok != test.wantGetOk {
				t.Errorf("PutStruct(%s, %v) returned ok=%v; want %v", test.key, test.data, ok, test.wantGetOk)
			}
			if diff := cmp.Diff(test.wantGet, stored); diff != "" {
				t.Errorf("PutStruct(%s, %v) produced diff:\n%s", test.key, test.data, diff)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	c := &Cache{}
	if c.Setup() != nil {
		t.Errorf("Cache returned unexpected setup: %v", c.Setup())
	}
}
