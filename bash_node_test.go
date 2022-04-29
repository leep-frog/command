package command

import (
	"fmt"
	"testing"
)

func TestBashNode(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *ExecuteTestCase
	}{
		// Generic tests
		{
			name: "bash command returns an error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[string]("s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantErr:    fmt.Errorf("failed to execute bash command: oops"),
				WantStderr: []string{"failed to execute bash command: oops"},
				RunResponses: []*FakeRun{
					{
						Err: fmt.Errorf("oops"),
					},
				},
			},
		},
		{
			name: "bash command prints stderr on error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[string]("s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantErr: fmt.Errorf("failed to execute bash command: oops"),
				WantStderr: []string{
					"un\ndeux\ntrois",
					"failed to execute bash command: oops",
				},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"one", "two", "three"},
						Stderr: []string{"un", "deux", "trois"},
						Err:    fmt.Errorf("oops"),
					},
				},
			},
		},
		{
			name: "bash command hides stderr on error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand("s", []string{"echo hello"}, HideStderr[string]())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantErr: fmt.Errorf("failed to execute bash command: oops"),
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"one", "two", "three"},
						Stderr: []string{"un", "deux", "trois"},
						Err:    fmt.Errorf("oops"),
					},
				},
			},
		},
		// String
		{
			name: "bash node for string",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[string]("s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"aloha"},
					},
				},
			},
		},
		{
			name: "bash node for string works with empty output",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[string]("s", []string{"don't echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"don't echo hello",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s": "",
				}},
				RunResponses: []*FakeRun{{}},
			},
		},
		{
			name: "successful command shows stderr",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[string]("s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				WantStderr: []string{"ahola"},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"aloha"},
						Stderr: []string{"ahola"},
					},
				},
			},
		},
		{
			name: "successful command hides stderr",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand("s", []string{"echo hello"}, HideStderr[string]())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"aloha"},
						Stderr: []string{"ahola"},
					},
				},
			},
		},
		// String list
		{
			name: "bash node for string list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[[]string]("s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s": []string{"aloha", "hello there", "howdy"},
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"aloha", "hello there", "howdy"},
					},
				},
			},
		},
		{
			name: "bash node for string list forwards stdout",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand("s", []string{"echo hello"}, ForwardStdout[[]string]())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantStdout: []string{
					"aloha\nhello there\nhowdy",
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": []string{"aloha", "hello there", "howdy"},
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"aloha", "hello there", "howdy"},
					},
				},
			},
		},
		{
			name: "bash node for string list gets empty new lines at end",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[[]string]("s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s": []string{"aloha", "hello there", "howdy", "", ""},
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"aloha", "hello there", "howdy", "", ""},
					},
				},
			},
		},
		// Int
		{
			name: "bash node for int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"echo 1248"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo 1248",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"i": 1248,
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"1248"},
					},
				},
			},
		},
		{
			name: "bash node for int works with empty output",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"don't echo 1248"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"don't echo 1248",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"i": 0,
				}},
				RunResponses: []*FakeRun{{}},
			},
		},
		{
			name: "error when not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"echo two"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"two"},
					},
				},
			},
		},
		// Int list
		{
			name: "bash node for int list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[[]int]("i", []string{"echo primes"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo primes",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"i": []int{2, 3, 5, 7},
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"2", "3", "5", "7"},
					},
				},
			},
		},
		{
			name: "error when not an int in list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[[]int]("i", []string{"echo two"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"2", "two", "200"},
					},
				},
			},
		},
		// Int
		{
			name: "bash node for int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"echo 1248"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo 1248",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"i": 1248,
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"1248"},
					},
				},
			},
		},
		{
			name: "error when not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"echo two"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"two"},
					},
				},
			},
		},
		// Int list
		{
			name: "bash node for int list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[[]int]("i", []string{"echo primes"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo primes",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"i": []int{2, 3, 5, 7},
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"2", "3", "5", "7"},
					},
				},
			},
		},
		{
			name: "error when not an int in list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[[]int]("i", []string{"echo two"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: []string{`strconv.Atoi: parsing "two": invalid syntax`},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"2", "two", "200"},
					},
				},
			},
		},
		// We don't need to test every value type, because we just use the
		// valueHandler interface.

		// Validators
		{
			name: "bash node with validators",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"echo 1248"}, NonNegative[int]())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo 1248",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"i": 1248,
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"1248"},
					},
				},
			},
		},
		{
			name: "bash node with failing validators",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"echo -1248"}, NonNegative[int]())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo -1248",
				}},
				WantStderr: []string{"validation failed: [NonNegative] value isn't non-negative"},
				WantErr:    fmt.Errorf("validation failed: [NonNegative] value isn't non-negative"),

				RunResponses: []*FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
		},
		{
			name: "bash node with failing validators and hidden stderr",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand[int]("i", []string{"echo -1248"}, NonNegative[int](), HideStderr[int]())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo -1248",
				}},
				WantErr: fmt.Errorf("validation failed: [NonNegative] value isn't non-negative"),
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			ExecuteTest(t, test.etc)
		})
	}
}
