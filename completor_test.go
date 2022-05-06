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
		c       *Completor[[]string]
		args    string
		want    []string
		wantErr error
	}
	for _, test := range []*testCase{
		{
			name: "nil completor returns nil",
		},
		{
			name: "nil fetcher returns nil",
			c:    &Completor[[]string]{},
		},
		{
			name: "doesn't complete if case mismatch with upper",
			args: "cmd A",
			c: &Completor[[]string]{
				Fetcher: &ListFetcher[[]string]{
					Options: []string{"abc", "Abc", "ABC"},
				},
			},
			want: []string{"ABC", "Abc"},
		},
		{
			name: "doesn't complete if case mismatch with lower",
			args: "cmd a",
			c: &Completor[[]string]{
				Fetcher: &ListFetcher[[]string]{
					Options: []string{"abc", "Abc", "ABC"},
				},
			},
			want: []string{"abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and upper",
			args: "cmd A",
			c: &Completor[[]string]{
				CaseInsensitive: true,
				Fetcher: &ListFetcher[[]string]{
					Options: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
				},
			},
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and lower",
			args: "cmd a",
			c: &Completor[[]string]{
				CaseInsensitive: true,
				Fetcher: &ListFetcher[[]string]{
					Options: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
				},
			},
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name:    "returns error",
			args:    "cmd A",
			wantErr: fmt.Errorf("bad news bears"),
			c: &Completor[[]string]{
				Fetcher: SimpleFetcher(func([]string, *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
					}, fmt.Errorf("bad news bears")
				}),
			},
		},
		{
			name: "completes only matching cases",
			args: "cmd A",
			c: &Completor[[]string]{
				Fetcher: SimpleFetcher(func([]string, *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
					}, nil
				}),
			},
			want: []string{"ABC", "Abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and upper",
			args: "cmd A",
			c: &Completor[[]string]{
				Fetcher: SimpleFetcher(func([]string, *Data) (*Completion, error) {
					return &Completion{
						CaseInsensitive: true,
						Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
					}, nil
				}),
			},
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and lower",
			args: "cmd a",
			c: &Completor[[]string]{
				Fetcher: SimpleFetcher(func([]string, *Data) (*Completion, error) {
					return &Completion{
						CaseInsensitive: true,
						Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
					}, nil
				}),
			},
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
			CompleteTest(t, &CompleteTestCase{
				Node:          SerialNodes(ListArg[string]("test", testDesc, 2, 5, test.c)),
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
			c := &Completor[[]string]{
				Fetcher: &ListFetcher[[]string]{
					Options: test.suggestions,
				},
			}
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

type fetcherTest[T any] struct {
	name          string
	f             Fetcher[[]T]
	singleF       Fetcher[T]
	args          string
	ptArgs        []string
	setup         func(*testing.T)
	cleanup       func(*testing.T)
	absErr        error
	commandBranch bool
	want          []string
}

func (test *fetcherTest[T]) run(t *testing.T) {
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
		if test.singleF != nil {
			completor := &Completor[T]{
				Fetcher: test.singleF,
			}
			got = Autocomplete(SerialNodes(Arg[T]("test", testDesc, completor)), test.args, test.ptArgs)
		} else {
			completor := &Completor[[]T]{
				Fetcher: test.f,
			}
			got = Autocomplete(SerialNodes(ListArg[T]("test", testDesc, 2, 5, completor)), test.args, test.ptArgs)
		}

		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("genericAutocomplete(%v) returned diff (-want, +got):\n%s", test.args, diff)
		}
	})
}

var (
	boolFetcherCases = []*fetcherTest[bool]{
		// BoolFetcher
		{
			name:    "bool fetcher returns value",
			singleF: &boolFetcher{},
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
	stringFetcherCases = []*fetcherTest[string]{
		{
			name: "list fetcher returns nil",
			f:    &ListFetcher[[]string]{},
		},
		{
			name: "list fetcher returns empty list",
			f: &ListFetcher[[]string]{
				Options: []string{},
			},
		},
		{
			name: "list fetcher returns list",
			f: &ListFetcher[[]string]{
				Options: []string{"first", "second", "third"},
			},
			want: []string{"first", "second", "third"},
		},
		// FileFetcher tests
		{
			name:   "file fetcher returns nil if failure fetching current directory",
			f:      &FileFetcher[[]string]{},
			absErr: fmt.Errorf("failed to fetch directory"),
		},
		{
			name: "file fetcher returns files with file types and directories",
			singleF: &FileFetcher[string]{
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
			name: "file fetcher handles empty directory",
			f:    &FileFetcher[[]string]{},
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
			name: "file fetcher works with string list arg",
			f:    &FileFetcher[[]string]{},
			args: "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		{
			name: "file fetcher works when distinct",
			f: &FileFetcher[[]string]{
				Distinct: true,
			},
			args: "cmd execute.go execu",
			want: []string{
				"execute_test.go",
			},
		},
		{
			name:    "file fetcher works with string arg",
			singleF: &FileFetcher[string]{},
			args:    "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		{
			name: "file fetcher returns nil if failure listing directory",
			f: &FileFetcher[[]string]{
				Directory: "does/not/exist",
			},
		},
		{
			name: "file fetcher returns files in the specified directory",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher returns files in the specified directory",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher ignores things from IgnoreFunc",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher returns files matching regex",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher requires prefix",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher ignores directories",
			f: &FileFetcher[[]string]{
				Directory:         "testing/dir2",
				IgnoreDirectories: true,
			},
			want: []string{
				"file",
				"file_",
			},
		},
		{
			name: "file fetcher ignores files",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher completes to directory",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/dir1",
			want: []string{
				"testing/dir1/",
				"testing/dir1/_",
			},
		},
		{
			name: "file fetcher completes to directory when starting dir specified",
			f: &FileFetcher[[]string]{
				Directory: "testing",
			},
			args: "cmd dir1",
			want: []string{
				"dir1/",
				"dir1/_",
			},
		},
		{
			name: "file fetcher shows contents of directory when ending with a separator",
			f:    &FileFetcher[[]string]{},
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
			name: "file fetcher completes to directory when ending with a separator and when starting dir specified",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher only shows basenames when multiple options with different next letter",
			f:    &FileFetcher[[]string]{},
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
			name: "file fetcher shows full names when multiple options with same next letter",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/d",
			want: []string{
				"testing/dir",
				"testing/dir_",
			},
		},
		{
			name: "file fetcher only shows basenames when multiple options and starting dir",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher handles directories with spaces",
			f:    &FileFetcher[[]string]{},
			args: `cmd testing/dir4/folder\ wit`,
			want: []string{
				`testing/dir4/folder\ with\ spaces/`,
				`testing/dir4/folder\ with\ spaces/_`,
			},
		},
		{
			name: "file fetcher handles directories with spaces when same argument",
			f:    &FileFetcher[[]string]{},
			args: `cmd testing/dir4/folder\ wit`,
			want: []string{
				`testing/dir4/folder\ with\ spaces/`,
				`testing/dir4/folder\ with\ spaces/_`,
			},
		},
		{
			name: "file fetcher can dive into folder with spaces",
			f:    &FileFetcher[[]string]{},
			args: `cmd testing/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		{
			name: "file fetcher can dive into folder with spaces when combined args",
			f:    &FileFetcher[[]string]{},
			args: `cmd testing/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		{
			name: "autocomplete fills in letters that are the same for all options",
			f:    &FileFetcher[[]string]{},
			args: `cmd testing/dir4/fo`,
			want: []string{
				"testing/dir4/folder",
				"testing/dir4/folder_",
			},
		},
		{
			name:          "file fetcher doesn't get filtered out when part of a CommandBranch",
			f:             &FileFetcher[[]string]{},
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
			name:          "file fetcher handles multiple options in directory",
			f:             &FileFetcher[[]string]{},
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
			f:             &FileFetcher[[]string]{},
			commandBranch: true,
			args:          "cmd testing/dI",
			want: []string{
				"testing/dir",
				"testing/dir_",
			},
		},
		{
			name:          "case insensitive recommends all without complete",
			f:             &FileFetcher[[]string]{},
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
			name:          "file fetcher ignores case",
			f:             &FileFetcher[[]string]{},
			commandBranch: true,
			args:          "cmd testing/cases/abc",
			want: []string{
				"testing/cases/abcde",
				"testing/cases/abcde_",
			},
		},
		{
			name:          "file fetcher sorting ignores cases when no file",
			f:             &FileFetcher[[]string]{},
			commandBranch: true,
			args:          "cmd testing/moreCases/",
			want: []string{
				"testing/moreCases/QW_",
				"testing/moreCases/QW__",
			},
		},
		{
			name:          "file fetcher sorting ignores cases when autofilling",
			f:             &FileFetcher[[]string]{},
			commandBranch: true,
			args:          "cmd testing/moreCases/q",
			want: []string{
				"testing/moreCases/qW_",
				"testing/moreCases/qW__",
			},
		},
		{
			name:          "file fetcher sorting ignores cases when not autofilling",
			f:             &FileFetcher[[]string]{},
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
			name: "file fetcher completes to case matched completion",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/meta",
			want: []string{
				"testing/metadata",
				"testing/metadata_",
			},
		},
		{
			name: "file fetcher completes to case matched completion",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/ME",
			want: []string{
				"testing/METADATA",
				"testing/METADATA_",
			},
		},
		{
			name: "file fetcher completes to something when no cases match",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/MeTa",
			want: []string{
				"testing/METADATA",
				"testing/METADATA_",
			},
		},
		{
			name: "file fetcher completes to case matched completion in current directory",
			f: &FileFetcher[[]string]{
				Directory: "testing",
			},
			args: "cmd meta",
			want: []string{
				"metadata",
				"metadata_",
			},
		},
		{
			name: "file fetcher completes to case matched completion in current directory",
			f: &FileFetcher[[]string]{
				Directory: "testing",
			},
			args: "cmd MET",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		{
			name: "file fetcher completes to something when no cases match in current directory",
			f: &FileFetcher[[]string]{
				Directory: "testing",
			},
			args: "cmd meTA",
			want: []string{
				"METADATA",
				"METADATA_",
			},
		},
		{
			name: "file fetcher doesn't complete when matches a prefix",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/METADATA",
			want: []string{
				"METADATA",
				"metadata_/",
				" ",
			},
		},
		{
			name: "file fetcher doesn't complete when matches a prefix file",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/metadata_/m",
			want: []string{
				"m1",
				"m2",
				" ",
			},
		},
		{
			name: "file fetcher returns complete match if distinct",
			f: &FileFetcher[[]string]{
				Distinct: true,
			},
			args: "cmd testing/metadata_/m1",
			want: []string{
				"testing/metadata_/m1",
			},
		},
		// Distinct file fetchers.
		{
			name: "file fetcher returns repeats if not distinct",
			f:    &FileFetcher[[]string]{},
			args: "cmd testing/three.txt testing/t",
			want: []string{"three.txt", "two.txt", " "},
		},
		{
			name: "file fetcher returns distinct",
			f: &FileFetcher[[]string]{
				Distinct: true,
			},
			args: "cmd testing/three.txt testing/t",
			want: []string{"testing/two.txt"},
		},
		{
			name: "file fetcher handles non with distinct",
			f: &FileFetcher[[]string]{
				Distinct: true,
			},
			args: "cmd testing/three.txt testing/two.txt testing/t",
		},
		{
			name: "file fetcher first level distinct partially completes",
			f: &FileFetcher[[]string]{
				Distinct: true,
			},
			args: "cmd comp",
			want: []string{"completor", "completor_"},
		},
		{
			name: "file fetcher first level distinct returns all options",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher first level distinct completes partial",
			f: &FileFetcher[[]string]{
				Distinct: true,
			},
			args: "cmd custom_nodes.go comp",
			want: []string{
				"completor",
				"completor_",
			},
		},
		{
			name: "file fetcher first level distinct suggests remaining",
			f: &FileFetcher[[]string]{
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
			name: "file fetcher first level distinct completes partial",
			f: &FileFetcher[[]string]{
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

func TestFetchers(t *testing.T) {
	for _, test := range stringFetcherCases {
		test.run(t)
	}
	for _, test := range boolFetcherCases {
		test.run(t)
	}
}
