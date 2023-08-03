package command

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFilepathLen(t *testing.T) {
	for _, test := range []struct {
		pathParts []string
		want      int
	}{
		{
			[]string{"."},
			1,
		},
		{
			// `./`
			[]string{".", ""},
			2,
		},
		{
			[]string{".."},
			1,
		},
		{
			// `../..`
			[]string{"..", ".."},
			2,
		},
		{
			// `../../`
			[]string{"..", "..", ""},
			3,
		},
		{
			// `../../`
			[]string{"..", "..", "a"},
			3,
		},
		{
			[]string{"abc", "def", "a"},
			3,
		},
		{
			[]string{"abc", "def", ".", "a"},
			4,
		},
		{
			[]string{"abc", "def", ".", "a", "other", "."},
			6,
		},
		// Absolute paths
		{
			[]string{"", ""},
			1,
		},
		{
			[]string{"", "abc"},
			1,
		},
		{
			[]string{"", "abc", ".", "def"},
			3,
		},
		{
			[]string{"", "abc", "def", "a"},
			3,
		},
		{
			[]string{"", "abc", "def", ".", "a"},
			4,
		},
		{
			[]string{"", "abc", "def", ".", "a", "other", "."},
			6,
		},
	} {
		path := strings.Join(test.pathParts, string(os.PathSeparator))
		t.Run(fmt.Sprintf("filepathDist(%q)", filepath.Join(path)), func(t *testing.T) {
			if got := filepathDepth(path); got != test.want {
				t.Errorf("filepathDist(%q) returned %d; want %d", path, got, test.want)
			}
		})
	}
}

