package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestAliasExecute(t *testing.T) {
	ac := &simpleAliasCLI{}
	for _, test := range []struct {
		name       string
		n          *Node
		args       []string
		am         map[string]map[string][]string
		wantStderr []string
		wantStdout []string
		wantErr    error
		wantEData  *ExecuteData
		wantData   *Data
		wantInput  *Input
		wantAC     *simpleAliasCLI
	}{
		{
			name:       "alias requires arg",
			n:          AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue(),
				},
			},
		},
		// Add alias tests.
		{
			name: "requires an alias value",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"a"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue(""),
				},
			},
			wantInput: &Input{
				args: []string{"a"},
			},
			wantErr:    fmt.Errorf("validation failed: [MinLength] value must be at least 1 character"),
			wantStderr: []string{"validation failed: [MinLength] value must be at least 1 character"},
		},
		// TODO: test empty alias.  Shouldn't allow? Otherwise, need to test it with
		// a lot of existing functionality.
		{
			name: "ignores execute data from children nodes",
			n: AliasNode("pioneer", ac, SerialNodes(StringNode("s", nil), SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
				ed.Executable = [][]string{
					{"ab", "cd"},
					{},
					{"e"},
				}
				ed.Executor = func(o Output, d *Data) error {
					o.Stdout("here we are")
					return o.Stderr("unfortunately")
				}
				return nil
			}, nil), nil)),
			args: []string{"a", "b", "c"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("b"),
					"s":     StringValue("c"),
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
			wantInput: &Input{
				args: []string{"a", "b", "c"},
			},
		},
		{
			name: "errors on empty alias",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"a", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue(""),
				},
			},
			wantInput: &Input{
				args: []string{"a", ""},
			},
			wantErr:    fmt.Errorf("validation failed: [MinLength] value must be at least 1 character"),
			wantStderr: []string{"validation failed: [MinLength] value must be at least 1 character"},
		},
		{
			name: "errors on too many values",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"a", "overload", "five", "four", "three", "two", "one"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("overload"),
					"sl":    StringListValue("five", "four", "three"),
				},
			},
			wantInput: &Input{
				args:      []string{"a", "overload", "five", "four", "three", "two", "one"},
				remaining: []int{5, 6},
			},
			wantErr:    fmt.Errorf("Unprocessed extra args: [two one]"),
			wantStderr: []string{"Unprocessed extra args: [two one]"},
		},
		{
			name: "adds empty alias list",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"a", "empty"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("empty"),
					"sl":    StringListValue(),
				},
			},
			wantInput: &Input{
				args: []string{"a", "empty"},
			},
			// TODO: find a way to prevent stderr from this. Either
			// - modify error definition
			// - remove output.Err function (and just have caller log errors to stderr)
			// - pass in modified output type in alias
			wantStderr: []string{"not enough arguments"},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"empty": nil,
					},
				},
			},
		},
		{
			name: "adds alias list when just enough",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"a", "bearMinimum", "grizzly"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly"),
				},
			},
			wantInput: &Input{
				args: []string{"a", "bearMinimum", "grizzly"},
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
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			am: map[string]map[string][]string{
				"pioneer": {
					"bearMinimum": nil,
				},
			},
			args: []string{"a", "bearMinimum", "grizzly"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
				},
			},
			wantInput: &Input{
				args:      []string{"a", "bearMinimum", "grizzly"},
				remaining: []int{2},
			},
			wantStderr: []string{`Alias "bearMinimum" already exists`},
			wantErr:    fmt.Errorf(`Alias "bearMinimum" already exists`),
		},
		{
			name: "adds alias list when maximum amount",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly", "teddy", "brown"),
				},
			},
			wantInput: &Input{
				args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
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
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil), IntNode("i", nil), FloatListNode("fl", 10, 0, nil))),
			args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly", "teddy", "brown"),
					"i":     IntValue(3),
					"fl":    FloatListValue(2.2, -1.1),
				},
			},
			wantInput: &Input{
				args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
					},
				},
			},
			wantStderr: []string{"not enough arguments"},
		},
		{
			name: "adds alias when doesn't reach nodes",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil), IntNode("i", nil), FloatListNode("fl", 10, 0, nil))),
			args: []string{"a", "bearMinimum", "grizzly"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly"),
					"i":     IntValue(0),
				},
			},
			wantInput: &Input{
				args: []string{"a", "bearMinimum", "grizzly"},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly"},
					},
				},
			},
			wantStderr: []string{"not enough arguments"},
		},
		{
			name: "adds alias for unbounded list",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, UnboundedList, nil), IntNode("i", nil), FloatListNode("fl", 10, 0, nil))),
			args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("grizzly", "teddy", "brown", "3", "2.2", "-1.1"),
					"i":     IntValue(0),
				},
			},
			wantInput: &Input{
				args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
			},
			wantAC: &simpleAliasCLI{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
					},
				},
			},
			wantStderr: []string{"not enough arguments"},
		},
		// Adding transformed arguments
		{
			name: "adds transformed arguments",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Transformer: SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
					return StringListValue("papa", "mama", "baby"), nil
				}),
			}))),
			args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
					"sl":    StringListValue("papa", "mama", "baby"),
				},
			},
			wantInput: &Input{
				args: []string{"a", "bearMinimum", "papa", "mama", "baby"},
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
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Transformer: SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
					return nil, fmt.Errorf("bad news bears")
				}),
			}))),
			args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringValue("bearMinimum"),
				},
			},
			wantInput: &Input{
				args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
			},
			wantStderr: []string{"Custom transformer failed: bad news bears"},
			wantErr:    fmt.Errorf("Custom transformer failed: bad news bears"),
		},
		// Executing node tests.
		// TODO: test that executable is returned and that executor is run (by adding output.Stdout(...) and ensuring it is included in test.wantStdout).
		{
			name: "Replaces alias with value",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"t"},
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("teddy"),
				},
			},
			wantInput: &Input{
				args: []string{"teddy"},
			},
		},
		{
			name: "Ignores non-alias values",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"tee"},
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("tee"),
				},
			},
			wantInput: &Input{
				args: []string{"tee"},
			},
		},
		{
			name: "Replaces only alias value",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"t", "grizzly"},
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("teddy", "grizzly"),
				},
			},
			wantInput: &Input{
				args: []string{"teddy", "grizzly"},
			},
		},
		{
			name: "Replaces with multiple values",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"t", "grizzly"},
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "brown"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("teddy", "brown", "grizzly"),
				},
			},
			wantInput: &Input{
				args: []string{"teddy", "brown", "grizzly"},
			},
		},
		{
			name: "Replaces with multiple values and transformers",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Transformer: UpperCaseTransformer(),
			}))),
			args: []string{"t", "grizzly"},
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "brown"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("TEDDY", "BROWN", "GRIZZLY"),
				},
			},
			wantInput: &Input{
				args: []string{"TEDDY", "BROWN", "GRIZZLY"},
			},
		},
		// Arg with alias opt tests
		// TODO: test only edits up to limit (here and completion).
		{
			name: "alias opt works with no aliases",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			})),
			args: []string{"zero"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("zero"),
				},
			},
			wantInput: &Input{
				args: []string{"zero"},
			},
		},
		{
			name: "alias opt replaces last argument",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			args: []string{"hello", "dee"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "d"),
				},
			},
			wantInput: &Input{
				args: []string{"hello", "d"},
			},
		},
		{
			name: "alias opt suggests args after replacement",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
					"t":   []string{"trois"},
				},
			},
			args: []string{"hello", "dee", "trois"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "d", "trois"),
				},
			},
			wantInput: &Input{
				args: []string{
					"hello",
					"d",
					"trois",
				},
			},
		},
		{
			name: "alias opt replaces multiple aliases with more than one value",
			n: SerialNodes(StringListNode("sl", 1, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
				},
			},
			args: []string{"f", "dee"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("four", "two", "deux"),
				},
			},
			wantInput: &Input{
				args: []string{"four", "two", "deux"},
			},
		},
		{
			name: "alias opt replaces values across multiple args",
			n: SerialNodes(
				StringListNode("sl", 1, 2, &ArgOpt{
					Alias: &AliasOpt{
						AliasName: "pioneer",
						AliasCLI:  ac,
					},
				}),
				StringNode("s", &ArgOpt{
					Alias: &AliasOpt{
						AliasName: "pioneer",
						AliasCLI:  ac,
					},
				}),
				OptionalStringNode("o", &ArgOpt{
					Alias: &AliasOpt{
						AliasName: "pioneer",
						AliasCLI:  ac,
					},
				}),
			),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
					"z":   []string{"zero"},
				},
			},
			args: []string{"un", "dee", "z", "f"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("un", "two", "deux"),
					"s":  StringValue("zero"),
					"o":  StringValue("four"),
				},
			},
			wantInput: &Input{
				args: []string{"un", "two", "deux", "zero", "four"},
			},
		},
		{
			name: "alias opt replaces multiple aliases intertwined with regular args more than one value",
			n: SerialNodes(StringListNode("sl", 1, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois", "five", "six"}},
				},
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			args: []string{"f", "zero", "zero", "n1", "dee", "n2", "n3", "t", "u", "n4", "n5", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""),
				},
			},
			wantInput: &Input{
				args: []string{"four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""},
			},
		},
		{
			name: "alias opt replaces last value",
			n: SerialNodes(StringListNode("sl", 1, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			args: []string{"f", "zero", "n1", "t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("four", "0", "n1", "three", "trois", "tres"),
				},
			},
			wantInput: &Input{
				args: []string{"four", "0", "n1", "three", "trois", "tres"},
			},
		},
		{
			name: "alias happens before transform",
			n: SerialNodes(StringListNode("sl", 1, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Transformer: UpperCaseTransformer(),
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			args: []string{"f", "zero", "n1", "t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("FOUR", "0", "N1", "THREE", "TROIS", "TRES"),
				},
			},
			wantInput: &Input{
				args: []string{"FOUR", "0", "N1", "THREE", "TROIS", "TRES"},
			},
		},
		{
			name: "fails if alias doesn't add enough args",
			n: SerialNodes(StringListNode("sl", 3, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
				},
			},
			args: []string{"dee"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("two", "deux"),
				},
			},
			wantInput: &Input{
				args: []string{"two", "deux"},
			},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
		},
		{
			name: "works if alias adds enough args",
			n: SerialNodes(StringListNode("sl", 3, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			})),
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres"},
				},
			},
			args: []string{"t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("three", "trois", "tres"),
				},
			},
			wantInput: &Input{
				args: []string{"three", "trois", "tres"},
			},
		},
		{
			name: "alias values bleed over into next argument",
			n: SerialNodes(StringListNode("sl", 3, 0, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			}), StringNode("s", nil), OptionalIntNode("i", nil)),
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III", "3"},
				},
			},
			args: []string{"t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("three", "trois", "tres"),
					"s":  StringValue("III"),
					"i":  IntValue(3),
				},
			},
			wantInput: &Input{
				args: []string{"three", "trois", "tres", "III", "3"},
			},
		},
		{
			name: "don't alias for later args",
			n: SerialNodes(StringListNode("sl", 3, 0, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			}), StringNode("s", nil), OptionalIntNode("i", nil)),
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres"},
				},
			},
			args: []string{"I", "II", "III", "t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("I", "II", "III"),
					"s":  StringValue("t"),
					"i":  IntValue(0),
				},
			},
			wantInput: &Input{
				args: []string{"I", "II", "III", "t"},
			},
		},
		// Get alias tests.
		{
			name: "Get alias requires argument",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"g"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue(),
				},
			},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
			wantInput: &Input{
				args: []string{"g"},
			},
		},
		{
			name: "Get alias handles missing alias type",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"g", "h"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue("h"),
				},
			},
			wantErr:    fmt.Errorf(`No aliases exist for alias type "pioneer"`),
			wantStderr: []string{`No aliases exist for alias type "pioneer"`},
			wantInput: &Input{
				args: []string{"g", "h"},
			},
		},
		{
			name: "Gets alias",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"g", "h", "i", "j", "k", "l", "m"},
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"k": []string{"alpha"},
					"m": []string{"one", "two", "three"},
					"z": []string{"omega"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue("h", "i", "j", "k", "l", "m"),
				},
			},
			wantStderr: []string{
				`Alias "j" does not exist`,
				`Alias "l" does not exist`,
			},
			wantStdout: []string{
				"h: ",
				"i: ",
				"k: alpha",
				"m: one two three",
			},
			wantInput: &Input{
				args: []string{"g", "h", "i", "j", "k", "l", "m"},
			},
		},
		// List aliases
		{
			name: "lists alias handles unset map",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"l"},
			wantInput: &Input{
				args: []string{"l"},
			},
		},
		{
			name: "lists aliases",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"l"},
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"k": []string{"alpha"},
					"m": []string{"one", "two", "three"},
					"z": []string{"omega"},
				},
			},
			wantStdout: []string{
				"h: ",
				"i: ",
				"k: alpha",
				"m: one two three",
				"z: omega",
			},
			wantInput: &Input{
				args: []string{"l"},
			},
		},
		// Search alias tests.
		{
			name:       "search alias requires argument",
			n:          AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args:       []string{"s"},
			wantStderr: []string{"not enough arguments"},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantData: &Data{
				Values: map[string]*Value{
					"regexp": StringListValue(),
				},
			},
			wantInput: &Input{
				args: []string{"s"},
			},
		},
		{
			name:       "search alias fails on invalid regexp",
			n:          AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args:       []string{"s", ":)"},
			wantStderr: []string{"Invalid regexp: error parsing regexp: unexpected ): `:)`"},
			wantErr:    fmt.Errorf("Invalid regexp: error parsing regexp: unexpected ): `:)`"),
			wantData: &Data{
				Values: map[string]*Value{
					"regexp": StringListValue(":)"),
				},
			},
			wantInput: &Input{
				args: []string{"s", ":)"},
			},
		},
		{
			name: "searches through aliases",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"s", "ga$"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"regexp": StringListValue("ga$"),
				},
			},
			wantStdout: []string{
				"j: bazzinga",
				"z: omega",
			},
			wantInput: &Input{
				args: []string{"s", "ga$"},
			},
		},
		{
			name: "searches through aliases with multiple",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"s", "a$", "^.: [aeiou]"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"regexp": StringListValue("a$", "^.: [aeiou]"),
				},
			},
			wantStdout: []string{
				"k: alpha",
				"z: omega",
			},
			wantInput: &Input{
				args: []string{"s", "a$", "^.: [aeiou]"},
			},
		},
		// Delete alias tests.
		{
			name: "Delete requires argument",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"d"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue(),
				},
			},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
			wantInput: &Input{
				args: []string{"d"},
			},
		},
		{
			name: "Delete returns error if alias group does not exist",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"d", "e"},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue("e"),
				},
			},
			wantErr:    fmt.Errorf("Alias group has no aliases yet."),
			wantStderr: []string{"Alias group has no aliases yet."},
			wantInput: &Input{
				args: []string{"d", "e"},
			},
		},
		{
			name: "Delete prints error if alias does not exist",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"d", "tee"},
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "grizzly"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue("tee"),
				},
			},
			wantInput: &Input{
				args: []string{"d", "tee"},
			},
			wantStderr: []string{`Alias "tee" does not exist`},
		},
		{
			name: "Deletes an alias",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"d", "t"},
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "grizzly"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue("t"),
				},
			},
			wantInput: &Input{
				args: []string{"d", "t"},
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
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			args: []string{"d", "t", "penguin", "colors", "bare"},
			am: map[string]map[string][]string{
				"pioneer": {
					"p":      []string{"polar", "pooh"},
					"colors": []string{"brown", "black"},
					"t":      []string{"teddy"},
					"g":      []string{"grizzly"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"ALIAS": StringListValue("t", "penguin", "colors", "bare"),
				},
			},
			wantStderr: []string{
				`Alias "penguin" does not exist`,
				`Alias "bare" does not exist`,
			},
			wantInput: &Input{
				args: []string{"d", "t", "penguin", "colors", "bare"},
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
			executeTest(t, test.n, test.args, test.wantErr, test.wantEData, test.wantData, test.wantInput, test.wantStdout, test.wantStderr)

			wac := test.wantAC
			if wac == nil {
				wac = &simpleAliasCLI{
					mp: originalMP,
				}
			}
			if diff := cmp.Diff(wac, ac, cmp.AllowUnexported(simpleAliasCLI{}), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("Alias.Execute(%v) incorrectly modified alias values:\n%s", test.args, diff)
			}
		})
	}
}

