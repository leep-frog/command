package sourcerer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
)

const (
	fakeFile          = "FAKE_FILE"
	fakeInputFile     = "FAKE_INPUT_FILE"
	usagePrefixString = "\n======= Command Usage ======="

	osLinux   = "linux"
	osWindows = "windows"
)

// TODO: Merge test methods
func TestGenerateBinaryNode(t *testing.T) {
	command.StubValue(t, &runtimeCaller, func(int) (uintptr, string, int, bool) {
		return 0, "/fake/source/location", 0, true
	})
	fakeGoExecutableFilePath := command.TempFile(t, "leepFrogSourcererTest")
	exeBaseName := filepath.Base(fakeGoExecutableFilePath.Name())

	type osCheck struct {
		wantStdout []string
		wantStderr []string
	}

	// We loop the OS here (and not in the test), so any underlying test data for
	// each test case is totally recreated (rather than re-using the same data
	// across tests which can be error prone and difficult to debug).
	for _, curOS := range []OS{Linux(), Windows()} {
		for _, test := range []struct {
			name              string
			clis              []CLI
			args              []string
			ignoreNosort      bool
			opts              []Option
			runtimeCallerMiss bool
			runCLI            bool
			osChecks          map[string]*osCheck
			commandStatFile   os.FileInfo
			commandStatErr    error
			wantErr           error
		}{
			{
				name: "generates source file when no CLIs",
				args: []string{"source", "leepFrogSource"},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _leepFrogSource_wrap_function {`,
							`function _custom_execute_leepFrogSource {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_leepFrogSource {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`}`, // wrap function end bracket
							`_leepFrogSource_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _leepFrogSource_wrap_function {`,
							`$_custom_autocomplete_leepFrogSource = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`}`, // wrap function end bracket
							`. _leepFrogSource_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name: "adds multiple Aliaser (singular) options at the end",
				args: []string{"source", "leepFrogSource"},
				opts: []Option{
					NewAliaser("a1", "do", "some", "stuff"),
					NewAliaser("otherAlias", "flaggable", "--args", "--at", "once"),
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _leepFrogSource_wrap_function {`,
							`function _custom_execute_leepFrogSource {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_leepFrogSource {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							strings.Join((&linux{}).aliaserGlobalAutocompleteFunction(fakeGoExecutableFilePath.Name()), "\n"),
							`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  local file="$(type do | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`  if [ -z "$file" ]; then`,
							`    echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
							`    return 1`,
							`  fi`,
							`fi`,
							``,
							``,
							`alias -- a1="do \"some\" \"stuff\""`,
							`function _custom_autocomplete_for_alias_a1 {`,
							`  _leep_frog_autocompleter "do" "some" "stuff"`,
							`}`,
							``,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_a1 -o nosort a1`,
							``,
							`local file="$(type flaggable | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  local file="$(type flaggable | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`  if [ -z "$file" ]; then`,
							`    echo Provided CLI "flaggable" is not a CLI generated with github.com/leep-frog/command`,
							`    return 1`,
							`  fi`,
							`fi`,
							``,
							``,
							`alias -- otherAlias="flaggable \"--args\" \"--at\" \"once\""`,
							`function _custom_autocomplete_for_alias_otherAlias {`,
							`  _leep_frog_autocompleter "flaggable" "--args" "--at" "once"`,
							`}`,
							``,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias`,
							``,
							`}`, // wrap function end bracket
							`_leepFrogSource_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _leepFrogSource_wrap_function {`,
							`$_custom_autocomplete_leepFrogSource = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`if (!(Test-Path alias:do) -or !(Get-Alias do | where {$_.DEFINITION -match "_custom_execute_"}).NAME) {`,
							`  throw "The CLI provided (do) is not a sourcerer-generated command"`,
							`}`,
							`function _sourcerer_alias_execute_a1 {`,
							`  $Local:functionName = "$((Get-Alias "do").DEFINITION)"`,
							`  Invoke-Expression ($Local:functionName + " " + "some" + " " + "stuff" + " " + $args)`,
							`}`,
							`$_sourcerer_alias_autocomplete_a1 = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "some" "stuff"') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							`(Get-Alias) | Where { $_.NAME -match '^a1$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias a1 _sourcerer_alias_execute_a1`,
							`Register-ArgumentCompleter -CommandName a1 -ScriptBlock $_sourcerer_alias_autocomplete_a1`,
							`if (!(Test-Path alias:flaggable) -or !(Get-Alias flaggable | where {$_.DEFINITION -match "_custom_execute_"}).NAME) {`,
							`  throw "The CLI provided (flaggable) is not a sourcerer-generated command"`,
							`}`,
							`function _sourcerer_alias_execute_otherAlias {`,
							`  $Local:functionName = "$((Get-Alias "flaggable").DEFINITION)"`,
							`  Invoke-Expression ($Local:functionName + " " + "--args" + " " + "--at" + " " + "once" + " " + $args)`,
							`}`,
							`$_sourcerer_alias_autocomplete_otherAlias = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "flaggable" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "--args" "--at" "once"') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							`(Get-Alias) | Where { $_.NAME -match '^otherAlias$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias otherAlias _sourcerer_alias_execute_otherAlias`,
							`Register-ArgumentCompleter -CommandName otherAlias -ScriptBlock $_sourcerer_alias_autocomplete_otherAlias`,
							`}`, // wrap function end bracket
							`. _leepFrogSource_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name: "only verifies each CLI once",
				args: []string{"source", "leepFrogSource"},
				opts: []Option{
					// Note the CLI in both of these is "do"
					NewAliaser("a1", "do", "some", "stuff"),
					NewAliaser("otherAlias", "do", "other", "stuff"),
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _leepFrogSource_wrap_function {`,
							`function _custom_execute_leepFrogSource {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_leepFrogSource {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							strings.Join((&linux{}).aliaserGlobalAutocompleteFunction(fakeGoExecutableFilePath.Name()), "\n"),
							`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  local file="$(type do | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`  if [ -z "$file" ]; then`,
							`    echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
							`    return 1`,
							`  fi`,
							`fi`,
							``,
							``,
							`alias -- a1="do \"some\" \"stuff\""`,
							`function _custom_autocomplete_for_alias_a1 {`,
							`  _leep_frog_autocompleter "do" "some" "stuff"`,
							`}`,
							``,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_a1 -o nosort a1`,
							``,
							// Note that we don't verify the `do` cli again here.
							// Instead, we just go straight into aliasing commands.
							`alias -- otherAlias="do \"other\" \"stuff\""`,
							`function _custom_autocomplete_for_alias_otherAlias {`,
							`  _leep_frog_autocompleter "do" "other" "stuff"`,
							`}`,
							``,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias`,
							``,
							`}`, // wrap function end bracket
							`_leepFrogSource_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _leepFrogSource_wrap_function {`,
							`$_custom_autocomplete_leepFrogSource = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`if (!(Test-Path alias:do) -or !(Get-Alias do | where {$_.DEFINITION -match "_custom_execute_"}).NAME) {`,
							`  throw "The CLI provided (do) is not a sourcerer-generated command"`,
							`}`,
							`function _sourcerer_alias_execute_a1 {`,
							`  $Local:functionName = "$((Get-Alias "do").DEFINITION)"`,
							`  Invoke-Expression ($Local:functionName + " " + "some" + " " + "stuff" + " " + $args)`,
							`}`,
							`$_sourcerer_alias_autocomplete_a1 = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "some" "stuff"') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							`(Get-Alias) | Where { $_.NAME -match '^a1$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias a1 _sourcerer_alias_execute_a1`,
							`Register-ArgumentCompleter -CommandName a1 -ScriptBlock $_sourcerer_alias_autocomplete_a1`,
							`function _sourcerer_alias_execute_otherAlias {`,
							`  $Local:functionName = "$((Get-Alias "do").DEFINITION)"`,
							`  Invoke-Expression ($Local:functionName + " " + "other" + " " + "stuff" + " " + $args)`,
							`}`,
							`$_sourcerer_alias_autocomplete_otherAlias = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "other" "stuff"') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							`(Get-Alias) | Where { $_.NAME -match '^otherAlias$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias otherAlias _sourcerer_alias_execute_otherAlias`,
							`Register-ArgumentCompleter -CommandName otherAlias -ScriptBlock $_sourcerer_alias_autocomplete_otherAlias`,
							`}`, // wrap function end bracket
							`. _leepFrogSource_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name: "adds Aliasers (plural) at the end",
				args: []string{"source", "leepFrogSource"},
				opts: []Option{
					Aliasers(map[string][]string{
						"a1":         {"do", "some", "stuff"},
						"otherAlias": {"flaggable", "--args", "--at", "once"},
					}),
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _leepFrogSource_wrap_function {`,
							`function _custom_execute_leepFrogSource {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_leepFrogSource {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							// TODO: Replace these with strings
							strings.Join((&linux{}).aliaserGlobalAutocompleteFunction(fakeGoExecutableFilePath.Name()), "\n"),
							`local file="$(type do | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  local file="$(type do | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`  if [ -z "$file" ]; then`,
							`    echo Provided CLI "do" is not a CLI generated with github.com/leep-frog/command`,
							`    return 1`,
							`  fi`,
							`fi`,
							``,
							``,
							`alias -- a1="do \"some\" \"stuff\""`,
							`function _custom_autocomplete_for_alias_a1 {`,
							`  _leep_frog_autocompleter "do" "some" "stuff"`,
							`}`,
							``,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_a1 -o nosort a1`,
							``,
							`local file="$(type flaggable | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  local file="$(type flaggable | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`  if [ -z "$file" ]; then`,
							`    echo Provided CLI "flaggable" is not a CLI generated with github.com/leep-frog/command`,
							`    return 1`,
							`  fi`,
							`fi`,
							``,
							``,
							`alias -- otherAlias="flaggable \"--args\" \"--at\" \"once\""`,
							`function _custom_autocomplete_for_alias_otherAlias {`,
							`  _leep_frog_autocompleter "flaggable" "--args" "--at" "once"`,
							`}`,
							``,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias`,
							``,
							`}`, // wrap function end bracket
							`_leepFrogSource_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _leepFrogSource_wrap_function {`,
							`$_custom_autocomplete_leepFrogSource = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`if (!(Test-Path alias:do) -or !(Get-Alias do | where {$_.DEFINITION -match "_custom_execute_"}).NAME) {`,
							`  throw "The CLI provided (do) is not a sourcerer-generated command"`,
							`}`,
							`function _sourcerer_alias_execute_a1 {`,
							`  $Local:functionName = "$((Get-Alias "do").DEFINITION)"`,
							`  Invoke-Expression ($Local:functionName + " " + "some" + " " + "stuff" + " " + $args)`,
							`}`,
							`$_sourcerer_alias_autocomplete_a1 = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "some" "stuff"') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							`(Get-Alias) | Where { $_.NAME -match '^a1$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias a1 _sourcerer_alias_execute_a1`,
							`Register-ArgumentCompleter -CommandName a1 -ScriptBlock $_sourcerer_alias_autocomplete_a1`,
							`if (!(Test-Path alias:flaggable) -or !(Get-Alias flaggable | where {$_.DEFINITION -match "_custom_execute_"}).NAME) {`,
							`  throw "The CLI provided (flaggable) is not a sourcerer-generated command"`,
							`}`,
							`function _sourcerer_alias_execute_otherAlias {`,
							`  $Local:functionName = "$((Get-Alias "flaggable").DEFINITION)"`,
							`  Invoke-Expression ($Local:functionName + " " + "--args" + " " + "--at" + " " + "once" + " " + $args)`,
							`}`,
							`$_sourcerer_alias_autocomplete_otherAlias = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "flaggable" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "--args" "--at" "once"') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							`(Get-Alias) | Where { $_.NAME -match '^otherAlias$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias otherAlias _sourcerer_alias_execute_otherAlias`,
							`Register-ArgumentCompleter -CommandName otherAlias -ScriptBlock $_sourcerer_alias_autocomplete_otherAlias`,
							`}`, // wrap function end bracket
							`. _leepFrogSource_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name: "generates source file with custom filename",
				args: []string{"source", "customOutputFile"},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _customOutputFile_wrap_function {`,
							`function _custom_execute_customOutputFile {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_customOutputFile {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`}`, // wrap function end bracket
							`_customOutputFile_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _customOutputFile_wrap_function {`,
							`$_custom_autocomplete_customOutputFile = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`}`, // wrap function end bracket
							`. _customOutputFile_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name: "generates source file with CLIs",
				args: []string{"source", "leepFrogSource"},
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _leepFrogSource_wrap_function {`,
							`function _custom_execute_leepFrogSource {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_leepFrogSource {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
							`  story`,
							`}`,
							``,
							`alias basic='o=$(mktemp) && _setup_for_basic_cli > $o && _custom_execute_leepFrogSource basic $o'`,
							"(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogSource -o nosort basic",
							`alias l='_custom_execute_leepFrogSource l'`,
							"(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogSource -o nosort l",
							"alias x='_custom_execute_leepFrogSource x'",
							"(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogSource -o nosort x",
							`}`, // wrap function end bracket
							`_leepFrogSource_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _leepFrogSource_wrap_function {`,
							`$_custom_autocomplete_leepFrogSource = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
							`  story`,
							`}`,
							``,
							`function _custom_execute_leepFrogSource_basic {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  $Local:setupTmpFile = New-TemporaryFile`,
							`  _setup_for_basic_cli > "$Local:setupTmpFile"`,
							`  Copy-Item "$Local:setupTmpFile" "$Local:setupTmpFile.txt"`,
							fmt.Sprintf(`  & %s execute "basic" $Local:tmpFile "$Local:setupTmpFile.txt" $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^basic$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias basic _custom_execute_leepFrogSource_basic`,
							`Register-ArgumentCompleter -CommandName basic -ScriptBlock $_custom_autocomplete_leepFrogSource`,
							``,
							`function _custom_execute_leepFrogSource_l {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s execute "l" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^l$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias l _custom_execute_leepFrogSource_l`,
							`Register-ArgumentCompleter -CommandName l -ScriptBlock $_custom_autocomplete_leepFrogSource`,
							``,
							`function _custom_execute_leepFrogSource_x {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s execute "x" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^x$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias x _custom_execute_leepFrogSource_x`,
							`Register-ArgumentCompleter -CommandName x -ScriptBlock $_custom_autocomplete_leepFrogSource`,
							`}`, // wrap function end bracket
							`. _leepFrogSource_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name: "generates source file with CLIs ignoring nosort",
				args: []string{"source", "leepFrogSource"},
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				ignoreNosort: true,
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _leepFrogSource_wrap_function {`,
							`function _custom_execute_leepFrogSource {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_leepFrogSource {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
							`  story`,
							`}`,
							``,
							`alias basic='o=$(mktemp) && _setup_for_basic_cli > $o && _custom_execute_leepFrogSource basic $o'`,
							"(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogSource  basic",
							`alias l='_custom_execute_leepFrogSource l'`,
							"(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogSource  l",
							"alias x='_custom_execute_leepFrogSource x'",
							"(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogSource  x",
							`}`, // wrap function end bracket
							`_leepFrogSource_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _leepFrogSource_wrap_function {`,
							`$_custom_autocomplete_leepFrogSource = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`function _setup_for_basic_cli {`,
							`  his`,
							`  story`,
							`}`,
							``,
							`function _custom_execute_leepFrogSource_basic {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							`  $Local:setupTmpFile = New-TemporaryFile`,
							`  _setup_for_basic_cli > "$Local:setupTmpFile"`,
							`  Copy-Item "$Local:setupTmpFile" "$Local:setupTmpFile.txt"`,
							fmt.Sprintf(`  & %s execute "basic" $Local:tmpFile "$Local:setupTmpFile.txt" $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^basic$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias basic _custom_execute_leepFrogSource_basic`,
							`Register-ArgumentCompleter -CommandName basic -ScriptBlock $_custom_autocomplete_leepFrogSource`,
							``,
							`function _custom_execute_leepFrogSource_l {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s execute "l" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^l$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias l _custom_execute_leepFrogSource_l`,
							`Register-ArgumentCompleter -CommandName l -ScriptBlock $_custom_autocomplete_leepFrogSource`,
							``,
							`function _custom_execute_leepFrogSource_x {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s execute "x" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^x$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias x _custom_execute_leepFrogSource_x`,
							`Register-ArgumentCompleter -CommandName x -ScriptBlock $_custom_autocomplete_leepFrogSource`,
							`}`, // wrap function end bracket
							`. _leepFrogSource_wrap_function`,
							``,
						},
					},
				},
			},
			// Test `builtin` keyword
			{
				name: "generates builtin source files",
				args: []string{"builtin", "source", "leepFrogBuiltIns"},
				// These should be ignored
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _leepFrogBuiltIns_wrap_function {`,
							`function _custom_execute_leepFrogBuiltIns {`,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  local tmpFile=$(mktemp)`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  %s builtin execute "$1" $tmpFile "${@:2}"`, fakeGoExecutableFilePath.Name()),
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
							``,
							`function _custom_autocomplete_leepFrogBuiltIns {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s builtin autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`alias aliaser='_custom_execute_leepFrogBuiltIns aliaser'`,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogBuiltIns -o nosort aliaser`,
							`alias gg='_custom_execute_leepFrogBuiltIns gg'`,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogBuiltIns -o nosort gg`,
							`alias goleep='_custom_execute_leepFrogBuiltIns goleep'`,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogBuiltIns -o nosort goleep`,
							`alias leep_debug='_custom_execute_leepFrogBuiltIns leep_debug'`,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogBuiltIns -o nosort leep_debug`,
							`alias sourcerer='_custom_execute_leepFrogBuiltIns sourcerer'`,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_leepFrogBuiltIns -o nosort sourcerer`,
							`}`, // wrap function end bracket
							`_leepFrogBuiltIns_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _leepFrogBuiltIns_wrap_function {`,
							`$_custom_autocomplete_leepFrogBuiltIns = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s builtin autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							``,
							`function _custom_execute_leepFrogBuiltIns_aliaser {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s builtin execute "aliaser" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^aliaser$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias aliaser _custom_execute_leepFrogBuiltIns_aliaser`,
							`Register-ArgumentCompleter -CommandName aliaser -ScriptBlock $_custom_autocomplete_leepFrogBuiltIns`,
							``,
							`function _custom_execute_leepFrogBuiltIns_gg {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s builtin execute "gg" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^gg$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias gg _custom_execute_leepFrogBuiltIns_gg`,
							`Register-ArgumentCompleter -CommandName gg -ScriptBlock $_custom_autocomplete_leepFrogBuiltIns`,
							``,
							`function _custom_execute_leepFrogBuiltIns_goleep {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s builtin execute "goleep" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^goleep$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias goleep _custom_execute_leepFrogBuiltIns_goleep`,
							`Register-ArgumentCompleter -CommandName goleep -ScriptBlock $_custom_autocomplete_leepFrogBuiltIns`,
							``,
							`function _custom_execute_leepFrogBuiltIns_leep_debug {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s builtin execute "leep_debug" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^leep_debug$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias leep_debug _custom_execute_leepFrogBuiltIns_leep_debug`,
							`Register-ArgumentCompleter -CommandName leep_debug -ScriptBlock $_custom_autocomplete_leepFrogBuiltIns`,
							``,
							`function _custom_execute_leepFrogBuiltIns_sourcerer {`,
							``,
							`  # tmpFile is the file to which we write ExecuteData.Executable`,
							`  $Local:tmpFile = New-TemporaryFile`,
							``,
							`  # Run the go-only code`,
							fmt.Sprintf(`  & %s builtin execute "sourcerer" $Local:tmpFile $args`, fakeGoExecutableFilePath.Name()),
							`  # Return error if failed`,
							`  If (!$?) {`,
							`    Write-Error "Go execution failed"`,
							`  } else {`,
							`    # If success, run the ExecuteData.Executable data`,
							`    Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
							`    . "$Local:tmpFile.ps1"`,
							`    If (!$?) {`,
							`      Write-Error "ExecuteData execution failed"`,
							`    }`,
							`  }`,
							`}`,
							``,
							`(Get-Alias) | Where { $_.NAME -match '^sourcerer$'} | ForEach-Object { del alias:${_} -Force }`,
							`Set-Alias sourcerer _custom_execute_leepFrogBuiltIns_sourcerer`,
							`Register-ArgumentCompleter -CommandName sourcerer -ScriptBlock $_custom_autocomplete_leepFrogBuiltIns`,
							`}`, // wrap function end bracket
							`. _leepFrogBuiltIns_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name:   "generates runCLI autocomplete source files using exeBaseName",
				args:   []string{"generate-autocomplete-setup"},
				runCLI: true,
				clis: []CLI{
					// TODO: Multiple CLIs/setup cause failure
					&testCLI{name: "basic"},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							fmt.Sprintf(`function _RunCLI_%s_autocomplete_wrap_function {`, exeBaseName),
							fmt.Sprintf(`function _custom_autocomplete_RunCLI%s {`, exeBaseName),
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete  "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							fmt.Sprintf(`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_RunCLI%s -o nosort %s`, exeBaseName, exeBaseName),
							`}`,
							fmt.Sprintf(`_RunCLI_%s_autocomplete_wrap_function`, exeBaseName),
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`function _RunCLI_%s_autocomplete_wrap_function {`, exeBaseName),
							fmt.Sprintf(`$_custom_autocomplete_RunCLI%s = {`, exeBaseName),
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete  --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							fmt.Sprintf(`Register-ArgumentCompleter -CommandName RunCLI%s -ScriptBlock $_custom_autocomplete_%s`, exeBaseName, exeBaseName),
							`}`,
							fmt.Sprintf(`. _RunCLI_%s_autocomplete_wrap_function`, exeBaseName),
							``,
						},
					},
				},
			},
			{
				name:   "generates runCLI autocomplete source files using custom alias",
				args:   []string{"generate-autocomplete-setup", "--alias", "abc"},
				runCLI: true,
				clis: []CLI{
					// TODO: Multiple CLIs/setup cause failure
					&testCLI{name: "basic"},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							`#!/bin/bash`,
							`function _RunCLI_abc_autocomplete_wrap_function {`,
							`function _custom_autocomplete_RunCLIabc {`,
							`  local tFile=$(mktemp)`,
							fmt.Sprintf(`  %s autocomplete  "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, fakeGoExecutableFilePath.Name()),
							`  local IFS=$'\n'`,
							`  COMPREPLY=( $(cat $tFile) )`,
							`  rm $tFile`,
							`}`,
							``,
							`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_RunCLIabc -o nosort abc`,
							`}`,
							`_RunCLI_abc_autocomplete_wrap_function`,
							``,
						},
					},
					osWindows: {
						wantStdout: []string{
							`function _RunCLI_abc_autocomplete_wrap_function {`,
							`$_custom_autocomplete_RunCLIabc = {`,
							`  param($wordToComplete, $commandAst, $compPoint)`,
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (& %s autocomplete  --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							`  }`,
							`}`,
							``,
							`Register-ArgumentCompleter -CommandName RunCLIabc -ScriptBlock $_custom_autocomplete_abc`,
							`}`,
							`. _RunCLI_abc_autocomplete_wrap_function`,
							``,
						},
					},
				},
			},
			{
				name:   "generates runCLI autocomplete fails if alias doesn't match regex",
				args:   []string{"generate-autocomplete-setup", "--alias", "ab c"},
				runCLI: true,
				clis: []CLI{
					// TODO: Multiple CLIs/setup cause failure
					&testCLI{name: "basic"},
				},
				wantErr: fmt.Errorf(`[MatchesRegex] value "ab c" doesn't match regex "^[a-zA-Z0-9]+$"`),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`[MatchesRegex] value "ab c" doesn't match regex "^[a-zA-Z0-9]+$"`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`[MatchesRegex] value "ab c" doesn't match regex "^[a-zA-Z0-9]+$"`,
							``,
						},
					},
				},
			},
			/* Useful for commenting out tests */
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					oschk = &osCheck{}
				}

				command.StubValue(t, &CurrentOS, curOS)
				command.StubValue(t, &commandStat, func(name string) (os.FileInfo, error) {
					return test.commandStatFile, test.commandStatErr
				})

				if test.ignoreNosort {
					command.StubValue(t, &NosortString, func() string { return "" })
				}
				o := command.NewFakeOutput()
				err := source(test.runCLI, test.clis, fakeGoExecutableFilePath.Name(), test.args, o, test.opts...)
				command.CmpError(t, "source(...)", test.wantErr, err)
				o.Close()

				// append to add a final newline (which should *always* be present).
				if diff := cmp.Diff(strings.Join(append(oschk.wantStdout, ""), "\n"), o.GetStdout()); diff != "" {
					t.Errorf("source(%v) sent incorrect data to stdout (-wamt, +got):\n%s", test.args, diff)
				}
				if diff := cmp.Diff(strings.Join(oschk.wantStderr, "\n"), o.GetStderr()); diff != "" {
					t.Errorf("source(%v) sent incorrect data to stderr (-wamt, +got):\n%s", test.args, diff)
				}
			})
		}
	}
}

func cmpFile(t *testing.T, prefix, filename string, want []string) {
	t.Helper()
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if want == nil {
		want = []string{""}
	} else {
		// In case there are any lines with multiple newline characters
		want = strings.Split(strings.Join(want, "\n"), "\n")
	}
	if diff := cmp.Diff(want, strings.Split(string(contents), "\n")); diff != "" {
		t.Errorf(prefix+": incorrect file contents (-want, +got):\n%s", diff)
	}
}

func TestSourcerer(t *testing.T) {
	selfRef := map[string]interface{}{}
	selfRef["self"] = selfRef
	f, err := os.CreateTemp("", "test-leep-frog")
	if err != nil {
		t.Fatalf("failed to create tmp file: %v", err)
	}
	fakeGoExecutableFilePath := command.TempFile(t, "leepFrogSourcerer-test")

	someCLI := &testCLI{
		name: "basic",
		processors: []command.Processor{
			command.Arg[string]("S", "desc"),
			command.ListArg[int]("IS", "ints", 2, 0),
			command.ListArg[float64]("FS", "floats", 0, command.UnboundedList),
		},
	}

	type osCheck struct {
		runtimeCallerMiss bool
		wantErr           error
		wantStdout        []string
		wantStderr        []string
		noStdoutNewline   bool
		noStderrNewline   bool
		wantCLIs          map[string]CLI
		wantOutput        []string
		wantFileContents  []string
	}

	// We loop the OS here (and not in the test), so any underlying test data for
	// each test case is totally recreated (rather than re-using the same data
	// across tests which can be error prone and difficult to debug).
	for _, curOS := range []OS{Linux(), Windows()} {
		for _, test := range []struct {
			name      string
			clis      []CLI
			args      []string
			uuids     []string
			cacheErrs []error
			runCLI    bool
			osCheck   *osCheck
			osChecks  map[string]*osCheck
			// We need to tsub osReadFile errors to be consistent across systems
			osReadFileStub        bool
			osReadFileResp        string
			osReadFileErr         error
			fakeInputFileContents []string
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
				name: "fails if unknown CLI",
				args: []string{"execute", "idk"},
				osCheck: &osCheck{
					wantStderr: []string{
						"validation for \"CLI\" failed: [MapArg] key (idk) is not in map; expected one of []",
					},
					wantErr: fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (idk) is not in map; expected one of []"),
				},
			},
			{
				name:      "fails if getCache error",
				cacheErrs: []error{fmt.Errorf("rats")},
				clis: []CLI{
					&testCLI{
						name: "basic",
					},
				},
				args: []string{"execute", "basic"},
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
				name: "fails if save error",
				clis: []CLI{
					&testCLI{
						name: "basic",
						f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							tc.MapStuff = selfRef
							tc.changed = true
							return nil
						},
					},
				},
				args: []string{"execute", "basic", fakeFile},
				osCheck: &osCheck{
					wantErr: fmt.Errorf("failed to save cli data: failed to save cli \"basic\": failed to marshal struct to json: json: unsupported value: encountered a cycle via map[string]interface {}"),
					wantStderr: []string{
						"failed to save cli data: failed to save cli \"basic\": failed to marshal struct to json: json: unsupported value: encountered a cycle via map[string]interface {}",
					},
					wantCLIs: map[string]CLI{
						"basic": &testCLI{
							MapStuff: selfRef,
						},
					},
				},
			},
			{
				name: "save fails if getCache error",
				clis: []CLI{
					&testCLI{
						name: "basic",
						f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							tc.MapStuff = selfRef
							tc.changed = true
							return nil
						},
					},
				},
				cacheErrs: []error{
					nil,                  // Successful load
					fmt.Errorf("whoops"), // Failed save
				},
				args: []string{"execute", "basic", fakeFile},
				osCheck: &osCheck{
					wantErr: fmt.Errorf("failed to save cli data: whoops"),
					wantStderr: []string{
						"failed to save cli data: whoops",
					},
					wantCLIs: map[string]CLI{
						"basic": &testCLI{
							MapStuff: selfRef,
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
				args:  []string{"execute", "basic", f.Name()},
				uuids: []string{"some-uuid"},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantOutput: []string{
							`#!/bin/bash`,
							"function _leep_execute_data_function_wrap_some_uuid {",
							"echo",
							"hello",
							"there",
							`}`,
							"_leep_execute_data_function_wrap_some_uuid",
							"",
						},
					},
					osWindows: {
						wantOutput: []string{
							"function _leep_execute_data_function_wrap_some_uuid {",
							"echo",
							"hello",
							"there",
							`}`,
							". _leep_execute_data_function_wrap_some_uuid",
							"",
						},
					},
				},
			},
			// Execute with usage tests
			{
				name: "Execute shows usage if help flag included with no other arguments",
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.Flag[string]("strFlag", 's', "strDesc"),
								command.Flag[string]("strFlag2", '2', "str2Desc"),
								command.BoolFlag("boolFlag", 'b', "bDesc"),
								command.BoolFlag("bool2Flag", command.FlagNoShortName, "b2Desc"),
							),
							command.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args: []string{"execute", "basic", fakeFile, "--help"},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"SL SL [ SL ] --bool2Flag --boolFlag|-b --strFlag|-s --strFlag2|-2",
							"",
							"Arguments:",
							"  SL: test",
							"",
							"Flags:",
							"      bool2Flag: b2Desc",
							"  [b] boolFlag: bDesc",
							"  [s] strFlag: strDesc",
							"  [2] strFlag2: str2Desc",
						}, "\n"),
					},
				},
			},
			{
				name: "Execute shows usage if help flag included with some arguments",
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.Flag[string]("strFlag", 's', "strDesc"),
								command.Flag[string]("strFlag2", '2', "str2Desc"),
								command.BoolFlag("boolFlag", 'b', "bDesc"),
								command.BoolFlag("bool2Flag", command.FlagNoShortName, "b2Desc"),
							),
							command.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args: []string{"execute", "basic", fakeFile, "--help", "un"},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"SL SL [ SL ] --bool2Flag --boolFlag|-b --strFlag|-s --strFlag2|-2",
							"",
							"Arguments:",
							"  SL: test",
							"",
							"Flags:",
							"      bool2Flag: b2Desc",
							"  [b] boolFlag: bDesc",
							"  [s] strFlag: strDesc",
							"  [2] strFlag2: str2Desc",
						}, "\n"),
					},
				},
			},
			{
				name: "Execute shows usage if all arguments provided",
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.Flag[string]("strFlag", 's', "strDesc"),
								command.Flag[string]("strFlag2", '2', "str2Desc"),
								command.BoolFlag("boolFlag", 'b', "bDesc"),
								command.BoolFlag("bool2Flag", command.FlagNoShortName, "b2Desc"),
							),
							command.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args: []string{"execute", "basic", fakeFile, "--help", "un", "deux"},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"--bool2Flag --boolFlag|-b --strFlag|-s --strFlag2|-2",
							"",
							"Flags:",
							"      bool2Flag: b2Desc",
							"  [b] boolFlag: bDesc",
							"  [s] strFlag: strDesc",
							"  [2] strFlag2: str2Desc",
						}, "\n"),
					},
				},
			},
			{
				name: "Execute shows usage if all arguments provided and some flags",
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.Flag[string]("strFlag", 's', "strDesc"),
								command.Flag[string]("strFlag2", '2', "str2Desc"),
								command.BoolFlag("boolFlag", 'b', "bDesc"),
								command.BoolFlag("bool2Flag", command.FlagNoShortName, "b2Desc"),
							),
							command.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args: []string{"execute", "basic", fakeFile, "-b", "un", "deux", "-s", "hi", "--help"},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"--bool2Flag --boolFlag|-b --strFlag|-s --strFlag2|-2",
							"",
							"Flags:",
							"      bool2Flag: b2Desc",
							"  [b] boolFlag: bDesc",
							"  [s] strFlag: strDesc",
							"  [2] strFlag2: str2Desc",
						}, "\n"),
					},
				},
			},
			{
				name: "Execute shows full usage if extra arguments provided",
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.Flag[string]("strFlag", 's', "strDesc"),
								command.Flag[string]("strFlag2", '2', "str2Desc"),
								command.BoolFlag("boolFlag", 'b', "bDesc"),
								command.BoolFlag("bool2Flag", command.FlagNoShortName, "b2Desc"),
							),
							command.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args: []string{"execute", "basic", fakeFile, "--help", "un", "deux", "trois", "quatre"},
				osCheck: &osCheck{
					// wantErr: fmt.Errorf("Unprocessed extra args: [quatre]"),
					wantStdout: []string{
						strings.Join([]string{
							"--bool2Flag --boolFlag|-b --strFlag|-s --strFlag2|-2",
							"",
							"Flags:",
							"      bool2Flag: b2Desc",
							"  [b] boolFlag: bDesc",
							"  [s] strFlag: strDesc",
							"  [2] strFlag2: str2Desc",
						}, "\n"),
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
							o.Stdoutf("stdout: %v\n", d.Values)
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
							o.Stdoutf("stdout: %v\n", d.Values)
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
						fmt.Sprintf(`stdout: map[SETUP_FILE:%s]`, command.FilepathAbs(t, "sourcerer.go")),
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
							o.Stdoutf("stdout: %v\n", d.Values)
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
						fmt.Sprintf(`stdout: map[SETUP_FILE:%s i:5]`, command.FilepathAbs(t, "sourcerer.go")),
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
						"validation for \"CLI\" failed: [MapArg] key (idk) is not in map; expected one of []\n",
					},
					wantErr:         fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (idk) is not in map; expected one of []"),
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
				name: "autocomplete handles no suggestions empty string along for completion",
				args: []string{"autocomplete", "basic", "63", "4", "cmd "},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.SimpleCompleter[string]()),
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{""},
				},
			},
			{
				name: "autocomplete handles single suggestion with SpacelssCompletion=true",
				args: []string{"autocomplete", "basic", "63", "5", "cmd h"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.CompleterFromFunc[string](func(s string, d *command.Data) (*command.Completion, error) {
								return &command.Completion{
									Suggestions:         []string{"howdy"},
									SpacelessCompletion: true,
								}, nil
							})),
						},
					},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							"howdy",
							"howdy_",
						},
					},
					osWindows: {
						wantStdout: []string{
							"howdy",
						},
					},
				},
			},
			{
				name: "autocomplete handles single suggestion with SpacelssCompletion=false",
				args: []string{"autocomplete", "basic", "63", "5", "cmd h"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.CompleterFromFunc[string](func(s string, d *command.Data) (*command.Completion, error) {
								return &command.Completion{
									Suggestions:         []string{"howdy"},
									SpacelessCompletion: false,
								}, nil
							})),
						},
					},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							"howdy",
						},
					},
					osWindows: {
						wantStdout: []string{
							"howdy ",
						},
					},
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
				name: "autocomplete does partial completion when --comp-line-file is set",
				args: []string{"autocomplete", "--comp-line-file", "basic", "63", "5", fakeInputFile},
				fakeInputFileContents: []string{
					"cmd b",
				},
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
				name: "autocomplete fails if --comp-line-file is not a file",
				args: []string{"autocomplete", "--comp-line-file", "basic", "63", "5", "not-a-file"},
				fakeInputFileContents: []string{
					"cmd b",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
						},
					},
				},
				osReadFileStub: true,
				osReadFileErr:  fmt.Errorf("read oops"),
				osCheck: &osCheck{
					wantStderr: []string{
						"Custom transformer failed: assumed COMP_LINE to be a file, but unable to read it: read oops",
					},
					wantErr: fmt.Errorf("Custom transformer failed: assumed COMP_LINE to be a file, but unable to read it: read oops"),
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
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: autocompleteSuggestions(
							"charlie",
						),
					},
					osWindows: {
						wantStdout: autocompleteSuggestions(
							"charlie ",
						),
					},
				},
			},
			{
				name: "autocomplete when COMP_POINT is equal to length of COMP_LINE",
				args: []string{"autocomplete", "basic", "63", "5", "cmd c"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							command.Arg[string]("z", "desz", command.SimpleCompleter[string]("un", "deux", "trois")),
						},
					},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: autocompleteSuggestions(
							"charlie",
						),
					},
					osWindows: {
						wantStdout: autocompleteSuggestions(
							"charlie ",
						),
					},
				},
			},
			{
				name: "autocomplete when COMP_POINT is greater than length of COMP_LINE (by 1)",
				args: []string{"autocomplete", "basic", "63", "6", "cmd c"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							command.Arg[string]("z", "desz", command.SimpleCompleter[string]("un", "deux", "trois")),
						},
					},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: autocompleteSuggestions(
							"deux",
							"trois",
							"un",
						),
					},
					osWindows: {
						wantStdout: autocompleteSuggestions(
							"deux",
							"trois",
							"un",
						),
					},
				},
			},
			{
				name: "autocomplete when COMP_POINT is greater than length of COMP_LINE (by 2)",
				args: []string{"autocomplete", "basic", "63", "7", "cmd c"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							command.Arg[string]("z", "desz", command.SimpleCompleter[string]("un", "deux", "trois")),
						},
					},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: autocompleteSuggestions(
							"deux",
							"trois",
							"un",
						),
					},
					osWindows: {
						wantStdout: autocompleteSuggestions(
							"deux",
							"trois",
							"un",
						),
					},
				},
			},
			{
				name: "autocomplete when COMP_POINT is greater than length of COMP_LINE with quoted space (by 1)",
				args: []string{"autocomplete", "basic", "63", "7", `cmd "c`},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.SimpleCompleter[string]("c alpha", "c bravo", "c charlie", "cheese", "baker")),
							command.Arg[string]("z", "desz", command.SimpleCompleter[string]("un", "deux", "trois")),
						},
					},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: autocompleteSuggestions(
							`"c alpha"`,
							`"c bravo"`,
							`"c charlie"`,
						),
					},
					osWindows: {
						wantStdout: autocompleteSuggestions(
							`"c alpha"`,
							`"c bravo"`,
							`"c charlie"`,
						),
					},
				},
			},
			{
				name: "autocomplete when COMP_POINT is greater than length of COMP_LINE with quoted space (by 2)",
				args: []string{"autocomplete", "basic", "63", "8", `cmd "c`},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("s", "desc", command.SimpleCompleter[string]("c alpha", "c bravo", "c charlie", "brown", "baker")),
							command.Arg[string]("z", "desz", command.SimpleCompleter[string]("un", "deux", "trois")),
						},
					},
				},
				osCheck: &osCheck{
					// No completions equivalent
					wantStdout: []string{""},
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
				name: "usage handles too many args with no errors",
				args: []string{"usage", "uec", "b", "un", "deux"},
				clis: []CLI{&usageErrCLI{}},
				osCheck: &osCheck{
					wantStdout: []string{""},
				},
			},
			{
				name: "usage handles too many args with flags",
				args: []string{"usage", "basic", "b", "un", "deux", "--sf", "hey"},
				clis: []CLI{&testCLI{
					name: "basic",
					processors: []command.Processor{command.FlagProcessor(
						command.BoolFlag("bf", 'b', "desc"),
						command.Flag[string]("sf", 's', "desc string"),
					)},
				}},
				osCheck: &osCheck{
					wantStdout: []string{
						"--bf|-b --sf|-s",
						"",
						"Flags:",
						"  [b] bf: desc",
						"  [s] sf: desc string",
					},
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
				args: []string{"usage", someCLI.name},
				osCheck: &osCheck{
					wantStdout: []string{strings.Join([]string{
						"S IS IS [ FS ... ]",
						"",
						"Arguments:",
						"  FS: floats",
						"  IS: ints",
						"  S: desc",
						"",
					}, "\n")},
					noStdoutNewline: true,
				},
			},
			{
				name: "usage handles non-usage error",
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.Arg[string]("S", "desc", command.Contains("ABC")),
						},
					},
				},
				args: []string{"usage", someCLI.name, "DEF"},
				osCheck: &osCheck{
					wantErr: fmt.Errorf(`validation for "S" failed: [Contains] value doesn't contain substring "ABC"`),
					wantStderr: []string{
						`validation for "S" failed: [Contains] value doesn't contain substring "ABC"`,
					},
				},
			},
			// Builtin command tests
			{
				name: "builtin usage doesn't work with provided CLIs",
				clis: []CLI{someCLI},
				args: []string{"builtin", "usage", someCLI.name},
				osCheck: &osCheck{
					wantStderr: []string{
						"validation for \"CLI\" failed: [MapArg] key (basic) is not in map; expected one of [aliaser gg goleep leep_debug sourcerer]",
					},
					wantErr:         fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (basic) is not in map; expected one of [aliaser gg goleep leep_debug sourcerer]"),
					noStdoutNewline: true,
				},
			},
			{
				name: "builtin usage works with builtin CLIs",
				clis: []CLI{someCLI},
				args: []string{"builtin", "usage", "gg"},
				osCheck: &osCheck{
					wantStdout: []string{
						"gg updates go packages from the github.com/leep-frog repository",
						"PACKAGE [ PACKAGE ... ]",
						"",
						"Arguments:",
						"  PACKAGE: Package name",
					},
				},
			},
			{
				name: "builtin execute doesn't work with provided CLIs",
				clis: []CLI{someCLI},
				args: []string{"builtin", "execute", someCLI.name},
				osCheck: &osCheck{
					wantStderr: []string{
						"validation for \"CLI\" failed: [MapArg] key (basic) is not in map; expected one of [aliaser gg goleep leep_debug sourcerer]",
					},
					wantErr:         fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (basic) is not in map; expected one of [aliaser gg goleep leep_debug sourcerer]"),
					noStdoutNewline: true,
				},
			},
			{
				name: "builtin execute works with builtin CLIs",
				clis: []CLI{someCLI},
				args: []string{"builtin", "execute", "aliaser", fakeFile, "bleh", "bloop", "er"},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantFileContents: []string{
							"function _leep_frog_autocompleter {",
							"  local tFile=$(mktemp)",
							fmt.Sprintf(`  %s autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`, fakeGoExecutableFilePath.Name()),
							"  local IFS='",
							"';",
							"  COMPREPLY=( $(cat $tFile) )",
							"  rm $tFile",
							"}",
							"",
							`local file="$(type bloop | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`if [ -z "$file" ]; then`,
							`  local file="$(type bloop | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
							`  if [ -z "$file" ]; then`,
							`    echo Provided CLI "bloop" is not a CLI generated with github.com/leep-frog/command`,
							"    return 1",
							`  fi`,
							"fi",
							"",
							"",
							`alias -- bleh="bloop \"er\""`,
							"function _custom_autocomplete_for_alias_bleh {",
							`  _leep_frog_autocompleter "bloop" "er"`,
							"}",
							"",
							"(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_bleh -o nosort bleh",
							"",
						},
					},
					osWindows: {
						wantFileContents: []string{
							`if (!(Test-Path alias:bloop) -or !(Get-Alias bloop | where {$_.DEFINITION -match "_custom_execute_"}).NAME) {`,
							`  throw "The CLI provided (bloop) is not a sourcerer-generated command"`,
							"}",
							"function _sourcerer_alias_execute_bleh {",
							`  $Local:functionName = "$((Get-Alias "bloop").DEFINITION)"`,
							`  Invoke-Expression ($Local:functionName + " " + "er" + " " + $args)`,
							"}",
							"$_sourcerer_alias_autocomplete_bleh = {",
							"  param($wordToComplete, $commandAst, $compPoint)",
							`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
							`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
							fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "bloop" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "er"') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
							`    "$_"`,
							"  }",
							"}",
							"(Get-Alias) | Where { $_.NAME -match '^bleh$'} | ForEach-Object { del alias:${_} -Force }",
							"Set-Alias bleh _sourcerer_alias_execute_bleh",
							"Register-ArgumentCompleter -CommandName bleh -ScriptBlock $_sourcerer_alias_autocomplete_bleh",
						},
					},
				},
			},
			{
				name: "builtin autocomplete doesn't work with provided CLIs",
				clis: []CLI{someCLI},
				args: []string{"builtin", "autocomplete", someCLI.name},
				osCheck: &osCheck{
					wantStderr: []string{
						"validation for \"CLI\" failed: [MapArg] key (basic) is not in map; expected one of [aliaser gg goleep leep_debug sourcerer]",
					},
					wantErr:         fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (basic) is not in map; expected one of [aliaser gg goleep leep_debug sourcerer]"),
					noStdoutNewline: true,
				},
			},
			{
				name: "builtin autocomplete works with builtin CLIs",
				clis: []CLI{someCLI},
				args: []string{"builtin", "autocomplete", "gg", "63", "5", "cmd c"},
				osCheck: &osCheck{
					wantStdout: []string{
						"cd",
						"command",
					},
				},
			},
			{
				name: "fails if runtimeCaller error",
				osCheck: &osCheck{
					runtimeCallerMiss: true,
					wantErr:           fmt.Errorf("failed to get source location: failed to fetch runtime.Caller"),
					wantStderr: []string{
						"failed to get source location: failed to fetch runtime.Caller",
					},
				},
			},
			// runCLI tests (should all result in errors)
			{
				name:   "runCLI fails if no CLIs provided",
				args:   []string{"builtin"},
				runCLI: true,
				clis:   []CLI{},
				osCheck: &osCheck{
					wantStderr: []string{
						"0 CLIs provided with RunCLI(); expected exactly one",
					},
					wantErr: fmt.Errorf("0 CLIs provided with RunCLI(); expected exactly one"),
				},
			},
			{
				name:   "runCLI fails if nil CLI provided",
				args:   []string{"builtin"},
				runCLI: true,
				clis:   []CLI{nil},
				osCheck: &osCheck{
					wantStderr: []string{
						"nil CLI provided at index 0",
					},
					wantErr: fmt.Errorf("nil CLI provided at index 0"),
				},
			},
			{
				name:   "runCLI fails if nil CLI in non-runCLI",
				args:   []string{"builtin"},
				runCLI: false,
				clis: []CLI{
					ToCLI("zero", nil),
					ToCLI("one", nil),
					nil,
					ToCLI("three", nil),
				},
				osCheck: &osCheck{
					wantStderr: []string{
						"nil CLI provided at index 2",
					},
					wantErr: fmt.Errorf("nil CLI provided at index 2"),
				},
			},
			{
				name:   "runCLI fails if multiple CLIs provided",
				args:   []string{"builtin"},
				runCLI: true,
				clis: []CLI{
					&testCLI{name: "basic"},
					&testCLI{name: "other"},
				},
				osCheck: &osCheck{
					wantStderr: []string{
						"2 CLIs provided with RunCLI(); expected exactly one",
					},
					wantErr: fmt.Errorf("2 CLIs provided with RunCLI(); expected exactly one"),
				},
			},
			{
				name:   "runCLI fails if provided with builtin",
				args:   []string{"builtin"},
				runCLI: true,
				clis: []CLI{
					&testCLI{name: "basic"},
				},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [builtin]",
					},
					wantErr: fmt.Errorf("Unprocessed extra args: [builtin]"),
				},
			},
			{
				name:   "runCLI fails if provided with source",
				args:   []string{"source"},
				runCLI: true,
				clis: []CLI{
					&testCLI{name: "basic"},
				},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [source]",
					},
					wantErr: fmt.Errorf("Unprocessed extra args: [source]"),
				},
			},
			{
				name:   "runCLI fails if provided with execute",
				args:   []string{"execute"},
				runCLI: true,
				clis: []CLI{
					&testCLI{name: "basic"},
				},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [execute]",
					},
					wantErr: fmt.Errorf("Unprocessed extra args: [execute]"),
				},
			},
			{
				name:   "runCLI fails if provided with usage",
				args:   []string{"usage"},
				runCLI: true,
				clis: []CLI{
					&testCLI{name: "basic"},
				},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [usage]",
					},
					wantErr: fmt.Errorf("Unprocessed extra args: [usage]"),
				},
			},
			{
				name:   "runCLI works with autocomplete",
				args:   []string{"autocomplete", "63", "4", "cmd "},
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.ListArg[string]("SS", "desc", 0, command.UnboundedList, command.SimpleCompleter[[]string]("abc", "def", "ghi")),
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						"abc",
						"def",
						"ghi",
					},
				},
			},
			{
				name:   "runCLI works with autocomplete and passthrough args",
				args:   []string{"autocomplete", "63", "4", "cmd ", "abc", "ghi"},
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.ListArg[string]("SS", "desc", 0, command.UnboundedList, command.SimpleDistinctCompleter[[]string]("abc", "def", "ghi")),
						},
					},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							"def",
						},
					},
					osWindows: {
						wantStdout: []string{
							"def ",
						},
					},
				},
			},
			{
				name:   "runCLI execution works (no `execute` branching keyword required)",
				args:   []string{"un", "--count", "6", "deux", "-b", "trois"},
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.BoolFlag("b", 'b', "B desc"),
								command.Flag[int]("count", command.FlagNoShortName, "Cnt desc"),
							),
							command.ListArg[string]("SS", "desc", 0, command.UnboundedList),
							&command.ExecutorProcessor{func(o command.Output, d *command.Data) error {
								o.Stdoutln(d.Values)
								return nil
							}},
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						"map[SS:[un deux trois] b:true count:6]",
					},
				},
			},
			{
				name:   "runCLI execution fails if extra args",
				args:   []string{"un", "--count", "6", "deux", "-b", "trois", "bleh"},
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.BoolFlag("b", 'b', "B desc"),
								command.Flag[int]("count", command.FlagNoShortName, "Cnt desc"),
							),
							command.ListArg[string]("SS", "desc", 0, 3),
							&command.ExecutorProcessor{func(o command.Output, d *command.Data) error {
								o.Stdoutln(d.Values)
								return nil
							}},
						},
					},
				},
				osCheck: &osCheck{
					wantStderr: []string{
						`Unprocessed extra args: [bleh]`,
						``,
						`======= Command Usage =======`,
						`[ SS SS SS ] --b|-b --count`,
						``,
						`Arguments:`,
						`  SS: desc`,
						``,
						`Flags:`,
						`  [b] b: B desc`,
						`      count: Cnt desc`,
					},
					wantErr: fmt.Errorf(`Unprocessed extra args: [bleh]`),
				},
			},
			{
				name:   "runCLI execution with help flag works",
				args:   []string{"--help"},
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.BoolFlag("b", 'b', "B desc"),
								command.Flag[int]("count", command.FlagNoShortName, "Cnt desc"),
							),
							command.ListArg[string]("SS", "desc", 0, 3),
							&command.ExecutorProcessor{func(o command.Output, d *command.Data) error {
								o.Stdoutln(d.Values)
								return nil
							}},
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						`[ SS SS SS ] --b|-b --count`,
						``,
						`Arguments:`,
						`  SS: desc`,
						``,
						`Flags:`,
						`  [b] b: B desc`,
						`      count: Cnt desc`,
					},
				},
			},
			{
				name:   "runCLI execution with help flag and some args works",
				args:   []string{"--help", "un", "--b"},
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.FlagProcessor(
								command.BoolFlag("b", 'b', "B desc"),
								command.Flag[int]("count", command.FlagNoShortName, "Cnt desc"),
							),
							command.ListArg[string]("SS", "desc", 0, 3),
							&command.ExecutorProcessor{func(o command.Output, d *command.Data) error {
								o.Stdoutln(d.Values)
								return nil
							}},
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						`--b|-b --count`,
						``,
						`Flags:`,
						`  [b] b: B desc`,
						`      count: Cnt desc`,
					},
				},
			},
			{
				name:   "runCLI fails if Setup isn't nil",
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name:  "basic",
						setup: []string{"s"},
					},
				},
				osCheck: &osCheck{
					wantErr: fmt.Errorf("Setup() must be empty when running via RunCLI() (supported only via Source())"),
					wantStderr: []string{
						"Setup() must be empty when running via RunCLI() (supported only via Source())",
					},
				},
			},
			{
				name:   "runCLI fails if ExecuteData is returned",
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
								ed.Executable = append(ed.Executable, "echo hi")
								return nil
							}, nil),
						},
					},
				},
				osCheck: &osCheck{
					wantErr: fmt.Errorf("ExecuteData.Executable is not supported via RunCLI() (use Source() instead)"),
					wantStderr: []string{
						"ExecuteData.Executable is not supported via RunCLI() (use Source() instead)",
					},
				},
			},
			// GoExecutableFilePath tests
			{
				name:   "RunCLI() gets goExecutableFilePath",
				runCLI: true,
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							ExecutableFileGetProcessor(),
						},
						f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							o.Stdoutln(d.Values)
							return nil
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						`map[GO_EXECUTABLE_FILE:osArgs-at-zero]`,
					},
				},
			},
			/* Useful for commenting out tests */
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				command.StubValue(t, &externalGoExecutableFilePath, "osArgs-at-zero")
				if test.osReadFileStub {
					command.StubValue(t, &osReadFile, func(b string) ([]byte, error) {
						return []byte(test.osReadFileResp), test.osReadFileErr
					})
				}
				command.StubValue(t, &CurrentOS, curOS)
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					oschk = test.osCheck
				}

				var uuidIdx int
				command.StubValue(t, &getUuid, func() string {
					r := test.uuids[uuidIdx]
					uuidIdx++
					return r
				})

				command.StubValue(t, &runtimeCaller, func(n int) (uintptr, string, int, bool) {
					return 0, "/fake/source/location/main.go", 0, !oschk.runtimeCallerMiss
				})

				if err := os.WriteFile(f.Name(), nil, 0644); err != nil {
					t.Fatalf("failed to clear file: %v", err)
				}

				fake := command.TempFile(t, "leepFrogSourcerer-test")
				for i, s := range test.args {
					if s == fakeFile {
						test.args[i] = fake.Name()
					}
				}

				if len(test.fakeInputFileContents) > 0 {
					fakeInput := command.TempFile(t, "leepFrogSourcerer-test")
					for i, s := range test.args {
						if s == fakeInputFile {
							test.args[i] = fakeInput.Name()
						}
					}
					if err := os.WriteFile(fakeInput.Name(), []byte(strings.Join(test.fakeInputFileContents, "\n")), 0644); err != nil {
						t.Fatalf("failed to write fake input file: %v", err)
					}
				}

				// Stub out real cache
				cash := cache.NewTestCache(t)
				command.StubValue(t, &getCache, func() (*cache.Cache, error) {
					if len(test.cacheErrs) == 0 {
						return cash, nil
					}
					e := test.cacheErrs[0]
					test.cacheErrs = test.cacheErrs[1:]
					return cash, e
				})

				// Run source command
				o := command.NewFakeOutput()
				err = source(test.runCLI, test.clis, fakeGoExecutableFilePath.Name(), test.args, o)
				command.CmpError(t, fmt.Sprintf("source(%v)", test.args), oschk.wantErr, err)
				o.Close()

				// Verify executeData file contains expected contents
				cmpFile(t, "Output file contents", fake.Name(), oschk.wantFileContents)

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

				if uuidIdx != len(test.uuids) {
					t.Errorf("Unnecessary uuid stubs. %d stubs, but only %d calls", len(test.uuids), uuidIdx)
				}

				// Check file contents
				cmpFile(t, "Sourcing produced incorrect file contents", f.Name(), oschk.wantOutput)

				// Check cli changes
				for _, c := range test.clis {
					if c == nil {
						continue
					}
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
	Stuff    string
	MapStuff map[string]interface{}
}

