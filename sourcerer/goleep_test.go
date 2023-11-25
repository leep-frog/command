package sourcerer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestGoLeep(t *testing.T) {
	for _, test := range []struct {
		name        string
		etc         *command.ExecuteTestCase
		writeToFile []string
		getTmpErr   error
		osenv       map[string]string
		wantOSEnv   map[string]string
	}{
		{
			name: "requires cli arg",
			etc: &command.ExecuteTestCase{
				WantStderr: "Argument \"CLI\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf("Argument \"CLI\" requires at least 1 argument, got 0"),
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): "",
				}},
			},
		},
		{
			name: "requires go-dir arg",
			etc: &command.ExecuteTestCase{
				Args:       []string{"c", "--go-dir"},
				WantStderr: "Argument \"go-dir\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "go-dir" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "runs with no go file",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"c",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						"execute",
						"c",
						`TMP_FILE`,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "c",
					goDirectory.Name():  "",
				}},
			},
		},
		{
			name: "runs other go dir",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"dc",
					"-d",
					filepath.Join("..", "testdata"),
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Dir:  filepath.Join("..", "testdata"),
					Args: []string{
						`run`,
						".",
						"execute",
						"dc",
						`TMP_FILE`,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "dc",
					goDirectory.Name():  filepath.Join("..", "testdata"),
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
					"ec",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						"execute",
						"ec",
						`TMP_FILE`,
					},
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"echo hello",
						"echo goodbye",
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name():  "",
					goleepCLIArg.Name(): "ec",
				}},
			},
		},
		{
			name: "passes along stdout and stderr",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"sc",
				},
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
				WantStdout: "hello there\ngeneral Kenobi\n",
				WantStderr: "goodbye then\ngeneral Grevious\n",
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						"execute",
						"sc",
						`TMP_FILE`,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "sc",
					goDirectory.Name():  "",
				}},
			},
		},
		{
			name: "handles shell command error",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"bc",
				},
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
				WantStdout: "hello there\ngeneral Kenobi\n",
				WantStderr: strings.Join([]string{
					"goodbye then",
					"general Grevious",
					"failed to run shell script: failed to execute shell command: bad news bears",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("failed to run shell script: failed to execute shell command: bad news bears"),
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						"execute",
						"bc",
						`TMP_FILE`,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "bc",
					goDirectory.Name():  "",
				}},
			},
		},
		{
			name: "passes extra args to command",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"c",
					"arg1",
					"arg2",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						"execute",
						"c",
						`TMP_FILE`,
						"arg1",
						"arg2",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "c",
					passAlongArgs.Name(): []string{
						"arg1",
						"arg2",
					},
					goDirectory.Name(): "",
				}},
			},
		},
		{
			name:      "handles getTmpFile error",
			getTmpErr: fmt.Errorf("whoops"),
			etc: &command.ExecuteTestCase{
				Args: []string{
					"c",
					"arg1",
					"arg2",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "c",
					passAlongArgs.Name(): []string{
						"arg1",
						"arg2",
					},
					goDirectory.Name(): "",
				}},
				WantStderr: "failed to create tmp file: whoops\n",
				WantErr:    fmt.Errorf("failed to create tmp file: whoops"),
			},
		},
		// Usage
		{
			name: "runs usage",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"c",
					"usage",
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						"usage",
						`"c"`,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "c",
					goDirectory.Name():  "",
				}},
			},
		},
		{
			name: "runs fails",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"c",
					"usage",
				},
				RunResponses: []*command.FakeRun{{
					Err: fmt.Errorf("oops"),
				}},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						"usage",
						`"c"`,
					},
				}},
				WantStderr: "failed to run goleep usage command: failed to execute shell command: oops\n",
				WantErr:    fmt.Errorf("failed to run goleep usage command: failed to execute shell command: oops"),
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "c",
					goDirectory.Name():  "",
				}},
			},
		},
		{
			name: "runs usage with go dir",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"c",
					"usage",
					"--go-dir",
					filepath.Join("..", "color"),
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Dir:  filepath.Join("..", "color"),
					Args: []string{
						`run`,
						`.`,
						"usage",
						`"c"`,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "c",
					goDirectory.Name():  filepath.Join("..", "color"),
				}},
			},
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			command.StubEnv(t, nil)

			// Stub files
			f, err := ioutil.TempFile("", "goleep-test")
			if err != nil {
				t.Fatalf("failed to create tmp file: %v", err)
			}
			command.StubValue(t, &getTmpFile, func() (*os.File, error) {
				return f, test.getTmpErr
			})

			if test.writeToFile != nil {
				if err := ioutil.WriteFile(f.Name(), []byte(strings.Join(test.writeToFile, "\n")), 0644); err != nil {
					t.Fatalf("failed to write to file: %v", err)
				}
			}

			for _, sets := range test.etc.WantRunContents {
				for i, a := range sets.Args {
					if a == "TMP_FILE" {
						sets.Args[i] = filepath.ToSlash(f.Name())
					}
				}
			}

			cli := &GoLeep{}
			test.etc.Node = cli.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, nil, cli)
		})
	}
}

func TestGoLeepAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name string
		ctc  *command.CompleteTestCase
	}{
		{
			name: "completes directories",
			ctc: &command.CompleteTestCase{
				Args: fmt.Sprintf("cmd c -d %s", filepath.Join("..", "c")),
				Want: &command.Autocompletion{
					Suggestions: []string{
						filepath.FromSlash("cache/"),
						filepath.FromSlash("color/"),
						filepath.FromSlash("commander/"),
						" ",
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): filepath.Join("..", "c"),
				}},
			},
		},
		{
			name: "completes a cli",
			ctc: &command.CompleteTestCase{
				Args: "cmd ",
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{"cliOne", "cliTwo", "cliThree"},
					},
				},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						`listCLIs`,
					},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"cliOne",
						"cliThree",
						"cliTwo",
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goleepCLIArg.Name(): "",
					goDirectory.Name():  "",
				}},
			},
		},
		{
			name: "completes empty args",
			ctc: &command.CompleteTestCase{
				Args: "cmd acli ",
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{"un", "deux", "trois"},
					},
				},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						`autocomplete`,
						`acli`,
						`63`,
						`13`,
						`dummyCommand `,
					},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"deux",
						"trois",
						"un",
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name():   "",
					goleepCLIArg.Name():  "acli",
					passAlongArgs.Name(): []string{""},
				}},
			},
		},
		{
			name: "completes present args with quotes",
			ctc: &command.CompleteTestCase{
				Args: "cmd aCLI abc d\"e'f",
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{"un", "deux", "trois", "de'finitely"},
					},
				},
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						`autocomplete`,
						`aCLI`,
						`63`,
						`21`,
						`dummyCommand abc de'f`,
					},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"de'finitely",
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name():   "",
					goleepCLIArg.Name():  "aCLI",
					passAlongArgs.Name(): []string{"abc", `de'f`},
				}},
			},
		},
		{
			name: "handles run response error",
			ctc: &command.CompleteTestCase{
				Args: "cmd someCLI ",
				RunResponses: []*command.FakeRun{
					{
						Err:    fmt.Errorf("whoops"),
						Stdout: []string{"un", "deux", "trois"},
					},
				},
				WantErr: fmt.Errorf(`failed to run goleep completion: failed to execute shell command: whoops`),
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						`autocomplete`,
						`someCLI`,
						`63`,
						`13`,
						`dummyCommand `,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name():   "",
					goleepCLIArg.Name():  "someCLI",
					passAlongArgs.Name(): []string{""},
				}},
			},
		},
		{
			name: "handles run response error with stderr",
			ctc: &command.CompleteTestCase{
				Args: "cmd someCLI ",
				RunResponses: []*command.FakeRun{
					{
						Err:    fmt.Errorf("whoops"),
						Stderr: []string{"argh", "matey"},
						Stdout: []string{"un", "deux", "trois"},
					},
				},
				WantErr: fmt.Errorf(strings.Join([]string{
					`failed to run goleep completion: failed to execute shell command: whoops`,
					``,
					`Stderr:`,
					`argh`,
					`matey`,
					``,
				}, "\n")),
				WantRunContents: []*command.RunContents{{
					Name: `go`,
					Args: []string{
						`run`,
						`.`,
						`autocomplete`,
						`someCLI`,
						`63`,
						`13`,
						`dummyCommand `,
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name():   "",
					goleepCLIArg.Name():  "someCLI",
					passAlongArgs.Name(): []string{""},
				}},
			},
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			cli := &GoLeep{}
			test.ctc.Node = cli.Node()
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestGoLeepMetadata(t *testing.T) {
	cli := &GoLeep{}
	if diff := cmp.Diff("goleep", cli.Name()); diff != "" {
		t.Errorf("Unexpected cli name (-want, +got):\n%s", diff)
	}

	if setup := cli.Setup(); setup != nil {
		t.Errorf("Expected cli.Setup() to be nil; got %v", setup)
	}
}

func TestUsage(t *testing.T) {
	command.UsageTest(t, &command.UsageTestCase{
		Node: (&GoLeep{}).Node(),
		WantString: []string{
			"Execute the provided go files",
			"CLI ┳ [ PASSTHROUGH_ARGS ... ] --go-dir|-d",
			"┏━━━┛",
			"┃   Get the usage of the provided go files",
			"┗━━ usage",
			"",
			"Arguments:",
			"  CLI: CLI to use",
			"  PASSTHROUGH_ARGS: Args to pass through to the command",
			"",
			"Flags:",
			"  [d] go-dir: Directory of package to run",
			"",
			"Symbols:",
			command.BranchDescWithDefault,
		},
	})
}
