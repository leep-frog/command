package command

import (
	"fmt"
	"runtime"
	"testing"
)

func TestBashNode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(runtime.GOOS, "bash tests do not work with windows")
	}
	for _, test := range []struct {
		name string
		etc  *ExecuteTestCase
	}{
		// Generic tests
		{
			name: "bash command returns an error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"})),
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
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"})),
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
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"}, HideStderr())),
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
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{
					"s": StringValue("aloha"),
				},
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
				Node: SerialNodes(BashCommand(StringType, "s", []string{"don't echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"don't echo hello",
				}},
				WantData: &Data{
					"s": StringValue(""),
				},
				RunResponses: []*FakeRun{{}},
			},
		},
		{
			name: "successful command shows stderr",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{
					"s": StringValue("aloha"),
				},
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
				Node: SerialNodes(BashCommand(StringType, "s", []string{"echo hello"}, HideStderr())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{
					"s": StringValue("aloha"),
				},
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
				Node: SerialNodes(BashCommand(StringListType, "s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{
					"s": StringListValue("aloha", "hello there", "howdy"),
				},
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
				Node: SerialNodes(BashCommand(StringListType, "s", []string{"echo hello"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{
					"s": StringListValue("aloha", "hello there", "howdy", "", ""),
				},
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
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo 1248"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo 1248",
				}},
				WantData: &Data{
					"i": IntValue(1248),
				},
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
				Node: SerialNodes(BashCommand(IntType, "i", []string{"don't echo 1248"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"don't echo 1248",
				}},
				WantData: &Data{
					"i": IntValue(0),
				},
				RunResponses: []*FakeRun{{}},
			},
		},
		{
			name: "error when not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo two"})),
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
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo primes"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo primes",
				}},
				WantData: &Data{
					"i": IntListValue(2, 3, 5, 7),
				},
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
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo two"})),
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
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo 1248"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo 1248",
				}},
				WantData: &Data{
					"i": IntValue(1248),
				},
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
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo two"})),
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
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo primes"})),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo primes",
				}},
				WantData: &Data{
					"i": IntListValue(2, 3, 5, 7),
				},
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
				Node: SerialNodes(BashCommand(IntListType, "i", []string{"echo two"})),
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
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo 1248"}, IntNonNegative())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo 1248",
				}},
				WantData: &Data{
					"i": IntValue(1248),
				},
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
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo -1248"}, IntNonNegative())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo -1248",
				}},
				WantStderr: []string{"validation failed: [IntNonNegative] value isn't non-negative"},
				WantErr:    fmt.Errorf("validation failed: [IntNonNegative] value isn't non-negative"),

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
				Node: SerialNodes(BashCommand(IntType, "i", []string{"echo -1248"}, IntNonNegative(), HideStderr())),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo -1248",
				}},
				WantErr: fmt.Errorf("validation failed: [IntNonNegative] value isn't non-negative"),
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
