package sourcerer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leep-frog/command"
)

type windows struct{}

func Windows() OS {
	return &windows{}
}

var (
	windowsRegisterCommandFormat = strings.Join([]string{
		`function %s {`,
		`  _custom_execute_%s %s $args`,
		`}`,
	}, "\n")
	windowsRegisterCommandWithSetupFormat = strings.Join([]string{
		`function %s {`,
		`  $Local:o = New-TemporaryFile`,
		`  %s > $Local:o`,
		`  _custom_execute_%s %s $Local:o $args`,
		`}`,
	}, "\n")
	windowsSetupFunctionFormat = strings.Join([]string{
		`function %s {`,
		`  %s`,
		`}`,
		``,
	}, "\n")
)

func (*windows) Name() string {
	return "windows"
}

func (w *windows) CreateGoFiles(sourceLocation string, targetName string) string {
	return strings.Join([]string{
		"Push-Location",
		fmt.Sprintf(`Set-Location "$(Split-Path %s)"`, sourceLocation),
		fmt.Sprintf("go build -o %s", filepath.Join("$env:GOPATH", "bin", fmt.Sprintf("_%s_runner.exe", targetName))),
		"Pop-Location",
		"",
	}, "\n")
}

func (w *windows) SourcererGoCLI(dir string, targetName string, loadFlag string) []string {
	return []string{
		"Push-Location",
		fmt.Sprintf("cd %q", dir),
		`$Local:tmpFile = New-TemporaryFile`,
		fmt.Sprintf("go run . source %q %s > $Local:tmpFile", targetName, loadFlag),
		`Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
		`. "$Local:tmpFile.ps1"`,
		`Pop-Location`,
	}
}

func (w *windows) RegisterCLIs(output command.Output, targetName string, clis []CLI) error {
	// Generate the autocomplete function
	output.Stdoutln(w.autocompleteFunction(targetName))

	sort.SliceStable(clis, func(i, j int) bool { return clis[i].Name() < clis[j].Name() })
	for _, cli := range clis {
		alias := cli.Name()

		output.Stdoutln(w.executeFunction(targetName, alias, cli.Setup()))

		// We sort ourselves, hence the no sort.
	}
	return nil
}

func (*windows) autocompleteFunction(targetName string) string {
	return strings.Join([]string{
		fmt.Sprintf("$_custom_autocomplete_%s = {", targetName),
		`  param($wordToComplete, $commandAst, $compPoint)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		// 0 for comp type
		fmt.Sprintf(`  (& $env:GOPATH\bin\_%s_runner.exe autocomplete ($commandAst.CommandElements | Select-Object -first 1) "0" $compPoint "$commandAst") | ForEach-Object {`, targetName),
		`    $_`,
		`  }`,
		"}",
		"",
	}, "\n")
}

func (*windows) executeFunction(targetName, cliName string, setup []string) string {
	runnerLine := fmt.Sprintf(`  & $env:GOPATH/bin/_%s_runner.exe execute %q $Local:tmpFile $args`, targetName, cliName)
	var prefix string
	if len(setup) > 0 {
		setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", cliName)
		prefix = strings.Join([]string{
			fmt.Sprintf(windowsSetupFunctionFormat, setupFunctionName, strings.Join(setup, "\n  ")),
		}, "\n")
		runnerLine = strings.Join([]string{
			`  $Local:setupTmpFile = New-TemporaryFile`,
			fmt.Sprintf(`  %s > $Local:setupTmpFile`, setupFunctionName),
			// Same as original command, but with the $Local:setupTmpFile provided as the first regular argument
			fmt.Sprintf(`  & $env:GOPATH/bin/_%s_runner.exe execute %q $Local:tmpFile $Local:setupTmpFile $args`, targetName, cliName),
		}, "\n")
	}
	return strings.Join([]string{
		prefix,
		fmt.Sprintf(`function _custom_execute_%s_%s {`, targetName, cliName),
		``,
		`  # tmpFile is the file to which we write ExecuteData.Executable`,
		`  $Local:tmpFile = New-TemporaryFile`,
		``,
		`  # Run the go-only code`,
		runnerLine,
		`  # Return error if failed`,
		// TODO: Use -ErrorAction (https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_commonparameters?view=powershell-7.3#-erroraction)
		`  If (!$?) { throw "Go execution failed" }`,
		``,
		`  # If success, run the ExecuteData.Executable data`,
		`  Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
		`  . "$Local:tmpFile.ps1"`,
		`  If (!$?) { throw "ExecuteData execution failed" }`,
		// TODO: Leave file as is if DebugEnvVar is set
		// `  Remove-Item "$Local:tmpFile"`,
		// `  Remove-Item "$Local:tmpFile.ps1"`,
		`}`,
		// fmt.Sprintf(`_custom_execute_%s $args`, targetName),
		``,
		fmt.Sprintf("Set-Alias %s _custom_execute_%s_%s", cliName, targetName, cliName),
		fmt.Sprintf("Register-ArgumentCompleter -CommandName %s -ScriptBlock $_custom_autocomplete_%s\n", cliName, targetName),
	}, "\n")
}

func (w *windows) HandleAutocompleteSuccess(output command.Output, suggestions []string) {
	output.Stdoutf("%s\n", strings.Join(suggestions, "\n"))
}

func (w *windows) HandleAutocompleteError(output command.Output, compType int, err error) {
	// Stderr gets hidden, so we need to write to stdout
	output.Stderrf("\nAutocomplete Error: %v", err)
	// Print another string so text isn't autocompleted to error text
	output.Stdoutln()
}

func (w *windows) FunctionWrap(fn string) string {
	return strings.Join([]string{
		"function _leep_execute_data_function_wrap {",
		fn,
		"}",
		"_leep_execute_data_function_wrap",
		"",
	}, "\n")
}

// TODO: Aliasers
func (w *windows) GlobalAliaserFunc(command.Output)         {}
func (w *windows) VerifyAliaser(command.Output, *Aliaser)   {}
func (w *windows) RegisterAliaser(command.Output, *Aliaser) {}

// TODO: Mancli
func (w *windows) Mancli(cli string) []string { return nil }
