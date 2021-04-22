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
			wantAC: &simpleAliasCLI{
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": nil,
					},
				},
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
			wantAC:     &simpleAliasCLI{},
			wantStderr: []string{"Custom transformer failed: bad news bears"},
			wantErr:    fmt.Errorf("Custom transformer failed: bad news bears"),
		},
		// Executing node tests.
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
			wantAC: &simpleAliasCLI{
				mp: map[string]map[string][]string{
					"pioneer": {
						"t": []string{"teddy"},
					},
				},
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
			wantAC: &simpleAliasCLI{
				mp: map[string]map[string][]string{
					"pioneer": {
						"t": []string{"teddy"},
					},
				},
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
			wantAC: &simpleAliasCLI{
				mp: map[string]map[string][]string{
					"pioneer": {
						"t": []string{"teddy"},
					},
				},
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
			wantAC: &simpleAliasCLI{
				mp: map[string]map[string][]string{
					"pioneer": {
						"t": []string{"teddy", "brown"},
					},
				},
			},
		},
		{
			name: "Replaces with multiple values and transformers",
			n: AliasNode("pioneer", ac, SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Transformer: SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
					r := make([]string, 0, len(v.StringList()))
					for _, v := range v.StringList() {
						r = append(r, strings.ToUpper(v))
					}
					return StringListValue(r...), nil
				}),
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
			wantAC: &simpleAliasCLI{
				mp: map[string]map[string][]string{
					"pioneer": {
						"t": []string{"teddy", "brown"},
					},
				},
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
			wantAC: &simpleAliasCLI{},
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
			wantAC: &simpleAliasCLI{},
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
			wantAC: &simpleAliasCLI{
				mp: map[string]map[string][]string{
					"pioneer": {
						"h": nil,
						"i": []string{},
						"k": []string{"alpha"},
						"m": []string{"one", "two", "three"},
						"z": []string{"omega"},
					},
				},
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
			wantAC: &simpleAliasCLI{},
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
			wantAC: &simpleAliasCLI{},
		},
		{
			name: "Delete prints error if alias does not exist",
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
	} {
		t.Run(test.name, func(t *testing.T) {
			ac.changed = false
			ac.mp = test.am
			ExecuteTest(t, test.n, test.args, test.wantErr, test.wantEData, test.wantData, test.wantInput, test.wantStdout, test.wantStderr)

			wac := test.wantAC
			if wac == nil {
				wac = &simpleAliasCLI{}
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
			want: []string{"a", "d", "deux", "g", "trois", "un"},
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
	} {
		t.Run(test.name, func(t *testing.T) {
			fmt.Println(test.name, "==============")
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

func ExecuteTest(t *testing.T, node *Node, args []string, wantErr error, want *ExecuteData, wantData *Data, wantInput *Input, wantStdout, wantStderr []string) {
	t.Helper()

	fo := NewFakeOutput()
	data := &Data{}
	input := ParseArgs(args)
	eData, err := execute(node, input, fo, data)
	if wantErr == nil && err != nil {
		t.Fatalf("execute(%v) returned error (%v) when shouldn't have", args, err)
	}
	if wantErr != nil {
		if err == nil {
			t.Fatalf("execute(%v) returned no error when should have returned %v", args, wantErr)
		} else if diff := cmp.Diff(wantErr.Error(), err.Error()); diff != "" {
			t.Errorf("execute(%v) returned unexpected error (-want, +got):\n%s", args, diff)
		}
	}

	if want == nil {
		want = &ExecuteData{}
	}
	if eData == nil {
		eData = &ExecuteData{}
	}
	if diff := cmp.Diff(want, eData, cmpopts.IgnoreFields(ExecuteData{}, "Executor")); diff != "" {
		t.Errorf("execute(%v) returned unexpected ExecuteData (-want, +got):\n%s", args, diff)
	}

	if wantData == nil {
		wantData = &Data{}
	}
	if diff := cmp.Diff(wantData, data); diff != "" {
		t.Errorf("execute(%v) returned unexpected Data (-want, +got):\n%s", args, diff)
	}

	if wantInput == nil {
		wantInput = &Input{}
	}
	if diff := cmp.Diff(wantInput, input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{})); diff != "" {
		t.Errorf("execute(%v) incorrectly modified input (-want, +got):\n%s", args, diff)
	}

	if diff := cmp.Diff(wantStdout, fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("execute(%v) sent wrong data to stdout (-want, +got):\n%s", args, diff)
	}
	if diff := cmp.Diff(wantStderr, fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("execute(%v) sent wrong data to stderr (-want, +got):\n%s", args, diff)
	}
}
