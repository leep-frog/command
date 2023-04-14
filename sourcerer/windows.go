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
		`  %s -and _custom_execute_%s %s $Local:o $args`,
		`}`,
	}, "\n")
	windowsSetupFunctionFormat = strings.Join([]string{
		`function %s {`,
		`  %s`,
		`}`,
		``,
	}, "\n")
)

func (w *windows) CreateGoFiles(sourceLocation string, targetName string) string {
	return strings.Join([]string{
		"pushd",
		fmt.Sprintf(`cd "$(Split-Path %s)"`, sourceLocation),
		fmt.Sprintf("go build -o %s", filepath.Join("$env:GOPATH", "bin", fmt.Sprintf("_%s_runner.exe", targetName))),
		"popd",
		"",
	}, "\n")
}

func (w *windows) SourcererGoCLI(dir string, targetName string, loadFlag string) []string {
	return []string{
		"pushd",
		fmt.Sprintf("cd %q", dir),
		`Local:tmpFile = New-TemporaryFile`,
		fmt.Sprintf("go run . source %q %s > $tmpFile && source $tmpFile ", targetName, loadFlag),
		"popd",
	}
}

func (w *windows) RegisterCLIs(output command.Output, targetName string, clis []CLI) error {
	// Generate the autocomplete function
	output.Stdoutln(w.autocompleteFunction(targetName))

	// The execute logic is put in an actual file so it can be used by other
	// bash environments that don't actually source sourcerer-related commands.
	efc := w.executeFileContents(targetName)

	// f, err := os.OpenFile(getExecuteFile(targetName), os.O_WRONLY|os.O_CREATE, command.CmdOS.DefaultFilePerm())
	// if err != nil {
	// return output.Stderrf("failed to open execute function file: %v\n", err)
	// }

	// if _, err := f.WriteString(efc); err != nil {
	// return output.Stderrf("failed to write to execute function file: %v\n", err)
	// }

	output.Stdoutln(efc)

	sort.SliceStable(clis, func(i, j int) bool { return clis[i].Name() < clis[j].Name() })
	for _, cli := range clis {
		alias := cli.Name()

		aliasCommand := fmt.Sprintf(windowsRegisterCommandFormat, alias, targetName, alias)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			output.Stdoutf(windowsSetupFunctionFormat, setupFunctionName, strings.Join(scs, "  \n  "))
			aliasCommand = fmt.Sprintf(windowsRegisterCommandWithSetupFormat, alias, setupFunctionName, targetName, alias)
		}

		output.Stdoutln(aliasCommand)

		// We sort ourselves, hence the no sort.
		// output.Stdoutf("complete -F _custom_autocomplete_%s %s %s\n", targetName, NosortString(), alias)
		output.Stdoutf("Register-ArgumentCompleter -CommandName %s -ScriptBlock $_custom_autocomplete_%s\n", alias, targetName)
	}
	return nil
}

func (*windows) autocompleteFunction(targetName string) string {
	return strings.Join([]string{
		fmt.Sprintf("$_custom_autocomplete_%s = {", targetName),
		// This order might be messed up because parameter name doesn't seem to be what I think it actually is
		`  param($wordToComplete, $commandAst, $compLine)`,
		// `  $Local:tFile = New-TemporaryFile`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		// 0 for comp type
		fmt.Sprintf(`  (& $env:GOPATH\bin\_%s_runner.exe autocomplete "$commandAst" "0" $compPoint "$commandAst") | ForEach-Object {`, targetName),
		`    $_`,
		`  }`,
		"}",
		"",
	}, "\n")
}

func (*windows) executeFileContents(targetName string) string {
	return strings.Join([]string{
		fmt.Sprintf(`function _custom_execute_%s {`, targetName),
		`  param ([String] $CLI)`,
		``,
		`  # tmpFile is the file to which we write ExecuteData.Executable`,
		`  $Local:tmpFile = New-TemporaryFile`,
		``,
		`  # Run the go-only code`,
		fmt.Sprintf(`  & $env:GOPATH/bin/_%s_runner.exe execute "$CLI" $tmpFile $args`, targetName),
		`  # Return error if failed`,
		// TODO: Use -ErrorAction (https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_commonparameters?view=powershell-7.3#-erroraction)
		`  If (!$?) { throw "Go execution failed" }`,
		``,
		`  # If success, run the ExecuteData.Executable data`,
		`  . $tmpFile`,
		`  If (!$?) { throw "Go execution failed" }`,
		// TODO: Leave file as is if DebugEnvVar is set
		`  Remove-Item $tmpFile`,
		`}`,
		// fmt.Sprintf(`_custom_execute_%s $args`, targetName),
		``,
	}, "\n")
}

func (w *windows) HandleAutocompleteSuccess(output command.Output, suggestions []string) {
	output.Stdoutf("%s\n", strings.Join(suggestions, "\n"))
}

func (w *windows) HandleAutocompleteError(output command.Output, compType int, err error) {
	// Stderr gets hidden, so we need to write to stdout
	output.Stdoutf("ERROR: %v\n", err)
	// Print another string so text isn't autocompleted to error text
	output.Stdoutln()
}

// TODO:
func (w *windows) FunctionWrap(string) string { return "" }

// TODO: Aliasers
func (w *windows) GlobalAliaserFunc(command.Output)         {}
func (w *windows) VerifyAliaser(command.Output, *Aliaser)   {}
func (w *windows) RegisterAliaser(command.Output, *Aliaser) {}

// TODO: Mancli
func (w *windows) Mancli(cli string) []string { return nil }
