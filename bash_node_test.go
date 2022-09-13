package command

import (
	"fmt"
	"strings"
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
				Node: SerialNodes(&BashCommand[string]{ArgName: "s", Contents: []string{"echo hello"}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantErr:    fmt.Errorf("failed to execute bash command: oops"),
				WantStderr: "failed to execute bash command: oops\n",
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
				Node: SerialNodes(&BashCommand[string]{ArgName: "s", Contents: []string{"echo hello"}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantErr: fmt.Errorf("failed to execute bash command: oops"),
				WantStderr: strings.Join([]string{
					"un\ndeux\ntrois",
					"failed to execute bash command: oops\n",
				}, ""),
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
				Node: SerialNodes(&BashCommand[string]{ArgName: "s", Contents: []string{"echo hello"}, HideStderr: true}),
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
				Node: SerialNodes(&BashCommand[string]{ArgName: "s", Contents: []string{"echo hello"}}),
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
				Node: SerialNodes(&BashCommand[string]{ArgName: "s", Contents: []string{"don't echo hello"}}),
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
				Node: SerialNodes(&BashCommand[string]{ArgName: "s", Contents: []string{"echo hello"}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				WantStderr: "ahola",
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
				Node: SerialNodes(&BashCommand[string]{ArgName: "s", Contents: []string{"echo hello"}, HideStderr: true}),
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
				Node: SerialNodes(&BashCommand[[]string]{ArgName: "s", Contents: []string{"echo hello"}}),
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
				Node: SerialNodes(&BashCommand[[]string]{ArgName: "s", Contents: []string{"echo hello"}, ForwardStdout: true}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo hello",
				}},
				WantStdout: "aloha\nhello there\nhowdy",
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
				Node: SerialNodes(&BashCommand[[]string]{ArgName: "s", Contents: []string{"echo hello"}}),
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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"echo 1248"}}),
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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"don't echo 1248"}}),
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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"echo two"}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
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
				Node: SerialNodes(&BashCommand[[]int]{ArgName: "i", Contents: []string{"echo primes"}}),
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
				Node: SerialNodes(&BashCommand[[]int]{ArgName: "i", Contents: []string{"echo two"}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"echo 1248"}}),
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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"echo two"}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
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
				Node: SerialNodes(&BashCommand[[]int]{ArgName: "i", Contents: []string{"echo primes"}}),
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
				Node: SerialNodes(&BashCommand[[]int]{ArgName: "i", Contents: []string{"echo two"}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo two",
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"echo 1248"}, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"echo -1248"}, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo -1248",
				}},
				WantStderr: "validation for \"i\" failed: [NonNegative] value isn't non-negative\n",
				WantErr:    fmt.Errorf("validation for \"i\" failed: [NonNegative] value isn't non-negative"),

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
				Node: SerialNodes(&BashCommand[int]{ArgName: "i", Contents: []string{"echo -1248"}, HideStderr: true, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo -1248",
				}},
				WantErr: fmt.Errorf("validation for \"i\" failed: [NonNegative] value isn't non-negative"),
				RunResponses: []*FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
		},
		// formatArgs tests
		{
			name: "bash node formats all input types",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					Arg[int]("i", testDesc),
					Arg[float64]("f", testDesc),
					Arg[bool]("b", testDesc),
					ListArg("sl", testDesc, 0, UnboundedList, Default([]string{"alpha", "beta", "gamma", "delta"})),
					ListArg("il", testDesc, 0, UnboundedList, Default([]int{1, 1, 2, 3, 5, 8})),
					ListArg("fl", testDesc, 0, UnboundedList, Default([]float64{13.21, 34.55})),
					&BashCommand[string]{ArgName: "bc", Contents: []string{
						"echo 1: %s && echo %s",
						"echo 2: %s %s",
						"echo list 1: %s",
						"echo list 2: %s",
						"echo list 3: %s",
					},
						FormatArgs: []BashDataStringer[string]{
							NewBashDataStringer[string](Arg[string]("s", testDesc), ""),
							NewBashDataStringer[string](Arg[int]("i", testDesc), ""),
							NewBashDataStringer[string](Arg[float64]("f", testDesc), ""),
							NewBashDataStringer[string](Arg[bool]("b", testDesc), ""),
							NewBashDataStringer[string](ListArg("sl", testDesc, 0, UnboundedList, Default([]string{"alpha", "beta", "gamma", "delta"})), " , "),
							NewBashDataStringer[string](ListArg("il", testDesc, 0, UnboundedList, Default([]int{1, 1, 2, 3, 5, 8})), ":"),
							NewBashDataStringer[string](ListArg("fl", testDesc, 0, UnboundedList, Default([]float64{13.21, 34.55})), " "),
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
			name: "bash node formatting fails on error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					&BashCommand[string]{ArgName: "bc", Contents: []string{
						"echo 1: %s",
					},
						FormatArgs: []BashDataStringer[string]{
							CustomBashDataStringer[string](func(d *Data) (string, error) {
								return "bleh", fmt.Errorf("ouch")
							}),
						},
					},
				),
				WantErr:    fmt.Errorf("failed to get string for bash formatting: ouch"),
				WantStderr: "failed to get string for bash formatting: ouch\n",
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			ExecuteTest(t, test.etc)
		})
	}
}
