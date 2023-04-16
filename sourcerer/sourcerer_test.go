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

	osLinux   = "linux"
	osWindows = "windows"
)

func TestGenerateBinaryNode(t *testing.T) {
	command.StubValue(t, &getSourceLoc, func() (string, error) {
		return "/fake/source/location", nil
	})

	type osCheck struct {
		wantStdout      []string
		wantStderr      []string
		wantExecuteFile []string
	}

	// We loop the OS here (and not in the test), so any underlying test data for
	// each test case is totally recreated (rather than re-using the same data
	// across tests which can be error prone and difficult to debug).
	for _, curOS := range []OS{Linux(), Windows()} {
		for _, test := range []struct {
			name            string
			clis            []CLI
			args            []string
			ignoreNosort    bool
			opts            []Option
			getSourceLocErr error
			osChecks        map[string]*osCheck
		}{
			{
				name: "generates source file when no CLIs",
				args: []string{"source"},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_leep-frog-source {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_leep-frog-source_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							`function _custom_autocomplete_leep-frog-source {`,
							`  local tFile=$(mktemp)`,
							`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
						},
					},
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`Push-Location`,
							`Set-Location "$(Split-Path /fake/source/location)"`,
							`go build -o $env:GOPATH\bin\_leep-frog-source_runner.exe`,
							`Pop-Location`,
							``,
							`$_custom_autocomplete_leep-frog-source = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_leep-frog-source_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
						},
					},
				},
			},
			{
				name: "adds multiple Aliaser (singular) options at the end",
				args: []string{"source"},
				opts: []Option{
					NewAliaser("a1", "do", "some", "stuff"),
					NewAliaser("otherAlias", "flaggable", "--args", "--at", "once"),
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_leep-frog-source {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_leep-frog-source_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							`function _custom_autocomplete_leep-frog-source {`,
							`  local tFile=$(mktemp)`,
							`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							(&linux{}).aliaserGlobalAutocompleteFunction(),
							`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
							`  return 1`,
							`fi`,
							``,
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
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`Push-Location`,
							`Set-Location "$(Split-Path /fake/source/location)"`,
							`go build -o $env:GOPATH\bin\_leep-frog-source_runner.exe`,
							`Pop-Location`,
							``,
							`$_custom_autocomplete_leep-frog-source = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_leep-frog-source_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
						},
					},
				},
			},
			{
				name: "load only flag doesn't generate binaries",
				args: []string{"source", "-l"},
				opts: []Option{
					NewAliaser("a1", "do", "some", "stuff"),
					NewAliaser("otherAlias", "flaggable", "--args", "--at", "once"),
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_leep-frog-source {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_leep-frog-source_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							(&linux{}).aliaserGlobalAutocompleteFunction(),
							`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
							`  return 1`,
							`fi`,
							``,
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
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`$_custom_autocomplete_leep-frog-source = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_leep-frog-source_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
						},
					},
				},
			},
			{
				name: "only verifies each CLI once",
				args: []string{"source"},
				opts: []Option{
					// Note the CLI in both of these is "do"
					NewAliaser("a1", "do", "some", "stuff"),
					NewAliaser("otherAlias", "do", "other", "stuff"),
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_leep-frog-source {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_leep-frog-source_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							`function _custom_autocomplete_leep-frog-source {`,
							`  local tFile=$(mktemp)`,
							`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							(&linux{}).aliaserGlobalAutocompleteFunction(),
							`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
							`  return 1`,
							`fi`,
							``,
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
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`Push-Location`,
							`Set-Location "$(Split-Path /fake/source/location)"`,
							`go build -o $env:GOPATH\bin\_leep-frog-source_runner.exe`,
							`Pop-Location`,
							``,
							`$_custom_autocomplete_leep-frog-source = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_leep-frog-source_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
						},
					},
				},
			},
			{
				name: "adds Aliasers (plural) at the end",
				args: []string{"source"},
				opts: []Option{
					Aliasers(map[string][]string{
						"a1":         {"do", "some", "stuff"},
						"otherAlias": {"flaggable", "--args", "--at", "once"},
					}),
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_leep-frog-source {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_leep-frog-source_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							`function _custom_autocomplete_leep-frog-source {`,
							`  local tFile=$(mktemp)`,
							`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							(&linux{}).aliaserGlobalAutocompleteFunction(),
							`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
							`  return 1`,
							`fi`,
							``,
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
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`Push-Location`,
							`Set-Location "$(Split-Path /fake/source/location)"`,
							`go build -o $env:GOPATH\bin\_leep-frog-source_runner.exe`,
							`Pop-Location`,
							``,
							`$_custom_autocomplete_leep-frog-source = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_leep-frog-source_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
						},
					},
				},
			},
			{
				name: "generates source file with custom filename",
				args: []string{"source", "custom-output_file"},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_custom-output_file {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_custom-output_file_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							`function _custom_autocomplete_custom-output_file {`,
							`  local tFile=$(mktemp)`,
							`  $GOPATH/bin/_custom-output_file_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
						},
					},
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`Push-Location`,
							`Set-Location "$(Split-Path /fake/source/location)"`,
							`go build -o $env:GOPATH\bin\_custom-output_file_runner.exe`,
							`Pop-Location`,
							``,
							`$_custom_autocomplete_custom-output_file = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_custom-output_file_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
						},
					},
				},
			},
			{
				name: "generates source file with CLIs",
				args: []string{"source"},
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_leep-frog-source {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_leep-frog-source_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							`function _custom_autocomplete_leep-frog-source {`,
							`  local tFile=$(mktemp)`,
							`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
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
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`Push-Location`,
							`Set-Location "$(Split-Path /fake/source/location)"`,
							`go build -o $env:GOPATH\bin\_leep-frog-source_runner.exe`,
							`Pop-Location`,
							``,
							`$_custom_autocomplete_leep-frog-source = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_leep-frog-source_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
							`  story`,
							`}`,
							``,
							`function _custom_execute_leep-frog-source_basic {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  $Local:setupTmpFile = New-TemporaryFile`,
							`  _setup_for_basic_cli > $Local:setupTmpFile`,
							`  & $env:GOPATH/bin/_leep-frog-source_runner.exe execute "basic" $Local:tmpFile $Local:setupTmpFile $args`,
							`  # Return error if failed`,
							`  If (!$?) { throw "Go execution failed" }`,
							``,
							`  # If success, run the ExecuteData.Executable data`,
							`  Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`  . "$Local:tmpFile.ps1"`,
							`  If (!$?) { throw "ExecuteData execution failed" }`,
							`}`,
							``,
							`Set-Alias basic _custom_execute_leep-frog-source_basic`,
							``,
							`Register-ArgumentCompleter -CommandName basic -ScriptBlock $_custom_autocomplete_leep-frog-source`,
							``,
							`function _custom_execute_leep-frog-source_l {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  & $env:GOPATH/bin/_leep-frog-source_runner.exe execute "l" $Local:tmpFile $args`,
							`  # Return error if failed`,
							`  If (!$?) { throw "Go execution failed" }`,
							``,
							`  # If success, run the ExecuteData.Executable data`,
							`  Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`  . "$Local:tmpFile.ps1"`,
							`  If (!$?) { throw "ExecuteData execution failed" }`,
							`}`,
							``,
							`Set-Alias l _custom_execute_leep-frog-source_l`,
							``,
							`Register-ArgumentCompleter -CommandName l -ScriptBlock $_custom_autocomplete_leep-frog-source`,
							``,
							`function _custom_execute_leep-frog-source_x {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  & $env:GOPATH/bin/_leep-frog-source_runner.exe execute "x" $Local:tmpFile $args`,
							`  # Return error if failed`,
							`  If (!$?) { throw "Go execution failed" }`,
							``,
							`  # If success, run the ExecuteData.Executable data`,
							`  Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`  . "$Local:tmpFile.ps1"`,
							`  If (!$?) { throw "ExecuteData execution failed" }`,
							`}`,
							``,
							`Set-Alias x _custom_execute_leep-frog-source_x`,
							``,
							`Register-ArgumentCompleter -CommandName x -ScriptBlock $_custom_autocomplete_leep-frog-source`,
						},
					},
				},
			},
			{
				name: "generates source file with CLIs ignoring nosort",
				args: []string{"source"},
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				ignoreNosort: true,
				osChecks: map[string]*osCheck{
					osLinux: {
						wantExecuteFile: []string{
							`function _custom_execute_leep-frog-source {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							`  $GOPATH/bin/_leep-frog-source_runner execute "$1" $tmpFile "${@:2}"`,
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
							``,
							`function _custom_autocomplete_leep-frog-source {`,
							`  local tFile=$(mktemp)`,
							`  $GOPATH/bin/_leep-frog-source_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
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
					osWindows: {
						wantExecuteFile: []string{""},
						wantStdout: []string{
							`Push-Location`,
							`Set-Location "$(Split-Path /fake/source/location)"`,
							`go build -o $env:GOPATH\bin\_leep-frog-source_runner.exe`,
							`Pop-Location`,
							``,
							`$_custom_autocomplete_leep-frog-source = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  (& $env:GOPATH\bin\_leep-frog-source_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`,
							`    $_`,
							`  }`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
							`  story`,
							`}`,
							``,
							`function _custom_execute_leep-frog-source_basic {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  $Local:setupTmpFile = New-TemporaryFile`,
							`  _setup_for_basic_cli > $Local:setupTmpFile`,
							`  & $env:GOPATH/bin/_leep-frog-source_runner.exe execute "basic" $Local:tmpFile $Local:setupTmpFile $args`,
							`  # Return error if failed`,
							`  If (!$?) { throw "Go execution failed" }`,
							``,
							`  # If success, run the ExecuteData.Executable data`,
							`  Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`  . "$Local:tmpFile.ps1"`,
							`  If (!$?) { throw "ExecuteData execution failed" }`,
							`}`,
							``,
							`Set-Alias basic _custom_execute_leep-frog-source_basic`,
							``,
							`Register-ArgumentCompleter -CommandName basic -ScriptBlock $_custom_autocomplete_leep-frog-source`,
							``,
							`function _custom_execute_leep-frog-source_l {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  & $env:GOPATH/bin/_leep-frog-source_runner.exe execute "l" $Local:tmpFile $args`,
							`  # Return error if failed`,
							`  If (!$?) { throw "Go execution failed" }`,
							``,
							`  # If success, run the ExecuteData.Executable data`,
							`  Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`  . "$Local:tmpFile.ps1"`,
							`  If (!$?) { throw "ExecuteData execution failed" }`,
							`}`,
							``,
							`Set-Alias l _custom_execute_leep-frog-source_l`,
							``,
							`Register-ArgumentCompleter -CommandName l -ScriptBlock $_custom_autocomplete_leep-frog-source`,
							``,
							`function _custom_execute_leep-frog-source_x {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  & $env:GOPATH/bin/_leep-frog-source_runner.exe execute "x" $Local:tmpFile $args`,
							`  # Return error if failed`,
							`  If (!$?) { throw "Go execution failed" }`,
							``,
							`  # If success, run the ExecuteData.Executable data`,
							`  Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`  . "$Local:tmpFile.ps1"`,
							`  If (!$?) { throw "ExecuteData execution failed" }`,
							`}`,
							``,
							`Set-Alias x _custom_execute_leep-frog-source_x`,
							``,
							`Register-ArgumentCompleter -CommandName x -ScriptBlock $_custom_autocomplete_leep-frog-source`,
						},
					},
				},
			},
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					t.Skipf("No osCheck set for this OS")
				}

				command.StubValue(t, &CurrentOS, curOS)

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
				if diff := cmp.Diff(strings.Join(append(oschk.wantStdout, ""), "\n"), o.GetStdout()); diff != "" {
					t.Errorf("source(%v) sent incorrect data to stdout (-wamt, +got):\n%s", test.args, diff)
				}
				if diff := cmp.Diff(strings.Join(oschk.wantStderr, "\n"), o.GetStderr()); diff != "" {
					t.Errorf("source(%v) sent incorrect data to stderr (-wamt, +got):\n%s", test.args, diff)
				}

				cmpFile(t, fmt.Sprintf("source(%v) created incorrect execute file", test.args), tmp.Name(), oschk.wantExecuteFile)
			})
		}
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

	type osCheck struct {
		wantErr         error
		wantStdout      []string
		wantStderr      []string
		noStdoutNewline bool
		noStderrNewline bool
		wantCLIs        map[string]CLI
		wantOutput      []string
	}

	// We loop the OS here (and not in the test), so any underlying test data for
	// each test case is totally recreated (rather than re-using the same data
	// across tests which can be error prone and difficult to debug).
	for _, curOS := range []OS{Linux(), Windows()} {
		for _, test := range []struct {
			name     string
			clis     []CLI
			args     []string
			cacheErr error
			osCheck  *osCheck
			osChecks map[string]*osCheck
		}{
			{
				name: "fails if invalid command branch",
				args: []string{"wizardry", "stuff"},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [wizardry stuff]",
					},
					wantErr: fmt.Errorf("Unprocessed extra args: [wizardry stuff]"),
				},
			},
			// Execute tests
			{
				name: "fails if no cli arg",
				args: []string{"execute"},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "CLI" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "fails if no cli arg other",
				args: []string{},
				osCheck: &osCheck{
					wantStderr: []string{
						"echo \"Executing a sourcerer.CLI directly through `go run` is tricky. Either generate a CLI or use the `goleep` command to directly run the file.\"",
					},
					wantErr: fmt.Errorf("echo \"Executing a sourcerer.CLI directly through `go run` is tricky. Either generate a CLI or use the `goleep` command to directly run the file.\""),
				},
			},
			{
				name: "fails if no file arg",
				args: []string{"execute", "bc"},
				clis: []CLI{ToCLI("bc", nil)},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "FILE" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "FILE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "fails if unknown CLI",
				args: []string{"execute", "idk"},
				osCheck: &osCheck{
					wantStderr: []string{
						"validation for \"CLI\" failed: [MapArg] key (idk) is not in map",
					},
					wantErr: fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (idk) is not in map"),
				},
			},
			{
				name:     "fails if getCache error",
				cacheErr: fmt.Errorf("rats"),
				clis: []CLI{
					&testCLI{
						name: "basic",
					},
				},
				args: []string{"execute", "basic", fakeFile},
				osCheck: &osCheck{
					wantErr:    fmt.Errorf("failed to load cache from environment variable: rats"),
					wantStderr: []string{"failed to load cache from environment variable: rats"},
				},
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
				args: []string{"execute", "basic", fakeFile},
				osCheck: &osCheck{
					wantStdout: []string{"Output:"},
				},
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
				args: []string{"execute", "basic", fakeFile},
				osCheck: &osCheck{
					wantStderr: []string{"oops"},
					wantErr:    fmt.Errorf("oops"),
				},
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
				args: []string{"execute", "basic", fakeFile, "un", "deux", "trois"},
				osCheck: &osCheck{
					wantStdout: []string{
						"Output:",
						`sl: [un deux trois]`,
					},
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
				args: []string{"execute", "basic", fakeFile, "un", "deux", "trois", "quatre"},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [trois quatre]",
						strings.Join([]string{
							usagePrefixString,
							"SL [ SL ]",
							"",
							"Arguments:",
							"  SL: test",
							"",
						}, "\n"),
					},
					wantErr:         fmt.Errorf("Unprocessed extra args: [trois quatre]"),
					noStderrNewline: true,
				},
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
				args: []string{"execute", "basic", fakeFile},
				osCheck: &osCheck{
					wantCLIs: map[string]CLI{
						"basic": &testCLI{
							Stuff: "things",
						},
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
				args: []string{"execute", "basic", f.Name()},
				osCheck: &osCheck{
					wantOutput: []string{
						"echo",
						"hello",
						"there",
					},
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
				args: []string{"execute", "basic", f.Name()},
				osCheck: &osCheck{
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
					"execute", "basic", fakeFile,
				},
				osCheck: &osCheck{
					wantErr: fmt.Errorf(`Argument "SETUP_FILE" requires at least 1 argument, got 0`),
					wantStderr: []string{
						`Argument "SETUP_FILE" requires at least 1 argument, got 0`,
					},
				},
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
					"basic",
					fakeFile,
					// SetupArg needs to be a real file, hence why it's this.
					"sourcerer.go",
				},
				osCheck: &osCheck{
					wantStdout: []string{
						// false is for data.complexecute
						fmt.Sprintf(`stdout: &{map[SETUP_FILE:%s] false}`, command.FilepathAbs(t, "sourcerer.go")),
					},
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
					"basic",
					fakeFile,
					// SetupArg needs to be a real file, hence why it's this.
					"sourcerer.go",
					"5",
				},
				osCheck: &osCheck{
					wantStdout: []string{
						// false is for data.complexecute
						fmt.Sprintf(`stdout: &{map[SETUP_FILE:%s i:5] false}`, command.FilepathAbs(t, "sourcerer.go")),
					},
				},
			},
			// Usage printing tests
			{
				name: "prints command usage for missing branch error",
				clis: []CLI{&usageErrCLI{}},
				args: []string{"execute", "uec", fakeFile},
				osCheck: &osCheck{
					wantStderr: []string{
						"Branching argument must be one of [a b]",
						uecUsage(),
					},
					wantErr:         fmt.Errorf("Branching argument must be one of [a b]"),
					noStderrNewline: true,
				},
			},
			{
				name: "prints command usage for bad branch arg error",
				clis: []CLI{&usageErrCLI{}},
				args: []string{"execute", "uec", fakeFile, "uh"},
				osCheck: &osCheck{
					wantStderr: []string{
						"Branching argument must be one of [a b]",
						uecUsage(),
					},
					wantErr:         fmt.Errorf("Branching argument must be one of [a b]"),
					noStderrNewline: true,
				},
			},
			{
				name: "prints command usage for missing args error",
				clis: []CLI{&usageErrCLI{}},
				args: []string{"execute", "uec", fakeFile, "b"},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "B_SL" requires at least 1 argument, got 0`,
						uecUsage(),
					},
					wantErr:         fmt.Errorf(`Argument "B_SL" requires at least 1 argument, got 0`),
					noStderrNewline: true,
				},
			},
			{
				name: "prints command usage for missing args error",
				clis: []CLI{&usageErrCLI{}},
				args: []string{"execute", "uec", fakeFile, "a", "un", "deux", "trois"},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [deux trois]",
						uecUsage(),
					},
					wantErr:         fmt.Errorf("Unprocessed extra args: [deux trois]"),
					noStderrNewline: true,
				},
			},
			// List CLI tests
			{
				name: "lists none",
				args: []string{ListBranchName},
				osCheck: &osCheck{
					wantStdout: []string{""},
				},
			},
			{
				name: "lists clis",
				args: []string{ListBranchName},
				clis: []CLI{
					&testCLI{name: "un"},
					&testCLI{name: "deux"},
					&testCLI{name: "trois"},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						"deux",
						"trois",
						"un",
					},
				},
			},
			// Autocomplete tests
			{
				name: "autocomplete requires cli name",
				args: []string{"autocomplete"},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "CLI" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "autocomplete requires comp_type",
				args: []string{"autocomplete", "uec"},
				clis: []CLI{&usageErrCLI{}},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "COMP_TYPE" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "COMP_TYPE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "autocomplete requires comp_point",
				args: []string{"autocomplete", "uec", "63"},
				clis: []CLI{&usageErrCLI{}},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "COMP_POINT" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "COMP_POINT" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "autocomplete requires comp_line",
				args: []string{"autocomplete", "uec", "63", "2"},
				clis: []CLI{&usageErrCLI{}},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "COMP_LINE" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "COMP_LINE" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "autocomplete doesn't require passthrough args",
				args: []string{"autocomplete", "basic", "63", "0", "h"},
				clis: []CLI{&testCLI{name: "basic"}},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantErr: fmt.Errorf("Unprocessed extra args: []"),
						wantStdout: []string{
							"\t",
							" ",
						},
						wantStderr: []string{
							"",
							"Autocomplete Error: Unprocessed extra args: []",
						},
						noStderrNewline: true,
					},
					osWindows: {
						wantErr: fmt.Errorf("Unprocessed extra args: []"),
						wantStdout: []string{
							"",
						},
						wantStderr: []string{
							"",
							"Autocomplete Error: Unprocessed extra args: []",
						},
						noStderrNewline: true,
					},
				},
			},
			{
				name: "autocomplete re-prints comp line",
				args: []string{"autocomplete", "basic", "63", "10", "hello ther"},
				clis: []CLI{&testCLI{name: "basic"}},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantErr: fmt.Errorf("Unprocessed extra args: [ther]"),
						wantStdout: []string{
							"\t",
							" ",
						},
						wantStderr: []string{
							"",
							"Autocomplete Error: Unprocessed extra args: [ther]",
						},
						noStderrNewline: true,
					},
					osWindows: {
						wantErr: fmt.Errorf("Unprocessed extra args: [ther]"),
						wantStdout: []string{
							"",
						},
						wantStderr: []string{
							"",
							"Autocomplete Error: Unprocessed extra args: [ther]",
						},
						noStderrNewline: true,
					},
				},
			},
			{
				name: "autocomplete doesn't re-print comp line if different COMP_TYPE",
				args: []string{"autocomplete", "basic", "64", "10", "hello ther"},
				clis: []CLI{&testCLI{name: "basic"}},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantErr: fmt.Errorf("Unprocessed extra args: [ther]"),
					},
					osWindows: {
						wantStdout: []string{""},
						wantStderr: []string{
							"",
							"Autocomplete Error: Unprocessed extra args: [ther]",
						},
						wantErr:         fmt.Errorf("Unprocessed extra args: [ther]"),
						noStderrNewline: true,
					},
				},
			},
			{
				name: "autocomplete requires valid cli",
				args: []string{"autocomplete", "idk", "63", "2", "a"},
				osCheck: &osCheck{
					wantStderr: []string{
						"validation for \"CLI\" failed: [MapArg] key (idk) is not in map\n",
					},
					wantErr:         fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (idk) is not in map"),
					noStderrNewline: true,
				},
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
				osCheck: &osCheck{
					wantStdout: autocompleteSuggestions(
						"alpha",
						"bravo",
						"charlie",
					),
				},
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
				osCheck: &osCheck{
					wantStdout: autocompleteSuggestions(
						"alpha",
						"bravo",
						"charlie",
					),
				},
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
				osCheck: &osCheck{
					wantStdout: autocompleteSuggestions(
						"baker",
						"bravo",
						"brown",
					),
				},
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
				osCheck: &osCheck{
					wantStdout: autocompleteSuggestions(
						"un",
						"deux",
						"trois",
					),
				},
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
				osCheck: &osCheck{
					wantStdout: autocompleteSuggestions(
						"charlie",
					),
				},
			},
			// Usage tests
			{
				name: "usage requires cli name",
				args: []string{"usage"},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "CLI" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
				},
			},
			{
				name: "usage fails if too many args",
				args: []string{"usage", "uec", "b", "un", "deux"},
				clis: []CLI{&usageErrCLI{}},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [deux]",
						uecUsage(),
					},
					wantErr:         fmt.Errorf("Unprocessed extra args: [deux]"),
					noStderrNewline: true,
				},
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
				osCheck: &osCheck{
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
			},
			/* Useful for commenting out tests */
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				command.StubValue(t, &CurrentOS, curOS)
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					oschk = test.osCheck
				}

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
					if test.cacheErr != nil {
						return nil, test.cacheErr
					}
					return cash, nil
				})

				// Run source command
				o := command.NewFakeOutput()
				err = source(test.clis, test.args, o)
				command.CmpError(t, fmt.Sprintf("source(%v)", test.args), oschk.wantErr, err)
				o.Close()

				// Check outputs

				// Make a separate variable so we don't edit variables on runs for different OS's.
				wantStdout, wantStderr := oschk.wantStdout, oschk.wantStderr
				if !oschk.noStdoutNewline {
					wantStdout = append(wantStdout, "")
				}
				if !oschk.noStderrNewline {
					wantStderr = append(wantStderr, "")
				}
				if diff := cmp.Diff(strings.Join(wantStdout, "\n"), o.GetStdout()); diff != "" {
					t.Errorf("source(%v) sent incorrect stdout (-want, +got):\n%s", test.args, diff)
				}
				if diff := cmp.Diff(strings.Join(wantStderr, "\n"), o.GetStderr()); diff != "" {
					t.Errorf("source(%v) sent incorrect stderr (-want, +got):\n%s", test.args, diff)
				}

				// Check file contents
				cmpFile(t, "Sourcing produced incorrect file contents", f.Name(), oschk.wantOutput)

				// Check cli changes
				for _, c := range test.clis {
					wantNew, wantChanged := oschk.wantCLIs[c.Name()]
					if wantChanged != c.Changed() {
						t.Errorf("CLI %q was incorrectly changed: want %v; got %v", c.Name(), wantChanged, c.Changed())
					}
					if wantChanged {
						if diff := cmp.Diff(wantNew, c, cmpopts.IgnoreUnexported(testCLI{})); diff != "" {
							t.Errorf("CLI %q was incorrectly updated: %v", c.Name(), diff)
						}
					}
					delete(oschk.wantCLIs, c.Name())
				}

				if len(oschk.wantCLIs) != 0 {
					for name := range oschk.wantCLIs {
						t.Errorf("Unknown CLI was supposed to change %q", name)
					}
				}
			})
		}
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
		``,
	}, "\n")
}
