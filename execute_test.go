package command

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type errorEdge struct {
	e error
}

func (ee *errorEdge) Next(*Input, *Data) (*Node, error) {
	return nil, ee.e
}

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
				args:      []string{"hello"},
				remaining: []int{0},
			},
		},
		// Single arg tests.
		{
			name:       "Fails if arg and no argument",
			node:       SerialNodes(StringNode("s", nil)),
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue(""),
				},
			},
		},
		{
			name: "Fails if edge fails",
			args: []string{"hello"},
			node: &Node{
				Processor: StringNode("s", nil),
				Edge: &errorEdge{
					e: fmt.Errorf("bad news bears"),
				},
			},
			wantInput: &Input{
				args: []string{"hello"},
			},
			wantErr: fmt.Errorf("bad news bears"),
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("hello"),
				},
			},
		},
		{
			name:       "Fails if int arg and no argument",
			node:       SerialNodes(IntNode("i", nil)),
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
			node:       SerialNodes(FloatNode("f", nil)),
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
			node: SerialNodes(StringNode("s", nil)),
			args: []string{"hello"},
			wantInput: &Input{
				args: []string{"hello"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("hello"),
				},
			},
		},
		{
			name: "Processes single int arg",
			node: SerialNodes(IntNode("i", nil)),
			args: []string{"123"},
			wantInput: &Input{
				args: []string{"123"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"i": IntValue(123),
				},
			},
		},
		{
			name: "Int arg fails if not an int",
			node: SerialNodes(IntNode("i", nil)),
			args: []string{"12.3"},
			wantInput: &Input{
				args: []string{"12.3"},
			},
			wantErr:    fmt.Errorf(`strconv.Atoi: parsing "12.3": invalid syntax`),
			wantStderr: []string{`strconv.Atoi: parsing "12.3": invalid syntax`},
		},
		{
			name: "Processes single float arg",
			node: SerialNodes(FloatNode("f", nil)),
			args: []string{"-12.3"},
			wantInput: &Input{
				args: []string{"-12.3"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"f": FloatValue(-12.3),
				},
			},
		},
		{
			name: "Float arg fails if not a float",
			node: SerialNodes(FloatNode("f", nil)),
			args: []string{"twelve"},
			wantInput: &Input{
				args: []string{"twelve"},
			},
			wantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
			wantStderr: []string{`strconv.ParseFloat: parsing "twelve": invalid syntax`},
		},
		// List args
		{
			name: "List fails if not enough args",
			node: SerialNodes(StringListNode("sl", 1, 1, nil)),
			args: []string{"hello", "there", "sir"},
			wantInput: &Input{
				args:      []string{"hello", "there", "sir"},
				remaining: []int{2},
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
			node: SerialNodes(StringListNode("sl", 1, 2, nil)),
			args: []string{"hello"},
			wantInput: &Input{
				args: []string{"hello"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello"),
				},
			},
		},
		{
			name: "Processes string list if some optional provided",
			node: SerialNodes(StringListNode("sl", 1, 2, nil)),
			args: []string{"hello", "there"},
			wantInput: &Input{
				args: []string{"hello", "there"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "there"),
				},
			},
		},
		{
			name: "Processes string list if max args provided",
			node: SerialNodes(StringListNode("sl", 1, 2, nil)),
			args: []string{"hello", "there", "maam"},
			wantInput: &Input{
				args: []string{"hello", "there", "maam"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "there", "maam"),
				},
			},
		},
		{
			name: "Unbounded string list fails if less than min provided",
			node: SerialNodes(StringListNode("sl", 4, UnboundedList, nil)),
			args: []string{"hello", "there", "kenobi"},
			wantInput: &Input{
				args: []string{"hello", "there", "kenobi"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "there", "kenobi"),
				},
			},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
		},
		{
			name: "Processes unbounded string list if min provided",
			node: SerialNodes(StringListNode("sl", 1, UnboundedList, nil)),
			args: []string{"hello"},
			wantInput: &Input{
				args: []string{"hello"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello"),
				},
			},
		},
		{
			name: "Processes unbounded string list if more than min provided",
			node: SerialNodes(StringListNode("sl", 1, UnboundedList, nil)),
			args: []string{"hello", "there", "kenobi"},
			wantInput: &Input{
				args: []string{"hello", "there", "kenobi"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("hello", "there", "kenobi"),
				},
			},
		},
		{
			name: "Processes int list",
			node: SerialNodes(IntListNode("il", 1, 2, nil)),
			args: []string{"1", "-23"},
			wantInput: &Input{
				args: []string{"1", "-23"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"il": IntListValue(1, -23),
				},
			},
		},
		{
			name: "Int list fails if an arg isn't an int",
			node: SerialNodes(IntListNode("il", 1, 2, nil)),
			args: []string{"1", "four", "-23"},
			wantInput: &Input{
				args: []string{"1", "four", "-23"},
			},
			wantErr:    fmt.Errorf(`strconv.Atoi: parsing "four": invalid syntax`),
			wantStderr: []string{`strconv.Atoi: parsing "four": invalid syntax`},
		},
		{
			name: "Processes float list",
			node: SerialNodes(FloatListNode("fl", 1, 2, nil)),
			args: []string{"0.1", "-2.3"},
			wantInput: &Input{
				args: []string{"0.1", "-2.3"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"fl": FloatListValue(0.1, -2.3),
				},
			},
		},
		{
			name: "Float list fails if an arg isn't an float",
			node: SerialNodes(FloatListNode("fl", 1, 2, nil)),
			args: []string{"0.1", "four", "-23"},
			wantInput: &Input{
				args: []string{"0.1", "four", "-23"},
			},
			wantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "four": invalid syntax`),
			wantStderr: []string{`strconv.ParseFloat: parsing "four": invalid syntax`},
		},
		// Multiple args
		{
			name: "Processes multiple args",
			node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil)),
			args: []string{"0", "1", "two", "0.3", "-4"},
			wantInput: &Input{
				args: []string{"0", "1", "two", "0.3", "-4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4),
				},
			},
		},
		{
			name: "Fails if extra args when multiple",
			node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil)),
			args: []string{"0", "1", "two", "0.3", "-4", "0.5", "6"},
			wantInput: &Input{
				remaining: []int{6},
				args:      []string{"0", "1", "two", "0.3", "-4", "0.5", "6"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4, 0.5),
				},
			},
			wantErr:    fmt.Errorf("Unprocessed extra args: [6]"),
			wantStderr: []string{"Unprocessed extra args: [6]"},
		},
		// Executor tests.
		{
			name: "executes with proper data",
			node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil), ExecutorNode(func(o Output, d *Data) error {
				var keys []string
				for k := range d.Values {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					o.Stdout("%s: %s", k, d.Values[k].Str())
				}
				return nil
			})),
			args: []string{"0", "1", "two", "0.3", "-4"},
			wantInput: &Input{
				args: []string{"0", "1", "two", "0.3", "-4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4),
				},
			},
			wantStdout: []string{
				"fl: 0.30, -4.00",
				"il: 0, 1",
				"s: two",
			},
		},
		{
			name: "executor error is returned",
			node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil), ExecutorNode(func(o Output, d *Data) error {
				return o.Stderr("bad news bears")
			})),
			args: []string{"0", "1", "two", "0.3", "-4"},
			wantInput: &Input{
				args: []string{"0", "1", "two", "0.3", "-4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4),
				},
			},
			wantStderr: []string{"bad news bears"},
			wantErr:    fmt.Errorf("bad news bears"),
		},
		// ArgValidator tests
		{
			name: "breaks when arg option is for invalid type",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, IntEQ(123))),
			},
			args: []string{"123"},
			wantInput: &Input{
				args: []string{"123"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("123"),
				},
			},
			wantStderr: []string{"validation failed: option can only be bound to arguments with type 3"},
			wantErr:    fmt.Errorf("validation failed: option can only be bound to arguments with type 3"),
		},
		// Contains
		{
			name: "contains works",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, Contains("good"))),
			},
			args: []string{"goodbye"},
			wantInput: &Input{
				args: []string{"goodbye"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("goodbye"),
				},
			},
		},
		{
			name: "contains fails",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, Contains("good"))),
			},
			args: []string{"hello"},
			wantInput: &Input{
				args: []string{"hello"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("hello"),
				},
			},
			wantStderr: []string{`validation failed: [Contains] value doesn't contain substring "good"`},
			wantErr:    fmt.Errorf(`validation failed: [Contains] value doesn't contain substring "good"`),
		},
		// MinLength
		{
			name: "MinLength works",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, MinLength(3))),
			},
			args: []string{"hello"},
			wantInput: &Input{
				args: []string{"hello"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("hello"),
				},
			},
		},
		{
			name: "MinLength works for exact count match",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, MinLength(3))),
			},
			args: []string{"hey"},
			wantInput: &Input{
				args: []string{"hey"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("hey"),
				},
			},
		},
		{
			name: "MinLength fails",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, MinLength(3))),
			},
			args: []string{"hi"},
			wantInput: &Input{
				args: []string{"hi"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("hi"),
				},
			},
			wantStderr: []string{`validation failed: [MinLength] value must be at least 3 characters`},
			wantErr:    fmt.Errorf(`validation failed: [MinLength] value must be at least 3 characters`),
		},
		// IntEQ
		{
			name: "IntEQ works",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntEQ(24))),
			},
			args: []string{"24"},
			wantInput: &Input{
				args: []string{"24"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(24),
				},
			},
		},
		{
			name: "IntEQ fails",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntEQ(24))),
			},
			args: []string{"25"},
			wantInput: &Input{
				args: []string{"25"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(25),
				},
			},
			wantStderr: []string{`validation failed: [IntEQ] value isn't equal to 24`},
			wantErr:    fmt.Errorf(`validation failed: [IntEQ] value isn't equal to 24`),
		},
		// IntNE
		{
			name: "IntNE works",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNE(24))),
			},
			args: []string{"25"},
			wantInput: &Input{
				args: []string{"25"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(25),
				},
			},
		},
		{
			name: "IntNE fails",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNE(24))),
			},
			args: []string{"24"},
			wantInput: &Input{
				args: []string{"24"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(24),
				},
			},
			wantStderr: []string{`validation failed: [IntNE] value isn't not equal to 24`},
			wantErr:    fmt.Errorf(`validation failed: [IntNE] value isn't not equal to 24`),
		},
		// IntLT
		{
			name: "IntLT works when less than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLT(25))),
			},
			args: []string{"24"},
			wantInput: &Input{
				args: []string{"24"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(24),
				},
			},
		},
		{
			name: "IntLT fails when equal to",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLT(25))),
			},
			args: []string{"25"},
			wantInput: &Input{
				args: []string{"25"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(25),
				},
			},
			wantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
			wantErr:    fmt.Errorf(`validation failed: [IntLT] value isn't less than 25`),
		},
		{
			name: "IntLT fails when greater than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLT(25))),
			},
			args: []string{"26"},
			wantInput: &Input{
				args: []string{"26"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(26),
				},
			},
			wantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
			wantErr:    fmt.Errorf(`validation failed: [IntLT] value isn't less than 25`),
		},
		// IntLTE
		{
			name: "IntLTE works when less than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLTE(25))),
			},
			args: []string{"24"},
			wantInput: &Input{
				args: []string{"24"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(24),
				},
			},
		},
		{
			name: "IntLTE works when equal to",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLTE(25))),
			},
			args: []string{"25"},
			wantInput: &Input{
				args: []string{"25"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(25),
				},
			},
		},
		{
			name: "IntLTE fails when greater than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLTE(25))),
			},
			args: []string{"26"},
			wantInput: &Input{
				args: []string{"26"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(26),
				},
			},
			wantStderr: []string{`validation failed: [IntLTE] value isn't less than or equal to 25`},
			wantErr:    fmt.Errorf(`validation failed: [IntLTE] value isn't less than or equal to 25`),
		},
		// IntGT
		{
			name: "IntGT fails when less than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGT(25))),
			},
			args: []string{"24"},
			wantInput: &Input{
				args: []string{"24"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(24),
				},
			},
			wantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
			wantErr:    fmt.Errorf(`validation failed: [IntGT] value isn't greater than 25`),
		},
		{
			name: "IntGT fails when equal to",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGT(25))),
			},
			args: []string{"25"},
			wantInput: &Input{
				args: []string{"25"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(25),
				},
			},
			wantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
			wantErr:    fmt.Errorf(`validation failed: [IntGT] value isn't greater than 25`),
		},
		{
			name: "IntGT works when greater than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGT(25))),
			},
			args: []string{"26"},
			wantInput: &Input{
				args: []string{"26"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(26),
				},
			},
		},
		// IntGTE
		{
			name: "IntGTE fails when less than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGTE(25))),
			},
			args: []string{"24"},
			wantInput: &Input{
				args: []string{"24"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(24),
				},
			},
			wantStderr: []string{`validation failed: [IntGTE] value isn't greater than or equal to 25`},
			wantErr:    fmt.Errorf(`validation failed: [IntGTE] value isn't greater than or equal to 25`),
		},
		{
			name: "IntGTE works when equal to",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGTE(25))),
			},
			args: []string{"25"},
			wantInput: &Input{
				args: []string{"25"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(25),
				},
			},
		},
		{
			name: "IntGTE works when greater than",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGTE(25))),
			},
			args: []string{"26"},
			wantInput: &Input{
				args: []string{"26"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(26),
				},
			},
		},
		// IntPositive
		{
			name: "IntPositive fails when negative",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntPositive())),
			},
			args: []string{"-1"},
			wantInput: &Input{
				args: []string{"-1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(-1),
				},
			},
			wantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
			wantErr:    fmt.Errorf(`validation failed: [IntPositive] value isn't positive`),
		},
		{
			name: "IntPositive fails when zero",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntPositive())),
			},
			args: []string{"0"},
			wantInput: &Input{
				args: []string{"0"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(0),
				},
			},
			wantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
			wantErr:    fmt.Errorf(`validation failed: [IntPositive] value isn't positive`),
		},
		{
			name: "IntPositive works when positive",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntPositive())),
			},
			args: []string{"1"},
			wantInput: &Input{
				args: []string{"1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(1),
				},
			},
		},
		// IntNegative
		{
			name: "IntNegative works when negative",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNegative())),
			},
			args: []string{"-1"},
			wantInput: &Input{
				args: []string{"-1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(-1),
				},
			},
		},
		{
			name: "IntNegative fails when zero",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNegative())),
			},
			args: []string{"0"},
			wantInput: &Input{
				args: []string{"0"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(0),
				},
			},
			wantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
			wantErr:    fmt.Errorf(`validation failed: [IntNegative] value isn't negative`),
		},
		{
			name: "IntNegative fails when positive",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNegative())),
			},
			args: []string{"1"},
			wantInput: &Input{
				args: []string{"1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(1),
				},
			},
			wantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
			wantErr:    fmt.Errorf(`validation failed: [IntNegative] value isn't negative`),
		},
		// IntNonNegative
		{
			name: "IntNonNegative fails when negative",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNonNegative())),
			},
			args: []string{"-1"},
			wantInput: &Input{
				args: []string{"-1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(-1),
				},
			},
			wantStderr: []string{`validation failed: [IntNonNegative] value isn't non-negative`},
			wantErr:    fmt.Errorf(`validation failed: [IntNonNegative] value isn't non-negative`),
		},
		{
			name: "IntNonNegative works when zero",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNonNegative())),
			},
			args: []string{"0"},
			wantInput: &Input{
				args: []string{"0"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(0),
				},
			},
		},
		{
			name: "IntNonNegative works when positive",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNonNegative())),
			},
			args: []string{"1"},
			wantInput: &Input{
				args: []string{"1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(1),
				},
			},
		},
		// FloatEQ
		{
			name: "FloatEQ works",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatEQ(2.4))),
			},
			args: []string{"2.4"},
			wantInput: &Input{
				args: []string{"2.4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
		},
		{
			name: "FloatEQ fails",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatEQ(2.4))),
			},
			args: []string{"2.5"},
			wantInput: &Input{
				args: []string{"2.5"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
			wantStderr: []string{`validation failed: [FloatEQ] value isn't equal to 2.40`},
			wantErr:    fmt.Errorf(`validation failed: [FloatEQ] value isn't equal to 2.40`),
		},
		// FloatNE
		{
			name: "FloatNE works",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNE(2.4))),
			},
			args: []string{"2.5"},
			wantInput: &Input{
				args: []string{"2.5"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
		},
		{
			name: "FloatNE fails",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNE(2.4))),
			},
			args: []string{"2.4"},
			wantInput: &Input{
				args: []string{"2.4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
			wantStderr: []string{`validation failed: [FloatNE] value isn't not equal to 2.40`},
			wantErr:    fmt.Errorf(`validation failed: [FloatNE] value isn't not equal to 2.40`),
		},
		// FloatLT
		{
			name: "FloatLT works when less than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
			},
			args: []string{"2.4"},
			wantInput: &Input{
				args: []string{"2.4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
		},
		{
			name: "FloatLT fails when equal to",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
			},
			args: []string{"2.5"},
			wantInput: &Input{
				args: []string{"2.5"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
			wantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
			wantErr:    fmt.Errorf(`validation failed: [FloatLT] value isn't less than 2.50`),
		},
		{
			name: "FloatLT fails when greater than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
			},
			args: []string{"2.6"},
			wantInput: &Input{
				args: []string{"2.6"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				},
			},
			wantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
			wantErr:    fmt.Errorf(`validation failed: [FloatLT] value isn't less than 2.50`),
		},
		// FloatLTE
		{
			name: "FloatLTE works when less than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLTE(2.5))),
			},
			args: []string{"2.4"},
			wantInput: &Input{
				args: []string{"2.4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
		},
		{
			name: "FloatLTE works when equal to",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLTE(2.5))),
			},
			args: []string{"2.5"},
			wantInput: &Input{
				args: []string{"2.5"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
		},
		{
			name: "FloatLTE fails when greater than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLTE(2.5))),
			},
			args: []string{"2.6"},
			wantInput: &Input{
				args: []string{"2.6"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				},
			},
			wantStderr: []string{`validation failed: [FloatLTE] value isn't less than or equal to 2.50`},
			wantErr:    fmt.Errorf(`validation failed: [FloatLTE] value isn't less than or equal to 2.50`),
		},
		// FloatGT
		{
			name: "FloatGT fails when less than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
			},
			args: []string{"2.4"},
			wantInput: &Input{
				args: []string{"2.4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
			wantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
			wantErr:    fmt.Errorf(`validation failed: [FloatGT] value isn't greater than 2.50`),
		},
		{
			name: "FloatGT fails when equal to",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
			},
			args: []string{"2.5"},
			wantInput: &Input{
				args: []string{"2.5"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
			wantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
			wantErr:    fmt.Errorf(`validation failed: [FloatGT] value isn't greater than 2.50`),
		},
		{
			name: "FloatGT works when greater than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
			},
			args: []string{"2.6"},
			wantInput: &Input{
				args: []string{"2.6"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				},
			},
		},
		// FloatGTE
		{
			name: "FloatGTE fails when less than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGTE(2.5))),
			},
			args: []string{"2.4"},
			wantInput: &Input{
				args: []string{"2.4"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
			wantStderr: []string{`validation failed: [FloatGTE] value isn't greater than or equal to 2.50`},
			wantErr:    fmt.Errorf(`validation failed: [FloatGTE] value isn't greater than or equal to 2.50`),
		},
		{
			name: "FloatGTE works when equal to",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGTE(2.5))),
			},
			args: []string{"2.5"},
			wantInput: &Input{
				args: []string{"2.5"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
		},
		{
			name: "FloatGTE works when greater than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGTE(2.5))),
			},
			args: []string{"2.6"},
			wantInput: &Input{
				args: []string{"2.6"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				},
			},
		},
		// FloatPositive
		{
			name: "FloatPositive fails when negative",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
			},
			args: []string{"-0.1"},
			wantInput: &Input{
				args: []string{"-0.1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				},
			},
			wantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
			wantErr:    fmt.Errorf(`validation failed: [FloatPositive] value isn't positive`),
		},
		{
			name: "FloatPositive fails when zero",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
			},
			args: []string{"0"},
			wantInput: &Input{
				args: []string{"0"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0),
				},
			},
			wantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
			wantErr:    fmt.Errorf(`validation failed: [FloatPositive] value isn't positive`),
		},
		{
			name: "FloatPositive works when positive",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
			},
			args: []string{"0.1"},
			wantInput: &Input{
				args: []string{"0.1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				},
			},
		},
		// FloatNegative
		{
			name: "FloatNegative works when negative",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNegative())),
			},
			args: []string{"-0.1"},
			wantInput: &Input{
				args: []string{"-0.1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				},
			},
		},
		{
			name: "FloatNegative fails when zero",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNegative())),
			},
			args: []string{"0"},
			wantInput: &Input{
				args: []string{"0"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0),
				},
			},
			wantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
			wantErr:    fmt.Errorf(`validation failed: [FloatNegative] value isn't negative`),
		},
		{
			name: "FloatNegative fails when positive",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNegative())),
			},
			args: []string{"0.1"},
			wantInput: &Input{
				args: []string{"0.1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				},
			},
			wantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
			wantErr:    fmt.Errorf(`validation failed: [FloatNegative] value isn't negative`),
		},
		// FloatNonNegative
		{
			name: "FloatNonNegative fails when negative",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
			},
			args: []string{"-0.1"},
			wantInput: &Input{
				args: []string{"-0.1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				},
			},
			wantStderr: []string{`validation failed: [FloatNonNegative] value isn't non-negative`},
			wantErr:    fmt.Errorf(`validation failed: [FloatNonNegative] value isn't non-negative`),
		},
		{
			name: "FloatNonNegative works when zero",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
			},
			args: []string{"0"},
			wantInput: &Input{
				args: []string{"0"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0),
				},
			},
		},
		{
			name: "FloatNonNegative works when positive",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
			},
			args: []string{"0.1"},
			wantInput: &Input{
				args: []string{"0.1"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				},
			},
		},
		// Flag nodes
		{
			name: "empty flag node works",
			node: &Node{Processor: NewFlagNode()},
		},
		{
			name: "flag node allows empty",
			node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
		},
		{
			name:       "flag node fails if no argument",
			node:       &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
			args:       []string{"--strFlag"},
			wantStderr: []string{"not enough arguments"},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantData: &Data{
				Values: map[string]*Value{
					"strFlag": StringValue(""),
				},
			},
			wantInput: &Input{
				args: []string{"--strFlag"},
			},
		},
		{
			name: "flag node parses flag",
			node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
			args: []string{"--strFlag", "hello"},
			wantData: &Data{
				Values: map[string]*Value{
					"strFlag": StringValue("hello"),
				},
			},
			wantInput: &Input{
				args: []string{"--strFlag", "hello"},
			},
		},
		{
			name: "flag node parses short name flag",
			node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
			args: []string{"-f", "hello"},
			wantData: &Data{
				Values: map[string]*Value{
					"strFlag": StringValue("hello"),
				},
			},
			wantInput: &Input{
				args: []string{"-f", "hello"},
			},
		},
		{
			name: "flag node parses flag in the middle",
			node: SerialNodes(
				NewFlagNode(StringFlag("strFlag", 'f', nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "--strFlag", "hello", "deux"},
			wantData: &Data{
				Values: map[string]*Value{
					"strFlag": StringValue("hello"),
					"filler":  StringListValue("un", "deux"),
				},
			},
			wantInput: &Input{
				args: []string{"un", "--strFlag", "hello", "deux"},
			},
		},
		{
			name: "flag node parses short name flag",
			node: SerialNodes(
				NewFlagNode(StringFlag("strFlag", 'f', nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"uno", "dos", "-f", "hello"},
			wantData: &Data{
				Values: map[string]*Value{
					"filler":  StringListValue("uno", "dos"),
					"strFlag": StringValue("hello"),
				},
			},
			wantInput: &Input{
				args: []string{"uno", "dos", "-f", "hello"},
			},
		},
		// Int flag
		{
			name: "parses int flag",
			node: SerialNodes(
				NewFlagNode(IntFlag("intFlag", 'f', nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "deux", "-f", "3", "quatre"},
			wantInput: &Input{
				args: []string{"un", "deux", "-f", "3", "quatre"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"filler":  StringListValue("un", "deux", "quatre"),
					"intFlag": IntValue(3),
				},
			},
		},
		{
			name: "handles invalid int flag value",
			node: SerialNodes(
				NewFlagNode(IntFlag("intFlag", 'f', nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "deux", "-f", "trois", "quatre"},
			wantInput: &Input{
				args:      []string{"un", "deux", "-f", "trois", "quatre"},
				remaining: []int{0, 1, 4},
			},
			wantStderr: []string{`strconv.Atoi: parsing "trois": invalid syntax`},
			wantErr:    fmt.Errorf(`strconv.Atoi: parsing "trois": invalid syntax`),
		},
		// Float flag
		{
			name: "parses float flag",
			node: SerialNodes(
				NewFlagNode(FloatFlag("floatFlag", 'f', nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"--floatFlag", "-1.2", "three"},
			wantInput: &Input{
				args: []string{"--floatFlag", "-1.2", "three"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"filler":    StringListValue("three"),
					"floatFlag": FloatValue(-1.2),
				},
			},
		},
		{
			name: "handles invalid float flag value",
			node: SerialNodes(
				NewFlagNode(FloatFlag("floatFlag", 'f', nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"--floatFlag", "twelve", "eleven"},
			wantInput: &Input{
				args:      []string{"--floatFlag", "twelve", "eleven"},
				remaining: []int{2},
			},
			wantStderr: []string{`strconv.ParseFloat: parsing "twelve": invalid syntax`},
			wantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
		},
		// Bool flag
		{
			name: "bool flag",
			node: SerialNodes(
				NewFlagNode(BoolFlag("boolFlag", 'b')),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"okay", "--boolFlag", "then"},
			wantInput: &Input{
				args: []string{"okay", "--boolFlag", "then"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"filler":   StringListValue("okay", "then"),
					"boolFlag": BoolValue(true),
				},
			},
		},
		{
			name: "short bool flag",
			node: SerialNodes(
				NewFlagNode(BoolFlag("boolFlag", 'b')),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"okay", "-b", "then"},
			wantInput: &Input{
				args: []string{"okay", "-b", "then"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"filler":   StringListValue("okay", "then"),
					"boolFlag": BoolValue(true),
				},
			},
		},
		// flag list tests
		{
			name: "flag list works",
			node: SerialNodes(
				NewFlagNode(StringListFlag("slFlag", 's', 2, 3, nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "--slFlag", "hello", "there"},
			wantInput: &Input{
				args: []string{"un", "--slFlag", "hello", "there"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"filler": StringListValue("un"),
					"slFlag": StringListValue("hello", "there"),
				},
			},
		},
		{
			name: "flag list fails if not enough",
			node: SerialNodes(
				NewFlagNode(StringListFlag("slFlag", 's', 2, 3, nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "--slFlag", "hello"},
			wantInput: &Input{
				args:      []string{"un", "--slFlag", "hello"},
				remaining: []int{0},
			},
			wantStderr: []string{"not enough arguments"},
			wantErr:    fmt.Errorf("not enough arguments"),
			wantData: &Data{
				Values: map[string]*Value{
					"slFlag": StringListValue("hello"),
				},
			},
		},
		// Int list
		{
			name: "int list works",
			node: SerialNodes(
				NewFlagNode(IntListFlag("ilFlag", 'i', 2, 3, nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "-i", "2", "4", "8", "16", "32", "64"},
			wantInput: &Input{
				args: []string{"un", "-i", "2", "4", "8", "16", "32", "64"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"filler": StringListValue("un", "64"),
					"ilFlag": IntListValue(2, 4, 8, 16, 32),
				},
			},
		},
		{
			name: "int list transform failure",
			node: SerialNodes(
				NewFlagNode(IntListFlag("ilFlag", 'i', 2, 3, nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "-i", "2", "4", "8", "16.0", "32", "64"},
			wantInput: &Input{
				args:      []string{"un", "-i", "2", "4", "8", "16.0", "32", "64"},
				remaining: []int{0, 7},
			},
			wantStderr: []string{`strconv.Atoi: parsing "16.0": invalid syntax`},
			wantErr:    fmt.Errorf(`strconv.Atoi: parsing "16.0": invalid syntax`),
		},
		// Float list
		{
			name: "float list works",
			node: SerialNodes(
				NewFlagNode(FloatListFlag("flFlag", 'f', 0, 3, nil)),
				StringListNode("filler", 1, 3, nil),
			),
			args: []string{"un", "-f", "2", "-4.4", "0.8", "16.16", "-32", "64"},
			wantInput: &Input{
				args: []string{"un", "-f", "2", "-4.4", "0.8", "16.16", "-32", "64"},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"filler": StringListValue("un", "16.16", "-32", "64"),
					"flFlag": FloatListValue(2, -4.4, 0.8),
				},
			},
		},
		{
			name: "float list transform failure",
			node: SerialNodes(
				NewFlagNode(FloatListFlag("flFlag", 'f', 0, 3, nil)),
				StringListNode("filler", 1, 2, nil),
			),
			args: []string{"un", "--flFlag", "2", "4", "eight", "16.0", "32", "64"},
			wantInput: &Input{
				args:      []string{"un", "--flFlag", "2", "4", "eight", "16.0", "32", "64"},
				remaining: []int{0, 5, 6, 7},
			},
			wantStderr: []string{`strconv.ParseFloat: parsing "eight": invalid syntax`},
			wantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "eight": invalid syntax`),
		},
		// Misc. flag tests
		{
			name: "processes multiple flags",
			node: SerialNodes(
				NewFlagNode(
					FloatListFlag("coordinates", 'c', 2, 0, nil),
					BoolFlag("boo", 'o'),
					StringListFlag("names", 'n', 1, 2, nil),
					IntFlag("rating", 'r', nil),
				),
				StringListNode("extra", 0, 10, nil),
			),
			args: []string{"its", "--boo", "a", "-r", "9", "secret", "-n", "greggar", "groog", "beggars", "--coordinates", "2.2", "4.4", "message."},
			wantInput: &Input{
				args: []string{"its", "--boo", "a", "-r", "9", "secret", "-n", "greggar", "groog", "beggars", "--coordinates", "2.2", "4.4", "message."},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"boo":         BoolValue(true),
					"extra":       StringListValue("its", "a", "secret", "message."),
					"names":       StringListValue("greggar", "groog", "beggars"),
					"coordinates": FloatListValue(2.2, 4.4),
					"rating":      IntValue(9),
				},
			},
		},
		// BranchNode tests
		{
			name: "branch node requires branch argument",
			node: BranchNode(map[string]*Node{
				"h": printNode("hello"),
				"b": printNode("goodbye"),
			}, nil),
			wantStderr: []string{"branching argument required"},
			wantErr:    fmt.Errorf("branching argument required"),
		},
		{
			name: "branch node requires matching branch argument",
			node: BranchNode(map[string]*Node{
				"h": printNode("hello"),
				"b": printNode("goodbye"),
			}, nil),
			args:       []string{"uh"},
			wantStderr: []string{"argument must be one of [b h]"},
			wantErr:    fmt.Errorf("argument must be one of [b h]"),
			wantInput: &Input{
				args:      []string{"uh"},
				remaining: []int{0},
			},
		},
		{
			name: "branch node forwards to proper node",
			node: BranchNode(map[string]*Node{
				"h": printNode("hello"),
				"b": printNode("goodbye"),
			}, nil),
			args:       []string{"h"},
			wantStdout: []string{"hello"},
			wantInput: &Input{
				args: []string{"h"},
			},
		},
		{
			name: "branch node forwards to default if none provided",
			node: BranchNode(map[string]*Node{
				"h": printNode("hello"),
				"b": printNode("goodbye"),
			}, printNode("default")),
			wantStdout: []string{"default"},
		},
		{
			name: "branch node forwards to default if unknown provided",
			node: BranchNode(map[string]*Node{
				"h": printNode("hello"),
				"b": printNode("goodbye"),
			}, SerialNodes(StringListNode("sl", 0, UnboundedList, nil), printArgsNode().Processor)),
			args:       []string{"good", "morning"},
			wantStdout: []string{"sl: good, morning"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("good", "morning"),
				},
			},
			wantInput: &Input{
				args: []string{"good", "morning"},
			},
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
			if diff := cmp.Diff(we, eData, cmpopts.IgnoreFields(ExecuteData{}, "Executor")); diff != "" {
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

			if diff := cmp.Diff(test.wantStdout, fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("execute(%v) sent wrong data to stdout (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("execute(%v) sent wrong data to stderr (-want, +got):\n%s", test.args, diff)
			}
		})
	}
}

func TestComplete(t *testing.T) {
	for _, test := range []struct {
		name           string
		node           *Node
		args           []string
		filepathAbs    string
		filepathAbsErr error
		want           []string
		wantData       *Data
	}{
		// Basic tests
		{
			name: "empty graph",
			node: &Node{},
		},
		{
			name: "returns suggestions of first node if empty",
			node: SerialNodes(
				StringNode("s", NewArgOpt(SimpleCompletor("un", "deux", "trois"), nil)),
				OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
				StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
			),
			want: []string{"deux", "trois", "un"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue(""),
				},
			},
		},
		{
			name: "returns suggestions of first node if up to first arg",
			node: SerialNodes(
				StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
				OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
				StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
			),
			args: []string{"t"},
			want: []string{"three", "two"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("t"),
				},
			},
		},
		{
			name: "returns suggestions of middle node if that's where we're at",
			node: SerialNodes(
				StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
				StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
			),
			args: []string{"three", ""},
			want: []string{"dos", "uno"},
			wantData: &Data{
				Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue(""),
				},
			},
		},
		{
			name: "returns suggestions of middle node if partial",
			node: SerialNodes(
				StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
				StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
			),
			args: []string{"three", "d"},
			want: []string{"dos"},
			wantData: &Data{
				Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("d"),
				},
			},
		},
		{
			name: "returns suggestions in list",
			node: SerialNodes(
				StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
				StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
			),
			args: []string{"three", "dos", ""},
			want: []string{"dos", "uno"},
			wantData: &Data{
				Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("dos", ""),
				},
			},
		},
		/*{
			name: "returns suggestions for last arg",
			node: SerialNodes(
				StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
				StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
			),
			args: []string{"three", "uno", "dos", ""},
			want: []string{"1", "2"},
			wantData: &Data{
				Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("uno", "dos"),
					"i":  IntValue(0),
				},
			},
		},*/
		{
			name: "returns nothing if iterate through all nodes",
			node: SerialNodes(
				StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
				StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
			),
			args: []string{"three", "uno", "dos", "1", "what", "now"},
			wantData: &Data{
				Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("uno", "dos"),
					"i":  IntValue(1),
				},
			},
		},
		{
			name: "works if empty and list starts",
			node: SerialNodes(
				StringListNode("sl", 1, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
			),
			want: []string{"dos", "uno"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue(""),
				},
			},
		},
		{
			name: "only returns suggestions matching prefix",
			node: SerialNodes(
				StringListNode("sl", 1, 2, NewArgOpt(SimpleCompletor("zzz-1", "zzz-2", "yyy-3", "zzz-4"), nil)),
			),
			args: []string{"zz"},
			want: []string{"zzz-1", "zzz-2", "zzz-4"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("zz"),
				},
			},
		},
		// Flag completion
		{
			name: "flag name gets completed if single hyphen at end",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"-"},
			want: []string{"--good", "--greeting", "--names", "-g", "-h", "-n"},
		},
		{
			name: "flag name gets completed if double hyphen at end",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"--"},
			want: []string{"--good", "--greeting", "--names"},
		},
		{
			name: "flag name gets completed if it's the only arg",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"1", "-"},
			want: []string{"--good", "--greeting", "--names", "-g", "-h", "-n"},
		},
		{
			name: "completes for single flag",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"1", "--greeting", "h"},
			want: []string{"hey", "hi"},
			wantData: &Data{
				Values: map[string]*Value{
					"greeting": StringValue("h"),
				},
			},
		},
		{
			name: "completes for single short flag",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"1", "-h", "he"},
			want: []string{"hey"},
			wantData: &Data{
				Values: map[string]*Value{
					"greeting": StringValue("he"),
				},
			},
		},
		{
			name: "completes for list flag",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"1", "-h", "hey", "other", "--names", ""},
			want: []string{"johnny", "ralph", "renee"},
			wantData: &Data{
				Values: map[string]*Value{
					"greeting": StringValue("hey"),
					"names":    StringListValue(""),
				},
			},
		},
		{
			name: "completes distinct secondary for list flag",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleDistinctCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"1", "-h", "hey", "other", "--names", "ralph", ""},
			want: []string{"johnny", "renee"},
			wantData: &Data{
				Values: map[string]*Value{
					"greeting": StringValue("hey"),
					"names":    StringListValue("ralph", ""),
				},
			},
		},
		{
			name: "completes last flag",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleDistinctCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
					FloatFlag("float", 'f', NewArgOpt(SimpleCompletor("1.23", "12.3", "123.4"), nil)),
				),
				IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
			),
			args: []string{"1", "-h", "hey", "other", "--names", "ralph", "renee", "johnny", "-f", ""},
			want: []string{"1.23", "12.3", "123.4"},
			wantData: &Data{
				Values: map[string]*Value{
					"greeting": StringValue("hey"),
					"names":    StringListValue("ralph", "renee", "johnny"),
					"float":    FloatValue(0),
				},
			},
		},
		{
			name: "completes arg if flag arg isn't at the end",
			node: SerialNodes(
				NewFlagNode(
					StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
					StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleDistinctCompletor("ralph", "johnny", "renee"), nil)),
					BoolFlag("good", 'g'),
				),
				StringListNode("i", 1, 2, NewArgOpt(SimpleCompletor("hey", "ooo"), nil)),
			),
			args: []string{"1", "-h", "hello", "beta", "--names", "ralph", "renee", "johnny", ""},
			want: []string{"hey", "ooo"},
			wantData: &Data{
				Values: map[string]*Value{
					"i":        StringListValue("1", "beta", ""),
					"greeting": StringValue("hello"),
					"names":    StringListValue("ralph", "renee", "johnny"),
				},
			},
		},
		// Transformer arg tests.
		{
			name: "handles nil option",
			node: &Node{
				Processor: StringNode("strArg", nil),
			},
			args: []string{"abc"},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("abc"),
				},
			},
		},
		{
			name: "list handles nil option",
			node: &Node{
				Processor: StringListNode("slArg", 1, 2, nil),
			},
			args: []string{"abc"},
			wantData: &Data{
				Values: map[string]*Value{
					"slArg": StringListValue("abc"),
				},
			},
		},
		{
			name: "transformer transforms values",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, FileTransformer())),
			},
			args: []string{filepath.Join("relative", "path.txt")},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue(filepath.Join("abso", "lutely", "relative", "path.txt")),
				},
			},
			filepathAbs: filepath.Join("abso", "lutely"),
		},
		{
			name: "handles transform error",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, FileTransformer())),
			},
			args: []string{filepath.Join("relative", "path.txt")},
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue(filepath.Join("relative", "path.txt")),
				},
			},
			filepathAbsErr: fmt.Errorf("bad news bears"),
		},
		{
			name: "handles transformer of incorrect type",
			node: &Node{
				Processor: IntNode("IntNode", NewArgOpt(nil, FileTransformer())),
			},
			args: []string{"123"},
			wantData: &Data{
				Values: map[string]*Value{
					"IntNode": IntValue(123),
				},
			},
		},
		{
			name: "transformer list transforms values",
			node: &Node{
				Processor: StringListNode("slArg", 1, 2, NewArgOpt(nil, FileListTransformer())),
			},
			args: []string{
				filepath.Join("relative", "path.txt"),
				filepath.Join("other.txt"),
			},
			wantData: &Data{
				Values: map[string]*Value{
					"slArg": StringListValue(
						filepath.Join("abso", "lutely", "relative", "path.txt"),
						filepath.Join("abso", "lutely", "other.txt"),
					),
				},
			},
			filepathAbs: filepath.Join("abso", "lutely"),
		},
		{
			name: "handles transform list error",
			node: &Node{
				Processor: StringListNode("slArg", 1, 2, NewArgOpt(nil, FileListTransformer())),
			},
			args: []string{
				filepath.Join("relative", "path.txt"),
				filepath.Join("other.txt"),
			},
			wantData: &Data{
				Values: map[string]*Value{
					"slArg": StringListValue(
						filepath.Join("relative", "path.txt"),
						filepath.Join("other.txt"),
					),
				},
			},
			filepathAbsErr: fmt.Errorf("bad news bears"),
		},
		{
			name: "handles list transformer of incorrect type",
			node: &Node{
				Processor: StringListNode("slArg", 1, 2, NewArgOpt(nil, FileTransformer())),
			},
			args: []string{"123"},
			wantData: &Data{
				Values: map[string]*Value{
					"slArg": StringListValue("123"),
				},
			},
		},
		// BranchNode completion tests.
		{
			name: "completes branch options",
			node: BranchNode(map[string]*Node{
				"a":     {},
				"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
				"bravo": {},
			}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts"), nil)))),
			want: []string{"a", "alpha", "bravo", "command", "default", "opts"},
			wantData: &Data{
				Values: map[string]*Value{
					"default": StringListValue(""),
				},
			},
		},
		{
			name: "completes for specific branch",
			node: BranchNode(map[string]*Node{
				"a":     {},
				"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
				"bravo": {},
			}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts"), nil)))),
			args: []string{"alpha", ""},
			want: []string{"other", "stuff"},
			wantData: &Data{
				Values: map[string]*Value{
					"hello": StringValue(""),
				},
			},
		},
		{
			name: "branch node doesn't complete if no default and no branch match",
			node: BranchNode(map[string]*Node{
				"a":     {},
				"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
				"bravo": {},
			}, nil),
			args: []string{"some", "thing", "else"},
		},
		{
			name: "completes branch options with partial completion",
			node: BranchNode(map[string]*Node{
				"a":     {},
				"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
				"bravo": {},
			}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts", "ahhhh"), nil)))),
			args: []string{"a"},
			want: []string{"a", "ahhhh", "alpha"},
			wantData: &Data{
				Values: map[string]*Value{
					"default": StringListValue("a"),
				},
			},
		},
		{
			name: "completes default options",
			node: BranchNode(map[string]*Node{
				"a":     {},
				"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
				"bravo": {},
			}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts"), nil)))),
			args: []string{"something", ""},
			wantData: &Data{
				Values: map[string]*Value{
					"default": StringListValue("something", ""),
				},
			},
			want: []string{"command", "default", "opts"},
		},
		// Commands with different value types.
		{
			name: "int arg gets completed",
			node: SerialNodes(IntNode("iArg", NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
			args: []string{"4"},
			want: []string{"45", "456", "468"},
			wantData: &Data{
				Values: map[string]*Value{
					"iArg": IntValue(4),
				},
			},
		},
		{
			name: "optional int arg gets completed",
			node: SerialNodes(OptionalIntNode("iArg", NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
			args: []string{"4"},
			want: []string{"45", "456", "468"},
			wantData: &Data{
				Values: map[string]*Value{
					"iArg": IntValue(4),
				},
			},
		},
		{
			name: "int list arg gets completed",
			node: SerialNodes(IntListNode("iArg", 2, 3, NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
			args: []string{"1", "4"},
			want: []string{"45", "456", "468"},
			wantData: &Data{
				Values: map[string]*Value{
					"iArg": IntListValue(1, 4),
				},
			},
		},
		{
			name: "int list arg gets completed if previous one was invalid",
			node: SerialNodes(IntListNode("iArg", 2, 3, NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
			args: []string{"one", "4"},
			want: []string{"45", "456", "468"},
			wantData: &Data{
				Values: map[string]*Value{
					"iArg": IntListValue(0, 4),
				},
			},
		},
		{
			name: "int list arg optional args get completed",
			node: SerialNodes(IntListNode("iArg", 2, 3, NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
			args: []string{"1", "2", "3", "4"},
			want: []string{"45", "456", "468"},
			wantData: &Data{
				Values: map[string]*Value{
					"iArg": IntListValue(1, 2, 3, 4),
				},
			},
		},
		{
			name: "float arg gets completed",
			node: SerialNodes(FloatNode("fArg", NewArgOpt(SimpleCompletor("12", "4.5", "45.6", "468", "7"), nil))),
			args: []string{"4"},
			want: []string{"4.5", "45.6", "468"},
			wantData: &Data{
				Values: map[string]*Value{
					"fArg": FloatValue(4),
				},
			},
		},
		{
			name: "float list arg gets completed",
			node: SerialNodes(FloatListNode("fArg", 1, 2, NewArgOpt(SimpleCompletor("12", "4.5", "45.6", "468", "7"), nil))),
			want: []string{"12", "4.5", "45.6", "468", "7"},
			wantData: &Data{
				Values: map[string]*Value{
					"fArg": FloatListValue(0),
				},
			},
		},
		/*{
			name: "bool arg gets completed",
			node: SerialNodes(BoolArg("bArg", true)),
			want: []string{"0", "1", "F", "FALSE", "False", "T", "TRUE", "True", "f", "false", "t", "true"},
			wantData: &Data{
				Values: map[string]*Value{
					"bArg": BoolValue(false),
				},
			},
		},
		/* Useful comment for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			fmt.Println("==========", test.name)
			oldAbs := filepathAbs
			filepathAbs = func(s string) (string, error) {
				return filepath.Join(test.filepathAbs, s), test.filepathAbsErr
			}
			defer func() { filepathAbs = oldAbs }()

			// TODO: remove test.wantOK
			got := Autocomplete(test.node, test.args)

			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Autocomplete(%v) produced incorrect completions (-want, +got):\n%s", test.args, diff)
			}
		})
	}
}

func printNode(s string) *Node {
	return &Node{
		Processor: ExecutorNode(func(output Output, _ *Data) error {
			output.Stdout(s)
			return nil
		}),
	}
}

func printArgsNode() *Node {
	return &Node{
		Processor: ExecutorNode(func(output Output, data *Data) error {
			for k, v := range data.Values {
				output.Stdout("%s: %s", k, v.Str())
			}
			return nil
		}),
	}
}
