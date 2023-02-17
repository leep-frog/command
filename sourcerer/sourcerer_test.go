package sourcerer

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
)

const (
	fakeFile          = "FAKE_FILE"
	usagePrefixString = "\n======= Command Usage ======="
)

func TestGenerateBinaryNode(t *testing.T) {
	command.StubValue(t, &getSourceLoc, func() (string, error) {
		return "/fake/source/location", nil
	})

	for _, test := range []struct {
		name            string
		clis            []CLI
		args            []string
		ignoreNosort    bool
		opts            []Option
		getSourceLocErr error
		wantStdout      []string
		wantStderr      []string
		wantExecuteFile []string
	}{
		{
			name: "generates source file when no CLIs",
			wantExecuteFile: []string{
				`function _custom_execute_leep-frog-source {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_leep-frog-source_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_leep-frog-source "$@"`,
				``,
			},
			wantStdout: []string{
				`pushd . > /dev/null`,
				`cd "$(dirname /fake/source/location)"`,
				`go build -o $GOPATH/bin/_leep-frog-source_runner`,
				`popd > /dev/null`,
				`function _custom_autocomplete_leep-frog-source {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
			},
		},
		{
			name: "adds multiple Aliaser (singular) options at the end",
			opts: []Option{
				NewAliaser("a1", "do", "some", "stuff"),
				NewAliaser("otherAlias", "flaggable", "--args", "--at", "once"),
			},
			wantExecuteFile: []string{
				`function _custom_execute_leep-frog-source {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_leep-frog-source_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_leep-frog-source "$@"`,
				``,
			},
			wantStdout: []string{
				`pushd . > /dev/null`,
				`cd "$(dirname /fake/source/location)"`,
				`go build -o $GOPATH/bin/_leep-frog-source_runner`,
				`popd > /dev/null`,
				`function _custom_autocomplete_leep-frog-source {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
				GlobalAutocompleteForAliasFunction,
				`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				``,
				`alias -- a1="do \"some\" \"stuff\""`,
				`function _custom_autocomplete_for_alias_a1 {`,
				`  _leep_frog_autocompleter "do" "some" "stuff"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_a1 -o nosort a1`,
				``,
				`local file="$(type flaggable | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI "flaggable" is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				``,
				`alias -- otherAlias="flaggable \"--args\" \"--at\" \"once\""`,
				`function _custom_autocomplete_for_alias_otherAlias {`,
				`  _leep_frog_autocompleter "flaggable" "--args" "--at" "once"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias`,
				``,
			},
		},
		{
			name: "load only flag doesn't generate binaries",
			args: []string{"-l"},
			opts: []Option{
				NewAliaser("a1", "do", "some", "stuff"),
				NewAliaser("otherAlias", "flaggable", "--args", "--at", "once"),
			},
			wantExecuteFile: []string{
				`function _custom_execute_leep-frog-source {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_leep-frog-source_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_leep-frog-source "$@"`,
				``,
			},
			wantStdout: []string{
				// `pushd . > /dev/null`,
				// `cd "$(dirname /fake/source/location)"`,
				// `go build -o $GOPATH/bin/_leep-frog-source_runner`,
				// `popd > /dev/null`,
				`function _custom_autocomplete_leep-frog-source {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
				GlobalAutocompleteForAliasFunction,
				`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				``,
				`alias -- a1="do \"some\" \"stuff\""`,
				`function _custom_autocomplete_for_alias_a1 {`,
				`  _leep_frog_autocompleter "do" "some" "stuff"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_a1 -o nosort a1`,
				``,
				`local file="$(type flaggable | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI "flaggable" is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				``,
				`alias -- otherAlias="flaggable \"--args\" \"--at\" \"once\""`,
				`function _custom_autocomplete_for_alias_otherAlias {`,
				`  _leep_frog_autocompleter "flaggable" "--args" "--at" "once"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias`,
				``,
			},
		},
		{
			name: "only verifies each CLI once",
			opts: []Option{
				// Note the CLI in both of these is "do"
				NewAliaser("a1", "do", "some", "stuff"),
				NewAliaser("otherAlias", "do", "other", "stuff"),
			},
			wantExecuteFile: []string{
				`function _custom_execute_leep-frog-source {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_leep-frog-source_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_leep-frog-source "$@"`,
				``,
			},
			wantStdout: []string{
				`pushd . > /dev/null`,
				`cd "$(dirname /fake/source/location)"`,
				`go build -o $GOPATH/bin/_leep-frog-source_runner`,
				`popd > /dev/null`,
				`function _custom_autocomplete_leep-frog-source {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
				GlobalAutocompleteForAliasFunction,
				`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				``,
				`alias -- a1="do \"some\" \"stuff\""`,
				`function _custom_autocomplete_for_alias_a1 {`,
				`  _leep_frog_autocompleter "do" "some" "stuff"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_a1 -o nosort a1`,
				``,
				// Note that we don't verify the `do` cli again here.
				// Instead, we just go straight into aliasing commands.
				`alias -- otherAlias="do \"other\" \"stuff\""`,
				`function _custom_autocomplete_for_alias_otherAlias {`,
				`  _leep_frog_autocompleter "do" "other" "stuff"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias`,
				``,
			},
		},
		{
			name: "adds Aliasers (plural) at the end",
			opts: []Option{
				Aliasers(map[string][]string{
					"a1":         {"do", "some", "stuff"},
					"otherAlias": {"flaggable", "--args", "--at", "once"},
				}),
			},
			wantExecuteFile: []string{
				`function _custom_execute_leep-frog-source {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_leep-frog-source_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_leep-frog-source "$@"`,
				``,
			},
			wantStdout: []string{
				`pushd . > /dev/null`,
				`cd "$(dirname /fake/source/location)"`,
				`go build -o $GOPATH/bin/_leep-frog-source_runner`,
				`popd > /dev/null`,
				`function _custom_autocomplete_leep-frog-source {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
				GlobalAutocompleteForAliasFunction,
				`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				``,
				`alias -- a1="do \"some\" \"stuff\""`,
				`function _custom_autocomplete_for_alias_a1 {`,
				`  _leep_frog_autocompleter "do" "some" "stuff"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_a1 -o nosort a1`,
				``,
				`local file="$(type flaggable | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI "flaggable" is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				``,
				`alias -- otherAlias="flaggable \"--args\" \"--at\" \"once\""`,
				`function _custom_autocomplete_for_alias_otherAlias {`,
				`  _leep_frog_autocompleter "flaggable" "--args" "--at" "once"`,
				`}`,
				``,
				`complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias`,
				``,
			},
		},
		{
			name: "generates source file with custom filename",
			args: []string{"custom-output_file"},
			wantExecuteFile: []string{
				`function _custom_execute_custom-output_file {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_custom-output_file_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_custom-output_file "$@"`,
				``,
			},
			wantStdout: []string{
				`pushd . > /dev/null`,
				`cd "$(dirname /fake/source/location)"`,
				`go build -o $GOPATH/bin/_custom-output_file_runner`,
				`popd > /dev/null`,
				`function _custom_autocomplete_custom-output_file {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_custom-output_file_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
			},
		},
		{
			name: "generates source file with CLIs",
			clis: append(SimpleCommands(map[string]string{
				"x": "exit",
				"l": "ls -la",
			}), &testCLI{name: "basic", setup: []string{"his", "story"}}),
			wantExecuteFile: []string{
				`function _custom_execute_leep-frog-source {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_leep-frog-source_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_leep-frog-source "$@"`,
				``,
			},
			wantStdout: []string{
				`pushd . > /dev/null`,
				`cd "$(dirname /fake/source/location)"`,
				`go build -o $GOPATH/bin/_leep-frog-source_runner`,
				`popd > /dev/null`,
				`function _custom_autocomplete_leep-frog-source {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
				`function _setup_for_basic_cli {`,
				`  his  `,
				`  story`,
				`}`,
				`alias basic='o=$(mktemp) && _setup_for_basic_cli > $o && source $GOPATH/bin/_custom_execute_leep-frog-source basic $o'`,
				"complete -F _custom_autocomplete_leep-frog-source -o nosort basic",
				`alias l='source $GOPATH/bin/_custom_execute_leep-frog-source l'`,
				"complete -F _custom_autocomplete_leep-frog-source -o nosort l",
				"alias x='source $GOPATH/bin/_custom_execute_leep-frog-source x'",
				"complete -F _custom_autocomplete_leep-frog-source -o nosort x",
			},
		},
		{
			name: "generates source file with CLIs ignoring nosort",
			clis: append(SimpleCommands(map[string]string{
				"x": "exit",
				"l": "ls -la",
			}), &testCLI{name: "basic", setup: []string{"his", "story"}}),
			ignoreNosort: true,
			wantExecuteFile: []string{
				`function _custom_execute_leep-frog-source {`,
				`  # tmpFile is the file to which we write ExecuteData.Executable`,
				`  local tmpFile=$(mktemp)`,
				``,
				`  # Run the go-only code`,
				`  $GOPATH/bin/_leep-frog-source_runner execute $tmpFile "$@"`,
				`  # Return the error code if go code terminated with an error`,
				`  local errorCode=$?`,
				`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
				``,
				`  # Otherwise, run the ExecuteData.Executable data`,
				`  source $tmpFile`,
				`  local errorCode=$?`,
				`  if [ -z "$LEEP_FROG_DEBUG" ]; then`,
				`    rm $tmpFile`,
				`  else`,
				`    echo $tmpFile`,
				`  fi`,
				`  return $errorCode`,
				`}`,
				`_custom_execute_leep-frog-source "$@"`,
				``,
			},
			wantStdout: []string{
				`pushd . > /dev/null`,
				`cd "$(dirname /fake/source/location)"`,
				`go build -o $GOPATH/bin/_leep-frog-source_runner`,
				`popd > /dev/null`,
				`function _custom_autocomplete_leep-frog-source {`,
				`  local tFile=$(mktemp)`,
				`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
				`  local IFS=$'\n'`,
				`  COMPREPLY=( $(cat $tFile) )`,
				`  rm $tFile`,
				`}`,
				`function _setup_for_basic_cli {`,
				`  his  `,
				`  story`,
				`}`,
				`alias basic='o=$(mktemp) && _setup_for_basic_cli > $o && source $GOPATH/bin/_custom_execute_leep-frog-source basic $o'`,
				"complete -F _custom_autocomplete_leep-frog-source  basic",
				`alias l='source $GOPATH/bin/_custom_execute_leep-frog-source l'`,
				"complete -F _custom_autocomplete_leep-frog-source  l",
				"alias x='source $GOPATH/bin/_custom_execute_leep-frog-source x'",
				"complete -F _custom_autocomplete_leep-frog-source  x",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := command.TempFile(t, "leep-frog-sourcerer-test")
			command.StubValue(t, &getExecuteFile, func(string) string {
				return tmp.Name()
			})
			if test.ignoreNosort {
				command.StubValue(t, &NosortString, func() string { return "" })
			}
			o := command.NewFakeOutput()
			source(test.clis, test.args, o, test.opts...)
			o.Close()

			if o.GetStderrByCalls() != nil {
				t.Errorf("source(%v) produced stderr when none was expected:\n%v", test.args, o.GetStderrByCalls())
			}

			// append to add a final newline (which should *always* be present).
			if diff := cmp.Diff(strings.Join(append(test.wantStdout, ""), "\n"), o.GetStdout()); diff != "" {
				t.Errorf("source(%v) sent incorrect data to stdout (-wamt, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(strings.Join(test.wantStderr, "\n"), o.GetStderr()); diff != "" {
				t.Errorf("source(%v) sent incorrect data to stderr (-wamt, +got):\n%s", test.args, diff)
			}

			cmpFile(t, fmt.Sprintf("source(%v) created incorrect execute file", test.args), tmp.Name(), test.wantExecuteFile)
		})
	}
}

func cmpFile(t *testing.T, prefix, filename string, want []string) {
	t.Helper()
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if want == nil {
		want = []string{""}
	}
	if diff := cmp.Diff(want, strings.Split(string(contents), "\n")); diff != "" {
		t.Errorf(prefix+": incorrect file contents (-want, +got):\n%s", diff)
	}
}

func TestSourcerer(t *testing.T) {
	f, err := ioutil.TempFile("", "test-leep-frog")
	if err != nil {
		t.Fatalf("failed to create tmp file: %v", err)
	}

	// use, err := command.Use((&sourcerer{
	// 	cliArg: command.MapArg("CLI", "", map[string]CLI{}, false),
	// }).Node(), command.ParseExecuteArgs(nil))
	// if err != nil {
	// 	t.Fatalf("Failed to generate sourcerer usage: %v", err)
	// }

	baseUsage := strings.Join([]string{
		usagePrefixString,
		`[ TARGET_NAME ] --load-only|-l`,
		``,
		`Arguments:`,
		`  TARGET_NAME: The name of the created target in $GOPATH/bin`,
		``,
		`Flags:`,
		`  [l] load-only: If set to true, the binaries are assumed to exist and only the aliases and completion setups are generated`,
	}, "\n")
	executeUsage := strings.Join([]string{
		usagePrefixString,
		`FILE CLI [ ARG ... ]`,
		``,
		`Arguments:`,
		`  ARG: Arguments that get passed through to relevant CLI command`,
		`  FILE: Temporary file for execution`,
		`    FileExists()`,
	}, "\n")
	_ = baseUsage + executeUsage
	for _, test := range []struct {
		name            string
		clis            []CLI
		args            []string
		wantErr         error
		wantStdout      []string
		wantStderr      []string
		noStdoutNewline bool
		noStderrNewline bool
		wantCLIs        map[string]CLI
		wantOutput      []string
	}{
		{
			name: "fails if invalid command branch",
			args: []string{"wizardry", "stuff"},
			wantStderr: []string{
				"Unprocessed extra args: [stuff]",
				baseUsage,
			},
			wantErr:         fmt.Errorf("Unprocessed extra args: [stuff]"),
			noStderrNewline: true,
		},
		// Execute tests
		{
			name: "fails if no file arg",
			args: []string{"execute"},
			wantStderr: []string{
				`Argument "FILE" requires at least 1 argument, got 0`,
				baseUsage,
			},
			wantErr:         fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name: "fails if no cli arg",
			args: []string{"execute", fakeFile},
			wantStderr: []string{
				`Argument "CLI" requires at least 1 argument, got 0`,
				baseUsage,
			},
			wantErr:         fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name: "fails if unknown CLI",
			args: []string{"execute", fakeFile, "idk"},
			wantStderr: []string{
				"validation for \"CLI\" failed: [MapArg] key is not in map",
			},
			wantErr: fmt.Errorf("validation for \"CLI\" failed: [MapArg] key is not in map"),
		},
		{
			name: "properly executes CLI",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						var keys []string
						for k := range d.Values {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						o.Stdoutln("Output:")
						for _, k := range keys {
							o.Stdoutf("%s: %s\f", k, d.Values[k])
						}
						return nil
					},
				},
			},
			args:       []string{"execute", fakeFile, "basic"},
			wantStdout: []string{"Output:"},
		},
		{
			name: "handles processing error",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						return o.Stderrln("oops")
					},
				},
			},
			args:       []string{"execute", fakeFile, "basic"},
			wantStderr: []string{"oops"},
			wantErr:    fmt.Errorf("oops"),
		},
		{
			name: "properly passes arguments to CLI",
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.ListArg[string]("sl", "test desc", 1, 4),
					},
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						var keys []string
						for k := range d.Values {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						o.Stdoutln("Output:")
						for _, k := range keys {
							o.Stdoutf("%s: %s\n", k, d.Values[k])
						}
						return nil
					},
				},
			},
			args: []string{"execute", fakeFile, "basic", "un", "deux", "trois"},
			wantStdout: []string{
				"Output:",
				`sl: [un deux trois]`,
			},
		},
		{
			name: "properly passes extra arguments to CLI",
			clis: []CLI{
				&testCLI{
					name:       "basic",
					processors: []command.Processor{command.ListArg[string]("SL", "test", 1, 1)},
				},
			},
			args: []string{"execute", fakeFile, "basic", "un", "deux", "trois", "quatre"},
			wantStderr: []string{
				"Unprocessed extra args: [trois quatre]",
				strings.Join([]string{
					usagePrefixString,
					"SL [ SL ]",
					"",
					"Arguments:",
					"  SL: test",
				}, "\n"),
			},
			wantErr:         fmt.Errorf("Unprocessed extra args: [trois quatre]"),
			noStderrNewline: true,
		},
		{
			name: "properly marks CLI as changed",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						tc.Stuff = "things"
						tc.changed = true
						return nil
					},
				},
			},
			args: []string{"execute", fakeFile, "basic"},
			wantCLIs: map[string]CLI{
				"basic": &testCLI{
					Stuff: "things",
				},
			},
		},
		{
			name: "writes execute data to file",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						ed.Executable = []string{"echo", "hello", "there"}
						return nil
					},
				},
			},
			args: []string{"execute", f.Name(), "basic"},
			wantOutput: []string{
				"echo",
				"hello",
				"there",
			},
		},
		{
			name: "writes function wrapped execute data to file",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						ed.Executable = []string{"echo", "hello", "there"}
						ed.FunctionWrap = true
						return nil
					},
				},
			},
			args: []string{"execute", f.Name(), "basic"},
			wantOutput: []string{
				"function _leep_execute_data_function_wrap {",
				"echo",
				"hello",
				"there",
				"}",
				"_leep_execute_data_function_wrap",
				"",
			},
		},
		// CLI with setup:
		{
			name: "SetupArg node is automatically added as required arg",
			clis: []CLI{
				&testCLI{
					name:  "basic",
					setup: []string{"his", "story"},
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						o.Stdoutf("stdout: %v\n", d)
						return nil
					},
				},
			},
			args: []string{
				"execute", fakeFile, "basic",
			},
			wantErr: fmt.Errorf(`Argument "SETUP_FILE" requires at least 1 argument, got 0`),
			wantStderr: []string{
				`Argument "SETUP_FILE" requires at least 1 argument, got 0`,
				usagePrefixString + "\n",
			},
			noStderrNewline: true,
		},
		{
			name: "SetupArg is properly populated",
			clis: []CLI{
				&testCLI{
					name:  "basic",
					setup: []string{"his", "story"},
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						o.Stdoutf("stdout: %v\n", d)
						return nil
					},
				},
			},
			args: []string{
				"execute",
				fakeFile,
				"basic",
				// SetupArg needs to be a real file, hence why it's this.
				"sourcerer.go",
			},
			wantStdout: []string{
				// false is for data.complexecute
				fmt.Sprintf(`stdout: &{map[SETUP_FILE:%s] false}`, command.FilepathAbs(t, "sourcerer.go")),
			},
		},
		{
			name: "args after SetupArg are properly populated",
			clis: []CLI{
				&testCLI{
					name:  "basic",
					setup: []string{"his", "story"},
					processors: []command.Processor{
						command.Arg[int]("i", "desc"),
					},
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						o.Stdoutf("stdout: %v\n", d)
						return nil
					},
				},
			},
			args: []string{
				"execute",
				fakeFile,
				"basic",
				// SetupArg needs to be a real file, hence why it's this.
				"sourcerer.go",
				"5",
			},
			wantStdout: []string{
				// false is for data.complexecute
				fmt.Sprintf(`stdout: &{map[SETUP_FILE:%s i:5] false}`, command.FilepathAbs(t, "sourcerer.go")),
			},
		},
		// Usage printing tests
		{
			name: "prints command usage for missing branch error",
			clis: []CLI{&usageErrCLI{}},
			args: []string{"execute", fakeFile, "uec"},
			wantStderr: []string{
				"Branching argument must be one of [a b]",
				uecUsage(),
			},
			wantErr:         fmt.Errorf("Branching argument must be one of [a b]"),
			noStderrNewline: true,
		},
		{
			name: "prints command usage for bad branch arg error",
			clis: []CLI{&usageErrCLI{}},
			args: []string{"execute", fakeFile, "uec", "uh"},
			wantStderr: []string{
				"Branching argument must be one of [a b]",
				uecUsage(),
			},
			wantErr:         fmt.Errorf("Branching argument must be one of [a b]"),
			noStderrNewline: true,
		},
		{
			name: "prints command usage for missing args error",
			clis: []CLI{&usageErrCLI{}},
			args: []string{"execute", fakeFile, "uec", "b"},
			wantStderr: []string{
				`Argument "B_SL" requires at least 1 argument, got 0`,
				uecUsage(),
			},
			wantErr:         fmt.Errorf(`Argument "B_SL" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name: "prints command usage for missing args error",
			clis: []CLI{&usageErrCLI{}},
			args: []string{"execute", fakeFile, "uec", "a", "un", "deux", "trois"},
			wantStderr: []string{
				"Unprocessed extra args: [deux trois]",
				uecUsage(),
			},
			wantErr:         fmt.Errorf("Unprocessed extra args: [deux trois]"),
			noStderrNewline: true,
		},
		// Autocomplete tests
		{
			name: "autocomplete requires cli name",
			args: []string{"autocomplete"},
			wantStderr: []string{
				`Argument "CLI" requires at least 1 argument, got 0`,
				baseUsage,
			},
			wantErr:         fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name: "autocomplete requires comp_type",
			args: []string{"autocomplete", "uec"},
			clis: []CLI{&usageErrCLI{}},
			wantStderr: []string{
				`Argument "COMP_TYPE" requires at least 1 argument, got 0`,
				baseUsage,
			},
			wantErr:         fmt.Errorf(`Argument "COMP_TYPE" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name: "autocomplete requires comp_point",
			args: []string{"autocomplete", "uec", "63"},
			clis: []CLI{&usageErrCLI{}},
			wantStderr: []string{
				`Argument "COMP_POINT" requires at least 1 argument, got 0`,
				baseUsage,
			},
			wantErr:         fmt.Errorf(`Argument "COMP_POINT" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name: "autocomplete requires comp_line",
			args: []string{"autocomplete", "uec", "63", "2"},
			clis: []CLI{&usageErrCLI{}},
			wantStderr: []string{
				`Argument "COMP_LINE" requires at least 1 argument, got 0`,
				baseUsage,
			},
			wantErr:         fmt.Errorf(`Argument "COMP_LINE" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name:    "autocomplete doesn't require passthrough args",
			args:    []string{"autocomplete", "basic", "63", "0", "h"},
			clis:    []CLI{&testCLI{name: "basic"}},
			wantErr: fmt.Errorf("Unprocessed extra args: []"),
			wantStdout: []string{
				" ",
				"\t",
			},
			wantStderr: []string{
				"",
				"Unprocessed extra args: []",
			},
			noStderrNewline: true,
		},
		{
			name:    "autocomplete re-prints comp line",
			args:    []string{"autocomplete", "basic", "63", "10", "hello ther"},
			clis:    []CLI{&testCLI{name: "basic"}},
			wantErr: fmt.Errorf("Unprocessed extra args: [ther]"),
			wantStdout: []string{
				" ",
				"\t",
			},
			wantStderr: []string{
				"",
				"Unprocessed extra args: [ther]",
			},
			noStderrNewline: true,
		},
		{
			name:    "autocomplete doesn't re-print comp line if different COMP_TYPE",
			args:    []string{"autocomplete", "basic", "64", "10", "hello ther"},
			clis:    []CLI{&testCLI{name: "basic"}},
			wantErr: fmt.Errorf("Unprocessed extra args: [ther]"),
		},
		{
			name: "autocomplete requires valid cli",
			args: []string{"autocomplete", "idk", "63", "2", "a"},
			wantStderr: []string{
				"validation for \"CLI\" failed: [MapArg] key is not in map\n",
			},
			wantErr:         fmt.Errorf("validation for \"CLI\" failed: [MapArg] key is not in map"),
			noStderrNewline: true,
		},
		{
			name: "autocomplete passes empty string along for completion",
			args: []string{"autocomplete", "basic", "63", "4", "cmd "},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"alpha",
				"bravo",
				"charlie",
			),
		},
		{
			name: "autocomplete doesn't complete passthrough args",
			args: []string{"autocomplete", "basic", "63", "4", "cmd ", "al"},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.ListArg[string]("s", "desc", 0, command.UnboundedList, command.SimpleCompleter[[]string]("alpha", "bravo", "charlie")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"alpha",
				"bravo",
				"charlie",
			),
		},
		/*{
			name: "autocomplete doesn't complete passthrough args",
			args: []string{"autocomplete", "basic", "0", "", "al"},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.ListArg[string]()
						command.Arg[string]("s", "desc",
							&command.Completer[string]{
								Fetcher: command.SimpleFetcher(func(t string, d *command.Data) (*command.Completion, error) {
									return nil, nil
								}),
							},
						),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"alpha",
				"bravo",
				"charlie",
			),
		},*/
		{
			name: "autocomplete does partial completion",
			args: []string{"autocomplete", "basic", "63", "5", "cmd b"},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"baker",
				"bravo",
				"brown",
			),
		},
		{
			name: "autocomplete goes along processors",
			args: []string{"autocomplete", "basic", "63", "6", "cmd a "},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
						command.Arg[string]("z", "desz", command.SimpleCompleter[string]("un", "deux", "trois")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"un",
				"deux",
				"trois",
			),
		},
		{
			name: "autocomplete does earlier completion if cpoint is smaller",
			args: []string{"autocomplete", "basic", "63", "5", "cmd c "},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
						command.Arg[string]("z", "desz", command.SimpleCompleter[string]("un", "deux", "trois")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"charlie",
			),
		},
		// Usage tests
		{
			name: "usage requires cli name",
			args: []string{"usage"},
			wantStderr: []string{
				`Argument "CLI" requires at least 1 argument, got 0`,
				baseUsage,
			},
			wantErr:         fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
			noStderrNewline: true,
		},
		{
			name: "usage fails if too many args",
			args: []string{"usage", "uec", "b", "un", "deux"},
			clis: []CLI{&usageErrCLI{}},
			wantStderr: []string{
				"Unprocessed extra args: [deux]",
				baseUsage,
			},
			wantErr:         fmt.Errorf("Unprocessed extra args: [deux]"),
			noStderrNewline: true,
		},
		{
			name: "usage prints command's usage",
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.Arg[string]("S", "desc"),
						command.ListArg[int]("IS", "ints", 2, 0),
						command.ListArg[float64]("FS", "floats", 0, command.UnboundedList),
					},
				},
			},
			args: []string{"usage", "basic"},
			wantStdout: []string{strings.Join([]string{
				"S IS IS [ FS ... ]",
				"",
				"Arguments:",
				"  FS: floats",
				"  IS: ints",
				"  S: desc",
			}, "\n")},
			noStdoutNewline: true,
		},
		/* Useful for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ioutil.WriteFile(f.Name(), nil, command.CmdOS.DefaultFilePerm()); err != nil {
				t.Fatalf("failed to clear file: %v", err)
			}

			fake, err := ioutil.TempFile("", "leep-frog-sourcerer-test")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			for i, s := range test.args {
				if s == fakeFile {
					test.args[i] = fake.Name()
				}
			}

			// Stub out real cache
			cash := cache.NewTestCache(t)
			command.StubValue(t, &getCache, func() (*cache.Cache, error) {
				return cash, nil
			})

			// Run source command
			o := command.NewFakeOutput()
			err = source(test.clis, test.args, o)
			command.CmpError(t, fmt.Sprintf("source(%v)", test.args), test.wantErr, err)
			o.Close()

			// Check outputs
			if !test.noStderrNewline {
				test.wantStderr = append(test.wantStderr, "")
			}
			if !test.noStdoutNewline {
				test.wantStdout = append(test.wantStdout, "")
			}
			if diff := cmp.Diff(strings.Join(test.wantStdout, "\n"), o.GetStdout()); diff != "" {
				t.Errorf("source(%v) sent incorrect stdout (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(strings.Join(test.wantStderr, "\n"), o.GetStderr()); diff != "" {
				t.Errorf("source(%v) sent incorrect stderr (-want, +got):\n%s", test.args, diff)
			}

			// Check file contents
			cmpFile(t, "Sourcing produced incorrect file contents", f.Name(), test.wantOutput)

			// Check cli changes
			for _, c := range test.clis {
				wantNew, wantChanged := test.wantCLIs[c.Name()]
				if wantChanged != c.Changed() {
					t.Errorf("CLI %q was incorrectly changed: want %v; got %v", c.Name(), wantChanged, c.Changed())
				}
				if wantChanged {
					if diff := cmp.Diff(wantNew, c, cmpopts.IgnoreUnexported(testCLI{})); diff != "" {
						t.Errorf("CLI %q was incorrectly updated: %v", c.Name(), diff)
					}
				}
				delete(test.wantCLIs, c.Name())
			}

			if len(test.wantCLIs) != 0 {
				for name := range test.wantCLIs {
					t.Errorf("Unknown CLI was supposed to change %q", name)
				}
			}
		})
	}
}

type testCLI struct {
	name       string
	processors []command.Processor
	f          func(*testCLI, *command.Input, command.Output, *command.Data, *command.ExecuteData) error
	changed    bool
	setup      []string
	// Used for json checking
	Stuff string
}

func (tc *testCLI) Name() string {
	return tc.name
}

func (tc *testCLI) UnmarshalJSON([]byte) error { return nil }
func (tc *testCLI) Node() command.Node {
	ns := append(tc.processors, command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		if tc.f != nil {
			return tc.f(tc, i, o, d, ed)
		}
		return nil
	}, nil))
	return command.SerialNodes(ns...)
}
func (tc *testCLI) Changed() bool   { return tc.changed }
func (tc *testCLI) Setup() []string { return tc.setup }

func autocompleteSuggestions(s ...string) []string {
	sort.Strings(s)
	return s
}

type usageErrCLI struct{}

func (uec *usageErrCLI) Name() string {
	return "uec"
}

func (uec *usageErrCLI) UnmarshalJSON([]byte) error { return nil }
func (uec *usageErrCLI) Node() command.Node {
	return &command.BranchNode{
		Branches: map[string]command.Node{
			"a": command.SerialNodes(command.ListArg[string]("A_SL", "str list", 0, 1)),
			"b": command.SerialNodes(command.ListArg[string]("B_SL", "str list", 1, 0)),
		},
		DefaultCompletion: true,
	}
}
func (uec *usageErrCLI) Changed() bool   { return false }
func (uec *usageErrCLI) Setup() []string { return nil }

func uecUsage() string {
	return strings.Join([]string{
		usagePrefixString,
		`<`,
		``,
		`  a [ A_SL ]`,
		``,
		`  b B_SL`,
		``,
		`Arguments:`,
		`  A_SL: str list`,
		`  B_SL: str list`,
		``,
		`Symbols:`,
		`  <: Start of subcommand branches`,
	}, "\n")
}
