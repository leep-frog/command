package sourcerer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/cache"
	"github.com/leep-frog/command/cache/cachetest"
	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/stubs"
	"github.com/leep-frog/command/internal/testutil"
)

const (
	fakeFile          = "FAKE_FILE"
	fakeInputFile     = "FAKE_INPUT_FILE"
	fakeFileContents  = "fake file contents"
	usagePrefixString = "\n======= Command Usage ======="

	osLinux   = "linux"
	osWindows = "windows"
)

func TestGenerateBinaryNode(t *testing.T) {
	testutil.StubValue(t, &runtimeCaller, func(int) (uintptr, string, int, bool) {
		return 0, testutil.FilepathAbs(t, "/", "fake", "source", "location"), 0, true
	})
	fakeGoExecutableFilePath := testutil.TempFile(t, "leepFrogSourcererTest")
	exeBaseName := filepath.Base(fakeGoExecutableFilePath.Name())
	_ = exeBaseName

	type osWriteFileArgs struct {
		File     string
		Contents []string
		FileMode os.FileMode
	}

	type osCheck struct {
		wantOsWriteFiles []*osWriteFileArgs
		wantStdout       []string
		wantStderr       []string
	}

	// We loop the OS here (and not in the test), so any underlying test data for
	// each test case is totally recreated (rather than re-using the same data
	// across tests which can be error prone and difficult to debug).
	for _, curOS := range []OS{Linux(), Windows()} {
		for _, test := range []struct {
			name              string
			cliTargetName     string
			clis              []CLI
			args              []string
			ignoreNosort      bool
			opts              []Option
			runtimeCallerMiss bool
			runCLI            bool
			osChecks          map[string]*osCheck
			osMkdirAllErrs    []error
			env               map[string]string
			wantErr           error

			wantOSReadFile  []string
			wantMkdirAll    []string
			osReadFileErr   error
			osWriteFileErrs []error
		}{
			{
				name:    "errors when empty target name",
				args:    []string{"source"},
				wantErr: fmt.Errorf(`Invalid target name: [MatchesRegex] value "" doesn't match regex "^[a-zA-Z0-9]+$"`),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`Invalid target name: [MatchesRegex] value "" doesn't match regex "^[a-zA-Z0-9]+$"`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`Invalid target name: [MatchesRegex] value "" doesn't match regex "^[a-zA-Z0-9]+$"`,
							``,
						},
					},
				},
			},
			{
				name:          "errors when invalid target name",
				cliTargetName: "some target name",
				args:          []string{"source"},
				wantErr:       fmt.Errorf(`Invalid target name: [MatchesRegex] value "some target name" doesn't match regex "^[a-zA-Z0-9]+$"`),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`Invalid target name: [MatchesRegex] value "some target name" doesn't match regex "^[a-zA-Z0-9]+$"`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`Invalid target name: [MatchesRegex] value "some target name" doesn't match regex "^[a-zA-Z0-9]+$"`,
							``,
						},
					},
				},
			},
			{
				name:          "errors when COMMAND_CLI_OUTPUT_DIR is not set",
				cliTargetName: "leepFrogSource",
				args:          []string{"source"},
				wantErr:       fmt.Errorf(`Environment variable COMMAND_CLI_OUTPUT_DIR is not set`),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`Environment variable COMMAND_CLI_OUTPUT_DIR is not set`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`Environment variable COMMAND_CLI_OUTPUT_DIR is not set`,
							``,
						},
					},
				},
			},
			{
				name:          "errors when output folder does not exist",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "/some-path",
				},
				args:    []string{"source"},
				wantErr: fmt.Errorf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] file %q does not exist`, testutil.FilepathAbs(t, "/some-path")),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							fmt.Sprintf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] file %q does not exist`, testutil.FilepathAbs(t, "/some-path")),
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							fmt.Sprintf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] file %q does not exist`, testutil.FilepathAbs(t, "/some-path")),
							``,
						},
					},
				},
			},
			{
				name:          "errors when output folder is not a directory",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "sourcerer.go",
				},
				args:    []string{"source"},
				wantErr: fmt.Errorf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] argument %q is a file`, testutil.FilepathAbs(t, "sourcerer.go")),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							fmt.Sprintf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] argument %q is a file`, testutil.FilepathAbs(t, "sourcerer.go")),
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							fmt.Sprintf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] argument %q is a file`, testutil.FilepathAbs(t, "sourcerer.go")),
							``,
						},
					},
				},
			},
			{
				name:          "fails if fail to make artifacts directory",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:           []string{"source"},
				osMkdirAllErrs: []error{fmt.Errorf("mkdir-all oops")},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts")},
				wantErr:        fmt.Errorf(`failed to make artifacts directory: mkdir-all oops`),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`failed to make artifacts directory: mkdir-all oops`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`failed to make artifacts directory: mkdir-all oops`,
							``,
						},
					},
				},
			},
			{
				name:          "fails if fail to make sourcerers directory",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:           []string{"source"},
				osMkdirAllErrs: []error{nil, fmt.Errorf("mkdir-all rats")},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				wantErr:        fmt.Errorf(`failed to make sourcerers directory: mkdir-all rats`),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`failed to make sourcerers directory: mkdir-all rats`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`failed to make sourcerers directory: mkdir-all rats`,
							``,
						},
					},
				},
			},
			{
				name:          "fails when osReadFileErr",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:           []string{"source"},
				osReadFileErr:  fmt.Errorf("read oops"),
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantErr:        fmt.Errorf(`failed to read executable file: read oops`),
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`failed to read executable file: read oops`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`failed to read executable file: read oops`,
							``,
						},
					},
				},
			},
			{
				name:          "fails when osWriteFileErr for copying binary file",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:            []string{"source"},
				wantOSReadFile:  []string{fakeGoExecutableFilePath.Name()},
				osWriteFileErrs: []error{fmt.Errorf("write binary whoops")},
				wantErr:         fmt.Errorf(`failed to copy executable file: write binary whoops`),
				wantMkdirAll:    []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								Contents: []string{fakeFileContents},
								FileMode: 0744,
							},
						},
						wantStderr: []string{
							`failed to copy executable file: write binary whoops`,
							``,
						},
					},
					osWindows: {
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								Contents: []string{fakeFileContents},
								FileMode: 0744,
							},
						},
						wantStderr: []string{
							`failed to copy executable file: write binary whoops`,
							``,
						},
					},
				},
			},
			{
				name:          "fails when osWriteFileErr for creating sourceable file",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:            []string{"source"},
				wantOSReadFile:  []string{fakeGoExecutableFilePath.Name()},
				osWriteFileErrs: []error{nil, fmt.Errorf("write sourceable whoops")},
				wantErr:         fmt.Errorf(`failed to write sourceable file contents: write sourceable whoops`),
				wantMkdirAll:    []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
						},
						wantStderr: []string{
							`failed to write sourceable file contents: write sourceable whoops`,
							``,
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
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
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
						},
						wantStderr: []string{
							`failed to write sourceable file contents: write sourceable whoops`,
							``,
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								Contents: []string{fakeFileContents},
								FileMode: 0744,
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			{
				name:          "generates source file when no CLIs",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:           []string{"source"},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `leepFrogSource_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
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
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			{
				name:          "hides output when quiet flag is provided",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:           []string{"source", "--quiet"},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							color.Apply("Successfully generated CLI data for leepFrogSource!", color.Green, color.Bold),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
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
						},
					},
					osWindows: {
						wantStdout: []string{
							color.Apply("Successfully generated CLI data for leepFrogSource!", color.Green, color.Bold),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			{
				name:          "adds multiple Aliaser (singular) options at the end",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"source"},
				opts: []Option{
					NewAliaser("a1", "do", "some", "stuff"),
					NewAliaser("otherAlias", "flaggable", "--args", "--at", "once"),
				},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `leepFrogSource_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  local IFS=$'\n'`,
									`  COMPREPLY=( $(cat $tFile) )`,
									`  rm $tFile`,
									`}`,
									``,
									`function _leep_frog_autocompleter {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  local IFS='`,
									`';`,
									`  COMPREPLY=( $(cat $tFile) )`,
									`  rm $tFile`,
									`}`,
									``,
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
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_for_alias_a1 -o nosort a1 ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
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
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									``,
									`}`, // wrap function end bracket
									`_leepFrogSource_wrap_function`,
									``,
								},
							},
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "some" "stuff"') | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "flaggable" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "--args" "--at" "once"') | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			{
				name:          "only verifies each CLI once",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"source"},
				opts: []Option{
					// Note the CLI in both of these is "do"
					NewAliaser("a1", "do", "some", "stuff"),
					NewAliaser("otherAlias", "do", "other", "stuff"),
				},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `leepFrogSource_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  local IFS=$'\n'`,
									`  COMPREPLY=( $(cat $tFile) )`,
									`  rm $tFile`,
									`}`,
									``,
									`function _leep_frog_autocompleter {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  local IFS='`,
									`';`,
									`  COMPREPLY=( $(cat $tFile) )`,
									`  rm $tFile`,
									`}`,
									``,
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
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_for_alias_a1 -o nosort a1 ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									``,
									// Note that we don't verify the `do` cli again here.
									// Instead, we just go straight into aliasing commands.
									`alias -- otherAlias="do \"other\" \"stuff\""`,
									`function _custom_autocomplete_for_alias_otherAlias {`,
									`  _leep_frog_autocompleter "do" "other" "stuff"`,
									`}`,
									``,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									``,
									`}`, // wrap function end bracket
									`_leepFrogSource_wrap_function`,
									``,
								},
							},
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "some" "stuff"') | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "other" "stuff"') | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			{
				name:          "adds Aliasers (plural) at the end",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"source"},
				opts: []Option{
					Aliasers(map[string][]string{
						"a1":         {"do", "some", "stuff"},
						"otherAlias": {"flaggable", "--args", "--at", "once"},
					}),
				},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `leepFrogSource_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  local IFS=$'\n'`,
									`  COMPREPLY=( $(cat $tFile) )`,
									`  rm $tFile`,
									`}`,
									``,
									`function _leep_frog_autocompleter {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  local IFS='`,
									`';`,
									`  COMPREPLY=( $(cat $tFile) )`,
									`  rm $tFile`,
									`}`,
									``,
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
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_for_alias_a1 -o nosort a1 ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
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
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_for_alias_otherAlias -o nosort otherAlias ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									``,
									`}`, // wrap function end bracket
									`_leepFrogSource_wrap_function`,
									``,
								},
							},
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "do" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "some" "stuff"') | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "flaggable" --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile "--args" "--at" "once"') | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			{
				name:          "generates source file with custom filename",
				cliTargetName: "customOutputFile",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:           []string{"source"},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "customOutputFile")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "customOutputFile_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `customOutputFile_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_customOutputFile_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "customOutputFile"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "customOutputFile_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _customOutputFile_wrap_function {`,
									`function _custom_execute_customOutputFile {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "customOutputFile")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_customOutputFile {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "customOutputFile")),
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
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "customOutputFile.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "customOutputFile_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "customOutputFile_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_customOutputFile_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "customOutputFile.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "customOutputFile_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _customOutputFile_wrap_function {`,
									`$_custom_autocomplete_customOutputFile = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "customOutputFile.exe")),
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
				},
			},
			{
				name:          "generates source file with CLIs",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"source"},
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `leepFrogSource_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
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
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogSource -o nosort basic ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`alias l='_custom_execute_leepFrogSource l'`,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogSource -o nosort l ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									"alias x='_custom_execute_leepFrogSource x'",
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogSource -o nosort x ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`}`, // wrap function end bracket
									`_leepFrogSource_wrap_function`,
									``,
								},
							},
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  & %s execute "basic" $Local:tmpFile "$Local:setupTmpFile.txt" $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  & %s execute "l" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  & %s execute "x" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			{
				name:          "generates source file with CLIs ignoring nosort",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"source"},
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				ignoreNosort:   true,
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `leepFrogSource_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogSource_wrap_function {`,
									`function _custom_execute_leepFrogSource {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogSource {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource")),
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
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogSource  basic ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`alias l='_custom_execute_leepFrogSource l'`,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogSource  l ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									"alias x='_custom_execute_leepFrogSource x'",
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogSource  x ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`}`, // wrap function end bracket
									`_leepFrogSource_wrap_function`,
									``,
								},
							},
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogSource_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogSource_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogSource_wrap_function {`,
									`$_custom_autocomplete_leepFrogSource = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  & %s execute "basic" $Local:tmpFile "$Local:setupTmpFile.txt" $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  & %s execute "l" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
									fmt.Sprintf(`  & %s execute "x" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogSource.exe")),
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
				},
			},
			// Test `builtin` keyword
			{
				name:          "builtin error bubbles up",
				cliTargetName: "myCustomBuiltIns",
				args:          []string{"builtin", "source"},
				// These should be ignored
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				wantErr: fmt.Errorf(`Environment variable COMMAND_CLI_OUTPUT_DIR is not set`),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							`Environment variable COMMAND_CLI_OUTPUT_DIR is not set`,
							``,
						},
					},
					osWindows: {
						wantStderr: []string{
							`Environment variable COMMAND_CLI_OUTPUT_DIR is not set`,
							``,
						},
					},
				},
			},
			{
				name:          "generates builtin source files",
				cliTargetName: "myCustomBuiltIns", // Note: this should be overriden in output below
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"builtin", "source"},
				// These should be ignored
				clis: []CLI{
					ToCLI("x", nil),
					ToCLI("l", nil),
					&testCLI{name: "basic", setup: []string{"his", "story"}},
				},
				wantOSReadFile: []string{fakeGoExecutableFilePath.Name()},
				wantMkdirAll:   []string{testutil.FilepathAbs(t, "cli-output-dir", "artifacts"), testutil.FilepathAbs(t, "cli-output-dir", "sourcerers")},
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogCLIBuiltIns_loader.sh")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`source %q`, testutil.FilepathAbs(t, `cli-output-dir`, `sourcerers`, `leepFrogCLIBuiltIns_loader.sh`)),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogCLIBuiltIns_CLIs() {`,
								`  pushd . > /dev/null`,
								fmt.Sprintf(`  cd %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . builtin source`,
								`  popd . > /dev/null`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogCLIBuiltIns_loader.sh"),
								FileMode: 0644,
								Contents: []string{
									`#!/bin/bash`,
									`function _leepFrogCLIBuiltIns_wrap_function {`,
									`function _custom_execute_leepFrogCLIBuiltIns {`,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  local tmpFile=$(mktemp)`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  %s builtin execute "$1" $tmpFile "${@:2}"`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns")),
									`  # Return the error code if go code terminated with an error`,
									`  local errorCode=$?`,
									`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
									``,
									`  # Otherwise, run the ExecuteData.Executable data`,
									`  source $tmpFile`,
									`  local errorCode=$?`,
									`  if [ -z "$COMMAND_CLI_DEBUG" ]; then`,
									`    rm $tmpFile`,
									`  else`,
									`    echo $tmpFile`,
									`  fi`,
									`  return $errorCode`,
									`}`,
									``,
									`function _custom_autocomplete_leepFrogCLIBuiltIns {`,
									`  local tFile=$(mktemp)`,
									fmt.Sprintf(`  %s builtin autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns")),
									`  local IFS=$'\n'`,
									`  COMPREPLY=( $(cat $tFile) )`,
									`  rm $tFile`,
									`}`,
									``,
									`alias aliaser='_custom_execute_leepFrogCLIBuiltIns aliaser'`,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogCLIBuiltIns -o nosort aliaser ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`alias gg='_custom_execute_leepFrogCLIBuiltIns gg'`,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogCLIBuiltIns -o nosort gg ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`alias goleep='_custom_execute_leepFrogCLIBuiltIns goleep'`,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogCLIBuiltIns -o nosort goleep ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`alias leep_debug='_custom_execute_leepFrogCLIBuiltIns leep_debug'`,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogCLIBuiltIns -o nosort leep_debug ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`alias sourcerer='_custom_execute_leepFrogCLIBuiltIns sourcerer'`,
									`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_leepFrogCLIBuiltIns -o nosort sourcerer ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
									`}`, // wrap function end bracket
									`_leepFrogCLIBuiltIns_wrap_function`,
									``,
								},
							},
						},
					},
					osWindows: {
						wantStdout: []string{
							fmt.Sprintf(`Binary file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe")),
							fmt.Sprintf(`Sourceable file created: %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogCLIBuiltIns_loader.ps1")),
							``,
							color.Apply(`All steps have completed successfully!`, color.Green, color.Bold),
							``,
							`Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:`,
							``,
							color.Apply(strings.Join([]string{
								`# Load all of your CLIs`,
								fmt.Sprintf(`. %q`, testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogCLIBuiltIns_loader.ps1")),
								``,
								`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
								`function _regenerate_leepFrogCLIBuiltIns_CLIs() {`,
								`  Push-Location`,
								fmt.Sprintf(`  Set-Location %q`, testutil.FilepathAbs(t, "/", "fake", "source")),
								`  go run . builtin source`,
								`  Pop-Location`,
								`}`,
							}, "\n"), color.Blue),
						},
						wantOsWriteFiles: []*osWriteFileArgs{
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe"),
								FileMode: 0744,
								Contents: []string{fakeFileContents},
							},
							{
								File:     testutil.FilepathAbs(t, "cli-output-dir", "sourcerers", "leepFrogCLIBuiltIns_loader.ps1"),
								FileMode: 0644,
								Contents: []string{
									`function _leepFrogCLIBuiltIns_wrap_function {`,
									`$_custom_autocomplete_leepFrogCLIBuiltIns = {`,
									`  param($wordToComplete, $commandAst, $compPoint)`,
									`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
									`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
									fmt.Sprintf(`  (& %s builtin autocomplete ($commandAst.CommandElements | Select-Object -first 1) --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe")),
									`    "$_"`,
									`  }`,
									`}`,
									``,
									``,
									`function _custom_execute_leepFrogCLIBuiltIns_aliaser {`,
									``,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  $Local:tmpFile = New-TemporaryFile`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  & %s builtin execute "aliaser" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe")),
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
									`Set-Alias aliaser _custom_execute_leepFrogCLIBuiltIns_aliaser`,
									`Register-ArgumentCompleter -CommandName aliaser -ScriptBlock $_custom_autocomplete_leepFrogCLIBuiltIns`,
									``,
									`function _custom_execute_leepFrogCLIBuiltIns_gg {`,
									``,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  $Local:tmpFile = New-TemporaryFile`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  & %s builtin execute "gg" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe")),
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
									`Set-Alias gg _custom_execute_leepFrogCLIBuiltIns_gg`,
									`Register-ArgumentCompleter -CommandName gg -ScriptBlock $_custom_autocomplete_leepFrogCLIBuiltIns`,
									``,
									`function _custom_execute_leepFrogCLIBuiltIns_goleep {`,
									``,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  $Local:tmpFile = New-TemporaryFile`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  & %s builtin execute "goleep" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe")),
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
									`Set-Alias goleep _custom_execute_leepFrogCLIBuiltIns_goleep`,
									`Register-ArgumentCompleter -CommandName goleep -ScriptBlock $_custom_autocomplete_leepFrogCLIBuiltIns`,
									``,
									`function _custom_execute_leepFrogCLIBuiltIns_leep_debug {`,
									``,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  $Local:tmpFile = New-TemporaryFile`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  & %s builtin execute "leep_debug" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe")),
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
									`Set-Alias leep_debug _custom_execute_leepFrogCLIBuiltIns_leep_debug`,
									`Register-ArgumentCompleter -CommandName leep_debug -ScriptBlock $_custom_autocomplete_leepFrogCLIBuiltIns`,
									``,
									`function _custom_execute_leepFrogCLIBuiltIns_sourcerer {`,
									``,
									`  # tmpFile is the file to which we write ExecuteData.Executable`,
									`  $Local:tmpFile = New-TemporaryFile`,
									``,
									`  # Run the go-only code`,
									fmt.Sprintf(`  & %s builtin execute "sourcerer" $Local:tmpFile $args`, testutil.FilepathAbs(t, "cli-output-dir", "artifacts", "leepFrogCLIBuiltIns.exe")),
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
									`Set-Alias sourcerer _custom_execute_leepFrogCLIBuiltIns_sourcerer`,
									`Register-ArgumentCompleter -CommandName sourcerer -ScriptBlock $_custom_autocomplete_leepFrogCLIBuiltIns`,
									`}`, // wrap function end bracket
									`. _leepFrogCLIBuiltIns_wrap_function`,
									``,
								},
							},
						},
					},
				},
			},
			{
				name:          "generate-autocomplete-setup with runCLI fails if multiple CLIs",
				cliTargetName: "leepFrogSource",
				args:          []string{"generate-autocomplete-setup"},
				runCLI:        true,
				clis: []CLI{
					&testCLI{name: "basic"},
					&testCLI{name: "other"},
				},
				wantErr: fmt.Errorf("2 CLIs provided with RunCLI(); expected exactly one"),
				osChecks: map[string]*osCheck{
					osLinux: {
						wantStderr: []string{
							"2 CLIs provided with RunCLI(); expected exactly one",
							"",
						},
					},
					osWindows: {
						wantStderr: []string{
							"2 CLIs provided with RunCLI(); expected exactly one",
							"",
						},
					},
				},
			},
			{
				name:          "generates runCLI autocomplete source files using exeBaseName",
				cliTargetName: "leepFrogSource",
				args:          []string{"generate-autocomplete-setup"},
				runCLI:        true,
				clis: []CLI{
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
							fmt.Sprintf(`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_RunCLI%s -o nosort %s ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`, exeBaseName, exeBaseName),
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
				name:          "generates runCLI autocomplete source files using custom alias",
				cliTargetName: "leepFrogSource",
				args:          []string{"generate-autocomplete-setup", "--alias", "abc"},
				runCLI:        true,
				clis: []CLI{
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
							`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_RunCLIabc -o nosort abc ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
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
				name:          "generates runCLI autocomplete fails if alias doesn't match regex",
				args:          []string{"generate-autocomplete-setup", "--alias", "ab c"},
				runCLI:        true,
				cliTargetName: "testCLIs",
				clis: []CLI{
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

				testutil.StubValue(t, &CurrentOS, curOS)
				var gotMkdirAllNames []string
				testutil.StubValue(t, &osMkdirAll, func(name string, perm fs.FileMode) error {
					gotMkdirAllNames = append(gotMkdirAllNames, name)
					if diff := cmp.Diff(fs.FileMode(0777), perm); diff != "" {
						t.Errorf("source(%v) sent incorrect permission value to os.MkdirAll (-want, +got):\n%s", test.args, diff)
					}
					if len(test.osMkdirAllErrs) > 0 {
						err := test.osMkdirAllErrs[0]
						test.osMkdirAllErrs = test.osMkdirAllErrs[1:]
						return err
					}
					return nil
				})

				var gotOSReadFile []string
				testutil.StubValue(t, &osReadFile, func(f string) ([]byte, error) {
					gotOSReadFile = append(gotOSReadFile, f)
					return []byte(fakeFileContents), test.osReadFileErr
				})
				var gotOSWriteFile []*osWriteFileArgs
				testutil.StubValue(t, &osWriteFile, func(f string, b []byte, fm fs.FileMode) error {
					gotOSWriteFile = append(gotOSWriteFile, &osWriteFileArgs{f, strings.Split(string(b), "\n"), fm})

					var err error
					if len(test.osWriteFileErrs) > 0 {
						err = test.osWriteFileErrs[0]
						test.osWriteFileErrs = test.osWriteFileErrs[1:]
					}
					return err
				})

				if test.ignoreNosort {
					testutil.StubValue(t, &NosortString, func() string { return "" })
				}
				o := commandtest.NewOutput()
				stubs.StubEnv(t, test.env)
				err := source(test.runCLI, test.cliTargetName, test.clis, fakeGoExecutableFilePath.Name(), test.args, o, test.opts...)
				testutil.CmpError(t, "source(...)", test.wantErr, err)
				o.Close()

				// append to add a final newline (which should *always* be present).
				if diff := cmp.Diff(strings.Join(append(oschk.wantStdout, ""), "\n"), o.GetStdout()); diff != "" {
					t.Errorf("source(%v) sent incorrect data to stdout (-want, +got):\n%s", test.args, diff)
				}
				if diff := cmp.Diff(strings.Join(oschk.wantStderr, "\n"), o.GetStderr()); diff != "" {
					t.Errorf("source(%v) sent incorrect data to stderr (-want, +got):\n%s", test.args, diff)
				}

				if diff := cmp.Diff(test.wantOSReadFile, gotOSReadFile); diff != "" {
					t.Errorf("source(%v) executed incorrect os.ReadFile commands (-want, +got):\n%s", test.args, diff)
				}

				if diff := cmp.Diff(oschk.wantOsWriteFiles, gotOSWriteFile); diff != "" {
					t.Errorf("source(%v) executed incorrect os.WriteFile commands (-want, +got):\n%s", test.args, diff)
				}

				if diff := cmp.Diff(test.wantMkdirAll, gotMkdirAllNames); diff != "" {
					t.Errorf("source(%v) executed incorrect os.MkdirAll commands (-want, +got):\n%s", test.args, diff)
				}
			})
		}
	}
}

func cmpFile(t *testing.T, prefix, filename string, want []string) {
	t.Helper()
	contents, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read File: %v", err)
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
	fakeGoExecutableFilePath := testutil.TempFile(t, "leepFrogSourcerer-test")

	someCLI := &testCLI{
		name: "basic",
		processors: []command.Processor{
			commander.Arg[string]("S", "desc"),
			commander.ListArg[int]("IS", "ints", 2, 0),
			commander.ListArg[float64]("FS", "floats", 0, command.UnboundedList),
		},
	}

	_ = someCLI

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
			name              string
			cliTargetName     string
			clis              []CLI
			args              []string
			env               map[string]string
			uuids             []string
			cacheErrs         []error
			wantGetCacheCalls []string
			runCLI            bool
			wantPanic         any
			osCheck           *osCheck
			osChecks          map[string]*osCheck
			// We need to stub osReadFile errors to be consistent across systems
			osReadFileStub        bool
			osReadFileResp        string
			osReadFileErr         error
			osExecutableErr       error
			fakeInputFileContents []string
		}{
			{
				name:          "fails if invalid target name",
				cliTargetName: "w t f",
				osCheck: &osCheck{
					wantStderr: []string{
						`Invalid target name: [MatchesRegex] value "w t f" doesn't match regex "^[a-zA-Z0-9]+$"`,
					},
					wantErr: fmt.Errorf(`Invalid target name: [MatchesRegex] value "w t f" doesn't match regex "^[a-zA-Z0-9]+$"`),
				},
			},
			{
				name:          "fails if invalid command branch",
				cliTargetName: "leepFrogSource",
				args:          []string{"wizardry", "stuff"},
				osCheck: &osCheck{
					wantStderr: []string{
						"Unprocessed extra args: [wizardry stuff]",
					},
					wantErr: fmt.Errorf("Unprocessed extra args: [wizardry stuff]"),
				},
			},
			// Execute tests
			{
				name:          "fails if no cli arg",
				cliTargetName: "leepFrogSource",
				args:          []string{"execute"},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "CLI" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
				},
			},
			{
				name:          "fails if no cli arg other",
				cliTargetName: "leepFrogSource",
				args:          []string{},
				osCheck: &osCheck{
					wantStderr: []string{
						"echo \"Executing a sourcerer.CLI directly through `go run` is tricky. Either generate a CLI or use the `goleep` command to directly run the file.\"",
					},
					wantErr: fmt.Errorf("echo \"Executing a sourcerer.CLI directly through `go run` is tricky. Either generate a CLI or use the `goleep` command to directly run the file.\""),
				},
			},
			{
				name:          "fails if unknown CLI",
				cliTargetName: "leepFrogSource",
				args:          []string{"execute", "idk"},
				osCheck: &osCheck{
					wantStderr: []string{
						"validation for \"CLI\" failed: [MapArg] key (idk) is not in map; expected one of []",
					},
					wantErr: fmt.Errorf("validation for \"CLI\" failed: [MapArg] key (idk) is not in map; expected one of []"),
				},
			},
			{
				name:          "fails if environment variable is not set",
				cliTargetName: "leepFrogSource",
				cacheErrs:     []error{fmt.Errorf("rats")},
				clis: []CLI{
					&testCLI{
						name: "basic",
					},
				},
				args: []string{"execute", "basic"},
				osCheck: &osCheck{
					wantErr:    fmt.Errorf("Environment variable COMMAND_CLI_OUTPUT_DIR is not set"),
					wantStderr: []string{"Environment variable COMMAND_CLI_OUTPUT_DIR is not set"},
				},
			},
			{
				name:          "fails if environment variable does not exist",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "some-path",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
					},
				},
				args: []string{"execute", "basic"},
				osCheck: &osCheck{
					wantErr:    fmt.Errorf("Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] file %q does not exist", testutil.FilepathAbs(t, "some-path")),
					wantStderr: []string{fmt.Sprintf("Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] file %q does not exist", testutil.FilepathAbs(t, "some-path"))},
				},
			},
			{
				name:          "fails if environment variable is not a directory",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "sourcerer_test.go",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
					},
				},
				args: []string{"execute", "basic"},
				osCheck: &osCheck{
					wantErr:    fmt.Errorf("Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] argument %q is a file", testutil.FilepathAbs(t, "sourcerer_test.go")),
					wantStderr: []string{fmt.Sprintf("Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] argument %q is a file", testutil.FilepathAbs(t, "sourcerer_test.go"))},
				},
			},
			{
				name:          "fails if getCache error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				cacheErrs: []error{fmt.Errorf("rats")},
				clis: []CLI{
					&testCLI{
						name: "basic",
					},
				},
				args:              []string{"execute", "basic"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantErr:    fmt.Errorf("failed to load cache from environment variable: rats"),
					wantStderr: []string{"failed to load cache from environment variable: rats"},
				},
			},
			{
				name:          "properly executes CLI",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				args:              []string{"execute", "basic", fakeFile},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{"Output:"},
				},
			},
			{
				name:          "handles processing error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							return o.Stderrln("oops")
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStderr: []string{"oops"},
					wantErr:    fmt.Errorf("oops"),
				},
			},
			{
				name:          "properly passes arguments to CLI",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.ListArg[string]("sl", "test desc", 1, 4),
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
				args:              []string{"execute", "basic", fakeFile, "un", "deux", "trois"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{
						"Output:",
						`sl: [un deux trois]`,
					},
				},
			},
			{
				name:          "properly passes extra arguments to CLI",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name:       "basic",
						processors: []command.Processor{commander.ListArg[string]("SL", "test", 1, 1)},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "un", "deux", "trois", "quatre"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "properly marks CLI as changed and saves",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				wantGetCacheCalls: []string{
					// load
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
					// save
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
				},
				osCheck: &osCheck{
					wantCLIs: map[string]CLI{
						"basic": &testCLI{
							Stuff: "things",
						},
					},
				},
			},
			{
				name: "saves even if execution error",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				cliTargetName: "leepFrogSource",
				clis: []CLI{
					&testCLI{
						name: "basic",
						f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							tc.Stuff = "things"
							tc.changed = true
							return o.Stderrln("whoops")
						},
					},
				},
				args: []string{"execute", "basic", fakeFile},
				wantGetCacheCalls: []string{
					// load
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
					// save
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
				},
				osCheck: &osCheck{
					wantCLIs: map[string]CLI{
						"basic": &testCLI{
							Stuff: "things",
						},
					},
					wantStderr: []string{
						"whoops",
					},
					wantErr: fmt.Errorf("whoops"),
				},
			},
			{
				name:          "fails if save error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				wantGetCacheCalls: []string{
					// load
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
					// save
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
				},
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
				name:          "save fails if getCache error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				wantGetCacheCalls: []string{
					// load
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
					// save
					testutil.FilepathAbs(t, "cli-output-dir", "cache"),
				},
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
				name:          "writes execute data to file",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executable = []string{"echo", "hello", "there"}
							return nil
						},
					},
				},
				args:              []string{"execute", "basic", f.Name()},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantOutput: []string{
						"echo",
						"hello",
						"there",
					},
				},
			},
			{
				name:          "writes function wrapped execute data to file",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				args:              []string{"execute", "basic", f.Name()},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				uuids:             []string{"some-uuid"},
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
				name:          "Execute shows usage if help flag included with no other arguments",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.Flag[string]("strFlag", 's', "strDesc"),
								commander.Flag[string]("strFlag2", '2', "str2Desc"),
								commander.BoolFlag("boolFlag", 'b', "bDesc"),
								commander.BoolFlag("bool2Flag", commander.FlagNoShortName, "b2Desc"),
							),
							commander.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "--help"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"SL SL [ SL ] --strFlag|-s STRFLAG --strFlag2|-2 STRFLAG2 --boolFlag|-b --bool2Flag",
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
				name:          "Usage handles usage error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							&commander.BranchNode{
								Branches: map[string]command.Node{
									"first": commander.SerialNodes(
										commander.ListArg[string]("F", "f", 1, 10),
									),
									"second": commander.SerialNodes(
										commander.ListArg[string]("S", "s", 2, 0),
									),
								},
							},
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "--help", "third"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantErr: fmt.Errorf("Branching argument must be one of [first second]"),
					wantStderr: []string{
						"Branching argument must be one of [first second]",
						"",
						"======= Command Usage =======",
						"┓",
						"┣━━ first F [ F F F F F F F F F F ]",
						"┃",
						"┗━━ second S S",
						"",
						"Arguments:",
						"  F: f",
						"  S: s",
					},
				},
			},
			{
				name:          "Usage handles non-usage error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
								return fmt.Errorf("rats")
							}),
							&commander.BranchNode{
								Branches: map[string]command.Node{
									"first": commander.SerialNodes(
										commander.ListArg[string]("F", "f", 1, 10),
									),
									"second": commander.SerialNodes(
										commander.ListArg[string]("S", "s", 2, 0),
									),
								},
							},
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "--help"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantErr:    fmt.Errorf("rats"),
					wantStderr: []string{"rats"},
				},
			},
			{
				name:          "Execute shows usage if help flag included with some arguments",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.Flag[string]("strFlag", 's', "strDesc"),
								commander.Flag[string]("strFlag2", '2', "str2Desc"),
								commander.BoolFlag("boolFlag", 'b', "bDesc"),
								commander.BoolFlag("bool2Flag", commander.FlagNoShortName, "b2Desc"),
							),
							commander.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "--help", "un"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"SL [ SL ] --strFlag|-s STRFLAG --strFlag2|-2 STRFLAG2 --boolFlag|-b --bool2Flag",
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
				name:          "Execute shows usage if all arguments provided",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.Flag[string]("strFlag", 's', "strDesc"),
								commander.Flag[string]("strFlag2", '2', "str2Desc"),
								commander.BoolFlag("boolFlag", 'b', "bDesc"),
								commander.BoolFlag("bool2Flag", commander.FlagNoShortName, "b2Desc"),
							),
							commander.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "--help", "un", "deux"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"--strFlag|-s STRFLAG --strFlag2|-2 STRFLAG2 --boolFlag|-b --bool2Flag",
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
				name:          "Execute shows usage if all arguments provided and some flags",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.Flag[string]("strFlag", 's', "strDesc"),
								commander.Flag[string]("strFlag2", '2', "str2Desc"),
								commander.BoolFlag("boolFlag", 'b', "bDesc"),
								commander.BoolFlag("bool2Flag", commander.FlagNoShortName, "b2Desc"),
							),
							commander.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "-b", "un", "deux", "-s", "hi", "--help"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{
						strings.Join([]string{
							"--strFlag2|-2 STRFLAG2 --bool2Flag",
							"",
							"Flags:",
							"      bool2Flag: b2Desc",
							"  [2] strFlag2: str2Desc",
						}, "\n"),
					},
				},
			},
			{
				name:          "Execute shows full usage if extra arguments provided",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.Flag[string]("strFlag", 's', "strDesc"),
								commander.Flag[string]("strFlag2", '2', "str2Desc"),
								commander.BoolFlag("boolFlag", 'b', "bDesc"),
								commander.BoolFlag("bool2Flag", commander.FlagNoShortName, "b2Desc"),
							),
							commander.ListArg[string]("SL", "test", 2, 1),
						},
					},
				},
				args:              []string{"execute", "basic", fakeFile, "--help", "un", "deux", "trois", "quatre"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					// wantErr: fmt.Errorf("Unprocessed extra args: [quatre]"),
					wantStdout: []string{
						strings.Join([]string{
							"--strFlag|-s STRFLAG --strFlag2|-2 STRFLAG2 --boolFlag|-b --bool2Flag",
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
				name:          "SetupArg node is automatically added as required arg",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantErr: fmt.Errorf(`Argument "SETUP_FILE" requires at least 1 argument, got 0`),
					wantStderr: []string{
						`Argument "SETUP_FILE" requires at least 1 argument, got 0`,
					},
				},
			},
			{
				name:          "SetupArg is properly populated",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{
						// false is for data.complexecute
						fmt.Sprintf(`stdout: map[SETUP_FILE:%s]`, testutil.FilepathAbs(t, "sourcerer.go")),
					},
				},
			},
			{
				name:          "args after SetupArg are properly populated",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis: []CLI{
					&testCLI{
						name:  "basic",
						setup: []string{"his", "story"},
						processors: []command.Processor{
							commander.Arg[int]("i", "desc"),
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
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{
						// false is for data.complexecute
						fmt.Sprintf(`stdout: map[SETUP_FILE:%s i:5]`, testutil.FilepathAbs(t, "sourcerer.go")),
					},
				},
			},
			// Usage printing tests
			{
				name:          "prints command usage for missing branch error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis:              []CLI{&usageErrCLI{}},
				args:              []string{"execute", "uec", fakeFile},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "prints command usage for bad branch arg error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis:              []CLI{&usageErrCLI{}},
				args:              []string{"execute", "uec", fakeFile, "uh"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "prints command usage for missing args error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis:              []CLI{&usageErrCLI{}},
				args:              []string{"execute", "uec", fakeFile, "b"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "prints command usage for missing args error",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				clis:              []CLI{&usageErrCLI{}},
				args:              []string{"execute", "uec", fakeFile, "a", "un", "deux", "trois"},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "lists none",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{ListBranchName},
				osCheck: &osCheck{
					wantStdout: []string{""},
				},
			},
			{
				name:          "lists clis",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				name:          "autocomplete requires cli name",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"autocomplete"},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "CLI" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "CLI" requires at least 1 argument, got 0`),
				},
			},
			{
				name:          "autocomplete requires comp_type",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:              []string{"autocomplete", "uec"},
				clis:              []CLI{&usageErrCLI{}},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "COMP_TYPE" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "COMP_TYPE" requires at least 1 argument, got 0`),
				},
			},
			{
				name:          "autocomplete requires comp_point",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:              []string{"autocomplete", "uec", "63"},
				clis:              []CLI{&usageErrCLI{}},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "COMP_POINT" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "COMP_POINT" requires at least 1 argument, got 0`),
				},
			},
			{
				name:          "autocomplete requires comp_line",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:              []string{"autocomplete", "uec", "63", "2"},
				clis:              []CLI{&usageErrCLI{}},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStderr: []string{
						`Argument "COMP_LINE" requires at least 1 argument, got 0`,
					},
					wantErr: fmt.Errorf(`Argument "COMP_LINE" requires at least 1 argument, got 0`),
				},
			},
			{
				name:          "autocomplete doesn't require passthrough args",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:              []string{"autocomplete", "basic", "63", "0", "h"},
				clis:              []CLI{&testCLI{name: "basic"}},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "autocomplete re-prints comp line",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:              []string{"autocomplete", "basic", "63", "10", "hello ther"},
				clis:              []CLI{&testCLI{name: "basic"}},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "autocomplete doesn't re-print comp line if different COMP_TYPE",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args:              []string{"autocomplete", "basic", "64", "10", "hello ther"},
				clis:              []CLI{&testCLI{name: "basic"}},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "autocomplete requires valid cli",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				name:          "autocomplete passes empty string along for completion",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"autocomplete", "basic", "63", "4", "cmd "},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie")),
						},
					},
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: autocompleteSuggestions(
						"alpha",
						"bravo",
						"charlie",
					),
				},
			},
			{
				name:          "autocomplete handles no suggestions empty string along for completion",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"autocomplete", "basic", "63", "4", "cmd "},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]()),
						},
					},
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				osCheck: &osCheck{
					wantStdout: []string{""},
				},
			},
			{
				name:          "autocomplete handles single suggestion with SpacelssCompletion=true",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"autocomplete", "basic", "63", "5", "cmd h"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.CompleterFromFunc[string](func(s string, d *command.Data) (*command.Completion, error) {
								return &command.Completion{
									Suggestions:         []string{"howdy"},
									SpacelessCompletion: true,
								}, nil
							})),
						},
					},
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "autocomplete handles single suggestion with SpacelssCompletion=false",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"autocomplete", "basic", "63", "5", "cmd h"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.CompleterFromFunc[string](func(s string, d *command.Data) (*command.Completion, error) {
								return &command.Completion{
									Suggestions:         []string{"howdy"},
									SpacelessCompletion: false,
								}, nil
							})),
						},
					},
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "autocomplete doesn't complete passthrough args",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"autocomplete", "basic", "63", "4", "cmd ", "al"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.ListArg[string]("s", "desc", 0, command.UnboundedList, commander.SimpleCompleter[[]string]("alpha", "bravo", "charlie")),
						},
					},
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args: []string{"autocomplete", "basic", "0", "", "al"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.ListArg[string]()
							commander.Arg[string]("s", "desc",
								&commander.Completer[string]{
									Fetcher: commander.SimpleFetcher(func(t string, d *command.Data) (*command.Completion, error) {
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
				name:          "autocomplete does partial completion",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "5", "cmd b"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
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
				name:          "autocomplete does partial completion when --comp-line-file is set",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "--comp-line-file", "basic", "63", "5", fakeInputFile},
				fakeInputFileContents: []string{
					"cmd b",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
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
				name:          "autocomplete fails if --comp-line-file is not a file",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "--comp-line-file", "basic", "63", "5", "not-a-file"},
				fakeInputFileContents: []string{
					"cmd b",
				},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
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
				name:          "autocomplete goes along processors",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "6", "cmd a "},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							commander.Arg[string]("z", "desz", commander.SimpleCompleter[string]("un", "deux", "trois")),
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
				name:          "autocomplete does earlier completion if cpoint is smaller",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "5", "cmd c "},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							commander.Arg[string]("z", "desz", commander.SimpleCompleter[string]("un", "deux", "trois")),
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
				name:          "autocomplete when COMP_POINT is equal to length of COMP_LINE",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "5", "cmd c"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							commander.Arg[string]("z", "desz", commander.SimpleCompleter[string]("un", "deux", "trois")),
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
				name:          "autocomplete when COMP_POINT is greater than length of COMP_LINE (by 1)",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "6", "cmd c"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							commander.Arg[string]("z", "desz", commander.SimpleCompleter[string]("un", "deux", "trois")),
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
				name:          "autocomplete when COMP_POINT is greater than length of COMP_LINE (by 2)",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "7", "cmd c"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("alpha", "bravo", "charlie", "brown", "baker")),
							commander.Arg[string]("z", "desz", commander.SimpleCompleter[string]("un", "deux", "trois")),
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
				name:          "autocomplete when COMP_POINT is greater than length of COMP_LINE with quoted space (by 1)",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "7", `cmd "c`},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("c alpha", "c bravo", "c charlie", "cheese", "baker")),
							commander.Arg[string]("z", "desz", commander.SimpleCompleter[string]("un", "deux", "trois")),
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
				name:          "autocomplete when COMP_POINT is greater than length of COMP_LINE with quoted space (by 2)",
				cliTargetName: "leepFrogSource",
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "basic", "63", "8", `cmd "c`},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.Arg[string]("s", "desc", commander.SimpleCompleter[string]("c alpha", "c bravo", "c charlie", "brown", "baker")),
							commander.Arg[string]("z", "desz", commander.SimpleCompleter[string]("un", "deux", "trois")),
						},
					},
				},
				osCheck: &osCheck{
					// No completions equivalent
					wantStdout: []string{""},
				},
			},
			// Builtin command tests
			{
				name:          "builtin execute doesn't work with provided CLIs",
				cliTargetName: "leepFrogSource",
				clis:          []CLI{someCLI},
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				name:          "builtin execute works with builtin CLIs",
				cliTargetName: "leepFrogSource",
				clis:          []CLI{someCLI},
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"builtin", "execute", "aliaser", fakeFile, "bleh", "bloop", "er"},
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
							`{ { type complete > /dev/null 2>&1 ; } && complete -F _custom_autocomplete_for_alias_bleh -o nosort bleh ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`,
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
				name:          "builtin autocomplete doesn't work with provided CLIs",
				cliTargetName: "leepFrogSource",
				clis:          []CLI{someCLI},
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
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
				name:          "builtin autocomplete works with builtin CLIs",
				cliTargetName: "leepFrogSource",
				clis:          []CLI{someCLI},
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"builtin", "autocomplete", "gg", "63", "5", "cmd c"},
				osCheck: &osCheck{
					wantStdout: []string{
						"cd",
						"command",
					},
				},
			},
			{
				name:          "fails if runtimeCaller error",
				cliTargetName: "leepFrogSource",
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
				name:          "runCLI fails if no CLIs provided",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"builtin"},
				clis: []CLI{},
				osCheck: &osCheck{
					wantStderr: []string{
						"0 CLIs provided with RunCLI(); expected exactly one",
					},
					wantErr: fmt.Errorf("0 CLIs provided with RunCLI(); expected exactly one"),
				},
			},
			{
				name:          "runCLI fails if nil CLI provided",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"builtin"},
				clis: []CLI{nil},
				osCheck: &osCheck{
					wantStderr: []string{
						"nil CLI provided at index 0",
					},
					wantErr: fmt.Errorf("nil CLI provided at index 0"),
				},
			},
			{
				name:          "runCLI fails if nil CLI in non-runCLI",
				cliTargetName: "leepFrogSource",
				runCLI:        false,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"builtin"},
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
				name:          "runCLI fails if multiple CLIs provided",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				args: []string{"builtin"},
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
				name:          "runCLI fails if invalid CLI root dir",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "ummmm",
				},
				clis: []CLI{
					&testCLI{name: "basic"},
				},
				osCheck: &osCheck{
					wantStderr: []string{
						fmt.Sprintf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] file %q does not exist`, testutil.FilepathAbs(t, "ummmm")),
					},
					wantErr: fmt.Errorf(`Invalid value for environment variable COMMAND_CLI_OUTPUT_DIR: [IsDir] file %q does not exist`, testutil.FilepathAbs(t, "ummmm")),
				},
			},
			{
				name:          "runCLI fails if provided with builtin",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"builtin"},
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
				name:          "runCLI fails if provided with source",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"source"},
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
				name:          "runCLI fails if provided with execute",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"execute"},
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
				name:          "runCLI fails if provided with usage",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"usage"},
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
				name:          "runCLI works with autocomplete",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "63", "4", "cmd "},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.ListArg[string]("SS", "desc", 0, command.UnboundedList, commander.SimpleCompleter[[]string]("abc", "def", "ghi")),
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
				name:          "runCLI works with autocomplete and passthrough args",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"autocomplete", "63", "4", "cmd ", "abc", "ghi"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.ListArg[string]("SS", "desc", 0, command.UnboundedList, commander.SimpleDistinctCompleter[[]string]("abc", "def", "ghi")),
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
				name:          "runCLI execution works (no `execute` branching keyword required)",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"un", "--count", "6", "deux", "-b", "trois"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.BoolFlag("b", 'b', "B desc"),
								commander.Flag[int]("count", commander.FlagNoShortName, "Cnt desc"),
							),
							commander.ListArg[string]("SS", "desc", 0, command.UnboundedList),
							&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
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
				name:          "runCLI execution works with other CLI root dir",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					// This is different from previous test
					RootDirectoryEnvVar: filepath.Join("cmd", "test_goleeper"),
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cmd", "test_goleeper", "cache")},
				args:              []string{"un", "--count", "6", "deux", "-b", "trois"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.BoolFlag("b", 'b', "B desc"),
								commander.Flag[int]("count", commander.FlagNoShortName, "Cnt desc"),
							),
							commander.ListArg[string]("SS", "desc", 0, command.UnboundedList),
							&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
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
				name:          "runCLI execution fails if extra args",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				args:              []string{"un", "--count", "6", "deux", "-b", "trois", "bleh"},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.BoolFlag("b", 'b', "B desc"),
								commander.Flag[int]("count", commander.FlagNoShortName, "Cnt desc"),
							),
							commander.ListArg[string]("SS", "desc", 0, 3),
							&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
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
						`[ SS SS SS ] --b|-b --count COUNT`,
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
				name:          "runCLI execution with help flag works",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				args:          []string{"--help"},
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.BoolFlag("b", 'b', "B desc"),
								commander.Flag[int]("count", commander.FlagNoShortName, "Cnt desc"),
							),
							commander.ListArg[string]("SS", "desc", 0, 3),
							&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
								o.Stdoutln(d.Values)
								return nil
							}},
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						`[ SS SS SS ] --b|-b --count COUNT`,
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
				name:          "runCLI execution with help flag and some args works",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				args:          []string{"--help", "un", "--b"},
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.FlagProcessor(
								commander.BoolFlag("b", 'b', "B desc"),
								commander.Flag[int]("count", commander.FlagNoShortName, "Cnt desc"),
							),
							commander.ListArg[string]("SS", "desc", 0, 3),
							&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
								o.Stdoutln(d.Values)
								return nil
							}},
						},
					},
				},
				osCheck: &osCheck{
					wantStdout: []string{
						`--count COUNT`,
						``,
						`Flags:`,
						`      count: Cnt desc`,
					},
				},
			},
			{
				name:          "runCLI fails if Setup isn't nil",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				name:          "runCLI fails if ExecuteData is returned",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
				clis: []CLI{
					&testCLI{
						name: "basic",
						processors: []command.Processor{
							commander.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
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
				name:          "RunCLI() gets goExecutableFilePath",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
			{
				name:          "RunCLI() fails if os.Executable error",
				cliTargetName: "leepFrogSource",
				runCLI:        true,
				env: map[string]string{
					RootDirectoryEnvVar: "cli-output-dir",
				},
				wantGetCacheCalls: []string{testutil.FilepathAbs(t, "cli-output-dir", "cache")},
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
				osExecutableErr: fmt.Errorf("goExe err"),
				osCheck: &osCheck{
					wantErr: fmt.Errorf("failed to get file from os.Executable(): goExe err"),
					wantStderr: []string{
						"failed to get file from os.Executable(): goExe err",
					},
				},
			},
			/* Useful for commenting out tests */
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				StubExecutableFile(t, "osArgs-at-zero", test.osExecutableErr)
				stubs.StubEnv(t, test.env)
				if test.osReadFileStub {
					testutil.StubValue(t, &osReadFile, func(b string) ([]byte, error) {
						return []byte(test.osReadFileResp), test.osReadFileErr
					})
				}
				testutil.StubValue(t, &CurrentOS, curOS)
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					oschk = test.osCheck
				}

				var uuidIdx int
				testutil.StubValue(t, &getUuid, func() string {
					r := test.uuids[uuidIdx]
					uuidIdx++
					return r
				})

				testutil.StubValue(t, &runtimeCaller, func(n int) (uintptr, string, int, bool) {
					return 0, "/fake/source/location/main.go", 0, !oschk.runtimeCallerMiss
				})

				if err := os.WriteFile(f.Name(), nil, 0644); err != nil {
					t.Fatalf("failed to clear file: %v", err)
				}

				fake := testutil.TempFile(t, "leepFrogSourcerer-test")
				for i, s := range test.args {
					if s == fakeFile {
						test.args[i] = fake.Name()
					}
				}

				if len(test.fakeInputFileContents) > 0 {
					fakeInput := testutil.TempFile(t, "leepFrogSourcerer-test")
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
				cash := cachetest.NewTestCache(t)
				var gotGetCacheCalls []string
				testutil.StubValue(t, &getCacheStub, func(s string) (*cache.Cache, error) {
					gotGetCacheCalls = append(gotGetCacheCalls, s)
					if len(test.cacheErrs) == 0 {
						return cash, nil
					}
					e := test.cacheErrs[0]
					test.cacheErrs = test.cacheErrs[1:]
					return cash, e
				})

				// Run source command
				o := commandtest.NewOutput()
				err = testutil.CmpPanic(t, "source()", func() error {
					return source(test.runCLI, test.cliTargetName, test.clis, fakeGoExecutableFilePath.Name(), test.args, o)
				}, test.wantPanic)
				testutil.CmpError(t, fmt.Sprintf("source(%v)", test.args), oschk.wantErr, err)
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

				if diff := cmp.Diff(test.wantGetCacheCalls, gotGetCacheCalls); diff != "" {
					t.Errorf("source(%v) made incorrect getCache calls (-want, +got):\n%s", test.args, diff)
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
	return commander.SerialNodes(append(tc.processors, commander.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
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
	return &commander.BranchNode{
		Branches: map[string]command.Node{
			"a": commander.SerialNodes(commander.ListArg[string]("A_SL", "str list", 0, 1)),
			"b": commander.SerialNodes(commander.ListArg[string]("B_SL", "str list", 1, 0)),
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
		`┣━━ a [ A_SL ]`,
		`┃`,
		`┗━━ b B_SL`,
		``,
		`Arguments:`,
		`  A_SL: str list`,
		`  B_SL: str list`,
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

// This set of tests ensures that commander.ChangeTest behaves the same as logic
// in sourcerer.go (specifically around saving a CLI regardless of error).
func TestCommandSave(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *commandtest.ExecuteTestCase
		cli  *testCLI
		want *testCLI
	}{
		{
			name: "saves a CLI",
			cli: &testCLI{
				f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					tc.Stuff = "new stuff"
					tc.changed = true
					return nil
				},
			},
			etc: &commandtest.ExecuteTestCase{},
			want: &testCLI{
				Stuff: "new stuff",
			},
		},
		{
			name: "saves a CLI even when error",
			cli: &testCLI{
				f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					tc.Stuff = "new stuff"
					tc.changed = true
					return o.Stderrln("whoops")
				},
			},
			etc: &commandtest.ExecuteTestCase{
				WantStderr: "whoops\n",
				WantErr:    fmt.Errorf("whoops"),
			},
			want: &testCLI{
				Stuff: "new stuff",
			},
		},
		{
			name: "doesn't save a non-changed CLI",
			cli: &testCLI{
				f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					tc.Stuff = "new stuff"
					tc.changed = false
					return nil
				},
			},
			etc: &commandtest.ExecuteTestCase{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.etc.Node = test.cli.Node()
			commandertest.ExecuteTest(t, test.etc)
			commandertest.ChangeTest(t, test.want, test.cli, cmpopts.IgnoreUnexported(testCLI{}))
		})
	}
}
