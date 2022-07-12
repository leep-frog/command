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

// TODO: do something similar to TestBoolFlag so we can
// run tests on other parameterized types.
func TestCompletors(t *testing.T) {
	type testCase struct {
		name    string
		c       Completor[[]string]
		args    string
		want    []string
		wantErr error
	}
	for _, test := range []*testCase{
		{
			name: "nil completor returns nil",
		},
		{
			name: "nil completor returns nil",
			c:    SimpleCompletor[[]string](),
		},
		{
			name: "doesn't complete if case mismatch with upper",
			args: "cmd A",
			c:    SimpleCompletor[[]string]("abc", "Abc", "ABC"),
			want: []string{"ABC", "Abc"},
		},
		{
			name: "doesn't complete if case mismatch with lower",
			args: "cmd a",
			c:    SimpleCompletor[[]string]("abc", "Abc", "ABC"),
			want: []string{"abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and upper",
			args: "cmd A",
			c: AsCompletor[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and lower",
			args: "cmd a",
			c: AsCompletor[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name:    "returns error",
			args:    "cmd A",
			wantErr: fmt.Errorf("bad news bears"),
			c: CompletorFromFunc(func([]string, *Data) (*Completion, error) {
				return &Completion{
					Suggestions: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
				}, fmt.Errorf("bad news bears")
			}),
		},
		{
			name: "completes only matching cases",
			args: "cmd A",
			c:    SimpleCompletor[[]string]("abc", "Abc", "ABC", "def", "Def", "DEF"),
			want: []string{"ABC", "Abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and upper",
			args: "cmd A",
			c: AsCompletor[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and lower",
			args: "cmd a",
			c: AsCompletor[[]string](&Completion{
				CaseInsensitive: true,
				Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
			}),
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "non-distinct completor returns duplicates",
			c:    SimpleCompletor[[]string]("first", "second", "third"),
			args: "cmd first second ",
			want: []string{"first", "second", "third"},
		},
		{
			name: "distinct completor does not return duplicates",
			c:    SimpleDistinctCompletor[[]string]("first", "second", "third"),
			args: "cmd first second ",
			want: []string{"third"},
		},
		// Delimiter tests
		/*{
			name: "completor works with ",
			c:    SimpleDistinctCompletor("first", "sec ond", "sec over"),
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

func TestBoolCompletor(t *testing.T) {
	CompleteTest(t, &CompleteTestCase{
		Node: SerialNodes(Arg[bool]("test", testDesc, BoolCompletor())),
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
			c := SimpleCompletor[[]string](test.suggestions...)
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

type completorTest[T any] struct {
	name          string
	c             Completor[[]T]
	singleC       Completor[T]
	args          string
	ptArgs        []string
	setup         func(*testing.T)
	cleanup       func(*testing.T)
	absErr        error
	commandBranch bool
	want          []string
}

func (test *completorTest[T]) Name() string {
	return test.name
}

func (test *completorTest[T]) run(t *testing.T) {
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

type completorTestInterface interface {
	run(*testing.T)
	Name() string
}

func TestTypedCompletors(t *testing.T) {
	for _, test := range []completorTestInterface{
		// Bool completor tests
		&completorTest[bool]{
			name:    "bool completor returns value",
			singleC: BoolCompletor(),
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
		// String completor tests
		&completorTest[string]{
			name: "list completor returns nil",
			c:    SimpleCompletor[[]string](),
		},
		&completorTest[string]{
			name: "list completor returns list",
			c:    SimpleCompletor[[]string]("first", "second", "third"),
			want: []string{"first", "second", "third"},
		},
		// FileCompletor tests
		&completorTest[string]{
			name:   "file completor returns nil if failure completing current directory",
			c:      &FileCompletor[[]string]{},
			absErr: fmt.Errorf("failed to fetch directory"),
		},
		&completorTest[string]{
			name: "file completor returns files with file types and directories",
			singleC: &FileCompletor[string]{
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
		&completorTest[string]{
			name: "file completor handles empty directory",
			c:    &FileCompletor[[]string]{},
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
		&completorTest[string]{
			name: "file completor works with string list arg",
			c:    &FileCompletor[[]string]{},
			args: "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		&completorTest[string]{
			name: "file completor works when distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd execute.go execu",
			want: []string{
				"execute_test.go",
			},
		},
		&completorTest[string]{
			name:    "file completor works with string arg",
			singleC: &FileCompletor[string]{},
			args:    "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		&completorTest[string]{
			name: "file completor returns nil if failure listing directory",
			c: &FileCompletor[[]string]{
				Directory: "does/not/exist",
			},
		},
		&completorTest[string]{
			name: "file completor returns files in the specified directory",
			c: &FileCompletor[[]string]{
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
		&completorTest[string]{
			name: "file completor returns files in the specified directory",
			c: &FileCompletor[[]string]{
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
		&completorTest[string]{
			name: "file completor ignores things from IgnoreFunc",
			c: &FileCompletor[[]string]{
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
		&completorTest[string]{
			name: "file completor returns files matching regex",
			c: &FileCompletor[[]string]{
				Directory: "testdata/dir1",
				Regexp:    regexp.MustCompile(".*.py$"),
			},
			want: []string{
				"fourth.py",
				"second.py",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor requires prefix",
			c: &FileCompletor[[]string]{
				Directory: "testdata/dir3",
			},
			args: "cmd th",
			want: []string{
				"that/",
				"this.txt",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor ignores directories",
			c: &FileCompletor[[]string]{
				Directory:         "testdata/dir2",
				IgnoreDirectories: true,
			},
			want: []string{
				"file",
				"file_",
			},
		},
		&completorTest[string]{
			name: "file completor ignores files",
			c: &FileCompletor[[]string]{
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
		&completorTest[string]{
			name: "file completor completes to directory",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/dir1",
			want: []string{
				"testdata/dir1/",
				"testdata/dir1/_",
			},
		},
		&completorTest[string]{
			name: "file completor completes to directory when starting dir specified",
			c: &FileCompletor[[]string]{
				Directory: "testdata",
			},
			args: "cmd dir1",
			want: []string{
				"dir1/",
				"dir1/_",
			},
		},
		&completorTest[string]{
			name: "file completor shows contents of directory when ending with a separator",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/dir1/",
			want: []string{
				"first.txt",
				"fourth.py",
				"second.py",
				"third.go",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor completes to directory when ending with a separator and when starting dir specified",
			c: &FileCompletor[[]string]{
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
		&completorTest[string]{
			name: "file completor only shows basenames when multiple options with different next letter",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/dir",
			want: []string{
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor shows full names when multiple options with same next letter",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/d",
			want: []string{
				"testdata/dir",
				"testdata/dir_",
			},
		},
		&completorTest[string]{
			name: "file completor only shows basenames when multiple options and starting dir",
			c: &FileCompletor[[]string]{
				Directory: "testdata/dir1",
			},
			args: "cmd f",
			want: []string{
				"first.txt",
				"fourth.py",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor handles directories with spaces",
			c:    &FileCompletor[[]string]{},
			args: `cmd testdata/dir4/folder\ wit`,
			want: []string{
				`testdata/dir4/folder\ with\ spaces/`,
				`testdata/dir4/folder\ with\ spaces/_`,
			},
		},
		&completorTest[string]{
			name: "file completor handles directories with spaces when same argument",
			c:    &FileCompletor[[]string]{},
			args: `cmd testdata/dir4/folder\ wit`,
			want: []string{
				`testdata/dir4/folder\ with\ spaces/`,
				`testdata/dir4/folder\ with\ spaces/_`,
			},
		},
		&completorTest[string]{
			name: "file completor can dive into folder with spaces",
			c:    &FileCompletor[[]string]{},
			args: `cmd testdata/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor can dive into folder with spaces when combined args",
			c:    &FileCompletor[[]string]{},
			args: `cmd testdata/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		&completorTest[string]{
			name: "autocomplete fills in letters that are the same for all options",
			c:    &FileCompletor[[]string]{},
			args: `cmd testdata/dir4/fo`,
			want: []string{
				"testdata/dir4/folder",
				"testdata/dir4/folder_",
			},
		},
		&completorTest[string]{
			name:          "file completor doesn't get filtered out when part of a CommandBranch",
			c:             &FileCompletor[[]string]{},
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
		&completorTest[string]{
			name:          "file completor handles multiple options in directory",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/dir1/f",
			want: []string{
				"first.txt",
				"fourth.py",
				" ",
			},
		},
		&completorTest[string]{
			name:          "case insensitive gets letters autofilled",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/dI",
			want: []string{
				"testdata/dir",
				"testdata/dir_",
			},
		},
		&completorTest[string]{
			name:          "case insensitive recommends all without complete",
			c:             &FileCompletor[[]string]{},
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
		&completorTest[string]{
			name:          "file completor ignores case",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/cases/abc",
			want: []string{
				"testdata/cases/abcde",
				"testdata/cases/abcde_",
			},
		},
		&completorTest[string]{
			name:          "file completor sorting ignores cases when no file",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/moreCases/",
			want: []string{
				"testdata/moreCases/QW_",
				"testdata/moreCases/QW__",
			},
		},
		&completorTest[string]{
			name:          "file completor sorting ignores cases when autofilling",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/moreCases/q",
			want: []string{
				"testdata/moreCases/qW_",
				"testdata/moreCases/qW__",
			},
		},
		&completorTest[string]{
			name:          "file completor sorting ignores cases when not autofilling",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testdata/moreCases/qW_t",
			want: []string{
				"qW_three.txt",
				"qw_TRES.txt",
				"Qw_two.txt",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor completes to case matched completion",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/meta",
			want: []string{
				"testdata/metadata",
				"testdata/metadata_",
			},
		},
		&completorTest[string]{
			name: "file completor completes to case matched completion",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/ME",
			want: []string{
				"testdata/METADATA",
				"testdata/METADATA_",
			},
		},
		&completorTest[string]{
			name: "file completor completes to something when no cases match",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/MeTa",
			want: []string{
				"testdata/METADATA",
				"testdata/METADATA_",
			},
		},
		&completorTest[string]{
			name: "file completor completes to case matched completion in current directory",
			c: &FileCompletor[[]string]{
				Directory: "testdata",
			},
			args: "cmd meta",
			want: []string{
				"metadata",
				"metadata_",
			},
		},
		&completorTest[string]{
			name: "file completor completes to case matched completion in current directory",
			c: &FileCompletor[[]string]{
				Directory: "testdata",
			},
			args: "cmd MET",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		&completorTest[string]{
			name: "file completor completes to something when no cases match in current directory",
			c: &FileCompletor[[]string]{
				Directory: "testdata",
			},
			args: "cmd meTA",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		&completorTest[string]{
			name: "file completor doesn't complete when matches a prefix",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/METADATA",
			want: []string{
				"METADATA",
				"metadata_/",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor doesn't complete when matches a prefix file",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/metadata_/m",
			want: []string{
				"m1",
				"m2",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor returns complete match if distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd testdata/metadata_/m1",
			want: []string{
				"testdata/metadata_/m1",
			},
		},
		// Distinct file completors.
		&completorTest[string]{
			name: "file completor returns repeats if not distinct",
			c:    &FileCompletor[[]string]{},
			args: "cmd testdata/three.txt testdata/t",
			want: []string{"three.txt", "two.txt", " "},
		},
		&completorTest[string]{
			name: "file completor returns distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd testdata/three.txt testdata/t",
			want: []string{"testdata/two.txt"},
		},
		&completorTest[string]{
			name: "file completor handles non with distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd testdata/three.txt testdata/two.txt testdata/t",
		},
		&completorTest[string]{
			name: "file completor first level distinct partially completes",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd comp",
			want: []string{"completor", "completor_"},
		},
		&completorTest[string]{
			name: "file completor first level distinct returns all options",
			c: &FileCompletor[[]string]{
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
				"completor.go",
				"completor_test.go",
				"custom_nodes.go",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor first level distinct completes partial",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd custom_nodes.go comp",
			want: []string{
				"completor",
				"completor_",
			},
		},
		&completorTest[string]{
			name: "file completor first level distinct suggests remaining",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd completor.go c",
			want: []string{
				"cache.go",
				"cache/",
				"cache_test.go",
				"cmd/",
				"color/",
				"commandtest.go",
				"completor_test.go",
				"custom_nodes.go",
				" ",
			},
		},
		&completorTest[string]{
			name: "file completor first level distinct completes partial",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd completor.go completor_test.go c",
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
		&completorTest[string]{
			name: "file completor works for absolute path",
			c:    &FileCompletor[[]string]{},
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
		&completorTest[string]{
			name: "file completor partial completes dir for absolute path",
			c:    &FileCompletor[[]string]{},
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
		&completorTest[string]{
			name: "file completor partial completes file for absolute path",
			c:    &FileCompletor[[]string]{},
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
		&completorTest[string]{
			name: "file completor works for absolute path with relative dir",
			c: &FileCompletor[[]string]{
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
		&completorTest[string]{
			name: "file completor partial completes dir for absolute path with relative dir",
			c: &FileCompletor[[]string]{
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
		&completorTest[string]{
			name: "file completor partial completes file for absolute path with relative dir",
			c: &FileCompletor[[]string]{
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
