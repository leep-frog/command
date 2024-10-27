package sourcerer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/leep-frog/command/command"
)

type windows struct{}

func Windows() OS {
	return &windows{}
}

var (
	windowsSetupFunctionFormat = strings.Join([]string{
		`function %s {`,
		`  %s`,
		`}`,
		``,
	}, "\n")
)

func (w *windows) setAlias(forAlias bool, alias, value, targetName string) []string {
	return []string{
		// Delete the alias if it exists
		fmt.Sprintf("(Get-Alias) | Where { $_.NAME -match '^%s$'} | ForEach-Object { del alias:${_} -Force }", alias),
		// Set the alias
		fmt.Sprintf("Set-Alias %s %s", alias, value),
		// Register the autocompleter
		w.registerArgumentCompleter(forAlias, alias, targetName),
	}
}

func (w *windows) registerArgumentCompleter(forAlias bool, alias, targetName string) string {
	return fmt.Sprintf("Register-ArgumentCompleter -CommandName %s -ScriptBlock $%s", alias, w.autocompleteFunctionName(forAlias, targetName))
}

func (*windows) Name() string {
	return "windows"
}

func (*windows) ExecutableFileSuffix() string {
	return ".exe"
}

func (w *windows) SourcererGoCLI(dir string, targetName string) []string {
	return []string{
		"Push-Location",
		fmt.Sprintf("cd %q", dir),
		`$Local:tmpFile = New-TemporaryFile`,
		fmt.Sprintf("go run . source %q > $Local:tmpFile", targetName),
		`Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
		`. "$Local:tmpFile.ps1"`,
		`Pop-Location`,
	}
}

func (w *windows) RegisterRunCLIAutocomplete(goExecutable, alias string) []string {
	targetName := fmt.Sprintf("RunCLI%s", alias)
	return []string{
		w.autocompleteFunction(true, false, goExecutable, targetName),
		w.registerArgumentCompleter(false, targetName, alias),
	}
}

func (w *windows) RegisterCLIs(builtin bool, goExecutable, targetName string, clis []CLI) []string {
	// Generate the autocomplete function
	r := []string{w.autocompleteFunction(false, builtin, goExecutable, targetName)}

	sort.SliceStable(clis, func(i, j int) bool { return clis[i].Name() < clis[j].Name() })
	for _, cli := range clis {
		alias := cli.Name()

		r = append(r, w.executeFunction(builtin, goExecutable, targetName, alias, cli.Setup()))

		// We sort ourselves, hence the no sort.
	}
	return r
}

func (*windows) getBranchString(builtin bool, branchName string) string {
	if builtin {
		return fmt.Sprintf("%s %s", BuiltInCommandParameter, branchName)
	}
	return branchName
}

func (w *windows) autocompleteFunction(runCLI bool, builtin bool, goExecutable, targetName string) string {
	var cliRef string
	if !runCLI {
		cliRef = "($commandAst.CommandElements | Select-Object -first 1)"
	}
	return strings.Join([]string{
		fmt.Sprintf("$%s = {", w.autocompleteFunctionName(false, targetName)),
		`  param($wordToComplete, $commandAst, $compPoint)`,

		// Passthrough args are in a file because powershell backwards slashes
		// can cause awkward behavior when passed as an argument (e.g. `some\file\path\` -> `some\file\path"`)
		// I believe an ending backslash causes the quote to be escaped (maybe?), and I'm
		// sure there are other edge cases caused by it, hence the decision to use
		// this workaround.
		`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
		`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
		// `  Write-Output $commandAst.ToString() > $Local:tmpPassthroughArgFile`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		// 0 for comp type
		fmt.Sprintf(`  (& %s %s %s --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile) | ForEach-Object {`, goExecutable, w.getBranchString(builtin, AutocompleteBranchName), cliRef),
		`    "$_"`,
		`  }`,
		"}",
		"",
	}, "\n")
}

func (w *windows) executeFunction(builtin bool, goExecutable, targetName, cliName string, setup []string) string {
	runnerLine := fmt.Sprintf(`  & %s %s %q $Local:tmpFile $args`, goExecutable, w.getBranchString(builtin, ExecuteBranchName), cliName)
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
			fmt.Sprintf(`  & %s execute %q $Local:tmpFile "$Local:setupTmpFile.txt" $args`, goExecutable, cliName),
		}, "\n")
	}
	return strings.Join(append([]string{
		prefix,
		fmt.Sprintf(`function %s {`, w.executeFunctionName(false, targetName, cliName)),
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
	}, w.setAlias(
		false,
		cliName,
		w.executeFunctionName(false, targetName, cliName),
		targetName,
	)...), "\n")
}

