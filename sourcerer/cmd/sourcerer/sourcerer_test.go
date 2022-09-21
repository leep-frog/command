package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name        string
		cli         sourcerer.CLI
		etc         *command.ExecuteTestCase
		writeToFile []string
		want        sourcerer.CLI
		osenv       map[string]string
		wantOSEnv   map[string]string
	}{
		// goleep tests
		{
			name: "requires go-dir arg",
			cli:  &GoLeep{},
			etc: &command.ExecuteTestCase{
				Args:       []string{"--go-dir"},
				WantStderr: "Argument \"go-dir\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "go-dir" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "runs with no go file",
			cli:  &GoLeep{},
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go run . execute TMP_FILE",
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): ".",
				}},
			},
		},
		{
			name: "runs other go dir",
			cli:  &GoLeep{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"-d",
					filepath.Join("..", "..", "..", "testdata"),
				},
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					fmt.Sprintf(`go run %s execute TMP_FILE`, filepath.Join("..", "..", "..", "testdata")),
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): filepath.Join("..", "..", "..", "testdata"),
				}},
			},
		},
		{
			name: "sets execute data to file contents",
			cli:  &GoLeep{},
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
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): ".",
				}},
			},
		},
		{
			name: "passes along stdout and stderr",
			cli:  &GoLeep{},
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
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): ".",
				}},
			},
		},
		{
			name: "handles bash command error",
			cli:  &GoLeep{},
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
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): ".",
				}},
			},
		},
		{
			name: "passes extra args to command",
			cli:  &GoLeep{},
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
				WantData: &command.Data{Values: map[string]interface{}{
					passAlongArgs.Name(): []string{
						"arg1",
						"arg2",
					},
					goDirectory.Name(): ".",
				}},
			},
		},
		// Usage
		{
			name: "runs usage",
			cli:  &GoLeep{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"usage",
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"go run . usage",
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): ".",
				}},
			},
		},
		{
			name: "runs usage with go dir",
			cli:  &GoLeep{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"usage",
					"--go-dir",
					filepath.Join("..", "..", "..", "color"),
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("go run %s usage", filepath.Join("..", "..", "..", "color")),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): filepath.Join("..", "..", "..", "color"),
				}},
			},
		},
		// gg tests
		{
			name: "Gets a package",
			cli:  &UpdateLeepPackageCommand{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"some-package",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					packageArg.Name(): []string{
						"some-package",
					},
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`local commitSha="$(git ls-remote git@github.com:leep-frog/some-package.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/some-package@$commitSha"`,
					},
				},
			},
		},
		{
			name: "Gets multiple packages",
			cli:  &UpdateLeepPackageCommand{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"ups",
					"fedex",
					"usps",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					packageArg.Name(): []string{
						"ups",
						"fedex",
						"usps",
					},
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`local commitSha="$(git ls-remote git@github.com:leep-frog/ups.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/ups@$commitSha"`,
						`local commitSha="$(git ls-remote git@github.com:leep-frog/fedex.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/fedex@$commitSha"`,
						`local commitSha="$(git ls-remote git@github.com:leep-frog/usps.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/usps@$commitSha"`,
					},
				},
			},
		},
		// mancli
		{
			name: "Gets usage",
			cli:  &UsageCommand{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"someCLI",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					usageCLIArg.Name(): "someCLI",
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						sourcerer.FileStringFromCLI("someCLI"),
						`if [ -z "$file" ]; then`,
						`  echo someCLI is not a CLI generated via github.com/leep-frog/command`,
						`  return 1`,
						`fi`,
						`  "$GOPATH/bin/_${file}_runner" usage someCLI`,
					},
				},
			},
		},
		// aliaser
		{
			name: "Creates aliaser with no passthrough args",
			cli:  &AliaserCommand{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"some-alias",
					"someCLI",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					aliasArg.Name():    "some-alias",
					aliasCLIArg.Name(): "someCLI",
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`function _leep_frog_autocompleter {`,
						`  local file="$(type "$1" | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`  local tFile=$(mktemp)`,
						`  $GOPATH/bin/_${file}_runner autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`,
						`  local IFS='`,
						`';`,
						`  COMPREPLY=( $(cat $tFile) )`,
						`  rm $tFile`,
						`}`,
						``,
						`local file="$(type someCLI | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`if [ -z "$file" ]; then`,
						`  echo Provided CLI "someCLI" is not a CLI generated with github.com/leep-frog/command`,
						`  return 1`,
						`fi`,
						``,
						`alias -- some-alias="someCLI "`,
						`function _custom_autocomplete_for_alias_some-alias {`,
						`  _leep_frog_autocompleter "someCLI" `,
						`}`,
						``,
						`complete -F _custom_autocomplete_for_alias_some-alias -o nosort some-alias`,
						``,
						``,
					},
				},
			},
		},
		{
			name: "Creates aliaser with passthrough args",
			cli:  &AliaserCommand{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					"some-alias",
					"someCLI",
					"un",
					"2",
					"trois",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					aliasArg.Name():    "some-alias",
					aliasCLIArg.Name(): "someCLI",
					aliasPTArg.Name(): []string{
						"un",
						"2",
						"trois",
					},
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`function _leep_frog_autocompleter {`,
						`  local file="$(type "$1" | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`  local tFile=$(mktemp)`,
						`  $GOPATH/bin/_${file}_runner autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`,
						`  local IFS='`,
						`';`,
						`  COMPREPLY=( $(cat $tFile) )`,
						`  rm $tFile`,
						`}`,
						``,
						`local file="$(type someCLI | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`if [ -z "$file" ]; then`,
						`  echo Provided CLI "someCLI" is not a CLI generated with github.com/leep-frog/command`,
						`  return 1`,
						`fi`,
						``,
						`alias -- some-alias="someCLI \"un\" \"2\" \"trois\""`,
						`function _custom_autocomplete_for_alias_some-alias {`,
						`  _leep_frog_autocompleter "someCLI" "un" "2" "trois"`,
						`}`,
						``,
						`complete -F _custom_autocomplete_for_alias_some-alias -o nosort some-alias`,
						``,
						``,
					},
				},
			},
		},
		// sourcerer
		{
			name: "Sources directory",
			cli:  &SourcererCommand{},
			etc: &command.ExecuteTestCase{
				Args: []string{
					filepath.Join("..", "..", "..", "testdata"),
					"ING",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					sourcererDirArg.Name():    command.FilepathAbs(t, "..", "..", "..", "testdata"),
					sourcererSuffixArg.Name(): "ING",
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"pushd . > /dev/null",
						fmt.Sprintf(`cd %q`, command.FilepathAbs(t, "..", "..", "..", "testdata")),
						`local tmpFile="$(mktemp)"`,
						`go run . "ING" > $tmpFile && source $tmpFile `,
						"popd > /dev/null",
					},
				},
			},
		},
		// Debugger tests
		{
			name: "Activates debug mode",
			cli:  &Debugger{},
			etc: &command.ExecuteTestCase{
				WantStdout: "Entering debug mode.\n",
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("export %q=%q", command.DebugEnvVar, "1"),
					},
				},
			},
		},
		{
			name: "Deactivates debug mode",
			cli:  &Debugger{},
			etc: &command.ExecuteTestCase{
				WantStdout: "Exiting debug mode.\n",
				Env: map[string]string{
					command.DebugEnvVar: "1",
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("unset %q", command.DebugEnvVar),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.DebugEnvVar: "1",
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
				return f, nil
			})

			if test.writeToFile != nil {
				if err := ioutil.WriteFile(f.Name(), []byte(strings.Join(test.writeToFile, "\n")), command.CmdOS.DefaultFilePerm()); err != nil {
					t.Fatalf("failed to write to file: %v", err)
				}
			}

			for _, sets := range test.etc.WantRunContents {
				for i, line := range sets {
					sets[i] = strings.ReplaceAll(line, "TMP_FILE", filepath.ToSlash(f.Name()))
				}
			}

			test.etc.Node = test.cli.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, test.want, test.cli)
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
				Args: fmt.Sprintf("cmd -d %s", filepath.Join("..", "..", "..", "c")),
				Want: []string{
					"cache/",
					"cmd/",
					"color/",
					" ",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name(): filepath.Join("..", "..", "..", "c"),
				}},
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
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name():   ".",
					passAlongArgs.Name(): []string{""},
				}},
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
				WantData: &command.Data{Values: map[string]interface{}{
					goDirectory.Name():   ".",
					passAlongArgs.Name(): []string{""},
				}},
			},
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			gl := &GoLeep{}
			test.ctc.Node = gl.Node()
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

func TestMetadata(t *testing.T) {
	for _, test := range []struct {
		cli sourcerer.CLI
	}{
		{
			cli: &SourcererCommand{},
		},
		{
			cli: &UpdateLeepPackageCommand{},
		},
		{
			cli: &UsageCommand{},
		},
		{
			cli: &AliaserCommand{},
		},
		{
			cli: &GoLeep{},
		},
	} {
		t.Run(test.cli.Name(), func(t *testing.T) {
			if test.cli.Setup() != nil {
				t.Errorf("CLI %s returned unexpected setup: %v", test.cli.Name(), test.cli.Setup())
			}
		})
	}
}