func (tc *testCLI) Name() string {
	return tc.name
}

func (tc *testCLI) UnmarshalJSON([]byte) error { return nil }
func (tc *testCLI) Node() command.Node {
	return command.SerialNodes(append(tc.processors, command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		if tc.f != nil {
			return tc.f(tc, i, o, d, ed)
		}
		return nil
	}, nil))...)
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
		`┓`,
		`┃`,
		`┣━━ a [ A_SL ]`,
		`┃`,
		`┗━━ b B_SL`,
		``,
		`Arguments:`,
		`  A_SL: str list`,
		`  B_SL: str list`,
		``,
		`Symbols:`,
		command.BranchDescWithoutDefault,
		``,
	}, "\n")
}

type fakeFileInfo struct {
	isDir bool
}

func (*fakeFileInfo) Name() string       { return "" }
func (*fakeFileInfo) Size() int64        { return 0 }
func (*fakeFileInfo) Mode() os.FileMode  { return 0 }
func (*fakeFileInfo) ModTime() time.Time { return time.Now() }
func (ffi *fakeFileInfo) IsDir() bool    { return ffi.isDir }
func (*fakeFileInfo) Sys() interface{}   { return nil }

var (
	fakeFI = &fakeFileInfo{}
)

type badUsage struct {
	command.Processor
	err error
}

func (b *badUsage) Usage(*command.Input, *command.Data, *command.Usage) error {
	return b.err
}
