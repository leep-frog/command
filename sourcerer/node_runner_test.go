package sourcerer

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
			name: "requires at least 1 go file",
			etc: &command.ExecuteTestCase{
				Args:       []string{"--GO_FILES"},
				WantStderr: []string{`Argument "GO_FILES" requires at least 1 argument, got 0`},
				WantErr:    fmt.Errorf(`Argument "GO_FILES" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "runs with no go file",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`goFiles="$(ls *.go | grep -v _test.go$)"`,
					"go run $goFiles execute TMP_FILE",
				}},
			},
		},
		{
			name: "runs single go file",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"-f",
					"main.go",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run main.go execute TMP_FILE`,
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
				Args: []string{
					"--GO_FILES",
					"main.go",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run main.go execute TMP_FILE`,
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
				WantStdout: []string{
					"hello there",
					"general Kenobi",
				},
				WantStderr: []string{
					"goodbye then\ngeneral Grevious",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					goFileGetter,
					`go run $goFiles execute TMP_FILE`,
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
					goFileGetter,
					`go run $goFiles execute TMP_FILE arg1 arg2`,
				}},
			},
		},
		{
			name: "handles extra go files",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"-f",
					"main.go",
					"other.go",
					"arg1",
					"arg2",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					`go run main.go other.go execute TMP_FILE arg1 arg2`,
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
						goFileGetter,
						"go run $goFiles usage",
					},
				},
			},
		},
		{
			name: "runs usage with go files",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"usage",
					"--GO_FILES",
					"main.go",
					"other.go",
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"go run main.go other.go usage"},
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
			oldGet := getTmpFile
			getTmpFile = func() (*os.File, error) {
				return f, nil
			}

			if test.writeToFile != nil {
				if err := ioutil.WriteFile(f.Name(), []byte(strings.Join(test.writeToFile, "\n")), 0644); err != nil {
					t.Fatalf("failed to write to file: %v", err)
				}
			}

			defer func() { getTmpFile = oldGet }()
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
			name: "completes go files",
			ctc: &command.CompleteTestCase{
				Args: "cmd --GO_FILES ",
				Want: []string{
					"node_runner.go",
					"node_runner_test.go",
					"sourcerer.go",
					"sourcerer_test.go",
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
					goFileGetter,
					`go run $goFiles autocomplete ""`,
				}},
				Want: []string{
					"deux",
					"trois",
					"un",
				},
			},
		},
		/* TODO: completion with list breaker at break
		{
			name: "completes distinct go files",
			ctc: &command.CompleteTestCase{
				Args: "cmd node_runner.go ",
				Want: []string{
					"node_runner_test.go",
				},
				WantData: &command.Data{
					Values: map[string]*command.Value{
						"GO_FILES": command.StringListValue("node_runner.go", ""),
					},
				},
			},
		},*/
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
			"< [ PASSTHROUGH_ARGS ... ] --GO_FILES|-f",
			"",
			"  Get the usage of the provided go files",
			"  usage --GO_FILES|-f",
			"",
			"Arguments:",
			"  PASSTHROUGH_ARGS: Args to pass through to the command",
			"",
			"Flags:",
			"  [f] GO_FILES: Go files to run",
			"",
			"Symbols:",
			command.BranchDesc,
		},
	})
}
