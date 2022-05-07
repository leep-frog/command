package command

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

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
			if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty()); diff != "" {
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

func (test *completorTest[T]) run(t *testing.T) {
	t.Run(test.name, func(t *testing.T) {
		if test.setup != nil {
			test.setup(t)
		}
		if test.cleanup != nil {
			defer test.cleanup(t)
		}
		oldAbs := filepathAbs
		filepathAbs = func(rel string) (string, error) {
			if test.absErr != nil {
				return "", test.absErr
			}
			return filepath.Abs(rel)
		}
		defer func() { filepathAbs = oldAbs }()

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

var (
	boolCompletorCases = []*completorTest[bool]{
		// BoolCompletor
		{
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
	}
	stringCompletorCases = []*completorTest[string]{
		{
			name: "list completor returns nil",
			c:    SimpleCompletor[[]string](),
		},
		{
			name: "list completor returns list",
			c:    SimpleCompletor[[]string]("first", "second", "third"),
			want: []string{"first", "second", "third"},
		},
		// FileCompletor tests
		{
			name:   "file completor returns nil if failure completing current directory",
			c:      &FileCompletor[[]string]{},
			absErr: fmt.Errorf("failed to fetch directory"),
		},
		{
			name: "file completor returns files with file types and directories",
			singleC: &FileCompletor[string]{
				FileTypes: []string{".mod", ".sum"},
			},
			args: "cmd ",
			want: []string{
				".git/",
				"cache/",
				"cmd/",
				"color/",
				"examples/",
				"go.mod",
				"go.sum",
				"sourcerer/",
				"testing/",
				" ",
			},
		},
		{
			name: "file completor handles empty directory",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/empty/",
			setup: func(t *testing.T) {
				if err := os.Mkdir("testing/empty", 0644); err != nil {
					t.Fatalf("failed to create empty directory")
				}
			},
			cleanup: func(t *testing.T) {
				if err := os.RemoveAll("testing/empty"); err != nil {
					t.Fatalf("failed to delete empty directory")
				}
			},
		},
		{
			name: "file completor works with string list arg",
			c:    &FileCompletor[[]string]{},
			args: "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		{
			name: "file completor works when distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd execute.go execu",
			want: []string{
				"execute_test.go",
			},
		},
		{
			name:    "file completor works with string arg",
			singleC: &FileCompletor[string]{},
			args:    "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		{
			name: "file completor returns nil if failure listing directory",
			c: &FileCompletor[[]string]{
				Directory: "does/not/exist",
			},
		},
		{
			name: "file completor returns files in the specified directory",
			c: &FileCompletor[[]string]{
				Directory: "testing",
			},
			setup: func(t *testing.T) {
				if err := os.Mkdir("testing/empty", 0644); err != nil {
					t.Fatalf("failed to create empty directory")
				}
			},
			cleanup: func(t *testing.T) {
				if err := os.RemoveAll("testing/empty"); err != nil {
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
		{
			name: "file completor returns files in the specified directory",
			c: &FileCompletor[[]string]{
				Directory: "testing/dir1",
			},
			want: []string{
				"first.txt",
				"fourth.py",
				"second.py",
				"third.go",
				" ",
			},
		},
		{
			name: "file completor ignores things from IgnoreFunc",
			c: &FileCompletor[[]string]{
				Directory: "testing/dir1",
				IgnoreFunc: func([]string, *Data) []string {
					return []string{"third.go", "other", "fourth.py"}
				},
			},
			want: []string{
				"first.txt",
				"second.py",
				" ",
			},
		},
		{
			name: "file completor returns files matching regex",
			c: &FileCompletor[[]string]{
				Directory: "testing/dir1",
				Regexp:    regexp.MustCompile(".*.py$"),
			},
			want: []string{
				"fourth.py",
				"second.py",
				" ",
			},
		},
		{
			name: "file completor requires prefix",
			c: &FileCompletor[[]string]{
				Directory: "testing/dir3",
			},
			args: "cmd th",
			want: []string{
				"that/",
				"this.txt",
				" ",
			},
		},
		{
			name: "file completor ignores directories",
			c: &FileCompletor[[]string]{
				Directory:         "testing/dir2",
				IgnoreDirectories: true,
			},
			want: []string{
				"file",
				"file_",
			},
		},
		{
			name: "file completor ignores files",
			c: &FileCompletor[[]string]{
				Directory:   "testing/dir2",
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
		{
			name: "file completor completes to directory",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/dir1",
			want: []string{
				"testing/dir1/",
				"testing/dir1/_",
			},
		},
		{
			name: "file completor completes to directory when starting dir specified",
			c: &FileCompletor[[]string]{
				Directory: "testing",
			},
			args: "cmd dir1",
			want: []string{
				"dir1/",
				"dir1/_",
			},
		},
		{
			name: "file completor shows contents of directory when ending with a separator",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/dir1/",
			want: []string{
				"first.txt",
				"fourth.py",
				"second.py",
				"third.go",
				" ",
			},
		},
		{
			name: "file completor completes to directory when ending with a separator and when starting dir specified",
			c: &FileCompletor[[]string]{
				Directory: "testing",
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
		{
			name: "file completor only shows basenames when multiple options with different next letter",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/dir",
			want: []string{
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
				" ",
			},
		},
		{
			name: "file completor shows full names when multiple options with same next letter",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/d",
			want: []string{
				"testing/dir",
				"testing/dir_",
			},
		},
		{
			name: "file completor only shows basenames when multiple options and starting dir",
			c: &FileCompletor[[]string]{
				Directory: "testing/dir1",
			},
			args: "cmd f",
			want: []string{
				"first.txt",
				"fourth.py",
				" ",
			},
		},
		{
			name: "file completor handles directories with spaces",
			c:    &FileCompletor[[]string]{},
			args: `cmd testing/dir4/folder\ wit`,
			want: []string{
				`testing/dir4/folder\ with\ spaces/`,
				`testing/dir4/folder\ with\ spaces/_`,
			},
		},
		{
			name: "file completor handles directories with spaces when same argument",
			c:    &FileCompletor[[]string]{},
			args: `cmd testing/dir4/folder\ wit`,
			want: []string{
				`testing/dir4/folder\ with\ spaces/`,
				`testing/dir4/folder\ with\ spaces/_`,
			},
		},
		{
			name: "file completor can dive into folder with spaces",
			c:    &FileCompletor[[]string]{},
			args: `cmd testing/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		{
			name: "file completor can dive into folder with spaces when combined args",
			c:    &FileCompletor[[]string]{},
			args: `cmd testing/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		{
			name: "autocomplete fills in letters that are the same for all options",
			c:    &FileCompletor[[]string]{},
			args: `cmd testing/dir4/fo`,
			want: []string{
				"testing/dir4/folder",
				"testing/dir4/folder_",
			},
		},
		{
			name:          "file completor doesn't get filtered out when part of a CommandBranch",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/dir",
			want: []string{
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
				" ",
			},
		},
		{
			name:          "file completor handles multiple options in directory",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/dir1/f",
			want: []string{
				"first.txt",
				"fourth.py",
				" ",
			},
		},
		{
			name:          "case insensitive gets letters autofilled",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/dI",
			want: []string{
				"testing/dir",
				"testing/dir_",
			},
		},
		{
			name:          "case insensitive recommends all without complete",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/DiR",
			want: []string{
				"dir1/",
				"dir2/",
				"dir3/",
				"dir4/",
				" ",
			},
		},
		{
			name:          "file completor ignores case",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/cases/abc",
			want: []string{
				"testing/cases/abcde",
				"testing/cases/abcde_",
			},
		},
		{
			name:          "file completor sorting ignores cases when no file",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/moreCases/",
			want: []string{
				"testing/moreCases/QW_",
				"testing/moreCases/QW__",
			},
		},
		{
			name:          "file completor sorting ignores cases when autofilling",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/moreCases/q",
			want: []string{
				"testing/moreCases/qW_",
				"testing/moreCases/qW__",
			},
		},
		{
			name:          "file completor sorting ignores cases when not autofilling",
			c:             &FileCompletor[[]string]{},
			commandBranch: true,
			args:          "cmd testing/moreCases/qW_t",
			want: []string{
				"qW_three.txt",
				"qw_TRES.txt",
				"Qw_two.txt",
				" ",
			},
		},
		{
			name: "file completor completes to case matched completion",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/meta",
			want: []string{
				"testing/metadata",
				"testing/metadata_",
			},
		},
		{
			name: "file completor completes to case matched completion",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/ME",
			want: []string{
				"testing/METADATA",
				"testing/METADATA_",
			},
		},
		{
			name: "file completor completes to something when no cases match",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/MeTa",
			want: []string{
				"testing/METADATA",
				"testing/METADATA_",
			},
		},
		{
			name: "file completor completes to case matched completion in current directory",
			c: &FileCompletor[[]string]{
				Directory: "testing",
			},
			args: "cmd meta",
			want: []string{
				"metadata",
				"metadata_",
			},
		},
		{
			name: "file completor completes to case matched completion in current directory",
			c: &FileCompletor[[]string]{
				Directory: "testing",
			},
			args: "cmd MET",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		{
			name: "file completor completes to something when no cases match in current directory",
			c: &FileCompletor[[]string]{
				Directory: "testing",
			},
			args: "cmd meTA",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		{
			name: "file completor doesn't complete when matches a prefix",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/METADATA",
			want: []string{
				"METADATA",
				"metadata_/",
				" ",
			},
		},
		{
			name: "file completor doesn't complete when matches a prefix file",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/metadata_/m",
			want: []string{
				"m1",
				"m2",
				" ",
			},
		},
		{
			name: "file completor returns complete match if distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd testing/metadata_/m1",
			want: []string{
				"testing/metadata_/m1",
			},
		},
		// Distinct file completors.
		{
			name: "file completor returns repeats if not distinct",
			c:    &FileCompletor[[]string]{},
			args: "cmd testing/three.txt testing/t",
			want: []string{"three.txt", "two.txt", " "},
		},
		{
			name: "file completor returns distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd testing/three.txt testing/t",
			want: []string{"testing/two.txt"},
		},
		{
			name: "file completor handles non with distinct",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd testing/three.txt testing/two.txt testing/t",
		},
		{
			name: "file completor first level distinct partially completes",
			c: &FileCompletor[[]string]{
				Distinct: true,
			},
			args: "cmd comp",
			want: []string{"completor", "completor_"},
		},
		{
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
		{
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
		{
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
		{
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
		/* Useful for commenting out tests */
	}
)

func TestTypedCompletors(t *testing.T) {
	for _, test := range stringCompletorCases {
		test.run(t)
	}
	for _, test := range boolCompletorCases {
		test.run(t)
	}
}
