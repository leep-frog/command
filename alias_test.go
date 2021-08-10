package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAliasExecute(t *testing.T) {
	ac := &simpleAliasCLI{}
	for _, test := range []struct {
		name   string
		etc    *ExecuteTestCase
		am     map[string]map[string][]string
		wantAC *simpleAliasCLI
	}{
		{
			name: "alias requires arg",
			etc: &ExecuteTestCase{
				Node:       AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		// Add alias tests.
		{
			name: "requires an alias value",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
					},
				},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		{
			name: "requires a non-empty alias value",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", ""},
				WantData: &Data{
					"ALIAS": StringValue(""),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{},
					},
				},
				WantErr:    fmt.Errorf("validation failed: [MinLength] value must be at least 1 character"),
				WantStderr: []string{"validation failed: [MinLength] value must be at least 1 character"},
			},
		},
		{
			name: "doesn't override add command",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", "a", "hello"},
				WantData: &Data{
					"ALIAS": StringValue("a"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: "a"},
						{value: "hello"},
					},
					remaining: []int{2},
				},
				WantErr:    fmt.Errorf("cannot create alias for reserved value"),
				WantStderr: []string{"cannot create alias for reserved value"},
			},
		},
		{
			name: "doesn't override delete command",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", "d"},
				WantData: &Data{
					"ALIAS": StringValue("d"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: "d"},
					},
				},
				WantErr:    fmt.Errorf("cannot create alias for reserved value"),
				WantStderr: []string{"cannot create alias for reserved value"},
			},
		},
		// We don't really need to test other overrides (like we do for add and
		// delete above) since the user can still delete and add if they
		// accidentally override.
		{
			name: "ignores execute data from children nodes",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringNode("s"), SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
					ed.Executable = []string{
						"ab cd",
						"",
						"e",
					}
					ed.Executor = func(o Output, d *Data) error {
						o.Stdout("here we are")
						return o.Stderr("unfortunately")
					}
					return nil
				}, nil), nil)),
				Args: []string{"a", "b", "c"},
				WantData: &Data{
					"ALIAS": StringValue("b"),
					"s":     StringValue("c"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "b"},
						{value: "c", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"b": []string{"c"},
					},
				},
			},
		},
		{
			name: "errors on empty alias",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", ""},
				WantData: &Data{
					"ALIAS": StringValue(""),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("validation failed: [MinLength] value must be at least 1 character"),
				WantStderr: []string{"validation failed: [MinLength] value must be at least 1 character"},
			},
		},
		{
			name: "errors on too many values",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", "overload", "five", "four", "three", "two", "one"},
				WantData: &Data{
					"ALIAS": StringValue("overload"),
					"sl":    StringListValue("five", "four", "three"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "overload"},
						{value: "five", snapshots: snapshotsMap(1)},
						{value: "four", snapshots: snapshotsMap(1)},
						{value: "three", snapshots: snapshotsMap(1)},
						{value: "two", snapshots: snapshotsMap(1)},
						{value: "one", snapshots: snapshotsMap(1)},
					},
					remaining: []int{5, 6},
				},
				WantErr:    fmt.Errorf("Unprocessed extra args: [two one]"),
				WantStderr: []string{"Unprocessed extra args: [two one]"},
			},
		},
		{
			name: "fails to add empty alias list",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", "empty"},
				WantData: &Data{
					"ALIAS": StringValue("empty"),
				},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "empty"},
					},
				},
			},
		},
		{
			name: "adds alias list when just enough",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly"},
					},
				},
			},
		},
		{
			name: "fails if alias already exists",
			am: map[string]map[string][]string{
				"pioneer": {
					"bearMinimum": nil,
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly"},
					},
					remaining: []int{2},
				},
				WantStderr: []string{`Alias "bearMinimum" already exists`},
				WantErr:    fmt.Errorf(`Alias "bearMinimum" already exists`),
			},
		},
		{
			name: "adds alias list when maximum amount",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly", "teddy", "brown"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown"},
					},
				},
			},
		},
		{
			name: "adds alias for multiple nodes",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2), IntNode("i"), FloatListNode("fl", 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly", "teddy", "brown"),
					"i":     IntValue(3),
					"fl":    FloatListValue(2.2, -1.1),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
						{value: "3", snapshots: snapshotsMap(1)},
						{value: "2.2", snapshots: snapshotsMap(1)},
						{value: "-1.1", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
					},
				},
			},
		},
		{
			name: "adds alias when doesn't reach nodes",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2), IntNode("i"), FloatListNode("fl", 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly"},
					},
				},
			},
		},
		{
			name: "adds alias for unbounded list",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, UnboundedList), IntNode("i"), FloatListNode("fl", 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly", "teddy", "brown", "3", "2.2", "-1.1"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
						{value: "3", snapshots: snapshotsMap(1)},
						{value: "2.2", snapshots: snapshotsMap(1)},
						{value: "-1.1", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
					},
				},
			},
		},
		// Adding transformed arguments
		{
			name: "adds transformed arguments",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
						return StringListValue("papa", "mama", "baby"), nil
					})))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("papa", "mama", "baby"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "papa", snapshots: snapshotsMap(1)},
						{value: "mama", snapshots: snapshotsMap(1)},
						{value: "baby", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"papa", "mama", "baby"},
					},
				},
			},
		},
		{
			name: "fails if transform error",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
						return nil, fmt.Errorf("bad news bears")
					})))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &Data{
					"ALIAS": StringValue("bearMinimum"),
				},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
					},
				},
				WantStderr: []string{"Custom transformer failed: bad news bears"},
				WantErr:    fmt.Errorf("Custom transformer failed: bad news bears"),
			},
		},
		// Executing node tests.
		// TODO: test that executable is returned and that executor is run (by adding output.Stdout(...) and ensuring it is included in test.WantStdout).
		{
			name: "Fails to replace alias with empty value",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{},
				},
			},
			etc: &ExecuteTestCase{
				Node:       AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args:       []string{"t", "grizzly", "other"},
				WantErr:    fmt.Errorf("alias has empty value"),
				WantStderr: []string{"alias has empty value"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "t"},
						{value: "grizzly"},
						{value: "other"},
					},
					remaining: []int{0, 1, 2},
				},
			},
		},
		{
			name: "Replaces alias with value",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"t"},
				WantData: &Data{
					"sl": StringListValue("teddy"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "teddy"},
					},
				},
			},
		},
		{
			name: "Ignores non-alias values",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"tee"},
				WantData: &Data{
					"sl": StringListValue("tee"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "tee"},
					},
				},
			},
		},
		{
			name: "Replaces only alias value",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"t", "grizzly"},
				WantData: &Data{
					"sl": StringListValue("teddy", "grizzly"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "teddy"},
						{value: "grizzly"},
					},
				},
			},
		},
		{
			name: "Replaces with multiple values",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "brown"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"t", "grizzly"},
				WantData: &Data{
					"sl": StringListValue("teddy", "brown", "grizzly"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "teddy"},
						{value: "brown"},
						{value: "grizzly"},
					},
				},
			},
		},
		{
			name: "Replaces with multiple values and transformers",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "brown"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, UpperCaseTransformer()))),
				Args: []string{"t", "grizzly"},
				WantData: &Data{
					"sl": StringListValue("TEDDY", "BROWN", "GRIZZLY"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "TEDDY"},
						{value: "BROWN"},
						{value: "GRIZZLY"},
					},
				},
			},
		},
		// Arg with alias opt tests
		{
			name: "alias opt works with no aliases",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, AliasOpt("pioneer", ac))),
				Args: []string{"zero"},
				WantData: &Data{
					"sl": StringListValue("zero"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "zero"},
					},
				},
			},
		},
		{
			name: "alias opt replaces last argument",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, AliasOpt("pioneer", ac))),
				Args: []string{"hello", "dee"},
				WantData: &Data{
					"sl": StringListValue("hello", "d"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "d"},
					},
				},
			},
		},
		{
			name: "alias opt suggests args after replacement",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
					"t":   []string{"trois"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2,
					AliasOpt("pioneer", ac),
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: []string{"hello", "dee", "trois"},
				WantData: &Data{
					"sl": StringListValue("hello", "d", "trois"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "d"},
						{value: "trois"},
					},
				},
			},
		},
		{
			name: "alias opt replaces multiple aliases with more than one value",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList,
					AliasOpt("pioneer", ac),
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: []string{"f", "dee"},
				WantData: &Data{
					"sl": StringListValue("four", "two", "deux"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "four"},
						{value: "two"},
						{value: "deux"},
					},
				},
			},
		},
		{
			name: "alias opt replaces values across multiple args",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
					"z":   []string{"zero"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("sl", 1, 2, AliasOpt("pioneer", ac)),
					StringNode("s", AliasOpt("pioneer", ac)),
					OptionalStringNode("o", AliasOpt("pioneer", ac)),
				),
				Args: []string{"un", "dee", "z", "f"},
				WantData: &Data{
					"sl": StringListValue("un", "two", "deux"),
					"s":  StringValue("zero"),
					"o":  StringValue("four"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "two"},
						{value: "deux"},
						{value: "zero"},
						{value: "four"},
					},
				},
			},
		},
		{
			name: "alias opt replaces multiple aliases intertwined with regular args more than one value",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, AliasOpt("pioneer", ac),
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois", "five", "six"}},
					})),
				Args: []string{"f", "zero", "zero", "n1", "dee", "n2", "n3", "t", "u", "n4", "n5", ""},
				WantData: &Data{
					"sl": StringListValue("four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "four"},
						{value: "0"},
						{value: "0"},
						{value: "n1"},
						{value: "two"},
						{value: "deux"},
						{value: "n2"},
						{value: "n3"},
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
						{value: "un"},
						{value: "n4"},
						{value: "n5"},
						{value: ""},
					},
				},
			},
		},
		{
			name: "alias opt replaces last value",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, AliasOpt("pioneer", ac))),
				Args: []string{"f", "zero", "n1", "t"},
				WantData: &Data{
					"sl": StringListValue("four", "0", "n1", "three", "trois", "tres"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "four"},
						{value: "0"},
						{value: "n1"},
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
					},
				},
			},
		},
		{
			name: "alias happens before transform",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, AliasOpt("pioneer", ac),
					UpperCaseTransformer(),
				)),
				Args: []string{"f", "zero", "n1", "t"},
				WantData: &Data{
					"sl": StringListValue("FOUR", "0", "N1", "THREE", "TROIS", "TRES"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "FOUR"},
						{value: "0"},
						{value: "N1"},
						{value: "THREE"},
						{value: "TROIS"},
						{value: "TRES"},
					},
				},
			},
		},
		{
			name: "fails if alias doesn't add enough args",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 3, UnboundedList, AliasOpt("pioneer", ac))),
				Args: []string{"dee"},
				WantData: &Data{
					"sl": StringListValue("two", "deux"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "two"},
						{value: "deux"},
					},
				},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		{
			name: "works if alias adds enough args",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 3, UnboundedList, AliasOpt("pioneer", ac))),
				Args: []string{"t"},
				WantData: &Data{
					"sl": StringListValue("three", "trois", "tres"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
					},
				},
			},
		},
		{
			name: "alias values bleed over into next argument",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III", "3"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 3, 0, AliasOpt("pioneer", ac)), StringNode("s"), OptionalIntNode("i")),
				Args: []string{"t"},
				WantData: &Data{
					"sl": StringListValue("three", "trois", "tres"),
					"s":  StringValue("III"),
					"i":  IntValue(3),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
						{value: "III"},
						{value: "3"},
					},
				},
			},
		},
		{
			name: "don't alias for later args",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 3, 0, AliasOpt("pioneer", ac)), StringNode("s"), OptionalIntNode("i")),
				Args: []string{"I", "II", "III", "t"},
				WantData: &Data{
					"sl": StringListValue("I", "II", "III"),
					"s":  StringValue("t"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "I"},
						{value: "II"},
						{value: "III"},
						{value: "t"},
					},
				},
			},
		},
		// Get alias tests.
		{
			name: "Get alias requires argument",
			etc: &ExecuteTestCase{
				Node:       AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args:       []string{"g"},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "g"},
					},
				},
			},
		},
		{
			name: "Get alias handles missing alias type",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"g", "h"},
				WantData: &Data{
					"ALIAS": StringListValue("h"),
				},
				WantErr:    fmt.Errorf(`No aliases exist for alias type "pioneer"`),
				WantStderr: []string{`No aliases exist for alias type "pioneer"`},
				wantInput: &Input{
					args: []*inputArg{
						{value: "g"},
						{value: "h"},
					},
				},
			},
		},
		{
			name: "Gets alias",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"k": []string{"alpha"},
					"m": []string{"one", "two", "three"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"g", "h", "i", "j", "k", "l", "m"},
				WantData: &Data{
					"ALIAS": StringListValue("h", "i", "j", "k", "l", "m"),
				},
				WantStderr: []string{
					`Alias "j" does not exist`,
					`Alias "l" does not exist`,
				},
				WantStdout: []string{
					"h: ",
					"i: ",
					"k: alpha",
					"m: one two three",
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "g"},
						{value: "h"},
						{value: "i"},
						{value: "j"},
						{value: "k"},
						{value: "l"},
						{value: "m"},
					},
				},
			},
		},
		// List aliases
		{
			name: "lists alias handles unset map",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"l"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "l"},
					},
				},
			},
		},
		{
			name: "lists aliases",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"k": []string{"alpha"},
					"m": []string{"one", "two", "three"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"l"},
				WantStdout: []string{
					"h: ",
					"i: ",
					"k: alpha",
					"m: one two three",
					"z: omega",
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "l"},
					},
				},
			},
		},
		// Search alias tests.
		{
			name: "search alias requires argument",
			etc: &ExecuteTestCase{
				Node:       AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args:       []string{"s"},
				WantStderr: []string{"not enough arguments"},
				WantErr:    fmt.Errorf("not enough arguments"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
					},
				},
			},
		},
		{
			name: "search alias fails on invalid regexp",
			etc: &ExecuteTestCase{
				Node:       AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args:       []string{"s", ":)"},
				WantStderr: []string{"Invalid regexp: error parsing regexp: unexpected ): `:)`"},
				WantErr:    fmt.Errorf("Invalid regexp: error parsing regexp: unexpected ): `:)`"),
				WantData: &Data{
					"regexp": StringListValue(":)"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
						{value: ":)"},
					},
				},
			},
		},
		{
			name: "searches through aliases",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"j": []string{"bazzinga"},
					"k": []string{"alpha"},
					"l": []string{"garage"},
					"m": []string{"one", "two", "three"},
					"n": []string{"four"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"s", "ga$"},
				WantData: &Data{
					"regexp": StringListValue("ga$"),
				},
				WantStdout: []string{
					"j: bazzinga",
					"z: omega",
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
						{value: "ga$"},
					},
				},
			},
		},
		{
			name: "searches through aliases with multiple",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"j": []string{"bazzinga"},
					"k": []string{"alpha"},
					"l": []string{"garage"},
					"m": []string{"one", "two", "three"},
					"n": []string{"four"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"s", "a$", "^.: [aeiou]"},
				WantData: &Data{
					"regexp": StringListValue("a$", "^.: [aeiou]"),
				},
				WantStdout: []string{
					"k: alpha",
					"z: omega",
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
						{value: "a$"},
						{value: "^.: [aeiou]"},
					},
				},
			},
		},
		// Delete alias tests.
		{
			name: "Delete requires argument",
			etc: &ExecuteTestCase{
				Node:       AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args:       []string{"d"},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
					},
				},
			},
		},
		{
			name: "Delete returns error if alias group does not exist",
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"d", "e"},
				WantData: &Data{
					"ALIAS": StringListValue("e"),
				},
				WantErr:    fmt.Errorf("Alias group has no aliases yet."),
				WantStderr: []string{"Alias group has no aliases yet."},
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "e"},
					},
				},
			},
		},
		{
			name: "Delete prints error if alias does not exist",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "grizzly"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"d", "tee"},
				WantData: &Data{
					"ALIAS": StringListValue("tee"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "tee"},
					},
				},
				WantStderr: []string{`Alias "tee" does not exist`},
			},
		},
		{
			name: "Deletes an alias",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "grizzly"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"d", "t"},
				WantData: &Data{
					"ALIAS": StringListValue("t"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "t"},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {},
				},
			},
		},
		{
			name: "Delete deletes multiple aliases",
			am: map[string]map[string][]string{
				"pioneer": {
					"p":      []string{"polar", "pooh"},
					"colors": []string{"brown", "black"},
					"t":      []string{"teddy"},
					"g":      []string{"grizzly"},
				},
			},
			etc: &ExecuteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: []string{"d", "t", "penguin", "colors", "bare"},
				WantData: &Data{
					"ALIAS": StringListValue("t", "penguin", "colors", "bare"),
				},
				WantStderr: []string{
					`Alias "penguin" does not exist`,
					`Alias "bare" does not exist`,
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "t"},
						{value: "penguin"},
						{value: "colors"},
						{value: "bare"},
					},
				},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"p": []string{"polar", "pooh"},
						"g": []string{"grizzly"},
					},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			originalMP := map[string]map[string][]string{}
			for k1, m := range test.am {
				originalMP[k1] = map[string][]string{}
				for k2, v := range m {
					originalMP[k1][k2] = v
				}
			}
			ac.changed = false
			ac.mp = test.am

			ExecuteTest(t, test.etc, &ExecuteTestOptions{testInput: true})
			ChangeTest(t, test.wantAC, ac, cmp.AllowUnexported(simpleAliasCLI{}))
		})
	}
}

