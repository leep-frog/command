package sourcerer

import (
	"fmt"
	"path/filepath"
	"regexp"
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
	windowsSetupFunctionFormat = strings.Join([]string{
		`function %s {`,
		`  %s`,
		`}`,
		``,
	}, "\n")
)

func (w *windows) setAlias(alias, value, completer string) string {
	return strings.Join([]string{
		// Delete the alias if it exists
		fmt.Sprintf("(Get-Alias) | Where { $_.NAME -match '^%s$'} | ForEach-Object { del alias:${_} -Force }", alias),
		// Set the alias
		fmt.Sprintf("Set-Alias %s %s", alias, value),
		// Register the autocompleter
		fmt.Sprintf("Register-ArgumentCompleter -CommandName %s -ScriptBlock $%s", alias, completer),
	}, "\n")
}

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

func (w *windows) executeFunction(targetName, cliName string, setup []string) string {
	runnerLine := fmt.Sprintf(`  & $env:GOPATH/bin/_%s_runner.exe execute %q $Local:tmpFile $args`, targetName, cliName)
	var prefix string
	if len(setup) > 0 {
		setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", cliName)
		prefix = strings.Join([]string{
			fmt.Sprintf(windowsSetupFunctionFormat, setupFunctionName, strings.Join(setup, "\n  ")),
		}, "\n")
		runnerLine = strings.Join([]string{
			`  $Local:setupTmpFile = New-TemporaryFile`,
			fmt.Sprintf(`  %s > "$Local:setupTmpFile"`, setupFunctionName),
			`  Copy-Item "$Local:setupTmpFile" "$Local:setupTmpFile.txt"`,
			// Same as original command, but with the $Local:setupTmpFile provided as the first regular argument
			fmt.Sprintf(`  & $env:GOPATH/bin/_%s_runner.exe execute %q $Local:tmpFile "$Local:setupTmpFile.txt" $args`, targetName, cliName),
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
		`  If (!$?) {`,
		`    Write-Error "Go execution failed"`,
		// We need the else (rather than using return or break)
		// so that the return status ($?) of the function is false.
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
		w.setAlias(
			cliName,
			fmt.Sprintf("_custom_execute_%s_%s", targetName, cliName),
			fmt.Sprintf("_custom_autocomplete_%s", targetName),
		),
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
func (w *windows) GlobalAliaserFunc(command.Output) {}
func (w *windows) VerifyAliaser(output command.Output, a *Aliaser) {
	output.Stdoutln(strings.Join(w.verifyAliaserCommand(a.cli), "\n"))
}

func (w *windows) verifyAliaserCommand(cli string) []string {
	return []string{
		fmt.Sprintf(`if (!(Test-Path alias:%s) -or !(Get-Alias %s | where {$_.DEFINITION -match "_custom_execute"}).NAME) {`, cli, cli),
		fmt.Sprintf(`  throw "The CLI provided (%s) is not a sourcerer-generated command"`, cli),
		`}`,
	}
}

func (w *windows) RegisterAliaser(output command.Output, a *Aliaser) {
	var qas []string
	for _, v := range a.values {
		qas = append(qas, fmt.Sprintf("%q", v))
	}
	quotedArgs := strings.Join(qas, " ")

	// Recursively passing `$args` sometimes lumps all args as one parameter. The expression
	// object is used in conjunction with `Invoke-Expression` to get around this issue.
	expression := `($Local:functionName $args)`
	if len(quotedArgs) > 0 {
		expression = fmt.Sprintf(`($Local:functionName + " " + %s + " " + $args)`, strings.Join(qas, ` + " " + `))
	}

	output.Stdoutln(strings.Join([]string{
		// Create the execute function
		fmt.Sprintf(`function _sourcerer_alias_execute_%s {`, a.alias),
		fmt.Sprintf(`  $Local:functionName = "$((Get-Alias %q).DEFINITION)"`, a.cli),
		fmt.Sprintf(`  Invoke-Expression %s`, expression),
		`}`,
		// Create the autocomplete function
		fmt.Sprintf(`$_sourcerer_alias_autocomplete_%s = {`, a.alias),
		`  param($wordToComplete, $commandAst, $compPoint)`,
		// targetNameArg ensures the target doesn't contain a '_' character
		fmt.Sprintf(`  $Local:def = ((Get-Alias %s).DEFINITION -split "_").Get(3)`, a.cli),
		fmt.Sprintf(`  (Invoke-Expression '& $env:GOPATH\bin\_${Local:def}_runner.exe autocomplete %q "0" $compPoint "$commandAst" %s') | ForEach-Object {`, a.cli, quotedArgs),
		`    $_`,
		`  }`,
		`}`,
		w.setAlias(
			a.alias,
			fmt.Sprintf("_sourcerer_alias_execute_%s", a.alias),
			fmt.Sprintf("_sourcerer_alias_autocomplete_%s", a.alias),
		),
	}, "\n"))
}

// TODO: Mancli

var (
	windowsMancliRegex = regexp.MustCompile("[\\s'\"`]")
)

func (w *windows) Mancli(cli string, args ...string) []string {
	// We can't use quotedArgs because this string is being used inside of a Windows string
	// and Windows uses backticks for escaping (not backslashes)
	// so we can't use built in go string format quoting.
	var formattedArgs []string
	for _, a := range args {
		formattedArgs = append(formattedArgs, windowsMancliRegex.ReplaceAllString(a, "_"))
	}

	return append(
		w.verifyAliaserCommand(cli),
		fmt.Sprintf(`$Local:targetName = (Get-Alias %s).DEFINITION.split("_")[3]`, cli),
		fmt.Sprintf(`Invoke-Expression "$env:GOPATH\bin\_${Local:targetName}_runner.exe usage %s %s"`, cli, strings.Join(formattedArgs, " ")),
	)
}

func (*windows) SetEnvVar(envVar, value string) string {
	return fmt.Sprintf("$env:%s = %q", envVar, value)
}

func (*windows) UnsetEnvVar(envVar string) string {
	return fmt.Sprintf("Remove-Item $env:%s", envVar)
}

func (*windows) AddsSpaceToSingleAutocompletion() bool {
	return false
}

func (*windows) ShellCommandFileRunner(file string) (string, []string) {
	return `powershell.exe`, []string{`-NoProfile`, file}
}
