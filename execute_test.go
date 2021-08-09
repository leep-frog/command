package command

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type errorEdge struct {
	e error
}

func (ee *errorEdge) Next(*Input, *Data) (*Node, error) {
	return nil, ee.e
}

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *ExecuteTestCase
	}{
		{
			name: "handles nil node",
		},
		{
			name: "fails if unprocessed args",
			etc: &ExecuteTestCase{
				Args:       []string{"hello"},
				WantErr:    fmt.Errorf("Unprocessed extra args: [hello]"),
				WantStderr: []string{"Unprocessed extra args: [hello]"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
					remaining: []int{0},
				},
			},
		},
		// Single arg tests.
		{
			name: "Fails if arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(StringNode("s", nil)),
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		{
			name: "Fails if edge fails",
			etc: &ExecuteTestCase{
				Args: []string{"hello"},
				Node: &Node{
					Processor: StringNode("s", nil),
					Edge: &errorEdge{
						e: fmt.Errorf("bad news bears"),
					},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantErr: fmt.Errorf("bad news bears"),
				WantData: &Data{
					Values: map[string]*Value{
						"s": StringValue("hello"),
					},
				},
			},
		},
		{
			name: "Fails if int arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(IntNode("i", nil)),
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		{
			name: "Fails if float arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(FloatNode("f", nil)),
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		{
			name: "Processes single string arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringNode("s", nil)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"s": StringValue("hello"),
					},
				},
			},
		},
		{
			name: "Processes single int arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntNode("i", nil)),
				Args: []string{"123"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"i": IntValue(123),
					},
				},
			},
		},
		{
			name: "Int arg fails if not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntNode("i", nil)),
				Args: []string{"12.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "12.3"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "12.3": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "12.3": invalid syntax`},
			},
		},
		{
			name: "Processes single float arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatNode("f", nil)),
				Args: []string{"-12.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-12.3"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"f": FloatValue(-12.3),
					},
				},
			},
		},
		{
			name: "Float arg fails if not a float",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatNode("f", nil)),
				Args: []string{"twelve"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "twelve"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
				WantStderr: []string{`strconv.ParseFloat: parsing "twelve": invalid syntax`},
			},
		},
		// List args
		{
			name: "List fails if not enough args",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 1, nil)),
				Args: []string{"hello", "there", "sir"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "sir"},
					},
					remaining: []int{2},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("hello", "there"),
					},
				},
				WantErr:    fmt.Errorf("Unprocessed extra args: [sir]"),
				WantStderr: []string{"Unprocessed extra args: [sir]"},
			},
		},
		{
			name: "Processes string list if minimum provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, nil)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("hello"),
					},
				},
			},
		},
		{
			name: "Processes string list if some optional provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, nil)),
				Args: []string{"hello", "there"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("hello", "there"),
					},
				},
			},
		},
		{
			name: "Processes string list if max args provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, 2, nil)),
				Args: []string{"hello", "there", "maam"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "maam"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("hello", "there", "maam"),
					},
				},
			},
		},
		{
			name: "Unbounded string list fails if less than min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 4, UnboundedList, nil)),
				Args: []string{"hello", "there", "kenobi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "kenobi"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("hello", "there", "kenobi"),
					},
				},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		{
			name: "Processes unbounded string list if min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, nil)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("hello"),
					},
				},
			},
		},
		{
			name: "Processes unbounded string list if more than min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", 1, UnboundedList, nil)),
				Args: []string{"hello", "there", "kenobi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "kenobi"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("hello", "there", "kenobi"),
					},
				},
			},
		},
		{
			name: "Processes int list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", 1, 2, nil)),
				Args: []string{"1", "-23"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
						{value: "-23"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"il": IntListValue(1, -23),
					},
				},
			},
		},
		{
			name: "Int list fails if an arg isn't an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", 1, 2, nil)),
				Args: []string{"1", "four", "-23"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
						{value: "four"},
						{value: "-23"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "four": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "four": invalid syntax`},
			},
		},
		{
			name: "Processes float list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatListNode("fl", 1, 2, nil)),
				Args: []string{"0.1", "-2.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
						{value: "-2.3"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"fl": FloatListValue(0.1, -2.3),
					},
				},
			},
		},
		{
			name: "Float list fails if an arg isn't an float",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatListNode("fl", 1, 2, nil)),
				Args: []string{"0.1", "four", "-23"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
						{value: "four"},
						{value: "-23"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "four": invalid syntax`),
				WantStderr: []string{`strconv.ParseFloat: parsing "four": invalid syntax`},
			},
		},
		// Multiple args
		{
			name: "Processes multiple args",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil)),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"il": IntListValue(0, 1),
						"s":  StringValue("two"),
						"fl": FloatListValue(0.3, -4),
					},
				},
			},
		},
		{
			name: "Fails if extra args when multiple",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil)),
				Args: []string{"0", "1", "two", "0.3", "-4", "0.5", "6"},
				wantInput: &Input{
					remaining: []int{6},
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
						{value: "0.5"},
						{value: "6"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"il": IntListValue(0, 1),
						"s":  StringValue("two"),
						"fl": FloatListValue(0.3, -4, 0.5),
					},
				},
				WantErr:    fmt.Errorf("Unprocessed extra args: [6]"),
				WantStderr: []string{"Unprocessed extra args: [6]"},
			},
		},
		// Executor tests.
		{
			name: "executes with proper data",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil), ExecutorNode(func(o Output, d *Data) error {
					var keys []string
					for k := range d.Values {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					for _, k := range keys {
						o.Stdout("%s: %s", k, d.Str(k))
					}
					return nil
				})),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"il": IntListValue(0, 1),
						"s":  StringValue("two"),
						"fl": FloatListValue(0.3, -4),
					},
				},
				WantStdout: []string{
					"fl: 0.30, -4.00",
					"il: 0, 1",
					"s: two",
				},
			},
		},
		{
			name: "executor error is returned",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", 2, 0, nil), StringNode("s", nil), FloatListNode("fl", 1, 2, nil), ExecutorNode(func(o Output, d *Data) error {
					return o.Stderr("bad news bears")
				})),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"il": IntListValue(0, 1),
						"s":  StringValue("two"),
						"fl": FloatListValue(0.3, -4),
					},
				},
				WantStderr: []string{"bad news bears"},
				WantErr:    fmt.Errorf("bad news bears"),
			},
		},
		// ArgValidator tests
		{
			name: "breaks when arg option is for invalid type",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, nil, IntEQ(123))),
				},
				Args: []string{"123"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("123"),
					},
				},
				WantStderr: []string{"validation failed: option can only be bound to arguments with type 3"},
				WantErr:    fmt.Errorf("validation failed: option can only be bound to arguments with type 3"),
			},
		},
		// Contains
		{
			name: "contains works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, nil, Contains("good"))),
				},
				Args: []string{"goodbye"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "goodbye"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("goodbye"),
					},
				},
			},
		},
		{
			name: "contains fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, nil, Contains("good"))),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("hello"),
					},
				},
				WantStderr: []string{`validation failed: [Contains] value doesn't contain substring "good"`},
				WantErr:    fmt.Errorf(`validation failed: [Contains] value doesn't contain substring "good"`),
			},
		},
		// MinLength
		{
			name: "MinLength works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, nil, MinLength(3))),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("hello"),
					},
				},
			},
		},
		{
			name: "MinLength works for exact count match",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, nil, MinLength(3))),
				},
				Args: []string{"hey"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hey"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("hey"),
					},
				},
			},
		},
		{
			name: "MinLength fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, nil, MinLength(3))),
				},
				Args: []string{"hi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hi"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("hi"),
					},
				},
				WantStderr: []string{`validation failed: [MinLength] value must be at least 3 characters`},
				WantErr:    fmt.Errorf(`validation failed: [MinLength] value must be at least 3 characters`),
			},
		},
		// IntEQ
		{
			name: "IntEQ works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntEQ(24))),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(24),
					},
				},
			},
		},
		{
			name: "IntEQ fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntEQ(24))),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(25),
					},
				},
				WantStderr: []string{`validation failed: [IntEQ] value isn't equal to 24`},
				WantErr:    fmt.Errorf(`validation failed: [IntEQ] value isn't equal to 24`),
			},
		},
		// IntNE
		{
			name: "IntNE works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNE(24))),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(25),
					},
				},
			},
		},
		{
			name: "IntNE fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNE(24))),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(24),
					},
				},
				WantStderr: []string{`validation failed: [IntNE] value isn't not equal to 24`},
				WantErr:    fmt.Errorf(`validation failed: [IntNE] value isn't not equal to 24`),
			},
		},
		// IntLT
		{
			name: "IntLT works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLT(25))),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(24),
					},
				},
			},
		},
		{
			name: "IntLT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLT(25))),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(25),
					},
				},
				WantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntLT] value isn't less than 25`),
			},
		},
		{
			name: "IntLT fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLT(25))),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(26),
					},
				},
				WantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntLT] value isn't less than 25`),
			},
		},
		// IntLTE
		{
			name: "IntLTE works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLTE(25))),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(24),
					},
				},
			},
		},
		{
			name: "IntLTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLTE(25))),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(25),
					},
				},
			},
		},
		{
			name: "IntLTE fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntLTE(25))),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(26),
					},
				},
				WantStderr: []string{`validation failed: [IntLTE] value isn't less than or equal to 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntLTE] value isn't less than or equal to 25`),
			},
		},
		// IntGT
		{
			name: "IntGT fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGT(25))),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(24),
					},
				},
				WantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntGT] value isn't greater than 25`),
			},
		},
		{
			name: "IntGT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGT(25))),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(25),
					},
				},
				WantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntGT] value isn't greater than 25`),
			},
		},
		{
			name: "IntGT works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGT(25))),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(26),
					},
				},
			},
		},
		// IntGTE
		{
			name: "IntGTE fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGTE(25))),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(24),
					},
				},
				WantStderr: []string{`validation failed: [IntGTE] value isn't greater than or equal to 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntGTE] value isn't greater than or equal to 25`),
			},
		},
		{
			name: "IntGTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGTE(25))),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(25),
					},
				},
			},
		},
		{
			name: "IntGTE works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntGTE(25))),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(26),
					},
				},
			},
		},
		// IntPositive
		{
			name: "IntPositive fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntPositive())),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(-1),
					},
				},
				WantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [IntPositive] value isn't positive`),
			},
		},
		{
			name: "IntPositive fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntPositive())),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(0),
					},
				},
				WantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [IntPositive] value isn't positive`),
			},
		},
		{
			name: "IntPositive works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntPositive())),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(1),
					},
				},
			},
		},
		// IntNegative
		{
			name: "IntNegative works when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNegative())),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(-1),
					},
				},
			},
		},
		{
			name: "IntNegative fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNegative())),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(0),
					},
				},
				WantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [IntNegative] value isn't negative`),
			},
		},
		{
			name: "IntNegative fails when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNegative())),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(1),
					},
				},
				WantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [IntNegative] value isn't negative`),
			},
		},
		// IntNonNegative
		{
			name: "IntNonNegative fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNonNegative())),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(-1),
					},
				},
				WantStderr: []string{`validation failed: [IntNonNegative] value isn't non-negative`},
				WantErr:    fmt.Errorf(`validation failed: [IntNonNegative] value isn't non-negative`),
			},
		},
		{
			name: "IntNonNegative works when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNonNegative())),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(0),
					},
				},
			},
		},
		{
			name: "IntNonNegative works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, nil, IntNonNegative())),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(1),
					},
				},
			},
		},
		// FloatEQ
		{
			name: "FloatEQ works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatEQ(2.4))),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.4),
					},
				},
			},
		},
		{
			name: "FloatEQ fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatEQ(2.4))),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.5),
					},
				},
				WantStderr: []string{`validation failed: [FloatEQ] value isn't equal to 2.40`},
				WantErr:    fmt.Errorf(`validation failed: [FloatEQ] value isn't equal to 2.40`),
			},
		},
		// FloatNE
		{
			name: "FloatNE works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNE(2.4))),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.5),
					},
				},
			},
		},
		{
			name: "FloatNE fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNE(2.4))),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.4),
					},
				},
				WantStderr: []string{`validation failed: [FloatNE] value isn't not equal to 2.40`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNE] value isn't not equal to 2.40`),
			},
		},
		// FloatLT
		{
			name: "FloatLT works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.4),
					},
				},
			},
		},
		{
			name: "FloatLT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.5),
					},
				},
				WantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatLT] value isn't less than 2.50`),
			},
		},
		{
			name: "FloatLT fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.6),
					},
				},
				WantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatLT] value isn't less than 2.50`),
			},
		},
		// FloatLTE
		{
			name: "FloatLTE works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLTE(2.5))),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.4),
					},
				},
			},
		},
		{
			name: "FloatLTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLTE(2.5))),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.5),
					},
				},
			},
		},
		{
			name: "FloatLTE fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLTE(2.5))),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.6),
					},
				},
				WantStderr: []string{`validation failed: [FloatLTE] value isn't less than or equal to 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatLTE] value isn't less than or equal to 2.50`),
			},
		},
		// FloatGT
		{
			name: "FloatGT fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.4),
					},
				},
				WantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatGT] value isn't greater than 2.50`),
			},
		},
		{
			name: "FloatGT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.5),
					},
				},
				WantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatGT] value isn't greater than 2.50`),
			},
		},
		{
			name: "FloatGT works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.6),
					},
				},
			},
		},
		// FloatGTE
		{
			name: "FloatGTE fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGTE(2.5))),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.4),
					},
				},
				WantStderr: []string{`validation failed: [FloatGTE] value isn't greater than or equal to 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatGTE] value isn't greater than or equal to 2.50`),
			},
		},
		{
			name: "FloatGTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGTE(2.5))),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.5),
					},
				},
			},
		},
		{
			name: "FloatGTE works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGTE(2.5))),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(2.6),
					},
				},
			},
		},
		// FloatPositive
		{
			name: "FloatPositive fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(-0.1),
					},
				},
				WantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [FloatPositive] value isn't positive`),
			},
		},
		{
			name: "FloatPositive fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(0),
					},
				},
				WantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [FloatPositive] value isn't positive`),
			},
		},
		{
			name: "FloatPositive works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(0.1),
					},
				},
			},
		},
		// FloatNegative
		{
			name: "FloatNegative works when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNegative())),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(-0.1),
					},
				},
			},
		},
		{
			name: "FloatNegative fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNegative())),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(0),
					},
				},
				WantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNegative] value isn't negative`),
			},
		},
		{
			name: "FloatNegative fails when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNegative())),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(0.1),
					},
				},
				WantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNegative] value isn't negative`),
			},
		},
		// FloatNonNegative
		{
			name: "FloatNonNegative fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(-0.1),
					},
				},
				WantStderr: []string{`validation failed: [FloatNonNegative] value isn't non-negative`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNonNegative] value isn't non-negative`),
			},
		},
		{
			name: "FloatNonNegative works when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(0),
					},
				},
			},
		},
		{
			name: "FloatNonNegative works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"flArg": FloatValue(0.1),
					},
				},
			},
		},
		// Flag nodes
		{
			name: "empty flag node works",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode()},
			},
		},
		{
			name: "flag node allows empty",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
			},
		},
		{
			name: "flag node fails if no argument",
			etc: &ExecuteTestCase{
				Node:       &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
				Args:       []string{"--strFlag"},
				WantStderr: []string{"not enough arguments"},
				WantErr:    fmt.Errorf("not enough arguments"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "--strFlag"},
					},
				},
			},
		},
		{
			name: "flag node parses flag",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
				Args: []string{"--strFlag", "hello"},
				WantData: &Data{
					Values: map[string]*Value{
						"strFlag": StringValue("hello"),
					},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--strFlag"},
						{value: "hello"},
					},
				},
			},
		},
		{
			name: "flag node parses short name flag",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', nil))},
				Args: []string{"-f", "hello"},
				WantData: &Data{
					Values: map[string]*Value{
						"strFlag": StringValue("hello"),
					},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-f"},
						{value: "hello"},
					},
				},
			},
		},
		{
			name: "flag node parses flag in the middle",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(StringFlag("strFlag", 'f', nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "--strFlag", "hello", "deux"},
				WantData: &Data{
					Values: map[string]*Value{
						"strFlag": StringValue("hello"),
						"filler":  StringListValue("un", "deux"),
					},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--strFlag"},
						{value: "hello"},
						{value: "deux"},
					},
				},
			},
		},
		{
			name: "flag node parses short name flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(StringFlag("strFlag", 'f', nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"uno", "dos", "-f", "hello"},
				WantData: &Data{
					Values: map[string]*Value{
						"filler":  StringListValue("uno", "dos"),
						"strFlag": StringValue("hello"),
					},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "uno"},
						{value: "dos"},
						{value: "-f"},
						{value: "hello"},
					},
				},
			},
		},
		// Int flag
		{
			name: "parses int flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(IntFlag("intFlag", 'f', nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "deux", "-f", "3", "quatre"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "deux"},
						{value: "-f"},
						{value: "3"},
						{value: "quatre"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"filler":  StringListValue("un", "deux", "quatre"),
						"intFlag": IntValue(3),
					},
				},
			},
		},
		{
			name: "handles invalid int flag value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(IntFlag("intFlag", 'f', nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "deux", "-f", "trois", "quatre"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "deux"},
						{value: "-f"},
						{value: "trois"},
						{value: "quatre"},
					},
					remaining: []int{0, 1, 4},
				},
				WantStderr: []string{`strconv.Atoi: parsing "trois": invalid syntax`},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "trois": invalid syntax`),
			},
		},
		// Float flag
		{
			name: "parses float flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(FloatFlag("floatFlag", 'f', nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"--floatFlag", "-1.2", "three"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--floatFlag"},
						{value: "-1.2"},
						{value: "three"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"filler":    StringListValue("three"),
						"floatFlag": FloatValue(-1.2),
					},
				},
			},
		},
		{
			name: "handles invalid float flag value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(FloatFlag("floatFlag", 'f', nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"--floatFlag", "twelve", "eleven"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--floatFlag"},
						{value: "twelve"},
						{value: "eleven"},
					},
					remaining: []int{2},
				},
				WantStderr: []string{`strconv.ParseFloat: parsing "twelve": invalid syntax`},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
			},
		},
		// Bool flag
		{
			name: "bool flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(BoolFlag("boolFlag", 'b')),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"okay", "--boolFlag", "then"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "okay"},
						{value: "--boolFlag"},
						{value: "then"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"filler":   StringListValue("okay", "then"),
						"boolFlag": BoolValue(true),
					},
				},
			},
		},
		{
			name: "short bool flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(BoolFlag("boolFlag", 'b')),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"okay", "-b", "then"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "okay"},
						{value: "-b"},
						{value: "then"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"filler":   StringListValue("okay", "then"),
						"boolFlag": BoolValue(true),
					},
				},
			},
		},
		// flag list tests
		{
			name: "flag list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(StringListFlag("slFlag", 's', 2, 3, nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "--slFlag", "hello", "there"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--slFlag"},
						{value: "hello"},
						{value: "there"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"filler": StringListValue("un"),
						"slFlag": StringListValue("hello", "there"),
					},
				},
			},
		},
		{
			name: "flag list fails if not enough",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(StringListFlag("slFlag", 's', 2, 3, nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "--slFlag", "hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--slFlag"},
						{value: "hello"},
					},
					remaining: []int{0},
				},
				WantStderr: []string{"not enough arguments"},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantData: &Data{
					Values: map[string]*Value{
						"slFlag": StringListValue("hello"),
					},
				},
			},
		},
		// Int list
		{
			name: "int list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(IntListFlag("ilFlag", 'i', 2, 3, nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "-i", "2", "4", "8", "16", "32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "-i"},
						{value: "2"},
						{value: "4"},
						{value: "8"},
						{value: "16"},
						{value: "32"},
						{value: "64"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"filler": StringListValue("un", "64"),
						"ilFlag": IntListValue(2, 4, 8, 16, 32),
					},
				},
			},
		},
		{
			name: "int list transform failure",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(IntListFlag("ilFlag", 'i', 2, 3, nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "-i", "2", "4", "8", "16.0", "32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "-i"},
						{value: "2"},
						{value: "4"},
						{value: "8"},
						{value: "16.0"},
						{value: "32"},
						{value: "64"},
					},
					remaining: []int{0, 7},
				},
				WantStderr: []string{`strconv.Atoi: parsing "16.0": invalid syntax`},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "16.0": invalid syntax`),
			},
		},
		// Float list
		{
			name: "float list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(FloatListFlag("flFlag", 'f', 0, 3, nil)),
					StringListNode("filler", 1, 3, nil),
				),
				Args: []string{"un", "-f", "2", "-4.4", "0.8", "16.16", "-32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "-f"},
						{value: "2"},
						{value: "-4.4"},
						{value: "0.8"},
						{value: "16.16"},
						{value: "-32"},
						{value: "64"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"filler": StringListValue("un", "16.16", "-32", "64"),
						"flFlag": FloatListValue(2, -4.4, 0.8),
					},
				},
			},
		},
		{
			name: "float list transform failure",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(FloatListFlag("flFlag", 'f', 0, 3, nil)),
					StringListNode("filler", 1, 2, nil),
				),
				Args: []string{"un", "--flFlag", "2", "4", "eight", "16.0", "32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--flFlag"},
						{value: "2"},
						{value: "4"},
						{value: "eight"},
						{value: "16.0"},
						{value: "32"},
						{value: "64"},
					},
					remaining: []int{0, 5, 6, 7},
				},
				WantStderr: []string{`strconv.ParseFloat: parsing "eight": invalid syntax`},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "eight": invalid syntax`),
			},
		},
		// Misc. flag tests
		{
			name: "processes multiple flags",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						FloatListFlag("coordinates", 'c', 2, 0, nil),
						BoolFlag("boo", 'o'),
						StringListFlag("names", 'n', 1, 2, nil),
						IntFlag("rating", 'r', nil),
					),
					StringListNode("extra", 0, 10, nil),
				),
				Args: []string{"its", "--boo", "a", "-r", "9", "secret", "-n", "greggar", "groog", "beggars", "--coordinates", "2.2", "4.4", "message."},
				wantInput: &Input{
					args: []*inputArg{
						{value: "its"},
						{value: "--boo"},
						{value: "a"},
						{value: "-r"},
						{value: "9"},
						{value: "secret"},
						{value: "-n"},
						{value: "greggar"},
						{value: "groog"},
						{value: "beggars"},
						{value: "--coordinates"},
						{value: "2.2"},
						{value: "4.4"},
						{value: "message."},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"boo":         BoolValue(true),
						"extra":       StringListValue("its", "a", "secret", "message."),
						"names":       StringListValue("greggar", "groog", "beggars"),
						"coordinates": FloatListValue(2.2, 4.4),
						"rating":      IntValue(9),
					},
				},
			},
		},
		// BranchNode tests
		{
			name: "branch node requires branch argument",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, nil, true),
				WantStderr: []string{"branching argument required"},
				WantErr:    fmt.Errorf("branching argument required"),
			},
		},
		{
			name: "branch node requires matching branch argument",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, nil, true),
				Args:       []string{"uh"},
				WantStderr: []string{"argument must be one of [b h]"},
				WantErr:    fmt.Errorf("argument must be one of [b h]"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "uh"},
					},
					remaining: []int{0},
				},
			},
		},
		{
			name: "branch node forwards to proper node",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, nil, true),
				Args:       []string{"h"},
				WantStdout: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "h"},
					},
				},
			},
		},
		{
			name: "branch node forwards to default if none provided",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, printNode("default"), true),
				WantStdout: []string{"default"},
			},
		},
		{
			name: "branch node forwards to default if unknown provided",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, SerialNodes(StringListNode("sl", 0, UnboundedList, nil), printArgsNode().Processor), true),
				Args:       []string{"good", "morning"},
				WantStdout: []string{"sl: good, morning"},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("good", "morning"),
					},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "good"},
						{value: "morning"},
					},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			ExecuteTest(t, test.etc, &ExecuteTestOptions{testInput: true})
		})
	}
}

func abc() *Node {
	return BranchNode(map[string]*Node{
		"t": AliasNode("TEST_ALIAS", nil,
			CacheNode("TEST_CACHE", nil, SerialNodes(
				&tt{},
				StringNode("PATH", &ArgOpt{Completor: SimpleCompletor("clh111", "abcd111")}),
				StringNode("TARGET", &ArgOpt{Completor: SimpleCompletor("clh222", "abcd222")}),
				StringNode("FUNC", &ArgOpt{Completor: SimpleCompletor("clh333", "abcd333")}),
			))),
	}, nil, false)
}

type tt struct{}

func (t *tt) Execute(input *Input, output Output, data *Data, e *ExecuteData) error {
	t.do(input)
	return nil
}

func (t *tt) do(input *Input) {
	if s, ok := input.Peek(); ok && strings.Contains(s, ":") {
		if ss := strings.Split(s, ":"); len(ss) == 2 {
			input.Pop()
			input.PushFront(ss...)
		}
	}
}

func (t *tt) Complete(input *Input, data *Data) *CompleteData {
	t.do(input)
	return nil
}

func TestComplete(t *testing.T) {
	for _, test := range []struct {
		name           string
		ctc            *CompleteTestCase
		filepathAbs    string
		filepathAbsErr error
	}{
		{
			name: "stuff",
			ctc: &CompleteTestCase{
				Node: abc(),
				Args: "cmd t clh:abc",
				Want: []string{"abcd222"},
				WantData: &Data{
					Values: map[string]*Value{
						"PATH":   StringValue("clh"),
						"TARGET": StringValue("abc"),
					},
				},
			},
		},

		// Basic tests
		{
			name: "empty graph",
			ctc: &CompleteTestCase{
				Node: &Node{},
			},
		},
		{
			name: "returns suggestions of first node if empty",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", NewArgOpt(SimpleCompletor("un", "deux", "trois"), nil)),
					OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
					StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				),
				Want: []string{"deux", "trois", "un"},
				WantData: &Data{
					Values: map[string]*Value{
						"s": StringValue(""),
					},
				},
			},
		},
		{
			name: "returns suggestions of first node if up to first arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
					OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
					StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				),
				Args: "cmd t",
				Want: []string{"three", "two"},
				WantData: &Data{
					Values: map[string]*Value{
						"s": StringValue("t"),
					},
				},
			},
		},
		{
			name: "returns suggestions of middle node if that's where we're at",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
					StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
					OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
				),
				Args: "cmd three ",
				Want: []string{"dos", "uno"},
				WantData: &Data{
					Values: map[string]*Value{
						"s":  StringValue("three"),
						"sl": StringListValue(""),
					},
				},
			},
		},
		{
			name: "returns suggestions of middle node if partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
					StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
					OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
				),
				Args: "cmd three d",
				Want: []string{"dos"},
				WantData: &Data{
					Values: map[string]*Value{
						"s":  StringValue("three"),
						"sl": StringListValue("d"),
					},
				},
			},
		},
		{
			name: "returns suggestions in list",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
					StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
					OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
				),
				Args: "cmd three dos ",
				Want: []string{"dos", "uno"},
				WantData: &Data{
					Values: map[string]*Value{
						"s":  StringValue("three"),
						"sl": StringListValue("dos", ""),
					},
				},
			},
		},
		{
			name: "returns suggestions for last arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
					StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
					OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
				),
				Args: "cmd three uno dos ",
				Want: []string{"1", "2"},
				WantData: &Data{
					Values: map[string]*Value{
						"s":  StringValue("three"),
						"sl": StringListValue("uno", "dos"),
					},
				},
			},
		},
		{
			name: "returns nothing if iterate through all nodes",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", NewArgOpt(SimpleCompletor("one", "two", "three"), nil)),
					StringListNode("sl", 0, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
					OptionalIntNode("i", NewArgOpt(SimpleCompletor("2", "1"), nil)),
				),
				Args: "cmd three uno dos 1 what now",
				WantData: &Data{
					Values: map[string]*Value{
						"s":  StringValue("three"),
						"sl": StringListValue("uno", "dos"),
						"i":  IntValue(1),
					},
				},
			},
		},
		{
			name: "works if empty and list starts",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListNode("sl", 1, 2, NewArgOpt(SimpleCompletor("uno", "dos"), nil)),
				),
				Want: []string{"dos", "uno"},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue(),
					},
				},
			},
		},
		{
			name: "only returns suggestions matching prefix",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListNode("sl", 1, 2, NewArgOpt(SimpleCompletor("zzz-1", "zzz-2", "yyy-3", "zzz-4"), nil)),
				),
				Args: "cmd zz",
				Want: []string{"zzz-1", "zzz-2", "zzz-4"},
				WantData: &Data{
					Values: map[string]*Value{
						"sl": StringListValue("zz"),
					},
				},
			},
		},
		// Flag completion
		{
			name: "flag name gets completed if single hyphen at end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd -",
				Want: []string{"--good", "--greeting", "--names", "-g", "-h", "-n"},
			},
		},
		{
			name: "flag name gets completed if double hyphen at end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd --",
				Want: []string{"--good", "--greeting", "--names"},
			},
		},
		{
			name: "flag name gets completed if it's the only arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd 1 -",
				Want: []string{"--good", "--greeting", "--names", "-g", "-h", "-n"},
			},
		},
		{
			name: "completes for single flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd 1 --greeting h",
				Want: []string{"hey", "hi"},
				WantData: &Data{
					Values: map[string]*Value{
						"greeting": StringValue("h"),
					},
				},
			},
		},
		{
			name: "completes for single short flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd 1 -h he",
				Want: []string{"hey"},
				WantData: &Data{
					Values: map[string]*Value{
						"greeting": StringValue("he"),
					},
				},
			},
		},
		{
			name: "completes for list flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd 1 -h hey other --names ",
				Want: []string{"johnny", "ralph", "renee"},
				WantData: &Data{
					Values: map[string]*Value{
						"greeting": StringValue("hey"),
						"names":    StringListValue(""),
					},
				},
			},
		},
		{
			name: "completes distinct secondary for list flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleDistinctCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd 1 -h hey other --names ralph ",
				Want: []string{"johnny", "renee"},
				WantData: &Data{
					Values: map[string]*Value{
						"greeting": StringValue("hey"),
						"names":    StringListValue("ralph", ""),
					},
				},
			},
		},
		{
			name: "completes last flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleDistinctCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
						FloatFlag("float", 'f', NewArgOpt(SimpleCompletor("1.23", "12.3", "123.4"), nil)),
					),
					IntNode("i", NewArgOpt(SimpleCompletor("1", "2"), nil)),
				),
				Args: "cmd 1 -h hey other --names ralph renee johnny -f ",
				Want: []string{"1.23", "12.3", "123.4"},
				WantData: &Data{
					Values: map[string]*Value{
						"greeting": StringValue("hey"),
						"names":    StringListValue("ralph", "renee", "johnny"),
					},
				},
			},
		},
		{
			name: "completes arg if flag arg isn't at the end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', NewArgOpt(SimpleCompletor("hey", "hi"), nil)),
						StringListFlag("names", 'n', 1, 2, NewArgOpt(SimpleDistinctCompletor("ralph", "johnny", "renee"), nil)),
						BoolFlag("good", 'g'),
					),
					StringListNode("i", 1, 2, NewArgOpt(SimpleCompletor("hey", "ooo"), nil)),
				),
				Args: "cmd 1 -h hello beta --names ralph renee johnny ",
				Want: []string{"hey", "ooo"},
				WantData: &Data{
					Values: map[string]*Value{
						"i":        StringListValue("1", "beta", ""),
						"greeting": StringValue("hello"),
						"names":    StringListValue("ralph", "renee", "johnny"),
					},
				},
			},
		},
		// Transformer arg tests.
		{
			name: "handles nil option",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", nil),
				},
				Args: "cmd abc",
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("abc"),
					},
				},
			},
		},
		{
			name: "list handles nil option",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", 1, 2, nil),
				},
				Args: "cmd abc",
				WantData: &Data{
					Values: map[string]*Value{
						"slArg": StringListValue("abc"),
					},
				},
			},
		},
		{
			name: "transformer doesn't transform value when ForComplete is false",
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringNode("strArg", NewArgOpt(nil, &simpleTransformer{
					vt: StringType,
					t: func(v *Value) (*Value, error) {
						return StringValue("newStuff"), nil
					},
					fc: false,
				}))),
				Args: "cmd abc",
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("abc"),
					},
				},
			},
		},
		{
			name: "transformer does transform value when ForComplete is true",
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringNode("strArg", NewArgOpt(nil, &simpleTransformer{
					vt: StringType,
					t: func(v *Value) (*Value, error) {
						return StringValue("newStuff"), nil
					},
					fc: true,
				}))),
				Args: "cmd abc",
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue("newStuff"),
					},
				},
			},
		},
		{
			name:        "FileTransformer doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, FileTransformer())),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue(filepath.Join("relative", "path.txt")),
					},
				},
			},
		},
		{
			name:        "FileTransformer for list doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("strArg", 1, 2, NewArgOpt(nil, FileListTransformer())),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringListValue(filepath.Join("relative", "path.txt")),
					},
				},
			},
		},
		{
			name:           "handles transform error",
			filepathAbsErr: fmt.Errorf("bad news bears"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", NewArgOpt(nil, FileTransformer())),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{
					Values: map[string]*Value{
						"strArg": StringValue(filepath.Join("relative", "path.txt")),
					},
				},
			},
		},
		{
			name: "handles transformer of incorrect type",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", NewArgOpt(nil, FileTransformer())),
				},
				Args: "cmd 123",
				WantData: &Data{
					Values: map[string]*Value{
						"IntNode": IntValue(123),
					},
				},
			},
		},
		{
			name:        "transformer list transforms values",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", 1, 2, NewArgOpt(nil, &simpleTransformer{
						vt: StringListType,
						fc: true,
						t: func(v *Value) (*Value, error) {
							var sl []string
							for _, s := range v.StringList() {
								sl = append(sl, fmt.Sprintf("_%s_", s))
							}
							return StringListValue(sl...), nil
						},
					})),
				},
				Args: "cmd uno dos",
				WantData: &Data{
					Values: map[string]*Value{
						"slArg": StringListValue(
							"_uno_",
							"_dos_",
						),
					},
				},
			},
		},
		{
			name:           "handles transform list error",
			filepathAbsErr: fmt.Errorf("bad news bears"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", 1, 2, NewArgOpt(nil, FileListTransformer())),
				},
				Args: fmt.Sprintf("cmd %s %s", filepath.Join("relative", "path.txt"), filepath.Join("other.txt")),
				WantData: &Data{
					Values: map[string]*Value{
						"slArg": StringListValue(
							filepath.Join("relative", "path.txt"),
							filepath.Join("other.txt"),
						),
					},
				},
			},
		},
		{
			name: "handles list transformer of incorrect type",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", 1, 2, NewArgOpt(nil, FileTransformer())),
				},
				Args: "cmd 123",
				WantData: &Data{
					Values: map[string]*Value{
						"slArg": StringListValue("123"),
					},
				},
			},
		},
		// BranchNode completion tests.
		{
			name: "completes branch options",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts"), nil))), true),
				Want: []string{"a", "alpha", "bravo", "command", "default", "opts"},
				WantData: &Data{
					Values: map[string]*Value{
						"default": StringListValue(),
					},
				},
			},
		},
		{
			name: "doesn't complete branch options if complete arg is false",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts"), nil))), false),
				Want: []string{"command", "default", "opts"},
				WantData: &Data{
					Values: map[string]*Value{
						"default": StringListValue(),
					},
				},
			},
		},
		{
			name: "completes for specific branch",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts"), nil))), true),
				Args: "cmd alpha ",
				Want: []string{"other", "stuff"},
				WantData: &Data{
					Values: map[string]*Value{
						"hello": StringValue(""),
					},
				},
			},
		},
		{
			name: "branch node doesn't complete if no default and no branch match",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
					"bravo": {},
				}, nil, true),
				Args: "cmd some thing else",
			},
		},
		{
			name: "completes branch options with partial completion",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts", "ahhhh"), nil))), true),
				Args: "cmd a",
				Want: []string{"a", "ahhhh", "alpha"},
				WantData: &Data{
					Values: map[string]*Value{
						"default": StringListValue("a"),
					},
				},
			},
		},
		{
			name: "completes default options",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", NewArgOpt(SimpleCompletor("other", "stuff"), nil))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", 1, 3, NewArgOpt(SimpleCompletor("default", "command", "opts"), nil))), true),
				Args: "cmd something ",
				WantData: &Data{
					Values: map[string]*Value{
						"default": StringListValue("something", ""),
					},
				},
				Want: []string{"command", "default", "opts"},
			},
		},
		// Commands with different value types.
		{
			name: "int arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntNode("iArg", NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
				Args: "cmd 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{
					Values: map[string]*Value{
						"iArg": IntValue(4),
					},
				},
			},
		},
		{
			name: "optional int arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(OptionalIntNode("iArg", NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
				Args: "cmd 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{
					Values: map[string]*Value{
						"iArg": IntValue(4),
					},
				},
			},
		},
		{
			name: "int list arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntListNode("iArg", 2, 3, NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
				Args: "cmd 1 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{
					Values: map[string]*Value{
						"iArg": IntListValue(1, 4),
					},
				},
			},
		},
		{
			name: "int list arg gets completed if previous one was invalid",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntListNode("iArg", 2, 3, NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
				Args: "cmd one 4",
				Want: []string{"45", "456", "468"},
			},
		},
		{
			name: "int list arg optional args get completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntListNode("iArg", 2, 3, NewArgOpt(SimpleCompletor("12", "45", "456", "468", "7"), nil))),
				Args: "cmd 1 2 3 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{
					Values: map[string]*Value{
						"iArg": IntListValue(1, 2, 3, 4),
					},
				},
			},
		},
		{
			name: "float arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(FloatNode("fArg", NewArgOpt(SimpleCompletor("12", "4.5", "45.6", "468", "7"), nil))),
				Args: "cmd 4",
				Want: []string{"4.5", "45.6", "468"},
				WantData: &Data{
					Values: map[string]*Value{
						"fArg": FloatValue(4),
					},
				},
			},
		},
		{
			name: "float list arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(FloatListNode("fArg", 1, 2, NewArgOpt(SimpleCompletor("12", "4.5", "45.6", "468", "7"), nil))),
				Want: []string{"12", "4.5", "45.6", "468", "7"},
				WantData: &Data{
					Values: map[string]*Value{
						"fArg": FloatListValue(),
					},
				},
			},
		},
		{
			name: "bool arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(BoolNode("bArg")),
				Want: []string{"0", "1", "F", "FALSE", "False", "T", "TRUE", "True", "f", "false", "t", "true"},
				WantData: &Data{
					Values: map[string]*Value{
						"bArg": BoolValue(false),
					},
				},
			},
		},
		/* Useful comment for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			oldAbs := filepathAbs
			filepathAbs = func(s string) (string, error) {
				return filepath.Join(test.filepathAbs, s), test.filepathAbsErr
			}
			defer func() { filepathAbs = oldAbs }()

			CompleteTest(t, test.ctc, nil)
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