func TestStringCompleters(t *testing.T) {
	type testCase struct {
		name    string
		c       Completer[[]string]
		args    string
		want    []string
		wantErr error
	}
	for _, test := range []*testCase{
		{
			name: "nil completer returns nil",
		},
		{
			name: "nil completer returns nil",
			c:    SimpleCompleter[[]string](),
		},
		{
			name: "doesn't complete if case mismatch with upper",
			args: "cmd A",
			c:    SimpleCompleter[[]string]("abc", "Abc", "ABC"),
			want: []string{"ABC", "Abc"},
		},
		{
			name: "doesn't complete if case mismatch with lower",
			args: "cmd a",
			c:    SimpleCompleter[[]string]("abc", "Abc", "ABC"),
			want: []string{"abc"},
		},
		{
			name: "completes all cases if completer.CaseInsensitive and upper",
			args: "cmd A",
			c: AsCompleter[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completer.CaseInsensitive and lower",
			args: "cmd a",
			c: AsCompleter[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name:    "returns error",
			args:    "cmd A",
			wantErr: fmt.Errorf("bad news bears"),
			c: CompleterFromFunc(func([]string, *Data) (*Completion, error) {
				return &Completion{
					Suggestions: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
				}, fmt.Errorf("bad news bears")
			}),
		},
		{
			name: "completes only matching cases",
			args: "cmd A",
			c:    SimpleCompleter[[]string]("abc", "Abc", "ABC", "def", "Def", "DEF"),
			want: []string{"ABC", "Abc"},
		},
		{
			name: "completes all cases if completer.CaseInsensitive and upper",
			args: "cmd A",
			c: AsCompleter[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completer.CaseInsensitive and lower",
			args: "cmd a",
			c: AsCompleter[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "non-distinct completer returns duplicates",
			c:    SimpleCompleter[[]string]("first", "second", "third"),
			args: "cmd first second ",
			want: []string{"first", "second", "third"},
		},
		{
			name: "distinct completer does not return duplicates",
			c:    SimpleDistinctCompleter[[]string]("first", "second", "third"),
			args: "cmd first second ",
			want: []string{"third"},
		},
		// CompleterWithOpts test
		{
			name: "CompleterWithOpts works",
			c: CompleterWithOpts(
				SimpleCompleter[[]string]("one", "two", "three", "Ten", "Twelve"),
				&Completion{
					Distinct:            true,
					CaseInsensitiveSort: true,
					CaseInsensitive:     true,
				},
			),
			args: "cmd Ten T",
			want: []string{
				"three",
				"Twelve",
				"two",
			},
		},
		// Delimiter tests
		/*{
			name: "completer works with ",
			c:    SimpleDistinctCompleter("first", "sec ond", "sec over"),
			args: "first", "sec",
			want: []string{"third"},
		},*/
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := []ArgumentOption[[]string]{}
			if test.c != nil {
				opts = append(opts, test.c)
			}
			CompleteTest(t, &CompleteTestCase{
				Node:          SerialNodes(ListArg("test", testDesc, 2, 5, opts...)),
				Args:          test.args,
				Want:          test.want,
				WantErr:       test.wantErr,
				SkipDataCheck: true,
			})
		})
	}
}

func TestBoolCompleter(t *testing.T) {
	CompleteTest(t, &CompleteTestCase{
		Node: SerialNodes(Arg[bool]("test", testDesc, BoolCompleter())),
		Args: "cmd ",
		Want: []string{
			"0",
			"1",
			"F",
			"FALSE",
			"False",
			"T",
			"TRUE",
			"True",
			"f",
			"false",
			"t",
			"true",
		},
		SkipDataCheck: true,
	})
}

func TestParseAndComplete(t *testing.T) {
	for _, test := range []struct {
		name        string
		args        string
		ptArgs      []string
		cursorIdx   int
		suggestions []string
		wantData    *Data
		want        []string
		wantErr     error
	}{
		{
			name: "handles empty array",
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{""},
			}},
		},
		{
			name: "multi-word options",
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{""},
			}},
			want: []string{
				"Fifth",
				`First\ Choice`,
				`Fourth\ Option`,
				`Second\ Thing`,
				`Third\ One`,
			},
		},
		{
			name: "last argument matches a multi-word option",
			args: "cmd Fo",
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"Fo"},
			}},
			want: []string{
				`Fourth\ Option`,
			},
		},
		{
			name: "last argument matches multiple multi-word options",
			args: "cmd F",
			suggestions: []string{
				"First Choice",
				"Fourth Option",
				"Fifth",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"F"},
			}},
			want: []string{
				"Fifth",
				`First\ Choice`,
				`Fourth\ Option`,
			},
		},
		{
			name: "args with double quotes count as single option and ignore single quote",
			args: `cmd "Greg's One" `,
			suggestions: []string{
				"Greg's One",
				"Greg's Two",
				"Greg's Three",
				"Greg's Four",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"Greg's One", ""},
			}},
			want: []string{
				`Greg's\ Four`,
				`Greg's\ One`,
				`Greg's\ Three`,
				`Greg's\ Two`,
			},
		},
		{
			name: "args with single quotes count as single option and ignore double quote",
			args: `cmd 'Greg"s Other"s' `,
			suggestions: []string{
				`Greg"s One`,
				`Greg"s Two`,
				`Greg"s Three`,
				`Greg"s Four`,
			},
			want: []string{
				`Greg"s\ Four`,
				`Greg"s\ One`,
				`Greg"s\ Three`,
				`Greg"s\ Two`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{`Greg"s Other"s`, ""},
			}},
		},
		{
			name: "completes properly if ending on double quote",
			args: `cmd "`,
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				`"First Choice"`,
				`"Fourth Option"`,
				`"Second Thing"`,
				`"Third One"`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{""},
			}},
		},
		{
			name: "completes properly if ending on double quote with previous option",
			args: `cmd hello "`,
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				`"First Choice"`,
				`"Fourth Option"`,
				`"Second Thing"`,
				`"Third One"`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"hello", ""},
			}},
		},
		{
			name: "completes properly if ending on single quote",
			args: `cmd "First Choice" '`,
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				"'First Choice'",
				"'Fourth Option'",
				"'Second Thing'",
				"'Third One'",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"First Choice", ""},
			}},
		},
		{
			name: "completes with single quotes if unclosed single quote",
			args: `cmd "First Choice" 'F`,
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				"'First Choice'",
				"'Fourth Option'",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"First Choice", "F"},
			}},
		},
		{
			name: "last argument is just a double quote",
			args: `cmd "`,
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				`"First Choice"`,
				`"Fourth Option"`,
				`"Second Thing"`,
				`"Third One"`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{""},
			}},
		},
		{
			name: "last argument is a double quote with words",
			args: `cmd "F`,
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				`"First Choice"`,
				`"Fourth Option"`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"F"},
			}},
		},
		{
			name: "double quote with single quote",
			args: `cmd "Greg's T`,
			suggestions: []string{
				"Greg's One",
				"Greg's Two",
				"Greg's Three",
				"Greg's Four",
			},
			want: []string{
				`"Greg's Three"`,
				`"Greg's Two"`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"Greg's T"},
			}},
		},
		{
			name: "last argument is just a single quote",
			args: "cmd '",
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				"'First Choice'",
				"'Fourth Option'",
				"'Second Thing'",
				"'Third One'",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{""},
			}},
		},
		{
			name: "last argument is a single quote with words",
			args: "cmd 'F",
			suggestions: []string{
				"First Choice",
				"Second Thing",
				"Third One",
				"Fourth Option",
				"Fifth",
			},
			want: []string{
				"Fifth",
				"'First Choice'",
				"'Fourth Option'",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"F"},
			}},
		},
		{
			name: "single quote with double quote",
			args: `cmd 'Greg"s T`,
			suggestions: []string{
				`Greg"s One`,
				`Greg"s Two`,
				`Greg"s Three`,
				`Greg"s Four`,
			},
			want: []string{
				`'Greg"s Three'`,
				`'Greg"s Two'`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{`Greg"s T`},
			}},
		},
		{
			name: "end with space",
			args: "cmd Attempt\\ One\\ ",
			suggestions: []string{
				"Attempt One Two",
				"Attempt OneTwo",
				"Three",
				"Three Four",
				"ThreeFour",
			},
			want: []string{
				`Attempt\ One\ Two`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"Attempt One "},
			}},
		},
		{
			name: "single and double words",
			args: "cmd Three",
			suggestions: []string{
				"Attempt One Two",
				"Attempt OneTwo",
				"Three",
				"Three Four",
				"ThreeFour",
			},
			want: []string{
				"Three",
				`Three\ Four`,
				"ThreeFour",
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"Three"},
			}},
		},
		{
			name: "handles backslashes before spaces",
			args: "cmd First\\ O",
			suggestions: []string{
				"First Of",
				"First One",
				"Second Thing",
				"Third One",
			},
			want: []string{
				`First\ Of`,
				`First\ One`,
			},
			wantData: &Data{Values: map[string]interface{}{
				"sl": []string{"First O"},
			}},
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			c := SimpleCompleter[[]string](test.suggestions...)
			n := SerialNodes(ListArg[string]("sl", testDesc, 0, UnboundedList, c))

			data := &Data{}
			got, err := autocomplete(n, test.args, test.ptArgs, data)
			if test.wantErr == nil && err != nil {
				t.Errorf("autocomplete(%v) returned error (%v) when shouldn't have", test.args, err)
			}
			if test.wantErr != nil {
				if err == nil {
					t.Errorf("autocomplete(%v) returned no error when should have returned %v", test.args, test.wantErr)
				} else if diff := cmp.Diff(test.wantErr.Error(), err.Error()); diff != "" {
					t.Errorf("autocomplete(%v) returned unexpected error (-want, +got):\n%s", test.args, diff)
				}
			}
			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Autocomplete(%s) produced incorrect completions (-want, +got):\n%s", test.args, diff)
			}

			wantData := test.wantData
			if wantData == nil {
				wantData = &Data{}
			}
			if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty(), cmpopts.IgnoreUnexported(Data{})); diff != "" {
				t.Errorf("Autocomplete(%s) improperly parsed args (-want, +got)\n:%s", test.args, diff)
			}
		})
	}
}

