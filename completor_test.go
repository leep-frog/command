package command

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCompletors(t *testing.T) {
	for _, test := range []struct {
		name string
		c    *Completor
		args string
		want []string
	}{
		{
			name: "nil completor returns nil",
		},
		{
			name: "nil fetcher returns nil",
			c:    &Completor{},
		},
		{
			name: "doesn't complete if case mismatch with upper",
			args: "cmd A",
			c: &Completor{
				SuggestionFetcher: &ListFetcher{
					Options: []string{"abc", "Abc", "ABC"},
				},
			},
			want: []string{"ABC", "Abc"},
		},
		{
			name: "doesn't complete if case mismatch with lower",
			args: "cmd a",
			c: &Completor{
				SuggestionFetcher: &ListFetcher{
					Options: []string{"abc", "Abc", "ABC"},
				},
			},
			want: []string{"abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and upper",
			args: "cmd A",
			c: &Completor{
				CaseInsensitive: true,
				SuggestionFetcher: &ListFetcher{
					Options: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
				},
			},
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and lower",
			args: "cmd a",
			c: &Completor{
				CaseInsensitive: true,
				SuggestionFetcher: &ListFetcher{
					Options: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
				},
			},
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes only matching cases",
			args: "cmd A",
			c: &Completor{
				SuggestionFetcher: SimpleFetcher(func(*Value, *Data) *Completion {
					return &Completion{
						Suggestions: []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
					}
				}),
			},
			want: []string{"ABC", "Abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and upper",
			args: "cmd A",
			c: &Completor{
				SuggestionFetcher: SimpleFetcher(func(*Value, *Data) *Completion {
					return &Completion{
						CaseInsensitive: true,
						Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
					}
				}),
			},
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "completes all cases if completor.CaseInsensitive and lower",
			args: "cmd a",
			c: &Completor{
				SuggestionFetcher: SimpleFetcher(func(*Value, *Data) *Completion {
					return &Completion{
						CaseInsensitive: true,
						Suggestions:     []string{"abc", "Abc", "ABC", "def", "Def", "DEF"},
					}
				}),
			},
			want: []string{"ABC", "Abc", "abc"},
		},
		{
			name: "bool completor returns bool values",
			c:    BoolCompletor(),
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
		{
			name: "non-distinct completor returns duplicates",
			c:    SimpleCompletor("first", "second", "third"),
			args: "cmd first second ",
			want: []string{"first", "second", "third"},
		},
		{
			name: "distinct completor does not return duplicates",
			c:    SimpleDistinctCompletor("first", "second", "third"),
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
			gn := SerialNodes(StringListNode("test", 2, 5, test.c))
			got := Autocomplete(gn, test.args)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("genericAutocomplete(%v, %v) returned diff (-want, +got):\n%s", gn, test.args, diff)
			}
		})
	}
}

func TestParseAndComplete(t *testing.T) {
	for _, test := range []struct {
		name        string
		args        string
		cursorIdx   int
		suggestions []string
		wantData    *Data
		want        []string
	}{
		{
			name: "handles empty array",
			wantData: &Data{
				"sl": StringListValue(),
			},
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
			wantData: &Data{
				"sl": StringListValue(),
			},
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
			wantData: &Data{
				"sl": StringListValue("Fo"),
			},
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
			wantData: &Data{
				"sl": StringListValue("F"),
			},
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
			wantData: &Data{
				"sl": StringListValue("Greg's One", ""),
			},
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
			wantData: &Data{
				"sl": StringListValue(`Greg"s Other"s`, ""),
			},
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
			wantData: &Data{
				"sl": StringListValue(""),
			},
		},
		{
			name: "completes properly if ending on double quote with previous option",
			// TODO: Should autocomplete just accept a string and it can parse the whole thing itself?
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
			wantData: &Data{
				"sl": StringListValue("hello", ""),
			},
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
			wantData: &Data{
				"sl": StringListValue("First Choice", ""),
			},
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
			wantData: &Data{
				"sl": StringListValue("First Choice", "F"),
			},
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
			wantData: &Data{
				"sl": StringListValue(""),
			},
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
			wantData: &Data{
				"sl": StringListValue("F"),
			},
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
			wantData: &Data{
				"sl": StringListValue("Greg's T"),
			},
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
			wantData: &Data{
				"sl": StringListValue(""),
			},
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
			wantData: &Data{
				"sl": StringListValue("F"),
			},
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
				// TODO: I think this may need backslashes like in the double quote case?
				// test this with actual commands and see what happens
				`'Greg"s Three'`,
				`'Greg"s Two'`,
			},
			wantData: &Data{
				"sl": StringListValue(`Greg"s T`),
			},
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
			wantData: &Data{
				"sl": StringListValue("Attempt One "),
			},
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
			wantData: &Data{
				"sl": StringListValue("Three"),
			},
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
			wantData: &Data{
				"sl": StringListValue("First O"),
			},
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &Completor{
				SuggestionFetcher: &ListFetcher{
					Options: test.suggestions,
				},
			}
			n := SerialNodes(StringListNode("sl", 0, UnboundedList, c))

			data := &Data{}
			got := autocomplete(n, test.args, data)
			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Autocomplete(%s) produced incorrect completions (-want, +got):\n%s", test.args, diff)
			}

			wantData := test.wantData
			if wantData == nil {
				wantData = &Data{}
			}
			if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("getCompleteData(%s) improperly parsed args (-want, +got)\n:%s", test.args, diff)
			}
		})
	}
}

