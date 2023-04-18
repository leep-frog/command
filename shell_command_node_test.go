package command

import (
	"fmt"
	"strings"
	"testing"
)

func TestShellCommand(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *ExecuteTestCase
	}{
		// Generic tests
		{
			name: "shell command returns an error",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
				WantErr:         fmt.Errorf("failed to execute shell command: oops"),
				WantStderr:      "failed to execute shell command: oops\n",
				RunResponses: []*FakeRun{
					{
						Err: fmt.Errorf("oops"),
					},
				},
			},
		},
		{
			name: "shell command prints stderr on error",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
				WantErr:         fmt.Errorf("failed to execute shell command: oops"),
				WantStderr: strings.Join([]string{
					"un",
					"deux",
					"trois",
					"failed to execute shell command: oops\n",
				}, "\n"),
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
			name: "shell command hides stderr on error",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}, HideStderr: true}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
				WantErr:         fmt.Errorf("failed to execute shell command: oops"),
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
			name: "shell node for string",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
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
			name: "shell node for string works with empty output",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "don't", Args: []string{"echo", "hello"}}),
				WantRunContents: []*RunContents{{"don't", []string{"echo", "hello"}}},
				WantData: &Data{Values: map[string]interface{}{
					"s": "",
				}},
				RunResponses: []*FakeRun{{}},
			},
		},
		{
			name: "successful command shows stderr",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
				WantData: &Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				WantStderr: "ahola\n",
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
				Node:            SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}, HideStderr: true}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
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
			name: "shell node for string list",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[[]string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
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
			name: "shell node for string list forwards stdout",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[[]string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}, ForwardStdout: true}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
				WantStdout:      "aloha\nhello there\nhowdy\n",
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
			name: "shell node with output streamer",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					&ShellCommand[[]string]{
						ArgName:       "s",
						CommandName:   "echo",
						Args:          []string{"hello"},
						ForwardStdout: true,
						OutputStreamProcessor: func(o Output, d *Data, s string) error {
							o.Stdoutf("Streamer received: %s", s)
							return nil
						},
					},
				),
				WantRunContents: []*RunContents{{"echo", []string{
					"hello",
				}}},
				WantStdout: strings.Join([]string{
					"aloha",
					"Streamer received: aloha",
					"hello there",
					"Streamer received: hello there",
					"howdy",
					"Streamer received: howdy",
					"",
				}, "\n"),
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
			name: "shell node for string list gets empty new lines at end",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[[]string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*RunContents{{"echo", []string{"hello"}}},
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
			name: "shell node for int",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"1248"}}),
				WantRunContents: []*RunContents{{"echo", []string{"1248"}}},
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
			name: "shell node for int works with empty output",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "don't", Args: []string{"echo", "1248"}}),
				WantRunContents: []*RunContents{{"don't", []string{"echo", "1248"}}},
				WantData: &Data{Values: map[string]interface{}{
					"i": 0,
				}},
				RunResponses: []*FakeRun{{}},
			},
		},
		{
			name: "error when not an int",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*RunContents{{"echo", []string{"two"}}},
				WantErr:         fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr:      "strconv.Atoi: parsing \"two\": invalid syntax\n",
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"two"},
					},
				},
			},
		},
		// Int list
		{
			name: "shell node for int list",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"primes"}}),
				WantRunContents: []*RunContents{{"echo", []string{"primes"}}},
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
				Node:            SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*RunContents{{"echo", []string{"two"}}},
				WantErr:         fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr:      "strconv.Atoi: parsing \"two\": invalid syntax\n",
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"2", "two", "200"},
					},
				},
			},
		},
		// Int
		{
			name: "shell node for int",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"1248"}}),
				WantRunContents: []*RunContents{{"echo", []string{"1248"}}},
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
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*RunContents{{"echo", []string{"two"}}},
				WantErr:         fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr:      "strconv.Atoi: parsing \"two\": invalid syntax\n",
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"two"},
					},
				},
			},
		},
		// Int list
		{
			name: "shell node for int list",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"primes"}}),
				WantRunContents: []*RunContents{{"echo", []string{"primes"}}},
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
				Node:            SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*RunContents{{"echo", []string{"two"}}},
				WantErr:         fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr:      "strconv.Atoi: parsing \"two\": invalid syntax\n",
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
			name: "shell node with validators",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"1248"}, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: []*RunContents{{"echo", []string{"1248"}}},
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
			name: "shell node with failing validators",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"-1248"}, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: []*RunContents{{"echo", []string{"-1248"}}},
				WantStderr:      "validation for \"i\" failed: [NonNegative] value isn't non-negative\n",
				WantErr:         fmt.Errorf("validation for \"i\" failed: [NonNegative] value isn't non-negative"),

				RunResponses: []*FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
		},
		{
			name: "shell node with failing validators and hidden stderr",
			etc: &ExecuteTestCase{
				Node:            SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"-1248"}, HideStderr: true, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: []*RunContents{{"echo", []string{"-1248"}}},
				WantErr:         fmt.Errorf("validation for \"i\" failed: [NonNegative] value isn't non-negative"),
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
		},
		// formatArgs tests
		/*{
			name: "shell node formats all input types",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					Arg[int]("i", testDesc),
					Arg[float64]("f", testDesc),
					Arg[bool]("b", testDesc),
					ListArg("sl", testDesc, 0, UnboundedList, Default([]string{"alpha", "beta", "gamma", "delta"})),
					ListArg("il", testDesc, 0, UnboundedList, Default([]int{1, 1, 2, 3, 5, 8})),
					ListArg("fl", testDesc, 0, UnboundedList, Default([]float64{13.21, 34.55})),
					&ShellCommand[string]{ArgName: "bc", Args: []string{
						"echo 1: %s && echo %s",
						"echo 2: %s %s",
						"echo list 1: %s",
						"echo list 2: %s",
						"echo list 3: %s",
					},
						FormatArgs: []ShellCommandDataStringer[string]{
							NewShellCommandDataStringer[string](Arg[string]("s", testDesc), ""),
							NewShellCommandDataStringer[string](Arg[int]("i", testDesc), ""),
							NewShellCommandDataStringer[string](Arg[float64]("f", testDesc), ""),
							NewShellCommandDataStringer[string](Arg[bool]("b", testDesc), ""),
							NewShellCommandDataStringer[string](ListArg("sl", testDesc, 0, UnboundedList, Default([]string{"alpha", "beta", "gamma", "delta"})), " , "),
							NewShellCommandDataStringer[string](ListArg("il", testDesc, 0, UnboundedList, Default([]int{1, 1, 2, 3, 5, 8})), ":"),
							NewShellCommandDataStringer[string](ListArg("fl", testDesc, 0, UnboundedList, Default([]float64{13.21, 34.55})), " "),
						},
					}),
				Args: []string{
					"stringVal",
					"9",
					"-12.345678",
					"true",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo 1: stringVal && echo 9",
					"echo 2: -12.345678 true",
					"echo list 1: alpha , beta , gamma , delta",
					"echo list 2: 1:1:2:3:5:8",
					"echo list 3: 13.21 34.55",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "stringVal",
					"i":  9,
					"f":  -12.345678,
					"b":  true,
					"sl": []string{"alpha", "beta", "gamma", "delta"},
					"il": []int{1, 1, 2, 3, 5, 8},
					"fl": []float64{13.21, 34.55},
					"bc": "-1248",
				}},
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
		},
		{
			name: "shell node formatting fails on error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					&ShellCommand[string]{ArgName: "bc", Args: []string{
						"echo 1: %s",
					},
						FormatArgs: []ShellCommandDataStringer[string]{
							CustomShellCommandDataStringer[string](func(d *Data) (string, error) {
								return "bleh", fmt.Errorf("ouch")
							}),
						},
					},
				),
				WantErr:    fmt.Errorf("failed to get string for shell formatting: ouch"),
				WantStderr: "failed to get string for shell formatting: ouch\n",
			},
		},*/
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			ExecuteTest(t, test.etc)
		})
	}
}
