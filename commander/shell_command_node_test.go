package commander

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/spycommandtest"
)

func TestShellCommand(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *commandtest.ExecuteTestCase
		ietc *spycommandtest.ExecuteTestCase
	}{
		// Generic tests
		{
			name: "shell command returns an error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantErr:    fmt.Errorf("failed to execute shell command: oops"),
				WantStderr: "failed to execute shell command: oops\n",
				RunResponses: []*commandtest.FakeRun{
					{
						Err: fmt.Errorf("oops"),
					},
				},
			},
		},
		{
			name: "shell command prints stderr on error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantErr: fmt.Errorf("failed to execute shell command: oops"),
				WantStderr: strings.Join([]string{
					"un",
					"deux",
					"trois",
					"failed to execute shell command: oops\n",
				}, "\n"),
				RunResponses: []*commandtest.FakeRun{
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}, HideStderr: true}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantErr: fmt.Errorf("failed to execute shell command: oops"),
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"one", "two", "three"},
						Stderr: []string{"un", "deux", "trois"},
						Err:    fmt.Errorf("oops"),
					},
				},
			},
		},
		{
			name: "shell command node runs in other directory",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{
					ArgName:     "s",
					CommandName: "echo",
					Args:        []string{"hello"},
					Dir:         filepath.Join("some", "other", "dir"),
				}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
					Dir:  filepath.Join("some", "other", "dir"),
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha"},
					},
				},
			},
		},
		{
			name: "shell command forwards stdin",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{
					ArgName:     "s",
					CommandName: "echo",
					Args:        []string{"hello"},
					Stdin:       strings.NewReader("hello there.\nGeneral Kenobi"),
				}),
				WantRunContents: []*commandtest.RunContents{{
					Name:          "echo",
					Args:          []string{"hello"},
					StdinContents: "hello there.\nGeneral Kenobi",
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha"},
					},
				},
			},
		},
		// String
		{
			name: "shell node for string",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha"},
					},
				},
			},
		},
		{
			name: "shell node for string with EchoCommand true",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{
					ArgName:     "s",
					CommandName: "echo",
					Args:        []string{"hello", "there"},
					EchoCommand: true,
				}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello", "there"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				WantStdout: "echo hello there\n",
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha"},
					},
				},
			},
		},
		{
			name: "shell node for string when empty ArgName",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{
					CommandName: "echo",
					Args:        []string{"hello", "there"},
				}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello", "there"},
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha"},
					},
				},
				// Empty WantData
			},
		},
		{
			name: "shell node for string works with empty output",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "don't", Args: []string{"echo", "hello"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "don't",
					Args: []string{"echo", "hello"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "",
				}},
				RunResponses: []*commandtest.FakeRun{{}},
			},
		},
		{
			name: "successful command shows stderr",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				WantStderr: "ahola\n",
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha"},
						Stderr: []string{"ahola"},
					},
				},
			},
		},
		{
			name: "successful command hides stderr",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}, HideStderr: true}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "aloha",
				}},
				RunResponses: []*commandtest.FakeRun{
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[[]string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": []string{"aloha", "hello there", "howdy"},
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha", "hello there", "howdy"},
					},
				},
			},
		},
		{
			name: "shell node for string list forwards stdout",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[[]string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}, ForwardStdout: true}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantStdout: "aloha\nhello there\nhowdy\n",
				WantData: &command.Data{Values: map[string]interface{}{
					"s": []string{"aloha", "hello there", "howdy"},
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha", "hello there", "howdy"},
					},
				},
			},
		},
		{
			name: "shell node with output streamer",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					&ShellCommand[[]string]{
						ArgName:       "s",
						CommandName:   "echo",
						Args:          []string{"hello"},
						ForwardStdout: true,
						OutputStreamProcessor: func(o command.Output, d *command.Data, b []byte) error {
							o.Stdoutf("Streamer received: %s", string(b))
							return nil
						},
					},
				),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{
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
				WantData: &command.Data{Values: map[string]interface{}{
					"s": []string{"aloha", "hello there", "howdy"},
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha", "hello there", "howdy"},
					},
				},
			},
		},
		{
			name: "shell node for string list gets empty new lines at end",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[[]string]{ArgName: "s", CommandName: "echo", Args: []string{"hello"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"hello"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": []string{"aloha", "hello there", "howdy", "", ""},
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"aloha", "hello there", "howdy", "", ""},
					},
				},
			},
		},
		// Int
		{
			name: "shell node for int",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"1248"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"1248"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 1248,
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"1248"},
					},
				},
			},
		},
		{
			name: "shell node for int works with empty output",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "don't", Args: []string{"echo", "1248"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "don't",
					Args: []string{"echo", "1248"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 0,
				}},
				RunResponses: []*commandtest.FakeRun{{}},
			},
		},
		{
			name: "error when not an int",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"two"},
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"two"},
					},
				},
			},
		},
		// Int list
		{
			name: "shell node for int list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"primes"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"primes"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": []int{2, 3, 5, 7},
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"2", "3", "5", "7"},
					},
				},
			},
		},
		{
			name: "error when not an int in list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"two"},
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"2", "two", "200"},
					},
				},
			},
		},
		// Int
		{
			name: "shell node for int",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"1248"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"1248"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 1248,
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"1248"},
					},
				},
			},
		},
		{
			name: "error when not an int",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"two"},
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"two"},
					},
				},
			},
		},
		// Int list
		{
			name: "shell node for int list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"primes"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"primes"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": []int{2, 3, 5, 7},
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"2", "3", "5", "7"},
					},
				},
			},
		},
		{
			name: "error when not an int in list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[[]int]{ArgName: "i", CommandName: "echo", Args: []string{"two"}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"two"},
				}},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"two\": invalid syntax\n",
				RunResponses: []*commandtest.FakeRun{
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"1248"}, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"1248"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 1248,
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"1248"},
					},
				},
			},
		},
		{
			name: "shell node with failing validators",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"-1248"}, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"-1248"},
				}},
				WantStderr: "validation for \"i\" failed: [NonNegative] value isn't non-negative\n",
				WantErr:    fmt.Errorf("validation for \"i\" failed: [NonNegative] value isn't non-negative"),
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
			},
		},
		{
			name: "shell node with failing validators and hidden stderr",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(&ShellCommand[int]{ArgName: "i", CommandName: "echo", Args: []string{"-1248"}, HideStderr: true, Validators: []*ValidatorOption[int]{NonNegative[int]()}}),
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"-1248"},
				}},
				WantErr: fmt.Errorf("validation for \"i\" failed: [NonNegative] value isn't non-negative"),
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
			},
		},
		// StdinPipeFn tests
		{
			name: "Fails without running command if StdinPipeFn returns error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					&ShellCommand[int]{
						ArgName:     "i",
						CommandName: "echo",
						Args:        []string{"-1248"},
						StdinPipeFn: func(w io.WriteCloser, err error) error {
							return fmt.Errorf("oopsies")
						},
					},
				),
				WantErr:    fmt.Errorf("failed to run StdinPipeFn: oopsies"),
				WantStderr: "failed to run StdinPipeFn: oopsies\n",
			},
		},
		{
			name: "Writes and returns error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					&ShellCommand[int]{
						ArgName:     "i",
						CommandName: "echo",
						Args:        []string{"-1248"},
						StdinPipeFn: func(w io.WriteCloser, _ error) error {
							if _, err := w.Write(([]byte("some text"))); err != nil {
								t.Fatalf("failed to write: %v", err)
							}
							return fmt.Errorf("oops 2")
						},
					},
				),
				WantErr:    fmt.Errorf("failed to run StdinPipeFn: oops 2"),
				WantStderr: "failed to run StdinPipeFn: oops 2\n",
			},
		},
		{
			name: "Successfully writes",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					&ShellCommand[int]{
						ArgName:       "i",
						CommandName:   "echo",
						Args:          []string{"-1248"},
						ForwardStdout: true,
						StdinPipeFn: func(w io.WriteCloser, _ error) error {
							if _, err := w.Write(([]byte("some text"))); err != nil {
								t.Fatalf("failed to write: %v", err)
							}
							return nil
						},
					},
				),
				WantRunContents: []*commandtest.RunContents{{
					Name:          "echo",
					Args:          []string{"-1248"},
					StdinContents: "some text",
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": -1248,
				}},
				WantStdout: "-1248\n",
			},
		},
		// formatArgs tests
		/*{
			name: "shell node formats all input types",
			etc: &commandtest.ExecuteTestCase{
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
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "stringVal",
					"i":  9,
					"f":  -12.345678,
					"b":  true,
					"sl": []string{"alpha", "beta", "gamma", "delta"},
					"il": []int{1, 1, 2, 3, 5, 8},
					"fl": []float64{13.21, 34.55},
					"bc": "-1248",
				}},
				RunResponses: []*commandtest.FakeRun{
					{
						Stdout: []string{"-1248"},
					},
				},
			},
		},
		{
			name: "shell node formatting fails on error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					&ShellCommand[string]{ArgName: "bc", Args: []string{
						"echo 1: %s",
					},
						FormatArgs: []ShellCommandDataStringer[string]{
							CustomShellCommandDataStringer[string](func(d *command.Data) (string, error) {
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
			executeTest(t, test.etc, test.ietc)
		})
	}
}