type completerTest[T any] struct {
	name           string
	c              Completer[[]T]
	singleC        Completer[T]
	args           string
	ptArgs         []string
	setup          func(*testing.T)
	cleanup        func(*testing.T)
	absErr         error
	commandBranch  bool
	doesntAddSpace bool
	want           []string
	wantErr        error
	getwd          func() (string, error)
	filepathRelErr error
}

func (test *completerTest[T]) Name() string {
	return test.name
}

func (test *completerTest[T]) run(t *testing.T) {
	t.Run(test.name, func(t *testing.T) {
		fos := &FakeOS{
			NoSpace: test.doesntAddSpace,
		}
		if test.getwd != nil {
			StubValue(t, &osGetwd, test.getwd)
		}
		if test.filepathRelErr != nil {
			StubValue(t, &filepathRel, func(a, b string) (string, error) {
				return "", test.filepathRelErr
			})
		}
		if test.setup != nil {
			test.setup(t)
		}
		if test.cleanup != nil {
			t.Cleanup(func() { test.cleanup(t) })
		}
		StubValue(t, &filepathAbs, func(rel string) (string, error) {
			if test.absErr != nil {
				return "", test.absErr
			}
			return filepath.Abs(rel)
		})

		var got []string
		var err error
		if test.singleC != nil {
			got, err = Autocomplete(SerialNodes(Arg[T]("test", testDesc, test.singleC)), test.args, test.ptArgs, fos)
		} else {
			got, err = Autocomplete(SerialNodes(ListArg[T]("test", testDesc, 2, 5, test.c)), test.args, test.ptArgs, fos)
		}

		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("genericAutocomplete(%v) returned diff (-want, +got):\n%s", test.args, diff)
		}
		CmpError(t, fmt.Sprintf("genericAutocomplete(%v)", test.args), test.wantErr, err)
	})
}

type completerTestInterface interface {
	run(*testing.T)
	Name() string
}

