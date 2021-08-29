package command

import (
	"fmt"
	"testing"
)

func TestBashNode(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *ExecuteTestCase
		frs  []*FakeRun
	}{
		// Generic tests
		{
			name: "bash command returns an error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"})),
				WantRunContents: [][]string{
					{"echo hello"},
				},
				WantErr:    fmt.Errorf("oops"),
				WantStderr: []string{"oops"},
			},
			frs: []*FakeRun{
				{
					Err: fmt.Errorf("oops"),
				},
			},
		},
		// String
		{
			name: "bash node for string",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"})),
				WantRunContents: [][]string{
					{"echo hello"},
				},
				WantData: &Data{
					"s": StringValue("aloha"),
				},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"aloha"},
				},
			},
		},
		// String list
		{
			name: "bash node for string list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(StringListType, "s", []string{"echo hello"})),
				WantRunContents: [][]string{
					{"echo hello"},
				},
				WantData: &Data{
					"s": StringListValue("aloha", "hello there", "howdy"),
				},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"aloha", "hello there", "howdy"},
				},
			},
		},
		// Int
		{
			name: "bash node for int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo 1248"})),
				WantRunContents: [][]string{
					{"echo 1248"},
				},
				WantData: &Data{
					"i": IntValue(1248),
				},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"1248"},
				},
			},
		},
		{
			name: "error when not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo two"})),
				WantRunContents: [][]string{
					{"echo two"},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"two"},
				},
			},
		},
		// Int list
		{
			name: "bash node for int list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo primes"})),
				WantRunContents: [][]string{
					{"echo primes"},
				},
				WantData: &Data{
					"i": IntListValue(2, 3, 5, 7),
				},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"2", "3", "5", "7"},
				},
			},
		},
		{
			name: "error when not an int in list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo two"})),
				WantRunContents: [][]string{
					{"echo two"},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"2", "two", "200"},
				},
			},
		},
		// Int
		{
			name: "bash node for int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo 1248"})),
				WantRunContents: [][]string{
					{"echo 1248"},
				},
				WantData: &Data{
					"i": IntValue(1248),
				},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"1248"},
				},
			},
		},
		{
			name: "error when not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo two"})),
				WantRunContents: [][]string{
					{"echo two"},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"two"},
				},
			},
		},
		// Int list
		{
			name: "bash node for int list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo primes"})),
				WantRunContents: [][]string{
					{"echo primes"},
				},
				WantData: &Data{
					"i": IntListValue(2, 3, 5, 7),
				},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"2", "3", "5", "7"},
				},
			},
		},
		{
			name: "error when not an int in list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo two"})),
				WantRunContents: [][]string{
					{"echo two"},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"2", "two", "200"},
				},
			},
		},
		// Validators
		{
			name: "bash node with validators",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo 1248"}, IntNonNegative())),
				WantRunContents: [][]string{
					{"echo 1248"},
				},
				WantData: &Data{
					"i": IntValue(1248),
				},
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"1248"},
				},
			},
		},
		{
			name: "bash node with failing validators",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo -1248"}, IntNonNegative())),
				WantRunContents: [][]string{
					{"echo -1248"},
				},
				WantStderr: []string{"validation failed: [IntNonNegative] value isn't non-negative"},
				WantErr:    fmt.Errorf("validation failed: [IntNonNegative] value isn't non-negative"),
			},
			frs: []*FakeRun{
				{
					Stdout: []string{"-1248"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ExecuteTest(t, test.etc, &ExecuteTestOptions{RunResponses: test.frs})
		})
	}
}
