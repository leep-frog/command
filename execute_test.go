package command

import (
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type errorEdge struct {
	e error
}

func (ee *errorEdge) Next(*Input, Output, *Data) (*Node, error) {
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
				args: []string{"hello"},
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
				pos:  1,
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
			node: SerialNodes(IntNode("i", nil)),
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
			node: SerialNodes(IntNode("i", nil)),
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
			node: SerialNodes(FloatNode("f", nil)),
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
			node: SerialNodes(FloatNode("f", nil)),
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
			node: SerialNodes(StringListNode("sl", 1, 1, nil)),
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
			node: SerialNodes(StringListNode("sl", 1, 2, nil)),
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
			node: SerialNodes(StringListNode("sl", 1, 2, nil)),
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
			node: SerialNodes(StringListNode("sl", 1, 2, nil)),
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
			name: "Unbounded string list fails if less than min provided",
			node: SerialNodes(StringListNode("sl", 4, UnboundedList, nil)),
			args: []string{"hello", "there", "kenobi"},
			wantInput: &Input{
				args: []string{"hello", "there", "kenobi"},
				pos:  3,
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
				pos:  1,
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
				pos:  3,
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
			node: SerialNodes(IntListNode("il", 1, 2, nil)),
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
			node: SerialNodes(FloatListNode("fl", 1, 2, nil)),
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
			node: SerialNodes(FloatListNode("fl", 1, 2, nil)),
			args: []string{"0.1", "four", "-23"},
			wantInput: &Input{
				args: []string{"0.1", "four", "-23"},
				pos:  3,
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
				pos:  5,
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
				args: []string{"0", "1", "two", "0.3", "-4", "0.5", "6"},
				pos:  6,
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
				pos:  5,
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
				pos:  5,
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
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("123"),
				},
			},
			wantStderr: []string{"validation failed: option can only be bound to arguments with type 3"},
		},
		// Contains
		/*{
			name: "contains works",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, Contains("good"))),
			},
			args: []string{"goodbye"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("hello"),
				},
			},
			wantStderr: []string{`validation failed: [Contains] value doesn't contain substring "good"`},
		},
		// MinLength
		{
			name: "MinLength works",
			node: &Node{
				Processor: StringNode("strArg", NewArgOpt(nil, nil, MinLength(3))),
			},
			args: []string{"hello"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"strArg": StringValue("hi"),
				},
			},
			wantStderr: []string{`validation failed: [MinLength] value must be at least 3 characters`},
		},
		// IntEQ
		{
			name: "IntEQ works",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntEQ(24))),
			},
			args: []string{"24"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(24),
				},
			},
		},
		{
			name: "IntEQ fails",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntEQ(24))),
			},
			args: []string{"25"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(25),
				},
			},
			wantStderr: []string{`validation failed: [IntEQ] value isn't equal to 24`},
		},
		// IntNE
		{
			name: "IntNE works",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNE(24))),
			},
			args: []string{"25"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(25),
				},
			},
		},
		{
			name: "IntNE fails",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNE(24))),
			},
			args: []string{"24"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(24),
				},
			},
			wantStderr: []string{`validation failed: [IntNE] value isn't not equal to 24`},
		},
		// IntLT
		{
			name: "IntLT works when less than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntLT(25))),
			},
			args: []string{"24"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(24),
				},
			},
		},
		{
			name: "IntLT fails when equal to",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntLT(25))),
			},
			args: []string{"25"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(25),
				},
			},
			wantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
		},
		{
			name: "IntLT fails when greater than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntLT(25))),
			},
			args: []string{"26"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(26),
				},
			},
			wantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
		},
		// IntLTE
		{
			name: "IntLTE works when less than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntLTE(25))),
			},
			args: []string{"24"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(24),
				},
			},
		},
		{
			name: "IntLTE works when equal to",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntLTE(25))),
			},
			args: []string{"25"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(25),
				},
			},
		},
		{
			name: "IntLTE fails when greater than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntLTE(25))),
			},
			args: []string{"26"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(26),
				},
			},
			wantStderr: []string{`validation failed: [IntLTE] value isn't less than or equal to 25`},
		},
		// IntGT
		{
			name: "IntGT fails when less than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntGT(25))),
			},
			args: []string{"24"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(24),
				},
			},
			wantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
		},
		{
			name: "IntGT fails when equal to",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntGT(25))),
			},
			args: []string{"25"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(25),
				},
			},
			wantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
		},
		{
			name: "IntGT works when greater than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntGT(25))),
			},
			args: []string{"26"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(26),
				},
			},
		},
		// IntGTE
		{
			name: "IntGTE fails when less than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntGTE(25))),
			},
			args: []string{"24"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(24),
				},
			},
			wantStderr: []string{`validation failed: [IntGTE] value isn't greater than or equal to 25`},
		},
		{
			name: "IntGTE works when equal to",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntGTE(25))),
			},
			args: []string{"25"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(25),
				},
			},
		},
		{
			name: "IntGTE works when greater than",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntGTE(25))),
			},
			args: []string{"26"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(26),
				},
			},
		},
		// IntPositive
		{
			name: "IntPositive fails when negative",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntPositive())),
			},
			args: []string{"-1"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(-1),
				},
			},
			wantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
		},
		{
			name: "IntPositive fails when zero",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntPositive())),
			},
			args: []string{"0"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(0),
				},
			},
			wantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
		},
		{
			name: "IntPositive works when positive",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntPositive())),
			},
			args: []string{"1"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(1),
				},
			},
		},
		// IntNegative
		{
			name: "IntNegative works when negative",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNegative())),
			},
			args: []string{"-1"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(-1),
				},
			},
		},
		{
			name: "IntNegative fails when zero",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNegative())),
			},
			args: []string{"0"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(0),
				},
			},
			wantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
		},
		{
			name: "IntNegative fails when positive",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNegative())),
			},
			args: []string{"1"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(1),
				},
			},
			wantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
		},
		// IntNonNegative
		{
			name: "IntNonNegative fails when negative",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNonNegative())),
			},
			args: []string{"-1"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(-1),
				},
			},
			wantStderr: []string{`validation failed: [IntNonNegative] value isn't non-negative`},
		},
		{
			name: "IntNonNegative works when zero",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNonNegative())),
			},
			args: []string{"0"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(0),
				},
			},
		},
		{
			name: "IntNonNegative works when positive",
			node: &Node{
				Processor: IntNode("intArg", NewArgOpt(nil, nil, IntNonNegative())),
			},
			args: []string{"1"},
			wantData: &Data{
				Values: map[string]*Value{
					"intArg": IntValue(1),
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
			wantStderr: []string{`validation failed: [FloatEQ] value isn't equal to 2.40`},
		},
		// FloatNE
		{
			name: "FloatNE works",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNE(2.4))),
			},
			args: []string{"2.5"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
			wantStderr: []string{`validation failed: [FloatNE] value isn't not equal to 2.40`},
		},
		// FloatLT
		{
			name: "FloatLT works when less than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
			},
			args: []string{"2.4"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
			wantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
		},
		{
			name: "FloatLT fails when greater than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLT(2.5))),
			},
			args: []string{"2.6"},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				},
			},
			wantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
		},
		// FloatLTE
		{
			name: "FloatLTE works when less than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatLTE(2.5))),
			},
			args: []string{"2.4"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				},
			},
			wantStderr: []string{`validation failed: [FloatLTE] value isn't less than or equal to 2.50`},
		},
		// FloatGT
		{
			name: "FloatGT fails when less than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
			},
			args: []string{"2.4"},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
			wantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
		},
		{
			name: "FloatGT fails when equal to",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
			},
			args: []string{"2.5"},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				},
			},
			wantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
		},
		{
			name: "FloatGT works when greater than",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGT(2.5))),
			},
			args: []string{"2.6"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				},
			},
			wantStderr: []string{`validation failed: [FloatGTE] value isn't greater than or equal to 2.50`},
		},
		{
			name: "FloatGTE works when equal to",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatGTE(2.5))),
			},
			args: []string{"2.5"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				},
			},
			wantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
		},
		{
			name: "FloatPositive fails when zero",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
			},
			args: []string{"0"},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0),
				},
			},
			wantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
		},
		{
			name: "FloatPositive works when positive",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatPositive())),
			},
			args: []string{"0.1"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0),
				},
			},
			wantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
		},
		{
			name: "FloatNegative fails when positive",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNegative())),
			},
			args: []string{"0.1"},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				},
			},
			wantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
		},
		// FloatNonNegative
		{
			name: "FloatNonNegative fails when negative",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
			},
			args: []string{"-0.1"},
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				},
			},
			wantStderr: []string{`validation failed: [FloatNonNegative] value isn't non-negative`},
		},
		{
			name: "FloatNonNegative works when zero",
			node: &Node{
				Processor: FloatNode("flArg", NewArgOpt(nil, nil, FloatNonNegative())),
			},
			args: []string{"0"},
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
			wantData: &Data{
				Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				},
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