func TestAliasComplete(t *testing.T) {
	ac := &simpleAliasCLI{}
	for _, test := range []struct {
		name     string
		n        *Node
		args     []string
		mp       map[string]map[string][]string
		wantData *Data
		want     []string
	}{
		{
			name: "suggests command names and arg suggestions",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			args: []string{""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue(""),
				},
			},
			want: []string{"a", "d", "deux", "g", "l", "s", "trois", "un"},
		},
		// Add alias test
		{
			name: "suggests nothing for alias",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			args: []string{"a", ""},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringValue(""),
				},
			},
		},
		{
			name: "suggests regular things after alias",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			args: []string{"a", "b", ""},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringValue("b"),
					"sl":         StringListValue(""),
				},
			},
			want: []string{"deux", "trois", "un"},
		},
		{
			name: "suggests regular things after alias",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			args: []string{"a", "b", ""},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringValue("b"),
					"sl":         StringListValue(""),
				},
			},
			want: []string{"deux", "trois", "un"},
		},
		// Get alias test
		{
			name: "get alias makes suggestions",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			args: []string{"g", ""},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringListValue(""),
				},
			},
			want: []string{"alpha", "alright", "any", "balloon", "bear"},
		},
		{
			name: "get alias makes partial suggestions",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			args: []string{"g", "b"},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringListValue("b"),
				},
			},
			want: []string{"balloon", "bear"},
		},
		{
			name: "get alias makes unique suggestions",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			args: []string{"g", "alright", "balloon", ""},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringListValue("alright", "balloon", ""),
				},
			},
			want: []string{"alpha", "any", "bear"},
		},
		// Delete alias test
		{
			name: "get alias makes suggestions",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			args: []string{"d", ""},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringListValue(""),
				},
			},
			want: []string{"alpha", "alright", "any", "balloon", "bear"},
		},
		{
			name: "get alias makes partial suggestions",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			args: []string{"d", "b"},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringListValue("b"),
				},
			},
			want: []string{"balloon", "bear"},
		},
		{
			name: "get alias makes unique suggestions",
			n:    AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, nil))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			args: []string{"d", "alright", "balloon", ""},
			wantData: &Data{
				Values: map[string]*Value{
					aliasArgName: StringListValue("alright", "balloon", ""),
				},
			},
			want: []string{"alpha", "any", "bear"},
		},
		// Execute alias tests
		{
			name: "suggests regular things for regular command",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			args: []string{"zero", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("zero", ""),
				},
			},
			want: []string{"deux", "trois", "un"},
		},
		{
			name: "doesn't replace last argument if it's one",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			args: []string{"dee"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("dee"),
				},
			},
		},
		{
			name: "suggests args after replacement",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			args: []string{"dee", "t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("d", "t"),
				},
			},
			want: []string{"trois"},
		},
		{
			name: "replaced args are considered in distinct ops",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			}))),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			args: []string{"dee", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("deux", ""),
				},
			},
			want: []string{"trois", "un"},
		},
		// Arg with alias opt tests
		{
			name: "alias opt suggests regular things for regular command",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			args: []string{"zero", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("zero", ""),
				},
			},
			want: []string{"deux", "trois", "un"},
		},
		{
			name: "alias opt doesn't replace last argument if it's one",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			args: []string{"hello", "dee"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "dee"),
				},
			},
		},
		{
			name: "alias opt suggests args after replacement",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			args: []string{"hello", "dee", "t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "d", "t"),
				},
			},
			want: []string{"trois"},
		},
		{
			name: "alias opt replaced args are considered in distinct ops",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			args: []string{"dee", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("deux", ""),
				},
			},
			want: []string{"trois", "un"},
		},
		{
			name: "alias opt replaces multiple args",
			n: SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
					"t":   []string{"trois"},
				},
			},
			args: []string{"dee", "t", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("deux", "trois", ""),
				},
			},
			want: []string{"un"},
		},
		{
			name: "alias opt replaces multiple aliases with more than one value",
			n: SerialNodes(StringListNode("sl", 1, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois"}},
				},
			})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
				},
			},
			args: []string{"f", "dee", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("four", "two", "deux", ""),
				},
			},
			want: []string{"trois", "un"},
		},
		{
			name: "alias opt replaces multiple aliases intertwined with regular args more than one value",
			n: SerialNodes(StringListNode("sl", 1, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois", "five", "six"}},
				},
			})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			args: []string{"f", "zero", "zero", "n1", "dee", "n2", "n3", "t", "u", "n4", "n5", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""),
				},
			},
			want: []string{"five", "six"},
		},
		{
			name: "alias opt doesn't replace last value",
			n: SerialNodes(StringListNode("sl", 1, UnboundedList, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
				Completor: &Completor{
					Distinct:          true,
					SuggestionFetcher: &ListFetcher{[]string{"un", "deux", "trois", "five", "six"}},
				},
			})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			args: []string{"f", "zero", "n1", "t"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("four", "0", "n1", "t"),
				},
			},
			want: []string{"trois"},
		},
		{
			name: "alias values bleed over into next argument for suggestion",
			n: SerialNodes(StringListNode("sl", 3, 0, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			}), StringNode("s", nil), StringNode("i", &ArgOpt{Completor: SimpleCompletor("alpha", "beta")})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			args: []string{"t", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("three", "trois", "tres"),
					"s":  StringValue("III"),
					"i":  StringValue(""),
				},
			},
			want: []string{"alpha", "beta"},
		},
		{
			name: "don't alias for later args",
			n: SerialNodes(StringListNode("sl", 3, 0, &ArgOpt{
				Alias: &AliasOpt{
					AliasName: "pioneer",
					AliasCLI:  ac,
				},
			}), StringNode("s", nil), StringNode("i", &ArgOpt{Completor: SimpleCompletor("alpha", "beta")})),
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			args: []string{"I", "II", "III", "t", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("I", "II", "III"),
					"s":  StringValue("t"),
					"i":  StringValue(""),
				},
			},
			want: []string{"alpha", "beta"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ac.mp = test.mp
			data := &Data{}
			input := ParseArgs(test.args)
			got := getCompleteData(test.n, input, data)
			var results []string
			if got != nil && got.Completion != nil {
				results = got.Completion.Process(input)
			}

			if diff := cmp.Diff(test.wantData, data, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("getCompleteData(%s) improperly parsed args (-want, +got)\n:%s", test.args, diff)
			}

			if diff := cmp.Diff(test.want, results); diff != "" {
				t.Errorf("getCompleteData(%s) returned incorrect suggestions (-want, +got):\n%s", test.args, diff)
			}
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

func (sac *simpleAliasCLI) MarkChanged() {
	sac.changed = true
}

func newSimpleAlias(existing map[string]map[string][]string) AliasCLI {
	return &simpleAliasCLI{
		mp: existing,
	}
}

func UpperCaseTransformer() ArgTransformer {
	return SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
		r := make([]string, 0, len(v.StringList()))
		for _, v := range v.StringList() {
			r = append(r, strings.ToUpper(v))
		}
		return StringListValue(r...), nil
	})
}