func TestTypedCompleters(t *testing.T) {
	testdataContents := func() []string {
		return []string{
			".surprise",
			filepath.FromSlash("cases/"),
			filepath.FromSlash("dir1/"),
			filepath.FromSlash("dir2/"),
			filepath.FromSlash("dir3/"),
			filepath.FromSlash("dir4/"),
			"four.txt",
			"METADATA",
			filepath.FromSlash("metadata_/"),
			filepath.FromSlash("moreCases/"),
			"one.txt",
			"three.txt",
			"two.txt",
			" ",
		}
	}
	testdataContentsExcept := func(t *testing.T, except string) []string {
		var r []string
		var excluded bool
		except = filepath.FromSlash(except)
		for _, s := range testdataContents() {
			if s != except {
				excluded = true
				r = append(r, s)
			}
		}
		if !excluded {
			t.Errorf("%q was not excluded", except)
		}
		return r
	}
	for _, test := range []completerTestInterface{
		// Bool completer tests
		&completerTest[bool]{
			name:    "bool completer returns value",
			singleC: BoolCompleter(),
			want: []string{
				"0",
				"1",
				"F",
				"FALSE",
				"False",
				"T",
				"TRUE",
				"True",
				"f",
				"false",
				"t",
				"true",
			},
		},
		// String completer tests
		&completerTest[string]{
			name: "list completer returns nil",
			c:    SimpleCompleter[[]string](),
		},
		&completerTest[string]{
			name: "list completer returns list",
			c:    SimpleCompleter[[]string]("first", "second", "third"),
			want: []string{"first", "second", "third"},
		},
		// FileCompleter tests
		&completerTest[string]{
			name:    "file completer returns nil if failure completing current directory",
			c:       &FileCompleter[[]string]{},
			absErr:  fmt.Errorf("failed to fetch directory"),
			wantErr: fmt.Errorf("failed to get absolute filepath: failed to fetch directory"),
		},
		&completerTest[string]{
			name: "file completer returns files with file types and directories",
			singleC: &FileCompleter[string]{
				FileTypes: []string{".mod", ".sum"},
			},
			args: "cmd ",
			want: []string{
				filepath.FromSlash(".git/"),
				filepath.FromSlash("_testdata_symlink/"),
				filepath.FromSlash("cache/"),
				filepath.FromSlash("cmd/"),
				filepath.FromSlash("color/"),
				filepath.FromSlash("docs/"),
				filepath.FromSlash("glog/"),
				"go.mod",
				"go.sum",
				filepath.FromSlash("sourcerer/"),
				filepath.FromSlash("testdata/"),
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer handles empty directory",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/empty/",
			setup: func(t *testing.T) {
				if err := os.Mkdir("testdata/empty", 0644); err != nil {
					t.Fatalf("failed to create empty directory")
				}
			},
			cleanup: func(t *testing.T) {
				if err := os.RemoveAll("testdata/empty"); err != nil {
					t.Fatalf("failed to delete empty directory")
				}
			},
		},
		&completerTest[string]{
			name: "file completer works with string list arg, and autofills letters with space",
			c:    &FileCompleter[[]string]{},
			args: "cmd execu",
			want: []string{
				"execut",
				"execut_",
			},
		},
		&completerTest[string]{
			name:           "file completer works with string list arg, and autofills letters with no space",
			c:              &FileCompleter[[]string]{},
			args:           "cmd execu",
			doesntAddSpace: true,
			want: []string{
				"execut",
			},
		},
		&completerTest[string]{
			name: "file completer works when distinct",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd execute.go execute_test.go ex",
			want: []string{
				"executor.go",
			},
		},
		&completerTest[string]{
			name:    "file completer works with string arg",
			singleC: &FileCompleter[string]{},
			args:    "cmd execu",
			want: []string{
				"execut",
				"execut_",
			},
		},
		&completerTest[string]{
			name: "file completer returns nil if failure listing directory",
			c: &FileCompleter[[]string]{
				Directory: "does/not/exist",
			},
			wantErr: fmt.Errorf("failed to read dir: open %s: The system cannot find the path specified.", FilepathAbs(t, "does/not/exist")),
		},
		&completerTest[string]{
			name: "file completer returns files in the specified directory",
			c: &FileCompleter[[]string]{
				Directory: "testdata",
			},
			setup: func(t *testing.T) {
				if err := os.Mkdir("testdata/empty", 0644); err != nil {
					t.Fatalf("failed to create empty directory")
				}
			},
			cleanup: func(t *testing.T) {
				if err := os.RemoveAll("testdata/empty"); err != nil {
					t.Fatalf("failed to delete empty directory")
				}
			},
			want: []string{
				".surprise",
				filepath.FromSlash("cases/"),
				filepath.FromSlash("dir1/"),
				filepath.FromSlash("dir2/"),
				filepath.FromSlash("dir3/"),
				filepath.FromSlash("dir4/"),
				filepath.FromSlash("empty/"),
				"four.txt",
				"METADATA",
				filepath.FromSlash("metadata_/"),
				filepath.FromSlash("moreCases/"),
				"one.txt",
				"three.txt",
				"two.txt",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer returns files in the specified directory",
			c: &FileCompleter[[]string]{
				Directory: "testdata/dir1",
			},
			want: []string{
				"first.txt",
				"fourth.py",
				"second.py",
				"third.go",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer ignores things from IgnoreFunc",
			c: &FileCompleter[[]string]{
				Directory: "testdata/dir1",
				IgnoreFunc: func(fp, s string, data *Data) bool {
					return s == "third.go" || s == "other" || s == "fourth.py"
				},
			},
			want: []string{
				"first.txt",
				"second.py",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer returns files matching regex",
			c: &FileCompleter[[]string]{
				Directory: "testdata/dir1",
				Regexp:    regexp.MustCompile(".*.py$"),
			},
			want: []string{
				"fourth.py",
				"second.py",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer requires prefix",
			c: &FileCompleter[[]string]{
				Directory: "testdata/dir3",
			},
			args: "cmd th",
			want: []string{
				filepath.FromSlash("that/"),
				"this.txt",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer ignores directories",
			c: &FileCompleter[[]string]{
				Directory:         "testdata/dir2",
				IgnoreDirectories: true,
			},
			want: []string{
				"file",
				"file_",
			},
		},
		&completerTest[string]{
			name: "file completer ignores files",
			c: &FileCompleter[[]string]{
				Directory:   "testdata/dir2",
				IgnoreFiles: true,
			},
			want: []string{
				filepath.FromSlash("childC/"),
				filepath.FromSlash("childD/"),
				filepath.FromSlash("subA/"),
				filepath.FromSlash("subB/"),
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer completes to directory",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/dir1",
			want: []string{
				fmt.Sprintf("testdata/dir1%c", filepath.Separator),
				fmt.Sprintf("testdata/dir1%c_", filepath.Separator),
			},
		},
		&completerTest[string]{
			name:           "file completer completes to directory when no space",
			c:              &FileCompleter[[]string]{},
			args:           "cmd testdata/dir1",
			doesntAddSpace: true,
			want: []string{
				fmt.Sprintf("testdata/dir1%c", filepath.Separator),
			},
		},
		&completerTest[string]{
			name: "file completer completes to directory when starting dir specified",
			c: &FileCompleter[[]string]{
				Directory: "testdata",
			},
			args: "cmd dir1",
			want: []string{
				filepath.FromSlash("dir1/"),
				filepath.FromSlash("dir1/_"),
			},
		},
		&completerTest[string]{
			name: "file completer shows contents of directory when ending with a separator",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/dir1/",
			want: []string{
				"first.txt",
				"fourth.py",
				"second.py",
				"third.go",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer completes to directory when ending with a separator and when starting dir specified",
			c: &FileCompleter[[]string]{
				Directory: "testdata",
			},
			args: "cmd dir1/",
			want: []string{
				"first.txt",
				"fourth.py",
				"second.py",
				"third.go",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer only shows basenames when multiple options with different next letter",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/dir",
			want: []string{
				filepath.FromSlash("dir1/"),
				filepath.FromSlash("dir2/"),
				filepath.FromSlash("dir3/"),
				filepath.FromSlash("dir4/"),
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer shows full names when multiple options with same next letter",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/d",
			want: []string{
				"testdata/dir",
				"testdata/dir_",
			},
		},
		&completerTest[string]{
			name: "file completer only shows basenames when multiple options and starting dir",
			c: &FileCompleter[[]string]{
				Directory: "testdata/dir1",
			},
			args: "cmd f",
			want: []string{
				"first.txt",
				"fourth.py",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer handles directories with spaces",
			c:    &FileCompleter[[]string]{},
			args: `cmd testdata/dir4/folder\ wit`,
			want: []string{
				fmt.Sprintf(`testdata/dir4/folder\ with\ spaces%c`, filepath.Separator),
				fmt.Sprintf(`testdata/dir4/folder\ with\ spaces%c_`, filepath.Separator),
			},
		},
		&completerTest[string]{
			name: "file completer handles directories with spaces when same argument",
			c:    &FileCompleter[[]string]{},
			args: `cmd testdata/dir4/folder\ wit`,
			want: []string{
				fmt.Sprintf(`testdata/dir4/folder\ with\ spaces%c`, filepath.Separator),
				fmt.Sprintf(`testdata/dir4/folder\ with\ spaces%c_`, filepath.Separator),
			},
		},
		&completerTest[string]{
			name: "file completer can dive into folder with spaces",
			c:    &FileCompleter[[]string]{},
			args: `cmd testdata/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer can dive into folder with spaces when combined args",
			c:    &FileCompleter[[]string]{},
			args: `cmd testdata/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer fills in letters that are the same for all options",
			c:    &FileCompleter[[]string]{},
			args: `cmd testdata/dir4/fo`,
			want: []string{
				"testdata/dir4/folder",
				"testdata/dir4/folder_",
			},
		},
		&completerTest[string]{
			name:          "file completer doesn't get filtered out when part of a CommandBranch",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/dir",
			want: []string{
				filepath.FromSlash("dir1/"),
				filepath.FromSlash("dir2/"),
				filepath.FromSlash("dir3/"),
				filepath.FromSlash("dir4/"),
				" ",
			},
		},
		&completerTest[string]{
			name:          "file completer handles multiple options in directory",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/dir1/f",
			want: []string{
				"first.txt",
				"fourth.py",
				" ",
			},
		},
		&completerTest[string]{
			name:          "case insensitive gets letters autofilled",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/dI",
			want: []string{
				"testdata/dir",
				"testdata/dir_",
			},
		},
		&completerTest[string]{
			name:          "case insensitive recommends all without complete",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/DiR",
			want: []string{
				filepath.FromSlash("dir1/"),
				filepath.FromSlash("dir2/"),
				filepath.FromSlash("dir3/"),
				filepath.FromSlash("dir4/"),
				" ",
			},
		},
		&completerTest[string]{
			name:          "file completer ignores case",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/cases/abc",
			want: []string{
				"testdata/cases/abcde",
				"testdata/cases/abcde_",
			},
		},
		&completerTest[string]{
			name:          "file completer sorting ignores cases when no file",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/moreCases/",
			want: []string{
				"testdata/moreCases/QW_",
				"testdata/moreCases/QW__",
			},
		},
		&completerTest[string]{
			name:          "file completer sorting ignores cases when autofilling",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/moreCases/q",
			want: []string{
				"testdata/moreCases/qW_",
				"testdata/moreCases/qW__",
			},
		},
		&completerTest[string]{
			name:          "file completer sorting ignores cases when not autofilling",
			c:             &FileCompleter[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/moreCases/qW_t",
			want: []string{
				"qW_three.txt",
				"qw_TRES.txt",
				"Qw_two.txt",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer completes to case matched completion",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/meta",
			want: []string{
				"testdata/metadata",
				"testdata/metadata_",
			},
		},
		&completerTest[string]{
			name: "file completer completes to case matched completion",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/ME",
			want: []string{
				"testdata/METADATA",
				"testdata/METADATA_",
			},
		},
		&completerTest[string]{
			name: "file completer completes to something when no cases match",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/MeTa",
			want: []string{
				"testdata/METADATA",
				"testdata/METADATA_",
			},
		},
		&completerTest[string]{
			name: "file completer completes to case matched completion in current directory",
			c: &FileCompleter[[]string]{
				Directory: "testdata",
			},
			args: "cmd meta",
			want: []string{
				"metadata",
				"metadata_",
			},
		},
		&completerTest[string]{
			name: "file completer completes to case matched completion in current directory",
			c: &FileCompleter[[]string]{
				Directory: "testdata",
			},
			args: "cmd MET",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		&completerTest[string]{
			name: "file completer completes to something when no cases match in current directory",
			c: &FileCompleter[[]string]{
				Directory: "testdata",
			},
			args: "cmd meTA",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		&completerTest[string]{
			name: "file completer doesn't complete when matches a prefix",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/METADATA",
			want: []string{
				"METADATA",
				filepath.FromSlash("metadata_/"),
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer doesn't complete when matches a prefix file",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/metadata_/m",
			want: []string{
				"m1",
				"m2",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer returns complete match if distinct",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd testdata/metadata_/m1",
			want: []string{
				"testdata/metadata_/m1",
			},
		},
		// Distinct file completers.
		&completerTest[string]{
			name: "file completer returns repeats if not distinct",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/three.txt testdata/t",
			want: []string{"three.txt", "two.txt", " "},
		},
		&completerTest[string]{
			name: "file completer returns distinct",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd testdata/three.txt testdata/t",
			want: []string{"testdata/two.txt"},
		},
		&completerTest[string]{
			name: "file completer handles non with distinct",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd testdata/three.txt testdata/two.txt testdata/t",
		},
		&completerTest[string]{
			name: "file completer first level distinct partially completes",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd comp",
			want: []string{"completer", "completer_"},
		},
		&completerTest[string]{
			name: "file completer first level distinct returns all options",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd c",
			want: []string{
				"cache.go",
				filepath.FromSlash("cache/"),
				"cache_test.go",
				filepath.FromSlash("cmd/"),
				filepath.FromSlash("color/"),
				"command.go",
				"commandtest.go",
				"completer.go",
				"completer_test.go",
				"conditional.go",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer first level distinct completes partial",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd command.go comp",
			want: []string{
				"completer",
				"completer_",
			},
		},
		&completerTest[string]{
			name: "file completer first level distinct suggests remaining",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd completer.go c",
			want: []string{
				"cache.go",
				filepath.FromSlash("cache/"),
				"cache_test.go",
				filepath.FromSlash("cmd/"),
				filepath.FromSlash("color/"),
				"command.go",
				"commandtest.go",
				"completer_test.go",
				"conditional.go",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer first level distinct completes partial",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd completer.go completer_test.go c",
			want: []string{
				"cache.go",
				filepath.FromSlash("cache/"),
				"cache_test.go",
				filepath.FromSlash("cmd/"),
				filepath.FromSlash("color/"),
				"command.go",
				"commandtest.go",
				"conditional.go",
				" ",
			},
		},
		// Absolute file completion tests
		&completerTest[string]{
			name: "file completer works for absolute path",
			c:    &FileCompleter[[]string]{},
			setup: fakeReadDir("/",
				fakeFile("file1"),
				fakeFile("file2"),
				fakeDir("dirA"),
				fakeDir("dirB"),
			),
			args: fmt.Sprintf("cmd %s", "/"),
			want: []string{
				filepath.FromSlash("dirA/"),
				filepath.FromSlash("dirB/"),
				"file1",
				"file2",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer partial completes dir for absolute path",
			c:    &FileCompleter[[]string]{},
			args: fmt.Sprintf("cmd %sd", "/"),
			setup: fakeReadDir("/",
				fakeFile("file1"),
				fakeFile("file2"),
				fakeDir("dirA"),
				fakeDir("dirB"),
			),
			want: []string{
				"/dir",
				"/dir_",
			},
		},
		&completerTest[string]{
			name: "file completer partial completes file for absolute path",
			c:    &FileCompleter[[]string]{},
			args: fmt.Sprintf("cmd %sf", "/"),
			setup: fakeReadDir("/",
				fakeFile("file1"),
				fakeFile("file2"),
				fakeDir("dirA"),
				fakeDir("dirB"),
			),
			want: []string{
				"/file",
				"/file_",
			},
		},
		// Absolute file with specified directory completion tests
		&completerTest[string]{
			name: "file completer works for absolute path with relative dir",
			c: &FileCompleter[[]string]{
				Directory: "some/dir/ectory",
			},
			setup: fakeReadDir("/",
				fakeFile("file1"),
				fakeFile("file2"),
				fakeDir("dirA"),
				fakeDir("dirB"),
			),
			args: "cmd /",
			want: []string{
				filepath.FromSlash("dirA/"),
				filepath.FromSlash("dirB/"),
				"file1",
				"file2",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer partial completes dir for absolute path with relative dir",
			c: &FileCompleter[[]string]{
				Directory: "some/dir/ectory",
			},
			args: "cmd /d",
			setup: fakeReadDir("/",
				fakeFile("file1"),
				fakeFile("file2"),
				fakeDir("dirA"),
				fakeDir("dirB"),
			),
			want: []string{
				"/dir",
				"/dir_",
			},
		},
		&completerTest[string]{
			name: "file completer partial completes file for absolute path with relative dir",
			c: &FileCompleter[[]string]{
				Directory: "some/dir/ectory",
			},
			args: "cmd /f",
			setup: fakeReadDir("/",
				fakeFile("file1"),
				fakeFile("file2"),
				fakeDir("dirA"),
				fakeDir("dirB"),
			),
			want: []string{
				"/file",
				"/file_",
			},
		},
		// ExcludePwd FileCompleter tests
		&completerTest[string]{
			name: "file completer returns all files if cwd contains FileCompleter.directory",
			// cwd = "."; Directory = "./testdata"
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args: "cmd ",
			want: testdataContents(),
		},
		&completerTest[string]{
			name:  "file completer returns all files if cwd equals FileCompleter.directory",
			getwd: func() (string, error) { return FilepathAbs(t, "testdata"), nil },
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args: "cmd ",
			want: testdataContents(),
		},
		&completerTest[string]{
			name:  "file completer does not return directory if cwd is inside of FileCompleter.directory",
			getwd: func() (string, error) { return FilepathAbs(t, "testdata", "dir2"), nil },
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args: "cmd ",
			want: testdataContentsExcept(t, "dir2/"),
		},
		&completerTest[string]{
			name:  "file completer does not return directory if cwd is nested inside of FileCompleter.directory",
			getwd: func() (string, error) { return FilepathAbs(t, "testdata", "metadata_", "other"), nil },
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args: "cmd ",
			want: testdataContentsExcept(t, "metadata_/"),
		},
		&completerTest[string]{
			name:  "file completer returns all if cwd is sibling directory",
			getwd: func() (string, error) { return FilepathAbs(t, "cache"), nil },
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args: "cmd ",
			want: testdataContents(),
		},
		&completerTest[string]{
			name:  "file completer returns all if cwd matches non-dir file",
			getwd: func() (string, error) { return FilepathAbs(t, "testdata", "four.txt"), nil },
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args: "cmd ",
			want: testdataContents(),
		},
		&completerTest[string]{
			name:  "file completer returns all if directory is substring of cwd",
			getwd: func() (string, error) { return FilepathAbs(t, "testdatabc"), nil },
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args: "cmd ",
			want: testdataContents(),
		},
		&completerTest[string]{
			name:  "file completer does not complete if failed to get pwd",
			getwd: func() (string, error) { return FilepathAbs(t, "", "metadata_", "other"), fmt.Errorf("ugh a bug") },
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args:    "cmd ",
			wantErr: fmt.Errorf("failed to get current working directory: ugh a bug"),
		},
		&completerTest[string]{
			name:           "file completer does not complete if failed to get relative filepath",
			getwd:          func() (string, error) { return FilepathAbs(t, "", "metadata_", "other"), nil },
			filepathRelErr: fmt.Errorf("unrelated"),
			singleC: &FileCompleter[string]{
				Directory:  "testdata",
				ExcludePwd: true,
			},
			args:    "cmd ",
			wantErr: fmt.Errorf("failed to get relative directory: unrelated"),
		},
		// MaxDepth FileCompleter tests
		&completerTest[string]{
			name: "file completer with negative max depth returns regular suggestions with slashes",
			singleC: &FileCompleter[string]{
				FileTypes: []string{".mod", ".sum"},
				MaxDepth:  -1,
			},
			args: "cmd ",
			want: []string{
				filepath.FromSlash(".git/"),
				filepath.FromSlash("_testdata_symlink/"),
				filepath.FromSlash("cache/"),
				filepath.FromSlash("cmd/"),
				filepath.FromSlash("color/"),
				filepath.FromSlash("docs/"),
				filepath.FromSlash("glog/"),
				"go.mod",
				"go.sum",
				filepath.FromSlash("sourcerer/"),
				filepath.FromSlash("testdata/"),
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer with 2 max depth returns regular suggestions with slashes",
			singleC: &FileCompleter[string]{
				FileTypes: []string{".mod", ".sum"},
				MaxDepth:  2,
			},
			args: "cmd ",
			want: []string{
				filepath.FromSlash(".git/"),
				filepath.FromSlash("_testdata_symlink/"),
				filepath.FromSlash("cache/"),
				filepath.FromSlash("cmd/"),
				filepath.FromSlash("color/"),
				filepath.FromSlash("docs/"),
				filepath.FromSlash("glog/"),
				"go.mod",
				"go.sum",
				filepath.FromSlash("sourcerer/"),
				filepath.FromSlash("testdata/"),
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 1 removes slashes from dirs",
			singleC: &FileCompleter[string]{
				FileTypes: []string{".mod", ".sum"},
				MaxDepth:  1,
			},
			args: "cmd ",
			want: []string{
				".git",
				"_testdata_symlink",
				"cache",
				"cmd",
				"color",
				"docs",
				"glog",
				"go.mod",
				"go.sum",
				"sourcerer",
				"testdata",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 1 completes partial",
			singleC: &FileCompleter[string]{
				MaxDepth:  1,
				FileTypes: []string{".mod", ".sum"},
			},
			args: "cmd go",
			want: []string{
				"go.",
				"go._",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 2 completes partial",
			singleC: &FileCompleter[string]{
				MaxDepth:  2,
				FileTypes: []string{".mod", ".sum"},
			},
			args: "cmd go",
			want: []string{
				"go.",
				"go._",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 1 completes fully",
			singleC: &FileCompleter[string]{
				MaxDepth:  1,
				FileTypes: []string{".mod", ".sum"},
			},
			args: "cmd te",
			want: []string{
				"testdata",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 2 completes with slash",
			singleC: &FileCompleter[string]{
				MaxDepth:  2,
				FileTypes: []string{".mod", ".sum"},
			},
			args: "cmd te",
			want: []string{
				filepath.FromSlash("testdata/"),
				filepath.FromSlash("testdata/_"),
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 1 completes sub directories still (and without slashes)",
			singleC: &FileCompleter[string]{
				MaxDepth: 1,
			},
			args: "cmd testdata/",
			want: []string{
				".surprise",
				"cases",
				"dir1",
				"dir2",
				"dir3",
				"dir4",
				"four.txt",
				"METADATA",
				"metadata_",
				"moreCases",
				"one.txt",
				"three.txt",
				"two.txt",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 2 completes sub directories without slashes",
			singleC: &FileCompleter[string]{
				MaxDepth: 2,
			},
			args: "cmd testdata/",
			want: []string{
				".surprise",
				"cases",
				"dir1",
				"dir2",
				"dir3",
				"dir4",
				"four.txt",
				"METADATA",
				"metadata_",
				"moreCases",
				"one.txt",
				"three.txt",
				"two.txt",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 1 completes partial",
			singleC: &FileCompleter[string]{
				MaxDepth: 1,
			},
			args: "cmd testdata/d",
			want: []string{
				"testdata/dir",
				"testdata/dir_",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 2 completes partial",
			singleC: &FileCompleter[string]{
				MaxDepth: 2,
			},
			args: "cmd testdata/d",
			want: []string{
				"testdata/dir",
				"testdata/dir_",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 1 completes sub directory fully with no slash",
			singleC: &FileCompleter[string]{
				MaxDepth: 1,
			},
			args: "cmd testdata/mo",
			want: []string{
				"testdata/moreCases",
			},
		},
		&completerTest[string]{
			name: "file completer with max depth 2 completes sub directory fully with no slash",
			singleC: &FileCompleter[string]{
				MaxDepth: 2,
			},
			args: "cmd testdata/mo",
			want: []string{
				"testdata/moreCases",
			},
		},
		/* Useful for commenting out tests. */
	} {
		test.run(t)
	}
}

func fakeDir(name string) os.FileInfo {
	return &fakeFileInfo{name, true}
}

func fakeFile(name string) os.FileInfo {
	return &fakeFileInfo{name, false}
}

func fakeReadDir(wantDir string, files ...fs.FileInfo) func(t *testing.T) {
	return func(t *testing.T) {
		StubValue(t, &ioutilReadDir, func(dir string) ([]fs.FileInfo, error) {
			if diff := cmp.Diff(wantDir, dir); diff != "" {
				t.Fatalf("ioutil.ReadDir received incorrect argument (-want, +got):\n%s", diff)
			}
			return files, nil
		})
	}
}

type fakeFileInfo struct {
	name  string
	isDir bool
}

func (fi fakeFileInfo) Name() string       { return fi.name }
func (fi fakeFileInfo) Size() int64        { return 0 }
func (fi fakeFileInfo) Mode() os.FileMode  { return 0 }
func (fi fakeFileInfo) ModTime() time.Time { return time.Now() }
func (fi fakeFileInfo) IsDir() bool        { return fi.isDir }
func (fi fakeFileInfo) Sys() interface{}   { return nil }
