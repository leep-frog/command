package command

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

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
			opts := []ArgOpt[[]string]{}
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
	name          string
	c             Completer[[]T]
	singleC       Completer[T]
	args          string
	ptArgs        []string
	setup         func(*testing.T)
	cleanup       func(*testing.T)
	absErr        error
	commandBranch bool
	want          []string
}

func (test *completerTest[T]) Name() string {
	return test.name
}

func (test *completerTest[T]) run(t *testing.T) {
	t.Run(test.name, func(t *testing.T) {
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
		if test.singleC != nil {
			got = Autocomplete(SerialNodes(Arg[T]("test", testDesc, test.singleC)), test.args, test.ptArgs)
		} else {
			got = Autocomplete(SerialNodes(ListArg[T]("test", testDesc, 2, 5, test.c)), test.args, test.ptArgs)
		}

		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("genericAutocomplete(%v) returned diff (-want, +got):\n%s", test.args, diff)
		}
	})
}

type completerTestInterface interface {
	run(*testing.T)
	Name() string
}

func TestTypedCompleters(t *testing.T) {
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
			name:   "file completer returns nil if failure completing current directory",
			c:      &FileCompleter[[]string]{},
			absErr: fmt.Errorf("failed to fetch directory"),
		},
		&completerTest[string]{
			name: "file completer returns files with file types and directories",
			singleC: &FileCompleter[string]{
				FileTypes: []string{".mod", ".sum"},
			},
			args: "cmd ",
			want: []string{
				".git/",
				"_testdata_symlink/",
				"cache/",
				"cmd/",
				"color/",
				"docs/",
				"go.mod",
				"go.sum",
				"sourcerer/",
				"testdata/",
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
			name: "file completer works with string list arg",
			c:    &FileCompleter[[]string]{},
			args: "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		&completerTest[string]{
			name: "file completer works when distinct",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd execute.go execu",
			want: []string{
				"execute_test.go",
			},
		},
		&completerTest[string]{
			name:    "file completer works with string arg",
			singleC: &FileCompleter[string]{},
			args:    "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		&completerTest[string]{
			name: "file completer returns nil if failure listing directory",
			c: &FileCompleter[[]string]{
				Directory: "does/not/exist",
			},
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
				"cases/",
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
				"empty/",
				"four.txt",
				"METADATA",
				"metadata_/",
				"moreCases/",
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
				"that/",
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
				"childC/",
				"childD/",
				"subA/",
				"subB/",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer completes to directory",
			c:    &FileCompleter[[]string]{},
			args: "cmd testdata/dir1",
			want: []string{
				"testdata/dir1/",
				"testdata/dir1/_",
			},
		},
		&completerTest[string]{
			name: "file completer completes to directory when starting dir specified",
			c: &FileCompleter[[]string]{
				Directory: "testdata",
			},
			args: "cmd dir1",
			want: []string{
				"dir1/",
				"dir1/_",
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
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
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
				`testdata/dir4/folder\ with\ spaces/`,
				`testdata/dir4/folder\ with\ spaces/_`,
			},
		},
		&completerTest[string]{
			name: "file completer handles directories with spaces when same argument",
			c:    &FileCompleter[[]string]{},
			args: `cmd testdata/dir4/folder\ wit`,
			want: []string{
				`testdata/dir4/folder\ with\ spaces/`,
				`testdata/dir4/folder\ with\ spaces/_`,
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
			name: "autocomplete fills in letters that are the same for all options",
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
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
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
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
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
				"metadata_/",
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
				"cache/",
				"cache_test.go",
				"cmd/",
				"color/",
				"commandtest.go",
				"completer.go",
				"completer_test.go",
				"custom_nodes.go",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer first level distinct completes partial",
			c: &FileCompleter[[]string]{
				Distinct: true,
			},
			args: "cmd custom_nodes.go comp",
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
				"cache/",
				"cache_test.go",
				"cmd/",
				"color/",
				"commandtest.go",
				"completer_test.go",
				"custom_nodes.go",
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
				"cache/",
				"cache_test.go",
				"cmd/",
				"color/",
				"commandtest.go",
				"custom_nodes.go",
				" ",
			},
		},
		// Absolute file completion tests
		&completerTest[string]{
			name: "file completer works for absolute path",
			c:    &FileCompleter[[]string]{},
			setup: fakeReadDir(cmdos.absStart(),
				fakeFile("file1"),
				fakeFile("file2"),
				fakeDir("dirA"),
				fakeDir("dirB"),
			),
			args: fmt.Sprintf("cmd %s", cmdos.absStart()),
			want: []string{
				"dirA/",
				"dirB/",
				"file1",
				"file2",
				" ",
			},
		},
		&completerTest[string]{
			name: "file completer partial completes dir for absolute path",
			c:    &FileCompleter[[]string]{},
			args: fmt.Sprintf("cmd %sd", cmdos.absStart()),
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
			args: fmt.Sprintf("cmd %sf", cmdos.absStart()),
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
				"dirA/",
				"dirB/",
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
