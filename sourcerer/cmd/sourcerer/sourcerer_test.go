package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leep-frog/command"
)

func TestNodeRunner(t *testing.T) {
	for _, test := range []struct {
		name        string
		etc         *command.ExecuteTestCase
		writeToFile []string
	}{
		{
			name: "requires go-dir arg",
			etc: &command.ExecuteTestCase{
				Args:       []string{"--go-dir"},
				WantStderr: "Argument \"go-dir\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "go-dir" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "runs with no go file",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go run . execute TMP_FILE",
				}},
			},
		},
		{
			name: "runs other go dir",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"-d",
					"../../../testdata",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run ../../../testdata execute TMP_FILE`,
				}},
			},
		},
		{
			name: "sets execute data to file contents",
			writeToFile: []string{
				"echo hello",
				"echo goodbye",
			},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run . execute TMP_FILE`,
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"echo hello",
						"echo goodbye",
					},
				},
			},
		},
		{
			name: "passes along stdout and stderr",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						"hello there",
						"general Kenobi",
					},
					Stderr: []string{
						"goodbye then",
						"general Grevious",
					},
				}},
				WantStdout: "hello there\ngeneral Kenobi",
				WantStderr: "goodbye then\ngeneral Grevious",
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run . execute TMP_FILE`,
				}},
			},
		},
		{
			name: "handles bash command error",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Err: fmt.Errorf("bad news bears"),
					Stdout: []string{
						"hello there",
						"general Kenobi",
					},
					Stderr: []string{
						"goodbye then",
						"general Grevious",
					},
				}},
				WantStdout: "hello there\ngeneral Kenobi",
				WantStderr: strings.Join([]string{
					"goodbye then\ngeneral Grevious",
					"failed to run bash script: failed to execute bash command: bad news bears\n",
				}, ""),
				WantErr: fmt.Errorf("failed to run bash script: failed to execute bash command: bad news bears"),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run . execute TMP_FILE`,
				}},
			},
		},
		{
			name: "passes extra args to command",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"arg1",
					"arg2",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run . execute TMP_FILE arg1 arg2`,
				}},
			},
		},
		{
			name: "runs usage",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"usage",
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"go run . usage",
					},
				},
			},
		},
		{
			name: "runs usage with go dir",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"usage",
					"--go-dir",
					"../../../color",
				},

				WantExecuteData: &command.ExecuteData{
					Executable: []string{"go run ../../../color usage"},
				},
			},
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			f, err := ioutil.TempFile("", "goleep-test")
			if err != nil {
				t.Fatalf("failed to create tmp file: %v", err)
			}
			command.StubValue(t, &getTmpFile, func() (*os.File, error) {
				return f, nil
			})

			if test.writeToFile != nil {
				if err := ioutil.WriteFile(f.Name(), []byte(strings.Join(test.writeToFile, "\n")), 0644); err != nil {
					t.Fatalf("failed to write to file: %v", err)
				}
			}

			for _, sets := range test.etc.WantRunContents {
				for i, line := range sets {
					sets[i] = strings.ReplaceAll(line, "TMP_FILE", filepath.ToSlash(f.Name()))
				}
			}

			gl := &GoLeep{}
			test.etc.Node = gl.Node()
			test.etc.SkipDataCheck = true
			command.ExecuteTest(t, test.etc)
		})
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name string
		ctc  *command.CompleteTestCase
	}{
		{
			name: "completes directories",
			ctc: &command.CompleteTestCase{
				Args: "cmd -d ../../../c",
				Want: []string{
					"cache/",
					"cmd/",
					"color/",
					" ",
				},
			},
		},
		{
			name: "completes args",
			ctc: &command.CompleteTestCase{
				Args: "cmd ",
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{"un", "deux", "trois"},
					},
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run . autocomplete ""`,
				}},
				Want: []string{
					"deux",
					"trois",
					"un",
				},
			},
		},
		{
			name: "handles run response error",
			ctc: &command.CompleteTestCase{
				Args: "cmd ",
				RunResponses: []*command.FakeRun{
					{
						Err:    fmt.Errorf("whoops"),
						Stdout: []string{"un", "deux", "trois"},
					},
				},
				WantErr: fmt.Errorf(`failed to execute bash command: whoops`),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run . autocomplete ""`,
				}},
			},
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			gl := &GoLeep{}
			test.ctc.Node = gl.Node()
			test.ctc.SkipDataCheck = true
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestUsage(t *testing.T) {
	command.UsageTest(t, &command.UsageTestCase{
		Node: (&GoLeep{}).Node(),
		WantString: []string{
			"Execute the provided go files",
			"< [ PASSTHROUGH_ARGS ... ] --go-dir|-d",
			"",
			"  Get the usage of the provided go files",
			"  usage --go-dir|-d",
			"",
			"Arguments:",
			"  PASSTHROUGH_ARGS: Args to pass through to the command",
			"",
			"Flags:",
			"  [d] go-dir: Directory of package to run",
			"",
			"Symbols:",
			command.BranchDesc,
		},
	})
}
