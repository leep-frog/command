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

func (ee *errorEdge) UsageNext() *Node {
	return nil
}

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name      string
		etc       *ExecuteTestCase
		postCheck func(*testing.T)
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
				Node:       SerialNodes(StringNode("s", testDesc)),
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
				WantStderr: []string{`Argument "s" requires at least 1 argument, got 0`},
			},
		},
		{
			name: "Fails if edge fails",
			etc: &ExecuteTestCase{
				Args: []string{"hello"},
				Node: &Node{
					Processor: StringNode("s", testDesc),
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
				WantData: &Data{Values: map[string]*Value{
					"s": StringValue("hello"),
				}},
			},
		},
		{
			name: "Fails if int arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(IntNode("i", testDesc)),
				WantErr:    fmt.Errorf(`Argument "i" requires at least 1 argument, got 0`),
				WantStderr: []string{`Argument "i" requires at least 1 argument, got 0`},
			},
		},
		{
			name: "Fails if float arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(FloatNode("f", testDesc)),
				WantErr:    fmt.Errorf(`Argument "f" requires at least 1 argument, got 0`),
				WantStderr: []string{`Argument "f" requires at least 1 argument, got 0`},
			},
		},
		// Default value tests
		{
			name: "Uses default if no arg provided",
			etc: &ExecuteTestCase{
				Node:      SerialNodes(OptionalStringNode("s", testDesc, StringDefault("settled"))),
				wantInput: &Input{},
				WantData: &Data{Values: map[string]*Value{
					"s": StringValue("settled"),
				}},
			},
		},
		{
			name: "Fails if default is the wrong type",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(OptionalStringNode("s", testDesc, IntDefault(1))),
				wantInput:  &Input{},
				WantStderr: []string{`Argument "s" has type String, but its default is of type Int`},
				WantErr:    fmt.Errorf(`Argument "s" has type String, but its default is of type Int`),
			},
		},
		{
			name: "Default doesn't fill in required argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(StringNode("s", testDesc, StringDefault("settled"))),
				wantInput:  &Input{},
				WantStderr: []string{`Argument "s" requires at least 1 argument, got 0`},
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
			},
		},
		// Simple arg tests
		{
			name: "Processes single string arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringNode("s", testDesc)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"s": StringValue("hello"),
				}},
			},
		},
		{
			name: "Processes single int arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntNode("i", testDesc)),
				Args: []string{"123"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"i": IntValue(123),
				}},
			},
		},
		{
			name: "Int arg fails if not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntNode("i", testDesc)),
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
				Node: SerialNodes(FloatNode("f", testDesc)),
				Args: []string{"-12.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-12.3"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"f": FloatValue(-12.3),
				}},
			},
		},
		{
			name: "Float arg fails if not a float",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatNode("f", testDesc)),
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
				Node: SerialNodes(StringListNode("sl", testDesc, 1, 1)),
				Args: []string{"hello", "there", "sir"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "sir"},
					},
					remaining: []int{2},
				},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("hello", "there"),
				}},
				WantErr:    fmt.Errorf("Unprocessed extra args: [sir]"),
				WantStderr: []string{"Unprocessed extra args: [sir]"},
			},
		},
		{
			name: "Processes string list if minimum provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", testDesc, 1, 2)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("hello"),
				}},
			},
		},
		{
			name: "Processes string list if some optional provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", testDesc, 1, 2)),
				Args: []string{"hello", "there"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("hello", "there"),
				}},
			},
		},
		{
			name: "Processes string list if max args provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", testDesc, 1, 2)),
				Args: []string{"hello", "there", "maam"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "maam"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("hello", "there", "maam"),
				}},
			},
		},
		{
			name: "Unbounded string list fails if less than min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", testDesc, 4, UnboundedList)),
				Args: []string{"hello", "there", "kenobi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "kenobi"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("hello", "there", "kenobi"),
				}},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 4 arguments, got 3`),
				WantStderr: []string{`Argument "sl" requires at least 4 arguments, got 3`},
			},
		},
		{
			name: "Processes unbounded string list if min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", testDesc, 1, UnboundedList)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("hello"),
				}},
			},
		},
		{
			name: "Processes unbounded string list if more than min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("sl", testDesc, 1, UnboundedList)),
				Args: []string{"hello", "there", "kenobi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "kenobi"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("hello", "there", "kenobi"),
				}},
			},
		},
		{
			name: "Processes int list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", testDesc, 1, 2)),
				Args: []string{"1", "-23"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
						{value: "-23"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"il": IntListValue(1, -23),
				}},
			},
		},
		{
			name: "Int list fails if an arg isn't an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", testDesc, 1, 2)),
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
				Node: SerialNodes(FloatListNode("fl", testDesc, 1, 2)),
				Args: []string{"0.1", "-2.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
						{value: "-2.3"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"fl": FloatListValue(0.1, -2.3),
				}},
			},
		},
		{
			name: "Float list fails if an arg isn't an float",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatListNode("fl", testDesc, 1, 2)),
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
				Node: SerialNodes(IntListNode("il", testDesc, 2, 0), StringNode("s", testDesc), FloatListNode("fl", testDesc, 1, 2)),
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
				WantData: &Data{Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4),
				}},
			},
		},
		{
			name: "Fails if extra args when multiple",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", testDesc, 2, 0), StringNode("s", testDesc), FloatListNode("fl", testDesc, 1, 2)),
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
				WantData: &Data{Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4, 0.5),
				}},
				WantErr:    fmt.Errorf("Unprocessed extra args: [6]"),
				WantStderr: []string{"Unprocessed extra args: [6]"},
			},
		},
		// Executor tests.
		{
			name: "Sets executable with SimpleExecutableNode",
			etc: &ExecuteTestCase{
				Node: SerialNodes(SimpleExecutableNode("hello", "there")),
				WantExecuteData: &ExecuteData{
					Executable: []string{"hello", "there"},
				},
			},
		},
		{
			name: "Sets executable with ExecutableNode",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("SL", "", 0, UnboundedList),
					ExecutableNode(func(o Output, d *Data) ([]string, error) {
						o.Stdout("hello")
						o.Stderr("there")
						return d.StringList("SL"), nil
					}),
				),
				Args:       []string{"abc", "def"},
				WantStdout: []string{"hello"},
				WantStderr: []string{"there"},
				WantExecuteData: &ExecuteData{
					Executable: []string{"abc", "def"},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"SL": StringListValue("abc", "def"),
					},
				},
			},
		},
		{
			name: "ExecutableNode returning error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("SL", "", 0, UnboundedList),
					ExecutableNode(func(o Output, d *Data) ([]string, error) {
						return d.StringList("SL"), fmt.Errorf("bad news bears")
					}),
				),
				Args:    []string{"abc", "def"},
				WantErr: fmt.Errorf("bad news bears"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
					},
				},
				WantData: &Data{
					Values: map[string]*Value{
						"SL": StringListValue("abc", "def"),
					},
				},
			},
		},
		{
			name: "Sets executable with processor",
			etc: &ExecuteTestCase{
				Node: SerialNodes(SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
					ed.Executable = []string{"hello", "there"}
					return nil
				}, nil)),
				WantExecuteData: &ExecuteData{
					Executable: []string{"hello", "there"},
				},
			},
		},
		{
			name: "executes with proper data",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", testDesc, 2, 0), StringNode("s", testDesc), FloatListNode("fl", testDesc, 1, 2), ExecutorNode(func(o Output, d *Data) error {
					keys := d.Keys()
					sort.Strings(keys)
					for _, k := range keys {
						o.Stdoutf("%s: %v", k, d.Get(k))
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
				WantData: &Data{Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4),
				}},
				WantStdout: []string{
					"fl: FloatListValue(0.30, -4.00)",
					"il: IntListValue(0, 1)",
					`s: StringValue("two")`,
				},
			},
		},
		{
			name: "executor error is returned",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("il", testDesc, 2, 0), StringNode("s", testDesc), FloatListNode("fl", testDesc, 1, 2), ExecutorNode(func(o Output, d *Data) error {
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
				WantData: &Data{Values: map[string]*Value{
					"il": IntListValue(0, 1),
					"s":  StringValue("two"),
					"fl": FloatListValue(0.3, -4),
				}},
				WantStderr: []string{"bad news bears"},
				WantErr:    fmt.Errorf("bad news bears"),
			},
		},
		// ArgValidator tests
		{
			name: "breaks when arg option is for invalid type",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, IntEQ(123)),
				},
				Args: []string{"123"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("123"),
				}},
				WantStderr: []string{"validation failed: option can only be bound to arguments with type Int"},
				WantErr:    fmt.Errorf("validation failed: option can only be bound to arguments with type Int"),
			},
		},

		// StringDoesNotEqual
		{
			name: "string dne works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, StringDoesNotEqual("bad")),
				},
				Args: []string{"good"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "good"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("good"),
				}},
			},
		},
		{
			name: "string dne fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, StringDoesNotEqual("bad")),
				},
				Args: []string{"bad"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "bad"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("bad"),
				}},
				WantStderr: []string{`validation failed: [StringDoesNotEqual] value cannot equal "bad"`},
				WantErr:    fmt.Errorf(`validation failed: [StringDoesNotEqual] value cannot equal "bad"`),
			},
		},
		// Contains
		{
			name: "contains works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, Contains("good")),
				},
				Args: []string{"goodbye"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "goodbye"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("goodbye"),
				}},
			},
		},
		{
			name: "contains fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, Contains("good")),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("hello"),
				}},
				WantStderr: []string{`validation failed: [Contains] value doesn't contain substring "good"`},
				WantErr:    fmt.Errorf(`validation failed: [Contains] value doesn't contain substring "good"`),
			},
		},
		{
			name: "AddOptions works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc).AddOptions(Contains("good")),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("hello"),
				}},
				WantStderr: []string{`validation failed: [Contains] value doesn't contain substring "good"`},
				WantErr:    fmt.Errorf(`validation failed: [Contains] value doesn't contain substring "good"`),
			},
		},
		// MatchesRegex
		{
			name: "matches regex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, MatchesRegex("a+b=?c")),
				},
				Args: []string{"equiation: aabcdef"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "equiation: aabcdef"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("equiation: aabcdef"),
				}},
			},
		},
		{
			name: "matches regex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, MatchesRegex(".*", "i+")),
				},
				Args: []string{"team"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "team"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("team"),
				}},
				WantStderr: []string{`validation failed: [MatchesRegex] value doesn't match regex "i+"`},
				WantErr:    fmt.Errorf(`validation failed: [MatchesRegex] value doesn't match regex "i+"`),
			},
		},
		// ListMatchesRegex
		{
			name: "ListMatchesRegex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, UnboundedList, ListMatchesRegex("a+b=?c", "^eq")),
				},
				Args: []string{"equiation: aabcdef"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "equiation: aabcdef"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue("equiation: aabcdef"),
				}},
			},
		},
		{
			name: "ListMatchesRegex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, UnboundedList, ListMatchesRegex(".*", "i+")),
				},
				Args: []string{"equiation: aabcdef", "oops"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "equiation: aabcdef"},
						{value: "oops"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue("equiation: aabcdef", "oops"),
				}},
				WantStderr: []string{`validation failed: [ListMatchesRegex] value "oops" doesn't match regex "i+"`},
				WantErr:    fmt.Errorf(`validation failed: [ListMatchesRegex] value "oops" doesn't match regex "i+"`),
			},
		},
		// IsRegex
		{
			name: "IsRegex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, IsRegex()),
				},
				Args: []string{".*"},
				wantInput: &Input{
					args: []*inputArg{
						{value: ".*"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue(".*"),
				}},
			},
		},
		{
			name: "IsRegex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, IsRegex()),
				},
				Args: []string{"*"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "*"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("*"),
				}},
				WantStderr: []string{"validation failed: [IsRegex] value isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"},
				WantErr:    fmt.Errorf("validation failed: [IsRegex] value isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"),
			},
		},
		// ListIsRegex
		{
			name: "ListIsRegex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, UnboundedList, ListIsRegex()),
				},
				Args: []string{".*", " +"},
				wantInput: &Input{
					args: []*inputArg{
						{value: ".*"},
						{value: " +"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue(".*", " +"),
				}},
			},
		},
		{
			name: "ListIsRegex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, UnboundedList, ListIsRegex()),
				},
				Args: []string{".*", "+"},
				wantInput: &Input{
					args: []*inputArg{
						{value: ".*"},
						{value: "+"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue(".*", "+"),
				}},
				WantStderr: []string{"validation failed: [ListIsRegex] value \"+\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `+`"},
				WantErr:    fmt.Errorf("validation failed: [ListIsRegex] value \"+\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `+`"),
			},
		},
		// FileExists and FilesExist
		{
			name: "FileExists works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, FileExists()),
				},
				Args: []string{"execute_test.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("execute_test.go"),
				}},
			},
		},
		{
			name: "FileExists fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, FileExists()),
				},
				Args: []string{"execute_test.gone"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.gone"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("execute_test.gone"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [FileExists] file "execute_test.gone" does not exist`),
				WantStderr: []string{`validation failed: [FileExists] file "execute_test.gone" does not exist`},
			},
		},
		{
			name: "FilesExist works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, FilesExist()),
				},
				Args: []string{"execute_test.go", "execute.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
						{value: "execute.go"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("execute_test.go", "execute.go"),
				}},
			},
		},
		{
			name: "FilesExist fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, FilesExist()),
				},
				Args: []string{"execute_test.go", "execute.gone"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
						{value: "execute.gone"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("execute_test.go", "execute.gone"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [FilesExist] file "execute.gone" does not exist`),
				WantStderr: []string{`validation failed: [FilesExist] file "execute.gone" does not exist`},
			},
		},
		// IsDir and AreDirs
		{
			name: "IsDir works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, IsDir()),
				},
				Args: []string{"testing"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testing"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("testing"),
				}},
			},
		},
		{
			name: "IsDir fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, IsDir()),
				},
				Args: []string{"tested"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "tested"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("tested"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [IsDir] file "tested" does not exist`),
				WantStderr: []string{`validation failed: [IsDir] file "tested" does not exist`},
			},
		},
		{
			name: "IsDir fails when not a directory",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, IsDir()),
				},
				Args: []string{"execute_test.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("execute_test.go"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [IsDir] argument "execute_test.go" is a file`),
				WantStderr: []string{`validation failed: [IsDir] argument "execute_test.go" is a file`},
			},
		},
		{
			name: "AreDirs works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, AreDirs()),
				},
				Args: []string{"testing", "cache"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testing"},
						{value: "cache"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("testing", "cache"),
				}},
			},
		},
		{
			name: "AreDirs fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, AreDirs()),
				},
				Args: []string{"testing", "cash"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testing"},
						{value: "cash"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("testing", "cash"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [AreDirs] file "cash" does not exist`),
				WantStderr: []string{`validation failed: [AreDirs] file "cash" does not exist`},
			},
		},
		{
			name: "AreDirs fails when not a directory",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, AreDirs()),
				},
				Args: []string{"testing", "execute.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testing"},
						{value: "execute.go"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("testing", "execute.go"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [AreDirs] argument "execute.go" is a file`),
				WantStderr: []string{`validation failed: [AreDirs] argument "execute.go" is a file`},
			},
		},
		// IsFile and AreFiles
		{
			name: "IsFile works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, IsFile()),
				},
				Args: []string{"execute.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("execute.go"),
				}},
			},
		},
		{
			name: "IsFile fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, IsFile()),
				},
				Args: []string{"tested"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "tested"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("tested"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [IsFile] file "tested" does not exist`),
				WantStderr: []string{`validation failed: [IsFile] file "tested" does not exist`},
			},
		},
		{
			name: "IsFile fails when not a file",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("S", testDesc, IsFile()),
				},
				Args: []string{"testing"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testing"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"S": StringValue("testing"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [IsFile] argument "testing" is a directory`),
				WantStderr: []string{`validation failed: [IsFile] argument "testing" is a directory`},
			},
		},
		{
			name: "AreFiles works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, AreFiles()),
				},
				Args: []string{"execute.go", "cache.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
						{value: "cache.go"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("execute.go", "cache.go"),
				}},
			},
		},
		{
			name: "AreFiles fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, AreFiles()),
				},
				Args: []string{"execute.go", "cash"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
						{value: "cash"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("execute.go", "cash"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [AreFiles] file "cash" does not exist`),
				WantStderr: []string{`validation failed: [AreFiles] file "cash" does not exist`},
			},
		},
		{
			name: "AreFiles fails when not a directory",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringListNode("SL", testDesc, 1, 3, AreFiles()),
				},
				Args: []string{"execute.go", "testing"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
						{value: "testing"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("execute.go", "testing"),
				}},
				WantErr:    fmt.Errorf(`validation failed: [AreFiles] argument "testing" is a directory`),
				WantStderr: []string{`validation failed: [AreFiles] argument "testing" is a directory`},
			},
		},
		// InList & string menu
		{
			name: "InList works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, InList("abc", "def", "ghi")),
				},
				Args: []string{"def"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "def"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("def"),
				}},
			},
		},
		{
			name: "InList fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, InList("abc", "def", "ghi")),
				},
				Args: []string{"jkl"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "jkl"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("jkl"),
				}},
				WantStderr: []string{`validation failed: [InList] argument must be one of [abc def ghi]`},
				WantErr:    fmt.Errorf(`validation failed: [InList] argument must be one of [abc def ghi]`),
			},
		},
		{
			name: "StringMenu works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringMenu("strArg", testDesc, "abc", "def", "ghi"),
				},
				Args: []string{"def"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "def"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("def"),
				}},
			},
		},
		{
			name: "StringMenu fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringMenu("strArg", testDesc, "abc", "def", "ghi"),
				},
				Args: []string{"jkl"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "jkl"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("jkl"),
				}},
				WantStderr: []string{`validation failed: [InList] argument must be one of [abc def ghi]`},
				WantErr:    fmt.Errorf(`validation failed: [InList] argument must be one of [abc def ghi]`),
			},
		},
		// MinLength
		{
			name: "MinLength works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, MinLength(3)),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("hello"),
				}},
			},
		},
		{
			name: "MinLength works for exact count match",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, MinLength(3)),
				},
				Args: []string{"hey"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hey"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("hey"),
				}},
			},
		},
		{
			name: "MinLength fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, MinLength(3)),
				},
				Args: []string{"hi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hi"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("hi"),
				}},
				WantStderr: []string{`validation failed: [MinLength] value must be at least 3 characters`},
				WantErr:    fmt.Errorf(`validation failed: [MinLength] value must be at least 3 characters`),
			},
		},
		// IntEQ
		{
			name: "IntEQ works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntEQ(24)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(24),
				}},
			},
		},
		{
			name: "IntEQ fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntEQ(24)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(25),
				}},
				WantStderr: []string{`validation failed: [IntEQ] value isn't equal to 24`},
				WantErr:    fmt.Errorf(`validation failed: [IntEQ] value isn't equal to 24`),
			},
		},
		// IntNE
		{
			name: "IntNE works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNE(24)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(25),
				}},
			},
		},
		{
			name: "IntNE fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNE(24)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(24),
				}},
				WantStderr: []string{`validation failed: [IntNE] value isn't not equal to 24`},
				WantErr:    fmt.Errorf(`validation failed: [IntNE] value isn't not equal to 24`),
			},
		},
		// IntLT
		{
			name: "IntLT works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntLT(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(24),
				}},
			},
		},
		{
			name: "IntLT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntLT(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(25),
				}},
				WantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntLT] value isn't less than 25`),
			},
		},
		{
			name: "IntLT fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntLT(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(26),
				}},
				WantStderr: []string{`validation failed: [IntLT] value isn't less than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntLT] value isn't less than 25`),
			},
		},
		// IntLTE
		{
			name: "IntLTE works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntLTE(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(24),
				}},
			},
		},
		{
			name: "IntLTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntLTE(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(25),
				}},
			},
		},
		{
			name: "IntLTE fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntLTE(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(26),
				}},
				WantStderr: []string{`validation failed: [IntLTE] value isn't less than or equal to 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntLTE] value isn't less than or equal to 25`),
			},
		},
		// IntGT
		{
			name: "IntGT fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntGT(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(24),
				}},
				WantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntGT] value isn't greater than 25`),
			},
		},
		{
			name: "IntGT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntGT(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(25),
				}},
				WantStderr: []string{`validation failed: [IntGT] value isn't greater than 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntGT] value isn't greater than 25`),
			},
		},
		{
			name: "IntGT works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntGT(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(26),
				}},
			},
		},
		// IntGTE
		{
			name: "IntGTE fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntGTE(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(24),
				}},
				WantStderr: []string{`validation failed: [IntGTE] value isn't greater than or equal to 25`},
				WantErr:    fmt.Errorf(`validation failed: [IntGTE] value isn't greater than or equal to 25`),
			},
		},
		{
			name: "IntGTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntGTE(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(25),
				}},
			},
		},
		{
			name: "IntGTE works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntGTE(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(26),
				}},
			},
		},
		// IntPositive
		{
			name: "IntPositive fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntPositive()),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(-1),
				}},
				WantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [IntPositive] value isn't positive`),
			},
		},
		{
			name: "IntPositive fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntPositive()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(0),
				}},
				WantStderr: []string{`validation failed: [IntPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [IntPositive] value isn't positive`),
			},
		},
		{
			name: "IntPositive works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntPositive()),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(1),
				}},
			},
		},
		// IntNegative
		{
			name: "IntNegative works when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNegative()),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(-1),
				}},
			},
		},
		{
			name: "IntNegative fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNegative()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(0),
				}},
				WantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [IntNegative] value isn't negative`),
			},
		},
		{
			name: "IntNegative fails when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNegative()),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(1),
				}},
				WantStderr: []string{`validation failed: [IntNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [IntNegative] value isn't negative`),
			},
		},
		// IntNonNegative
		{
			name: "IntNonNegative fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNonNegative()),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(-1),
				}},
				WantStderr: []string{`validation failed: [IntNonNegative] value isn't non-negative`},
				WantErr:    fmt.Errorf(`validation failed: [IntNonNegative] value isn't non-negative`),
			},
		},
		{
			name: "IntNonNegative works when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNonNegative()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(0),
				}},
			},
		},
		{
			name: "IntNonNegative works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, IntNonNegative()),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(1),
				}},
			},
		},
		// FloatEQ
		{
			name: "FloatEQ works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatEQ(2.4)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				}},
			},
		},
		{
			name: "FloatEQ fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatEQ(2.4)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				}},
				WantStderr: []string{`validation failed: [FloatEQ] value isn't equal to 2.40`},
				WantErr:    fmt.Errorf(`validation failed: [FloatEQ] value isn't equal to 2.40`),
			},
		},
		// FloatNE
		{
			name: "FloatNE works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNE(2.4)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				}},
			},
		},
		{
			name: "FloatNE fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNE(2.4)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				}},
				WantStderr: []string{`validation failed: [FloatNE] value isn't not equal to 2.40`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNE] value isn't not equal to 2.40`),
			},
		},
		// FloatLT
		{
			name: "FloatLT works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatLT(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				}},
			},
		},
		{
			name: "FloatLT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatLT(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				}},
				WantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatLT] value isn't less than 2.50`),
			},
		},
		{
			name: "FloatLT fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatLT(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				}},
				WantStderr: []string{`validation failed: [FloatLT] value isn't less than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatLT] value isn't less than 2.50`),
			},
		},
		// FloatLTE
		{
			name: "FloatLTE works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatLTE(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				}},
			},
		},
		{
			name: "FloatLTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatLTE(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				}},
			},
		},
		{
			name: "FloatLTE fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatLTE(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				}},
				WantStderr: []string{`validation failed: [FloatLTE] value isn't less than or equal to 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatLTE] value isn't less than or equal to 2.50`),
			},
		},
		// FloatGT
		{
			name: "FloatGT fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatGT(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				}},
				WantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatGT] value isn't greater than 2.50`),
			},
		},
		{
			name: "FloatGT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatGT(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				}},
				WantStderr: []string{`validation failed: [FloatGT] value isn't greater than 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatGT] value isn't greater than 2.50`),
			},
		},
		{
			name: "FloatGT works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatGT(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				}},
			},
		},
		// FloatGTE
		{
			name: "FloatGTE fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatGTE(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.4),
				}},
				WantStderr: []string{`validation failed: [FloatGTE] value isn't greater than or equal to 2.50`},
				WantErr:    fmt.Errorf(`validation failed: [FloatGTE] value isn't greater than or equal to 2.50`),
			},
		},
		{
			name: "FloatGTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatGTE(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.5),
				}},
			},
		},
		{
			name: "FloatGTE works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatGTE(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(2.6),
				}},
			},
		},
		// FloatPositive
		{
			name: "FloatPositive fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatPositive()),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				}},
				WantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [FloatPositive] value isn't positive`),
			},
		},
		{
			name: "FloatPositive fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatPositive()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(0),
				}},
				WantStderr: []string{`validation failed: [FloatPositive] value isn't positive`},
				WantErr:    fmt.Errorf(`validation failed: [FloatPositive] value isn't positive`),
			},
		},
		{
			name: "FloatPositive works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatPositive()),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				}},
			},
		},
		// FloatNegative
		{
			name: "FloatNegative works when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNegative()),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				}},
			},
		},
		{
			name: "FloatNegative fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNegative()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(0),
				}},
				WantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNegative] value isn't negative`),
			},
		},
		{
			name: "FloatNegative fails when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNegative()),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				}},
				WantStderr: []string{`validation failed: [FloatNegative] value isn't negative`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNegative] value isn't negative`),
			},
		},
		// FloatNonNegative
		{
			name: "FloatNonNegative fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNonNegative()),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(-0.1),
				}},
				WantStderr: []string{`validation failed: [FloatNonNegative] value isn't non-negative`},
				WantErr:    fmt.Errorf(`validation failed: [FloatNonNegative] value isn't non-negative`),
			},
		},
		{
			name: "FloatNonNegative works when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNonNegative()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(0),
				}},
			},
		},
		{
			name: "FloatNonNegative works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: FloatNode("flArg", testDesc, FloatNonNegative()),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"flArg": FloatValue(0.1),
				}},
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
				Node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', testDesc))},
			},
		},
		{
			name: "flag node fails if no argument",
			etc: &ExecuteTestCase{
				Node:       &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', testDesc))},
				Args:       []string{"--strFlag"},
				WantStderr: []string{`Argument "strFlag" requires at least 1 argument, got 0`},
				WantErr:    fmt.Errorf(`Argument "strFlag" requires at least 1 argument, got 0`),
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
				Node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', testDesc))},
				Args: []string{"--strFlag", "hello"},
				WantData: &Data{Values: map[string]*Value{
					"strFlag": StringValue("hello"),
				}},
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
				Node: &Node{Processor: NewFlagNode(StringFlag("strFlag", 'f', testDesc))},
				Args: []string{"-f", "hello"},
				WantData: &Data{Values: map[string]*Value{
					"strFlag": StringValue("hello"),
				}},
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
					NewFlagNode(StringFlag("strFlag", 'f', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "--strFlag", "hello", "deux"},
				WantData: &Data{Values: map[string]*Value{
					"strFlag": StringValue("hello"),
					"filler":  StringListValue("un", "deux"),
				}},
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
					NewFlagNode(StringFlag("strFlag", 'f', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
				),
				Args: []string{"uno", "dos", "-f", "hello"},
				WantData: &Data{Values: map[string]*Value{
					"filler":  StringListValue("uno", "dos"),
					"strFlag": StringValue("hello"),
				}},
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
					NewFlagNode(IntFlag("intFlag", 'f', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
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
				WantData: &Data{Values: map[string]*Value{
					"filler":  StringListValue("un", "deux", "quatre"),
					"intFlag": IntValue(3),
				}},
			},
		},
		{
			name: "handles invalid int flag value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(IntFlag("intFlag", 'f', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
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
					NewFlagNode(FloatFlag("floatFlag", 'f', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
				),
				Args: []string{"--floatFlag", "-1.2", "three"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--floatFlag"},
						{value: "-1.2"},
						{value: "three"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"filler":    StringListValue("three"),
					"floatFlag": FloatValue(-1.2),
				}},
			},
		},
		{
			name: "handles invalid float flag value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(FloatFlag("floatFlag", 'f', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
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
					NewFlagNode(BoolFlag("boolFlag", 'b', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
				),
				Args: []string{"okay", "--boolFlag", "then"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "okay"},
						{value: "--boolFlag"},
						{value: "then"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"filler":   StringListValue("okay", "then"),
					"boolFlag": TrueValue(),
				}},
			},
		},
		{
			name: "short bool flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(BoolFlag("boolFlag", 'b', testDesc)),
					StringListNode("filler", testDesc, 1, 2),
				),
				Args: []string{"okay", "-b", "then"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "okay"},
						{value: "-b"},
						{value: "then"},
					},
				},
				WantData: &Data{Values: map[string]*Value{
					"filler":   StringListValue("okay", "then"),
					"boolFlag": TrueValue(),
				}},
			},
		},
		// flag list tests
		{
			name: "flag list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(StringListFlag("slFlag", 's', testDesc, 2, 3)),
					StringListNode("filler", testDesc, 1, 2),
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
				WantData: &Data{Values: map[string]*Value{
					"filler": StringListValue("un"),
					"slFlag": StringListValue("hello", "there"),
				}},
			},
		},
		{
			name: "flag list fails if not enough",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(StringListFlag("slFlag", 's', testDesc, 2, 3)),
					StringListNode("filler", testDesc, 1, 2),
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
				WantStderr: []string{`Argument "slFlag" requires at least 2 arguments, got 1`},
				WantErr:    fmt.Errorf(`Argument "slFlag" requires at least 2 arguments, got 1`),
				WantData: &Data{Values: map[string]*Value{
					"slFlag": StringListValue("hello"),
				}},
			},
		},
		// Int list
		{
			name: "int list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(IntListFlag("ilFlag", 'i', testDesc, 2, 3)),
					StringListNode("filler", testDesc, 1, 2),
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
				WantData: &Data{Values: map[string]*Value{
					"filler": StringListValue("un", "64"),
					"ilFlag": IntListValue(2, 4, 8, 16, 32),
				}},
			},
		},
		{
			name: "int list transform failure",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(IntListFlag("ilFlag", 'i', testDesc, 2, 3)),
					StringListNode("filler", testDesc, 1, 2),
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
					NewFlagNode(FloatListFlag("flFlag", 'f', testDesc, 0, 3)),
					StringListNode("filler", testDesc, 1, 3),
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
				WantData: &Data{Values: map[string]*Value{
					"filler": StringListValue("un", "16.16", "-32", "64"),
					"flFlag": FloatListValue(2, -4.4, 0.8),
				}},
			},
		},
		{
			name: "float list transform failure",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(FloatListFlag("flFlag", 'f', testDesc, 0, 3)),
					StringListNode("filler", testDesc, 1, 2),
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
						FloatListFlag("coordinates", 'c', testDesc, 2, 0),
						BoolFlag("boo", 'o', testDesc),
						StringListFlag("names", 'n', testDesc, 1, 2),
						IntFlag("rating", 'r', testDesc),
					),
					StringListNode("extra", testDesc, 0, 10),
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
				WantData: &Data{Values: map[string]*Value{
					"boo":         TrueValue(),
					"extra":       StringListValue("its", "a", "secret", "message."),
					"names":       StringListValue("greggar", "groog", "beggars"),
					"coordinates": FloatListValue(2.2, 4.4),
					"rating":      IntValue(9),
				}},
			},
		},
		// Transformer tests.

		{
			name: "args get transformed",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringNode("strArg", testDesc, Transformer(StringType, func(v *Value) (*Value, error) {
						return StringValue(strings.ToUpper(v.ToString())), nil
					}, false)),
					IntNode("intArg", testDesc, Transformer(IntType, func(v *Value) (*Value, error) {
						return IntValue(10 * v.ToInt()), nil
					}, false)),
				),
				Args: []string{"hello", "12"},
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("HELLO"),
					"intArg": IntValue(120),
				}},
				wantInput: &Input{
					args: []*inputArg{{value: "HELLO"}, {value: "120"}},
				},
			},
		},
		{
			name: "failure if transformer is for the wrong type",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringNode("strArg", testDesc, Transformer(IntType, func(v *Value) (*Value, error) {
						return StringValue(strings.ToUpper(v.ToString())), nil
					}, false)),
				),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{{value: "hello"}},
				},
				WantErr:    fmt.Errorf("Transformer of type Int cannot be applied to a value with type String"),
				WantStderr: []string{"Transformer of type Int cannot be applied to a value with type String"},
			},
		},
		// Stdoutln tests
		{
			name: "stdoutln works",
			etc: &ExecuteTestCase{
				Node: printlnNode(true, "one", 2, 3.0),
				WantStdout: []string{
					"one 2 3",
				},
			},
		},
		{
			name: "stderrln works",
			etc: &ExecuteTestCase{
				Node: printlnNode(false, "uh", 0),
				WantStderr: []string{
					"uh 0",
				},
				WantErr: fmt.Errorf("uh 0"),
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
				WantStderr: []string{"Branching argument must be one of [b h]"},
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
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
				WantStderr: []string{"Branching argument must be one of [b h]"},
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
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
				}, SerialNodes(StringListNode("sl", testDesc, 0, UnboundedList), printArgsNode().Processor), true),
				Args:       []string{"good", "morning"},
				WantStdout: []string{`sl: StringListValue("good", "morning")`},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("good", "morning"),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "good"},
						{value: "morning"},
					},
				},
			},
		},
		// NodeRepeater tests
		{
			name: "NodeRepeater fails if not enough",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(3, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2"),
					"values": IntListValue(100, 200),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: []string{`Argument "KEY" requires at least 1 argument, got 0`},
			},
		},
		{
			name: "NodeRepeater fails if middle node doen't have enough",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 1)),
				Args: []string{"k1", "100", "k2"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2"),
					"values": IntListValue(100),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
					},
				},
				WantErr:    fmt.Errorf(`Argument "VALUE" requires at least 1 argument, got 0`),
				WantStderr: []string{`Argument "VALUE" requires at least 1 argument, got 0`},
			},
		},
		{
			name: "NodeRepeater fails if too many",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1"),
					"values": IntListValue(100),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
					remaining: []int{2, 3},
				},
				WantErr:    fmt.Errorf(`Unprocessed extra args: [k2 200]`),
				WantStderr: []string{`Unprocessed extra args: [k2 200]`},
			},
		},
		{
			name: "NodeRepeater accepts minimum when no optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2"),
					"values": IntListValue(100, 200),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts minimum when optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 3)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2"),
					"values": IntListValue(100, 200),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts minimum when unlimited optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 3)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2"),
					"values": IntListValue(100, 200),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts maximum when no optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2"),
					"values": IntListValue(100, 200),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts maximum when optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 1)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2"),
					"values": IntListValue(100, 200),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater with unlimited optional accepts a bunch",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, UnboundedList)),
				Args: []string{"k1", "100", "k2", "200", "k3", "300", "k4", "400", "...", "0", "kn", "999"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("k1", "k2", "k3", "k4", "...", "kn"),
					"values": IntListValue(100, 200, 300, 400, 0, 999),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
						{value: "k3"},
						{value: "300"},
						{value: "k4"},
						{value: "400"},
						{value: "..."},
						{value: "0"},
						{value: "kn"},
						{value: "999"},
					},
				},
			},
		},
		// ListBreaker tests
		{
			name: "Handles broken list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi")),
					StringListNode("SL2", testDesc, 0, UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Values: map[string]*Value{
					"SL":  StringListValue("abc", "def"),
					"SL2": StringListValue("ghi", "jkl"),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "List breaker before min value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 3, UnboundedList, ListUntilSymbol("ghi")),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("abc", "def"),
				}},
				WantErr:    fmt.Errorf(`Argument "SL" requires at least 3 arguments, got 2`),
				WantStderr: []string{`Argument "SL" requires at least 3 arguments, got 2`},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
					remaining: []int{2, 3},
				},
			},
		},
		{
			name: "Handles broken list with discard",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi", DiscardBreaker())),
					StringListNode("SL2", testDesc, 0, UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Values: map[string]*Value{
					"SL":  StringListValue("abc", "def"),
					"SL2": StringListValue("jkl"),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "Handles unbroken list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi")),
					StringListNode("SL2", testDesc, 0, UnboundedList),
				),
				Args: []string{"abc", "def", "ghif", "jkl"},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("abc", "def", "ghif", "jkl"),
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghif"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "Fails if arguments required after broken list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi")),
					StringListNode("SL2", testDesc, 1, UnboundedList),
				),
				Args: []string{"abc", "def", "ghif", "jkl"},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("abc", "def", "ghif", "jkl"),
				}},
				WantErr:    fmt.Errorf(`Argument "SL2" requires at least 1 argument, got 0`),
				WantStderr: []string{`Argument "SL2" requires at least 1 argument, got 0`},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghif"},
						{value: "jkl"},
					},
				},
			},
		},
		// StringListListNode tests
		{
			name: "StringListListNode works if no breakers",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Interfaces: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def", "ghi", "jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListNode works with unbounded list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl"},
				WantData: &Data{Interfaces: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListNode works with bounded list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl"},
				WantData: &Data{Interfaces: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListNode works if ends with operator",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl", "|"},
				WantData: &Data{Interfaces: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
						{value: "|"},
					},
				},
			},
		},
		{
			name: "StringListListNode fails if extra args",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl", "|", "other", "stuff"},
				WantData: &Data{Interfaces: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
						{value: "|"},
						{value: "other"},
						{value: "stuff"},
					},
					remaining: []int{8, 9},
				},
				WantErr:    fmt.Errorf("Unprocessed extra args: [other stuff]"),
				WantStderr: []string{"Unprocessed extra args: [other stuff]"},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.etc == nil {
				test.etc = &ExecuteTestCase{}
			}
			test.etc.testInput = true
			ExecuteTest(t, test.etc)
		})
	}
}