const (
	exFnPrefix      = "_custom_execute_"
	exFnAliasPrefix = "_sourcerer_alias_execute_"
	acFnPrefix      = "_custom_autocomplete_"
	acFnAliasPrefix = "_sourcerer_alias_autocomplete_"
)

func (w *windows) executeFunctionName(forAlias bool, targetName, cliName string) string {
	if forAlias {
		return fmt.Sprintf("%s%s", exFnAliasPrefix, cliName)
	}
	return fmt.Sprintf("%s%s_%s", exFnPrefix, targetName, cliName)
}

func (w *windows) autocompleteFunctionName(forAlias bool, targetName string) string {
	prefix := acFnPrefix
	if forAlias {
		prefix = acFnAliasPrefix
	}
	return fmt.Sprintf("%s%s", prefix, targetName)
}

func (w *windows) HandleAutocompleteSuccess(output command.Output, autocompletion *command.Autocompletion) {
	// Add a trailing space because powershell doesn't do that for us for single-guaranteed completions
	if len(autocompletion.Suggestions) == 1 && !autocompletion.SpacelessCompletion {
		autocompletion.Suggestions[0] = fmt.Sprintf("%s ", autocompletion.Suggestions[0])
	}
	output.Stdoutf("%s\n", strings.Join(autocompletion.Suggestions, "\n"))
}

func (w *windows) HandleAutocompleteError(output command.Output, compType int, err error) {
	// Stderr gets hidden, so we need to write to stdout
	output.Stderrf("\nAutocomplete Error: %v", err)
	// Print another string so text isn't autocompleted to error text
	output.Stdoutln()
}

func (w *windows) FunctionWrap(name string, fn string) string {
	return strings.Join([]string{
		fmt.Sprintf("function %s {", name),
		fn,
		"}",
		// . name so it runs in the same shell
		fmt.Sprintf(". %s", name),
		"",
	}, "\n")
}

func (w *windows) GlobalAliaserFunc(goExecutable string) []string { return nil }
func (w *windows) VerifyAliaser(a *Aliaser) []string {
	return w.verifyAliaserCommand(a.cli)
}

func (w *windows) verifyAliaserCommand(cli string) []string {
	return []string{
		fmt.Sprintf(`if (!(Test-Path alias:%s) -or !(Get-Alias %s | where {$_.DEFINITION -match "%s"}).NAME) {`, cli, cli, exFnPrefix),
		fmt.Sprintf(`  throw "The CLI provided (%s) is not a sourcerer-generated command"`, cli),
		`}`,
	}
}

func (w *windows) RegisterAliaser(goExecutable string, a *Aliaser) []string {
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

	aliaserTarget := "aliaser"

	return append([]string{
		// Create the execute function
		fmt.Sprintf(`function %s {`, w.executeFunctionName(true, aliaserTarget, a.alias)),
		fmt.Sprintf(`  $Local:functionName = "$((Get-Alias %q).DEFINITION)"`, a.cli),
		fmt.Sprintf(`  Invoke-Expression %s`, expression),
		`}`,
		// Create the autocomplete function
		fmt.Sprintf(`$%s = {`, w.autocompleteFunctionName(true, a.alias)),
		`  param($wordToComplete, $commandAst, $compPoint)`,
		`  $Local:tmpPassthroughArgFile = New-TemporaryFile`,
		`  [IO.File]::WriteAllText($Local:tmpPassthroughArgFile, $commandAst.ToString())`,
		fmt.Sprintf(`  (Invoke-Expression '& %s %s %q --comp-line-file "0" $compPoint $Local:tmpPassthroughArgFile %s') | ForEach-Object {`, goExecutable, AutocompleteBranchName, a.cli, quotedArgs),
		`    "$_"`,
		`  }`,
		`}`,
	}, w.setAlias(
		true,
		a.alias,
		w.executeFunctionName(true, aliaserTarget, a.alias),
		a.alias,
	)...)
}

func (*windows) SetEnvVar(envVar, value string) string {
	return fmt.Sprintf("$env:%s = %q", envVar, value)
}

func (*windows) UnsetEnvVar(envVar string) string {
	return fmt.Sprintf("Remove-Item $env:%s", envVar)
}

func (*windows) ShellCommandFileRunner(file string) (string, []string) {
	return `powershell.exe`, []string{`-NoProfile`, file}
}
