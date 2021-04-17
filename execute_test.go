package command

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name       string
		node       *Node
		args       []string
		wantStderr []string
		wantStdout []string
		wantErr    error
		want       *ExecuteData
		wantData   *Data
		wantInput  *Input
	}{
		{
			name: "handles nil node",
		},
		{
			name:       "fails if unprocessed args",
			args:       []string{"hello"},
			wantErr:    fmt.Errorf("Unprocessed extra args: [hello]"),
			wantStderr: []string{"Unprocessed extra args: [hello]"},
			wantInput: &Input{
				args: []string{"hello"},
			},
		},
		// Single arg tests.
		{
			name:       "Fails if arg and no argument",
			node:       SerialNodes(StringNode("s")),
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue(""),
				},
			},
		},
		{
			name:       "Fails if int arg and no argument",
			node:       SerialNodes(IntNode("i")),
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
			wantData: &Data{
				Values: map[string]*Value{
					"i": IntValue(0),
				},
			},
		},
		{
			name:       "Fails if float arg and no argument",
			node:       SerialNodes(FloatNode("f")),
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
			wantData: &Data{
				Values: map[string]*Value{
					"f": FloatValue(0),
				},
			},
		},
		{
			name: "Processes single string arg",
			node: SerialNodes(StringNode("s")),
			args: []string{"hello"},
			wantInput: &Input{
				args: []string{"hello"},
				pos:  1,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("hello"),
				},
			},
		},
		{
			name: "Processes single int arg",
			node: SerialNodes(IntNode("i")),
			args: []string{"123"},
			wantInput: &Input{
				args: []string{"123"},
				pos:  1,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"i": IntValue(123),
				},
			},
		},
		{
			name: "Int arg fails if not an int",
			node: SerialNodes(IntNode("i")),
			args: []string{"12.3"},
			wantInput: &Input{
				args: []string{"12.3"},
				pos:  1,
			},
			wantErr:    fmt.Errorf(`strconv.Atoi: parsing "12.3": invalid syntax`),
			wantStderr: []string{`strconv.Atoi: parsing "12.3": invalid syntax`},
		},
		{
			name: "Processes single float arg",
			node: SerialNodes(FloatNode("f")),
			args: []string{"-12.3"},
			wantInput: &Input{
				args: []string{"-12.3"},
				pos:  1,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"f": FloatValue(-12.3),
				},
			},
		},
		{
			name: "Float arg fails if not a float",
			node: SerialNodes(FloatNode("f")),
			args: []string{"twelve"},
			wantInput: &Input{
				args: []string{"twelve"},
				pos:  1,
			},
			wantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
			wantStderr: []string{`strconv.ParseFloat: parsing "twelve": invalid syntax`},
		},
		// List args
		{
			name: "List fails if not enough args",
			node: SerialNodes(StringListNode("sl", 1, 1)),
			args: []string{"hello", "there", "sir"},
			wantInput: &Input{
				args: []string{"hello", "there", "sir"},
				pos:  2,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "there"),
				},
			},
			wantErr:    fmt.Errorf("Unprocessed extra args: [sir]"),
			wantStderr: []string{"Unprocessed extra args: [sir]"},
		},
		{
			name: "Processes string list if minimum provided",
			node: SerialNodes(StringListNode("sl", 1, 2)),
			args: []string{"hello"},
			wantInput: &Input{
				args: []string{"hello"},
				pos:  1,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello"),
				},
			},
		},
		{
			name: "Processes string list if some optional provided",
			node: SerialNodes(StringListNode("sl", 1, 2)),
			args: []string{"hello", "there"},
			wantInput: &Input{
				args: []string{"hello", "there"},
				pos:  2,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "there"),
				},
			},
		},
		{
			name: "Processes string list if max args provided",
			node: SerialNodes(StringListNode("sl", 1, 2)),
			args: []string{"hello", "there", "maam"},
			wantInput: &Input{
				args: []string{"hello", "there", "maam"},
				pos:  3,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "there", "maam"),
				},
			},
		},
		{
			name: "Processes int list",
			node: SerialNodes(IntListNode("il", 1, 2)),
			args: []string{"1", "-23"},
			wantInput: &Input{
				args: []string{"1", "-23"},
				pos:  2,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"il": IntListValue(1, -23),
				},
			},
		},
		{
			name: "Int list fails if an arg isn't an int",
			node: SerialNodes(IntListNode("il", 1, 2)),
			args: []string{"1", "four", "-23"},
			wantInput: &Input{
				args: []string{"1", "four", "-23"},
				pos:  3,
			},
			wantErr:    fmt.Errorf(`strconv.Atoi: parsing "four": invalid syntax`),
			wantStderr: []string{`strconv.Atoi: parsing "four": invalid syntax`},
		},
		{
			name: "Processes float list",
			node: SerialNodes(FloatListNode("fl", 1, 2)),
			args: []string{"0.1", "-2.3"},
			wantInput: &Input{
				args: []string{"0.1", "-2.3"},
				pos:  2,
			},
			wantData: &Data{
				Values: map[string]*Value{
					"fl": FloatListValue(0.1, -2.3),
				},
			},
		},
		{
			name: "Float list fails if an arg isn't an float",
			node: SerialNodes(FloatListNode("fl", 1, 2)),
			args: []string{"0.1", "four", "-23"},
			wantInput: &Input{
				args: []string{"0.1", "four", "-23"},
				pos:  3,
			},
			wantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "four": invalid syntax`),
			wantStderr: []string{`strconv.ParseFloat: parsing "four": invalid syntax`},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			fo := NewFakeOutput()
			data := &Data{}
			input := ParseArgs(test.args)
			eData, err := execute(test.node, input, fo, data)
			if test.wantErr == nil && err != nil {
				t.Fatalf("execute(%v) returned error (%v) when shouldn't have", test.args, err)
			}
			if test.wantErr != nil {
				if err == nil {
					t.Fatalf("execute(%v) returned no error when should have returned %v", test.args, test.wantErr)
				} else if diff := cmp.Diff(test.wantErr.Error(), err.Error()); diff != "" {
					t.Errorf("execute(%v) returned unexpected error (-want, +got):\n%s", test.args, diff)
				}
			}

			we := test.want
			if we == nil {
				we = &ExecuteData{}
			}
			if eData == nil {
				eData = &ExecuteData{}
			}
			if diff := cmp.Diff(we, eData); diff != "" {
				t.Errorf("execute(%v) returned unexpected ExecuteData (-want, +got):\n%s", test.args, diff)
			}

			wd := test.wantData
			if test.wantData == nil {
				wd = &Data{}
			}
			if diff := cmp.Diff(wd, data); diff != "" {
				t.Errorf("execute(%v) returned unexpected Data (-want, +got):\n%s", test.args, diff)
			}

			wi := test.wantInput
			if wi == nil {
				wi = &Input{}
			}
			if diff := cmp.Diff(wi, input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{})); diff != "" {
				t.Errorf("execute(%v) incorrectly modified input (-want, +got):\n%s", test.args, diff)
			}

			if diff := cmp.Diff(test.wantStdout, fo.stdout, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("execute(%v) sent wrong data to stdout (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, fo.stderr, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("execute(%v) sent wrong data to stderr (-want, +got):\n%s", test.args, diff)
			}
		})
	}
}
