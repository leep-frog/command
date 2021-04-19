package command

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestAlias(t *testing.T) {
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
			n:          AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil), IntNode("i", nil), FloatListNode("fl", 10, 0, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, 2, nil), IntNode("i", nil), FloatListNode("fl", 10, 0, nil)), ac, "pioneer"),
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
			n:    AliasNode(SerialNodes(StringListNode("sl", 1, UnboundedList, nil), IntNode("i", nil), FloatListNode("fl", 10, 0, nil)), ac, "pioneer"),
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
			n: AliasNode(SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Transformer: SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
					return StringListValue("papa", "mama", "baby"), nil
				}),
			})), ac, "pioneer"),
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
			n: AliasNode(SerialNodes(StringListNode("sl", 1, 2, &ArgOpt{
				Transformer: SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
					return nil, fmt.Errorf("bad news bears")
				}),
			})), ac, "pioneer"),
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