func TestFetchers(t *testing.T) {
	for _, test := range []struct {
		name          string
		f             Fetcher
		distinct      bool
		args          string
		absErr        error
		stringArg     bool
		commandBranch bool
		want          []string
	}{
		{
			name: "list fetcher returns nil",
			f:    &ListFetcher{},
		},
		{
			name: "list fetcher returns empty list",
			f: &ListFetcher{
				Options: []string{},
			},
		},
		{
			name: "list fetcher returns list",
			f: &ListFetcher{
				Options: []string{"first", "second", "third"},
			},
			want: []string{"first", "second", "third"},
		},
		// BoolFetcher
		{
			name: "bool fetcher returns value",
			f:    &boolFetcher{},
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
		// FileFetcher tests
		{
			name:   "file fetcher returns nil if failure fetching current directory",
			f:      &FileFetcher{},
			absErr: fmt.Errorf("failed to fetch directory"),
		},
		// TODO: automatically create the empty directory
		// at beginning of this test (since not tracked by git).
		{
			name: "file fetcher handles empty directory",
			f:    &FileFetcher{},
			args: "cmd testing/empty/",
		},
		{
			name: "file fetcher works with string list arg",
			f:    &FileFetcher{},
			args: "cmd execu",
			want: []string{
				"execute",
				"execute_",
			},
		},
		{
			name:     "file fetcher works when distinct",
			distinct: true,
			f: &FileFetcher{
				Distinct: true,
			},
			args: "cmd execute.go execu",
			want: []string{
				"execute_test.go",
			},
		},
		{
			name:      "file fetcher works with string arg",
			f:         &FileFetcher{},
			args:      "cmd execu",
			stringArg: true,
			want: []string{
				"execute",
				"execute_",
			},
		},
		{
			name: "file fetcher returns nil if failure listing directory",
			f: &FileFetcher{
				Directory: "does/not/exist",
			},
		},
		{
			name: "file fetcher returns files in the specified directory",
			f: &FileFetcher{
				Directory: "testing",
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
			f: &FileFetcher{
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
			name: "file fetcher returns files matching regex",
			f: &FileFetcher{
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
			f: &FileFetcher{
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
			f: &FileFetcher{
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
			f: &FileFetcher{
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
			f:    &FileFetcher{},
			args: "cmd testing/dir1",
			want: []string{
				"testing/dir1/",
				"testing/dir1/_",
			},
		},
		{
			name: "file fetcher completes to directory when starting dir specified",
			f: &FileFetcher{
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
			f:    &FileFetcher{},
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
			f: &FileFetcher{
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
			f:    &FileFetcher{},
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
			f:    &FileFetcher{},
			args: "cmd testing/d",
			want: []string{
				"testing/dir",
				"testing/dir_",
			},
		},
		{
			name: "file fetcher only shows basenames when multiple options and starting dir",
			f: &FileFetcher{
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
			f:    &FileFetcher{},
			args: `cmd testing/dir4/folder\ wit`,
			want: []string{
				`testing/dir4/folder\ with\ spaces/`,
				`testing/dir4/folder\ with\ spaces/_`,
			},
		},
		{
			name: "file fetcher handles directories with spaces when same argument",
			f:    &FileFetcher{},
			args: `cmd testing/dir4/folder\ wit`,
			want: []string{
				`testing/dir4/folder\ with\ spaces/`,
				`testing/dir4/folder\ with\ spaces/_`,
			},
		},
		{
			name: "file fetcher can dive into folder with spaces",
			f:    &FileFetcher{},
			args: `cmd testing/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		{
			name: "file fetcher can dive into folder with spaces when combined args",
			f:    &FileFetcher{},
			args: `cmd testing/dir4/folder\ with\ spaces/`,
			want: []string{
				"goodbye.go",
				"hello.txt",
				" ",
			},
		},
		{
			name: "autocomplete fills in letters that are the same for all options",
			f:    &FileFetcher{},
			args: `cmd testing/dir4/fo`,
			want: []string{
				"testing/dir4/folder",
				"testing/dir4/folder_",
			},
		},
		{
			name:          "file fetcher doesn't get filtered out when part of a CommandBranch",
			f:             &FileFetcher{},
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
			f:             &FileFetcher{},
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
			f:             &FileFetcher{},
			commandBranch: true,
			args:          "cmd testing/dI",
			want: []string{
				"testing/dir",
				"testing/dir_",
			},
		},
		{
			name:          "case insensitive recommends all without complete",
			f:             &FileFetcher{},
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
			f:             &FileFetcher{},
			commandBranch: true,
			args:          "cmd testing/cases/abc",
			want: []string{
				"testing/cases/abcde",
				"testing/cases/abcde_",
			},
		},
		{
			name:          "file fetcher sorting ignores cases when no file",
			f:             &FileFetcher{},
			commandBranch: true,
			args:          "cmd testing/moreCases/",
			want: []string{
				"testing/moreCases/QW_",
				"testing/moreCases/QW__",
			},
		},
		{
			name:          "file fetcher sorting ignores cases when autofilling",
			f:             &FileFetcher{},
			commandBranch: true,
			args:          "cmd testing/moreCases/q",
			want: []string{
				"testing/moreCases/qW_",
				"testing/moreCases/qW__",
			},
		},
		{
			name:          "file fetcher sorting ignores cases when not autofilling",
			f:             &FileFetcher{},
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
			f:    &FileFetcher{},
			args: "cmd testing/meta",
			want: []string{
				"testing/metadata",
				"testing/metadata_",
			},
		},
		{
			name: "file fetcher completes to case matched completion",
			f:    &FileFetcher{},
			args: "cmd testing/ME",
			want: []string{
				"testing/METADATA",
				"testing/METADATA_",
			},
		},
		{
			name: "file fetcher completes to something when no cases match",
			f:    &FileFetcher{},
			args: "cmd testing/MeTa",
			want: []string{
				"testing/METADATA",
				"testing/METADATA_",
			},
		},
		{
			name: "file fetcher completes to case matched completion in current directory",
			f: &FileFetcher{
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
			f: &FileFetcher{
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
			f: &FileFetcher{
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
			f:    &FileFetcher{},
			args: "cmd testing/METADATA",
			want: []string{
				"METADATA",
				"metadata_/",
				" ",
			},
		},
		{
			name: "file fetcher doesn't complete when matches a prefix file",
			f:    &FileFetcher{},
			args: "cmd testing/metadata_/m",
			want: []string{
				"m1",
				"m2",
				" ",
			},
		},
		{
			name:     "file fetcher returns complete match if distinct",
			f:        &FileFetcher{},
			distinct: true,
			args:     "cmd testing/metadata_/m1",
			want: []string{
				"testing/metadata_/m1",
			},
		},
		// Distinct file fetchers.
		{
			name: "file fetcher returns repeats if not distinct",
			f:    &FileFetcher{},
			args: "cmd testing/three.txt testing/t",
			want: []string{"three.txt", "two.txt", " "},
		},
		{
			name: "file fetcher returns distinct",
			f: &FileFetcher{
				Distinct: true,
			},
			args: "cmd testing/three.txt testing/t",
			want: []string{"testing/two.txt"},
		},
		{
			name: "file fetcher handles non with distinct",
			f: &FileFetcher{
				Distinct: true,
			},
			args: "cmd testing/three.txt testing/two.txt testing/t",
		},
		{
			name: "file fetcher first level distinct partially completes",
			f: &FileFetcher{
				Distinct: true,
			},
			args: "cmd comp",
			want: []string{"completor", "completor_"},
		},
		{
			name: "file fetcher first level distinct returns all options",
			f: &FileFetcher{
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
			f: &FileFetcher{
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
			f: &FileFetcher{
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
			f: &FileFetcher{
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
	} {
		t.Run(test.name, func(t *testing.T) {
			oldAbs := filepathAbs
			filepathAbs = func(rel string) (string, error) {
				if test.absErr != nil {
					return "", test.absErr
				}
				return filepath.Abs(rel)
			}
			defer func() { filepathAbs = oldAbs }()

			completor := &Completor{
				SuggestionFetcher: test.f,
				Distinct:          test.distinct,
			}

			gn := SerialNodes(StringListNode("test", 2, 5, completor))
			if test.stringArg {
				gn = SerialNodes(StringNode("test", completor))
			}

			got := Autocomplete(gn, test.args)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("genericAutocomplete(%v) returned diff (-want, +got):\n%s", test.args, diff)
			}
		})
	}
}