func abc() *Node {
	return BranchNode(map[string]*Node{
		"t": AliasNode("TEST_ALIAS", nil,
			CacheNode("TEST_CACHE", nil, SerialNodes(
				&tt{},
				StringNode("PATH", testDesc, SimpleCompletor("clh111", "abcd111")),
				StringNode("TARGET", testDesc, SimpleCompletor("clh222", "abcd222")),
				StringNode("FUNC", testDesc, SimpleCompletor("clh333", "abcd333")),
			))),
	}, nil, false)
}

type tt struct{}

func (t *tt) Usage(*Usage) {}
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

func (t *tt) Complete(input *Input, data *Data) (*Completion, error) {
	t.do(input)
	return nil, nil
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
				WantData: &Data{Values: map[string]*Value{
					"PATH":   StringValue("clh"),
					"TARGET": StringValue("abc"),
				}},
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
					StringNode("s", testDesc, SimpleCompletor("un", "deux", "trois")),
					OptionalIntNode("i", testDesc, SimpleCompletor("2", "1")),
					StringListNode("sl", testDesc, 0, 2, SimpleCompletor("uno", "dos")),
				),
				Want: []string{"deux", "trois", "un"},
				WantData: &Data{Values: map[string]*Value{
					"s": StringValue(""),
				}},
			},
		},
		{
			name: "returns suggestions of first node if up to first arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", testDesc, SimpleCompletor("one", "two", "three")),
					OptionalIntNode("i", testDesc, SimpleCompletor("2", "1")),
					StringListNode("sl", testDesc, 0, 2, SimpleCompletor("uno", "dos")),
				),
				Args: "cmd t",
				Want: []string{"three", "two"},
				WantData: &Data{Values: map[string]*Value{
					"s": StringValue("t"),
				}},
			},
		},
		{
			name: "returns suggestions of middle node if that's where we're at",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", testDesc, SimpleCompletor("one", "two", "three")),
					StringListNode("sl", testDesc, 0, 2, SimpleCompletor("uno", "dos")),
					OptionalIntNode("i", testDesc, SimpleCompletor("2", "1")),
				),
				Args: "cmd three ",
				Want: []string{"dos", "uno"},
				WantData: &Data{Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue(""),
				}},
			},
		},
		{
			name: "returns suggestions of middle node if partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", testDesc, SimpleCompletor("one", "two", "three")),
					StringListNode("sl", testDesc, 0, 2, SimpleCompletor("uno", "dos")),
					OptionalIntNode("i", testDesc, SimpleCompletor("2", "1")),
				),
				Args: "cmd three d",
				Want: []string{"dos"},
				WantData: &Data{Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("d"),
				}},
			},
		},
		{
			name: "returns suggestions in list",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", testDesc, SimpleCompletor("one", "two", "three")),
					StringListNode("sl", testDesc, 0, 2, SimpleCompletor("uno", "dos")),
					OptionalIntNode("i", testDesc, SimpleCompletor("2", "1")),
				),
				Args: "cmd three dos ",
				Want: []string{"dos", "uno"},
				WantData: &Data{Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("dos", ""),
				}},
			},
		},
		{
			name: "returns suggestions for last arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", testDesc, SimpleCompletor("one", "two", "three")),
					StringListNode("sl", testDesc, 0, 2, SimpleCompletor("uno", "dos")),
					OptionalIntNode("i", testDesc, SimpleCompletor("2", "1")),
				),
				Args: "cmd three uno dos ",
				Want: []string{"1", "2"},
				WantData: &Data{Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("uno", "dos"),
				}},
			},
		},
		{
			name: "returns nothing if iterate through all nodes",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("s", testDesc, SimpleCompletor("one", "two", "three")),
					StringListNode("sl", testDesc, 0, 2, SimpleCompletor("uno", "dos")),
					OptionalIntNode("i", testDesc, SimpleCompletor("2", "1")),
				),
				Args: "cmd three uno dos 1 what now",
				WantData: &Data{Values: map[string]*Value{
					"s":  StringValue("three"),
					"sl": StringListValue("uno", "dos"),
					"i":  IntValue(1),
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [what now]"),
			},
		},
		{
			name: "works if empty and list starts",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListNode("sl", testDesc, 1, 2, SimpleCompletor("uno", "dos")),
				),
				Want: []string{"dos", "uno"},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue(),
				}},
			},
		},
		{
			name: "only returns suggestions matching prefix",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListNode("sl", testDesc, 1, 2, SimpleCompletor("zzz-1", "zzz-2", "yyy-3", "zzz-4")),
				),
				Args: "cmd zz",
				Want: []string{"zzz-1", "zzz-2", "zzz-4"},
				WantData: &Data{Values: map[string]*Value{
					"sl": StringListValue("zz"),
				}},
			},
		},
		// Ensure completion iteration stops if necessary.
		{
			name: "stop iterating if a completion returns nil",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("PATH", "dd", &Completor{
						SuggestionFetcher: SimpleFetcher(func(v *Value, d *Data) (*Completion, error) {
							return nil, nil
						}),
					}),
					StringListNode("SUB_PATH", "stc", 0, UnboundedList, SimpleCompletor("un", "deux", "trois")),
				),
				Args: "cmd p",
				WantData: &Data{Values: map[string]*Value{
					"PATH": StringValue("p"),
				}},
			},
		},
		{
			name: "stop iterating if a completion returns an error",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringNode("PATH", "dd", &Completor{
						SuggestionFetcher: SimpleFetcher(func(v *Value, d *Data) (*Completion, error) {
							return nil, fmt.Errorf("ruh-roh")
						}),
					}),
					StringListNode("SUB_PATH", "stc", 0, UnboundedList, SimpleCompletor("un", "deux", "trois")),
				),
				Args:    "cmd p",
				WantErr: fmt.Errorf("ruh-roh"),
				WantData: &Data{Values: map[string]*Value{
					"PATH": StringValue("p"),
				}},
			},
		},
		// Flag completion
		{
			name: "flag name gets completed if single hyphen at end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
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
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
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
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
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
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
				),
				Args: "cmd 1 --greeting h",
				Want: []string{"hey", "hi"},
				WantData: &Data{Values: map[string]*Value{
					"greeting": StringValue("h"),
				}},
			},
		},
		{
			name: "completes for single short flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
				),
				Args: "cmd 1 -h he",
				Want: []string{"hey"},
				WantData: &Data{Values: map[string]*Value{
					"greeting": StringValue("he"),
				}},
			},
		},
		{
			name: "completes for list flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ",
				Want: []string{"johnny", "ralph", "renee"},
				WantData: &Data{Values: map[string]*Value{
					"greeting": StringValue("hey"),
					"names":    StringListValue(""),
				}},
			},
		},
		{
			name: "completes distinct secondary for list flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleDistinctCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ralph ",
				Want: []string{"johnny", "renee"},
				WantData: &Data{Values: map[string]*Value{
					"greeting": StringValue("hey"),
					"names":    StringListValue("ralph", ""),
				}},
			},
		},
		{
			name: "completes last flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleDistinctCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
						FloatFlag("float", 'f', testDesc, SimpleCompletor("1.23", "12.3", "123.4")),
					),
					IntNode("i", testDesc, SimpleCompletor("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ralph renee johnny -f ",
				Want: []string{"1.23", "12.3", "123.4"},
				WantData: &Data{Values: map[string]*Value{
					"greeting": StringValue("hey"),
					"names":    StringListValue("ralph", "renee", "johnny"),
				}},
			},
		},
		{
			name: "completes arg if flag arg isn't at the end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						StringFlag("greeting", 'h', testDesc, SimpleCompletor("hey", "hi")),
						StringListFlag("names", 'n', testDesc, 1, 2, SimpleDistinctCompletor("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					StringListNode("i", testDesc, 1, 2, SimpleCompletor("hey", "ooo")),
				),
				Args: "cmd 1 -h hello beta --names ralph renee johnny ",
				Want: []string{"hey", "ooo"},
				WantData: &Data{Values: map[string]*Value{
					"i":        StringListValue("1", "beta", ""),
					"greeting": StringValue("hello"),
					"names":    StringListValue("ralph", "renee", "johnny"),
				}},
			},
		},
		// Transformer arg tests.
		{
			name: "handles nil option",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc),
				},
				Args: "cmd abc",
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("abc"),
				}},
			},
		},
		{
			name: "list handles nil option",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, 2),
				},
				Args: "cmd abc",
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue("abc"),
				}},
			},
		},
		{
			name: "transformer does transform value when ForComplete is true",
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringNode("strArg", testDesc, Transformer(StringType, func(v *Value) (*Value, error) { return StringValue("newStuff"), nil }, true))),
				Args: "cmd abc",
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue("newStuff"),
				}},
			},
		},
		{
			name:        "FileTransformer doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, FileTransformer()),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue(filepath.Join("relative", "path.txt")),
				}},
			},
		},
		{
			name:        "FileTransformer for list doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("strArg", testDesc, 1, 2, FileListTransformer()),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringListValue(filepath.Join("relative", "path.txt")),
				}},
			},
		},
		{
			name:           "handles transform error",
			filepathAbsErr: fmt.Errorf("bad news bears"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringNode("strArg", testDesc, FileTransformer()),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{Values: map[string]*Value{
					"strArg": StringValue(filepath.Join("relative", "path.txt")),
				}},
			},
		},
		{
			name: "handles transformer of incorrect type",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: IntNode("IntNode", testDesc, FileTransformer()),
				},
				Args: "cmd 123",
				WantData: &Data{Values: map[string]*Value{
					"IntNode": IntValue(123),
				}},
			},
		},
		{
			name:        "transformer list transforms values",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, 2, Transformer(StringListType, func(v *Value) (*Value, error) {
						var sl []string
						for _, s := range v.ToStringList() {
							sl = append(sl, fmt.Sprintf("_%s_", s))
						}
						return StringListValue(sl...), nil
					}, true)),
				},
				Args: "cmd uno dos",
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue(
						"_uno_",
						"_dos_",
					),
				}},
			},
		},
		{
			name:           "handles transform list error",
			filepathAbsErr: fmt.Errorf("bad news bears"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, 2, FileListTransformer()),
				},
				Args: fmt.Sprintf("cmd %s %s", filepath.Join("relative", "path.txt"), filepath.Join("other.txt")),
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue(
						filepath.Join("relative", "path.txt"),
						filepath.Join("other.txt"),
					),
				}},
			},
		},
		{
			name: "handles list transformer of incorrect type",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: StringListNode("slArg", testDesc, 1, 2, FileTransformer()),
				},
				Args: "cmd 123",
				WantData: &Data{Values: map[string]*Value{
					"slArg": StringListValue("123"),
				}},
			},
		},
		// BranchNode completion tests.
		{
			name: "completes branch options",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", testDesc, 1, 3, SimpleCompletor("default", "command", "opts"))), true),
				Want: []string{"a", "alpha", "bravo", "command", "default", "opts"},
				WantData: &Data{Values: map[string]*Value{
					"default": StringListValue(),
				}},
			},
		},
		{
			name: "doesn't complete branch options if complete arg is false",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", testDesc, 1, 3, SimpleCompletor("default", "command", "opts"))), false),
				Want: []string{"command", "default", "opts"},
				WantData: &Data{Values: map[string]*Value{
					"default": StringListValue(),
				}},
			},
		},
		{
			name: "completes for specific branch",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", testDesc, 1, 3, SimpleCompletor("default", "command", "opts"))), true),
				Args: "cmd alpha ",
				Want: []string{"other", "stuff"},
				WantData: &Data{Values: map[string]*Value{
					"hello": StringValue(""),
				}},
			},
		},
		{
			name: "branch node doesn't complete if no default and no branch match",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, nil, true),
				Args:    "cmd some thing else",
				WantErr: fmt.Errorf("Branching argument must be one of [a alpha bravo]"),
			},
		},
		{
			name: "branch node returns default node error if branch completion is false",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(SimpleProcessor(nil, func(i *Input, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("bad news bears")
				})), false),
				Args:    "cmd ",
				WantErr: fmt.Errorf("bad news bears"),
			},
		},
		{
			name: "branch node returns default node error and branch completions",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(SimpleProcessor(nil, func(i *Input, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("bad news bears")
				})), true),
				Args:    "cmd ",
				Want:    []string{"a", "alpha", "bravo"},
				WantErr: fmt.Errorf("bad news bears"),
			},
		},
		{
			name: "completes branch options with partial completion",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", testDesc, 1, 3, SimpleCompletor("default", "command", "opts", "ahhhh"))), true),
				Args: "cmd a",
				Want: []string{"a", "ahhhh", "alpha"},
				WantData: &Data{Values: map[string]*Value{
					"default": StringListValue("a"),
				}},
			},
		},
		{
			name: "completes default options",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalStringNode("hello", testDesc, SimpleCompletor("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(StringListNode("default", testDesc, 1, 3, SimpleCompletor("default", "command", "opts"))), true),
				Args: "cmd something ",
				WantData: &Data{Values: map[string]*Value{
					"default": StringListValue("something", ""),
				}},
				Want: []string{"command", "default", "opts"},
			},
		},
		// StringMenu tests.
		{
			name: "StringMenu completes choices",
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringMenu("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd ",
				Want: []string{"abc", "def", "ghi"},
				WantData: &Data{Values: map[string]*Value{
					"sm": StringValue(""),
				}},
			},
		},
		{
			name: "StringMenu completes partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(StringMenu("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd g",
				Want: []string{"ghi"},
				WantData: &Data{Values: map[string]*Value{
					"sm": StringValue("g"),
				}},
			},
		},
		// Commands with different value types.
		{
			name: "int arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntNode("iArg", testDesc, SimpleCompletor("12", "45", "456", "468", "7"))),
				Args: "cmd 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]*Value{
					"iArg": IntValue(4),
				}},
			},
		},
		{
			name: "optional int arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(OptionalIntNode("iArg", testDesc, SimpleCompletor("12", "45", "456", "468", "7"))),
				Args: "cmd 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]*Value{
					"iArg": IntValue(4),
				}},
			},
		},
		{
			name: "int list arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntListNode("iArg", testDesc, 2, 3, SimpleCompletor("12", "45", "456", "468", "7"))),
				Args: "cmd 1 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]*Value{
					"iArg": IntListValue(1, 4),
				}},
			},
		},
		{
			name: "int list arg gets completed if previous one was invalid",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntListNode("iArg", testDesc, 2, 3, SimpleCompletor("12", "45", "456", "468", "7"))),
				Args: "cmd one 4",
				Want: []string{"45", "456", "468"},
			},
		},
		{
			name: "int list arg optional args get completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(IntListNode("iArg", testDesc, 2, 3, SimpleCompletor("12", "45", "456", "468", "7"))),
				Args: "cmd 1 2 3 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]*Value{
					"iArg": IntListValue(1, 2, 3, 4),
				}},
			},
		},
		{
			name: "float arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(FloatNode("fArg", testDesc, SimpleCompletor("12", "4.5", "45.6", "468", "7"))),
				Args: "cmd 4",
				Want: []string{"4.5", "45.6", "468"},
				WantData: &Data{Values: map[string]*Value{
					"fArg": FloatValue(4),
				}},
			},
		},
		{
			name: "float list arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(FloatListNode("fArg", testDesc, 1, 2, SimpleCompletor("12", "4.5", "45.6", "468", "7"))),
				Want: []string{"12", "4.5", "45.6", "468", "7"},
				WantData: &Data{Values: map[string]*Value{
					"fArg": FloatListValue(),
				}},
			},
		},
		{
			name: "bool arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(BoolNode("bArg", testDesc)),
				Want: []string{"0", "1", "F", "FALSE", "False", "T", "TRUE", "True", "f", "false", "t", "true"},
				WantData: &Data{Values: map[string]*Value{
					"bArg": FalseValue(),
				}},
			},
		},
		// NodeRepeater
		{
			name: "NodeRepeater completes first node",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Want: []string{"alpha", "bravo", "brown", "charlie"},
				WantData: &Data{Values: map[string]*Value{
					"keys": StringListValue(""),
				}},
			},
		},
		{
			name: "NodeRepeater completes first node partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Args: "cmd b",
				Want: []string{"bravo", "brown"},
				WantData: &Data{Values: map[string]*Value{
					"keys": StringListValue("b"),
				}},
			},
		},
		{
			name: "NodeRepeater completes second node",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Args: "cmd brown ",
				Want: []string{"1", "121", "1213121"},
				WantData: &Data{Values: map[string]*Value{
					"keys": StringListValue("brown"),
				}},
			},
		},
		{
			name: "NodeRepeater completes second node partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Args: "cmd brown 12",
				Want: []string{"121", "1213121"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("brown"),
					"values": IntListValue(12),
				}},
			},
		},
		{
			name: "NodeRepeater completes second required iteration",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 0)),
				Args: "cmd brown 12 c",
				Want: []string{"charlie"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("brown", "c"),
					"values": IntListValue(12),
				}},
			},
		},
		{
			name: "NodeRepeater completes optional iteration",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 1)),
				Args: "cmd brown 12 charlie 21 alpha 1",
				Want: []string{"1", "121", "1213121"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("brown", "charlie", "alpha"),
					"values": IntListValue(12, 21, 1),
				}},
			},
		},
		{
			name: "NodeRepeater completes unbounded optional iteration",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, UnboundedList)),
				Args: "cmd brown 12 charlie 21 alpha 100 delta 98 b",
				Want: []string{"bravo", "brown"},
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("brown", "charlie", "alpha", "delta", "b"),
					"values": IntListValue(12, 21, 100, 98),
				}},
			},
		},
		{
			name: "NodeRepeater doesn't complete beyond repeated iterations",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 1)),
				Args: "cmd brown 12 charlie 21 alpha 100 b",
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("brown", "charlie", "alpha"),
					"values": IntListValue(12, 21, 100),
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [b]"),
			},
		},
		{
			name: "NodeRepeater works if fully processed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 1), StringNode("S", testDesc, SimpleCompletor("un", "deux", "trois"))),
				Args: "cmd brown 12 charlie 21 alpha 100",
				WantData: &Data{Values: map[string]*Value{
					"keys":   StringListValue("brown", "charlie", "alpha"),
					"values": IntListValue(12, 21, 100),
				}},
			},
		},
		// ListBreaker tests
		{
			name: "Suggests things after broken list",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi"), SimpleCompletor("un", "deux", "trois")),
					StringListNode("SL2", testDesc, 0, UnboundedList, SimpleCompletor("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: []string{"one", "three", "two"},
				WantData: &Data{Values: map[string]*Value{
					"SL":  StringListValue("abc", "def"),
					"SL2": StringListValue("ghi", ""),
				}},
			},
		},
		{
			name: "Suggests things after broken list with discard",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi", DiscardBreaker()), SimpleCompletor("un", "deux", "trois")),
					StringListNode("SL2", testDesc, 0, UnboundedList, SimpleCompletor("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: []string{"one", "three", "two"},
				WantData: &Data{Values: map[string]*Value{
					"SL":  StringListValue("abc", "def"),
					"SL2": StringListValue(""),
				}},
			},
		},
		{
			name: "Suggests things before list is broken",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi"), SimpleCompletor("un", "deux", "trois", "uno")),
					StringListNode("SL2", testDesc, 0, UnboundedList, SimpleCompletor("one", "two", "three")),
				),
				Args: "cmd abc def un",
				Want: []string{"un", "uno"},
				WantData: &Data{Values: map[string]*Value{
					"SL": StringListValue("abc", "def", "un"),
				}},
			},
		},
		// StringListListNode
		{
			name: "StringListListNode works if no breakers",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList, SimpleCompletor("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: []string{"one", "three", "two"},
				WantData: &Data{Interfaces: map[string]interface{}{
					"SLL": [][]string{{"abc", "def", "ghi", ""}},
				}},
			},
		},
		{
			name: "StringListListNode works with breakers",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList, SimpleCompletor("one", "two", "three")),
				),
				Args: "cmd abc def | ghi t",
				Want: []string{"three", "two"},
				WantData: &Data{Interfaces: map[string]interface{}{
					"SLL": [][]string{{"abc", "def"}, {"ghi", "t"}},
				}},
			},
		},
		{
			name: "completes args after StringListListNode",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 1, SimpleCompletor("one", "two", "three")),
					StringNode("S", testDesc, SimpleCompletor("un", "deux", "trois")),
				),
				Args: "cmd abc def | ghi | ",
				Want: []string{"deux", "trois", "un"},
				WantData: &Data{
					Interfaces: map[string]interface{}{
						"SLL": [][]string{{"abc", "def"}, {"ghi"}},
					},
					Values: map[string]*Value{
						"S": StringValue(""),
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

			CompleteTest(t, test.ctc)
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

func printlnNode(stdout bool, a ...interface{}) *Node {
	return &Node{
		Processor: ExecutorNode(func(output Output, _ *Data) error {
			if !stdout {
				return output.Stderrln(a...)
			}
			output.Stdoutln(a...)
			return nil
		}),
	}
}

func printArgsNode() *Node {
	return &Node{
		Processor: ExecutorNode(func(output Output, data *Data) error {
			for k, v := range data.Values {
				output.Stdoutf("%s: %v", k, v)
			}
			return nil
		}),
	}
}

func sampleRepeaterNode(minN, optionalN int) Processor {
	return NodeRepeater(SerialNodes(
		StringNode("KEY", testDesc, CustomSetter(func(v *Value, d *Data) {
			if !d.HasArg("keys") {
				d.Set("keys", StringListValue(v.ToString()))
			} else {
				d.Set("keys", StringListValue(append(d.StringList("keys"), v.ToString())...))
			}
		}), SimpleCompletor("alpha", "bravo", "charlie", "brown")),
		IntNode("VALUE", testDesc, CustomSetter(func(v *Value, d *Data) {
			if !d.HasArg("values") {
				d.Set("values", IntListValue(v.ToInt()))
			} else {
				d.Set("values", IntListValue(append(d.IntList("values"), v.ToInt())...))
			}
		}), SimpleCompletor("1", "121", "1213121")),
	), minN, optionalN)
}

func TestRunNodes(t *testing.T) {
	sum := SerialNodes(
		Description("Adds A and B"),
		IntNode("A", "The first value"),
		IntNode("B", "The second value"),
		ExecutorNode(func(o Output, d *Data) error {
			o.Stdoutln(d.Int("A") + d.Int("B"))
			return nil
		}),
	)
	for _, test := range []struct {
		name string
		rtc  *RunNodeTestCase
	}{
		// execute tests (without keyword)
		{
			name: "no keyword requires arguments",
			rtc: &RunNodeTestCase{
				Node: sum,
				WantStderr: []string{
					`Argument "A" requires at least 1 argument, got 0`,
					GetUsage(sum).String(),
				},
				WantErr: fmt.Errorf(`Argument "A" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "no keyword fails with extra",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"5", "7", "9"},
				WantStderr: []string{
					`Unprocessed extra args: [9]`,
					GetUsage(sum).String(),
				},
				WantErr: fmt.Errorf(`Unprocessed extra args: [9]`),
			},
		},
		{
			name: "successfully runs node",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"5", "7"},
				WantStdout: []string{
					"12",
				},
			},
		},
		// execute tests with keyword
		{
			name: "execute requires arguments",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"execute", "TMP_FILE"},
				WantStderr: []string{
					`Argument "A" requires at least 1 argument, got 0`,
					GetUsage(sum).String(),
				},
				WantErr: fmt.Errorf(`Argument "A" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "execute fails with extra",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"execute", "TMP_FILE", "5", "7", "9"},
				WantStderr: []string{
					`Unprocessed extra args: [9]`,
					GetUsage(sum).String(),
				},
				WantErr: fmt.Errorf(`Unprocessed extra args: [9]`),
			},
		},
		{
			name: "successfully runs node",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"execute", "TMP_FILE", "5", "7"},
				WantStdout: []string{
					"12",
				},
			},
		},
		{
			name: "execute data",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					SimpleExecutableNode(
						"echo hello",
						"echo there",
					),
				),
				Args: []string{"execute", "TMP_FILE"},
				WantFileContents: []string{
					"echo hello",
					"echo there",
				},
			},
		},
		// Autocomplete tests
		{
			name: "autocompletes empty",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					StringListNode("SL_ARG", "", 1, UnboundedList, SimpleCompletor("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", ""},
				WantStdout: []string{
					"four",
					"one",
					"three",
					"two",
				},
			},
		},
		{
			name: "autocompletes empty with command",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					StringListNode("SL_ARG", "", 1, UnboundedList, SimpleCompletor("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", "cmd "},
				WantStdout: []string{
					"four",
					"one",
					"three",
					"two",
				},
			},
		},
		{
			name: "autocompletes partial arg",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					StringListNode("SL_ARG", "", 1, UnboundedList, SimpleCompletor("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", "cmd t"},
				WantStdout: []string{
					"three",
					"two",
				},
			},
		},
		{
			name: "autocompletes later args",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					StringListNode("SL_ARG", "", 1, UnboundedList, SimpleCompletor("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", "cmd three f"},
				WantStdout: []string{
					"four",
				},
			},
		},
		{
			name: "autocompletes nothing if past last arg",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					StringListNode("SL_ARG", "", 1, 0, SimpleCompletor("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", "cmd three f"},
			},
		},
		// Usage tests
		{
			name: "prints usage",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"usage"},
				WantStdout: []string{
					GetUsage(sum).String(),
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.rtc.SkipDataCheck = true
			RunNodeTest(t, test.rtc)
		})
	}
}