func TestAliasComplete(t *testing.T) {
	ac := &simpleAliasCLI{}
	for _, test := range []struct {
		name string
		ctc  *CompleteTestCase
		mp   map[string]map[string][]string
	}{
		{
			name: "suggests arg suggestions, but not command names",
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd ",
				WantData: &Data{
					"sl": StringListValue(""),
				},
				Want: []string{"deux", "trois", "un"},
			},
		},
		// Add alias test
		{
			name: "suggests nothing for alias",
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd a ",
				WantData: &Data{
					aliasArgName: StringValue(""),
				},
			},
		},
		{
			name: "fails if empty alias",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha": nil,
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd alpha b ",
			},
		},
		{
			name: "suggests regular things after alias",
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd a b ",
				WantData: &Data{
					aliasArgName: StringValue("b"),
					"sl":         StringListValue(""),
				},
				Want: []string{"deux", "trois", "un"},
			},
		},
		{
			name: "suggests regular things after alias",
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd a b ",
				WantData: &Data{
					aliasArgName: StringValue("b"),
					"sl":         StringListValue(""),
				},
				Want: []string{"deux", "trois", "un"},
			},
		},
		// Get alias test
		{
			name: "get alias makes suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: "cmd g ",
				WantData: &Data{
					aliasArgName: StringListValue(""),
				},
				Want: []string{"alpha", "alright", "any", "balloon", "bear"},
			},
		},
		{
			name: "get alias makes partial suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: "cmd g b",
				WantData: &Data{
					aliasArgName: StringListValue("b"),
				},
				Want: []string{"balloon", "bear"},
			},
		},
		{
			name: "get alias makes unique suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: "cmd g alright balloon ",
				WantData: &Data{
					aliasArgName: StringListValue("alright", "balloon", ""),
				},
				Want: []string{"alpha", "any", "bear"},
			},
		},
		// Delete alias test
		{
			name: "get alias makes suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: "cmd d ",
				WantData: &Data{
					aliasArgName: StringListValue(""),
				},
				Want: []string{"alpha", "alright", "any", "balloon", "bear"},
			},
		},
		{
			name: "get alias makes partial suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: "cmd d b",
				WantData: &Data{
					aliasArgName: StringListValue("b"),
				},
				Want: []string{"balloon", "bear"},
			},
		},
		{
			name: "get alias makes unique suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2))),
				Args: "cmd d alright balloon ",
				WantData: &Data{
					aliasArgName: StringListValue("alright", "balloon", ""),
				},
				Want: []string{"alpha", "any", "bear"},
			},
		},
		// Execute alias tests
		{
			name: "suggests regular things for regular command",
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd zero ",
				WantData: &Data{
					"sl": StringListValue("zero", ""),
				},
				Want: []string{"deux", "trois", "un"},
			},
		},
		{
			name: "doesn't replace last argument if it's one",
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd dee",
				WantData: &Data{
					"sl": StringListValue("dee"),
				},
			},
		},
		{
			name: "suggests args after replacement",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd dee t",
				WantData: &Data{
					"sl": StringListValue("d", "t"),
				},
				Want: []string{"trois"},
			},
		},
		{
			name: "replaced args are considered in distinct ops",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			ctc: &CompleteTestCase{
				Node: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2,
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					}))),
				Args: "cmd dee ",
				WantData: &Data{
					"sl": StringListValue("deux", ""),
				},
				Want: []string{"trois", "un"},
			},
		},
		// Arg with alias opt tests
		{
			name: "alias opt suggests regular things for regular command",
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, AliasOpt("pioneer", ac),
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: "cmd zero ",
				WantData: &Data{
					"sl": StringListValue("zero", ""),
				},
				Want: []string{"deux", "trois", "un"},
			},
		},
		{
			name: "alias opt doesn't replace last argument if it's one",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, AliasOpt("pioneer", ac),
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: "cmd hello dee",
				WantData: &Data{
					"sl": StringListValue("hello", "dee"),
				},
			},
		},
		{
			name: "alias opt suggests args after replacement",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, AliasOpt("pioneer", ac),
					&Completor{
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: "cmd hello dee t",
				WantData: &Data{
					"sl": StringListValue("hello", "d", "t"),
				},
				Want: []string{"trois"},
			},
		},
		{
			name: "alias opt replaced args are considered in distinct ops",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, AliasOpt("pioneer", ac),
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: "cmd dee ",
				WantData: &Data{
					"sl": StringListValue("deux", ""),
				},
				Want: []string{"trois", "un"},
			},
		},
		{
			name: "alias opt replaces multiple args",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
					"t":   []string{"trois"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, AliasOpt("pioneer", ac),
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: "cmd dee t ",
				WantData: &Data{
					"sl": StringListValue("deux", "trois", ""),
				},
				Want: []string{"un"},
			},
		},
		{
			name: "alias opt replaces multiple aliases with more than one value",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, AliasOpt("pioneer", ac),
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
					})),
				Args: "cmd f dee ",
				WantData: &Data{
					"sl": StringListValue("four", "two", "deux", ""),
				},
				Want: []string{"trois", "un"},
			},
		},
		{
			name: "alias opt replaces multiple aliases intertwined with regular args more than one value",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, AliasOpt("pioneer", ac),
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois", "five", "six"}},
					})),
				Args: "cmd f zero zero n1 dee n2 n3 t u n4 n5 ",
				WantData: &Data{
					"sl": StringListValue("four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""),
				},
				Want: []string{"five", "six"},
			},
		},
		{
			name: "alias opt doesn't replace last value",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, AliasOpt("pioneer", ac),
					&Completor{
						Distinct:          true,
						SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois", "five", "six"}},
					})),
				Args: "cmd f zero n1 t",
				WantData: &Data{
					"sl": StringListValue("four", "0", "n1", "t"),
				},
				Want: []string{"trois"},
			},
		},
		{
			name: "alias values bleed over into next argument for suggestion",
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 3, 0, AliasOpt("pioneer", ac)), StringNode("s"), StringNode("i", SimpleCompletor("alpha", "beta"))),
				Args: "cmd t ",
				WantData: &Data{
					"sl": StringListValue("three", "trois", "tres"),
					"s":  StringValue("III"),
					"i":  StringValue(""),
				},
				Want: []string{"alpha", "beta"},
			},
		},
		{
			name: "don't alias for later args",
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringListNode("sl", 3, 0, AliasOpt("pioneer", ac)), StringNode("s"), StringNode("i", SimpleCompletor("alpha", "beta"))),
				Args: "cmd I II III t ",
				WantData: &Data{
					"sl": StringListValue("I", "II", "III"),
					"s":  StringValue("t"),
					"i":  StringValue(""),
				},
				Want: []string{"alpha", "beta"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ac.mp = test.mp
			CompleteTest(t, test.ctc, nil)
		})
	}
}

type simpleAliasCLI struct {
	mp      map[string]map[string][]string
	changed bool
}

func (sac *simpleAliasCLI) AliasMap() map[string]map[string][]string {
	if sac.mp == nil {
		sac.mp = map[string]map[string][]string{}
	}
	return sac.mp
}

func (sac *simpleAliasCLI) Changed() bool {
	return sac.changed
}

func (sac *simpleAliasCLI) MarkChanged() {
	sac.changed = true
}

func newSimpleAlias(existing map[string]map[string][]string) AliasCLI {
	return &simpleAliasCLI{
		mp: existing,
	}
}

func UpperCaseTransformer() ArgOpt {
	return &simpleTransformer{
		vt: StringListType,
		t: func(v *Value) (*Value, error) {
			r := make([]string, 0, len(v.StringList()))
			for _, v := range v.StringList() {
				r = append(r, strings.ToUpper(v))
			}
			return StringListValue(r...), nil
		},
	}
}
