package sourcerer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

const (
	zshEnvVar = "COMMAND_CLI_ZSH"
)

var (
	// NosortString returns the complete option to ignore sorting.
	// It returns nothing if the IGNORE_NOSORT environment variable is set.
	NosortString = func() string {
		if _, ignore := os.LookupEnv("IGNORE_NOSORT"); ignore {
			return ""
		}
		return "-o nosort"
	}
)

type linux struct{}

func Linux() OS {
	return &linux{}
}

var (
	linuxRegisterCommandFormat          = "alias %s='_custom_execute_%s %s'"
	linuxRegisterCommandWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && _custom_execute_%s %s $o'"
	linuxSetupFunctionFormat            = strings.Join([]string{
		`function %s {`,
		`  %s`,
		"}",
		"",
	}, "\n")
)

func (l *linux) Name() string {
	return "linux"
}

func (l *linux) ExecutableFileSuffix() string {
	return ""
}

func (l *linux) SourceableFile(target string) string {
	return fmt.Sprintf("%s_loader.sh", target)
}

func (l *linux) SourceSetup(sourceableFile, targetName, goRunSourceCommand, userDir string) []string {
	return []string{
		`# Load all of your CLIs`,
		fmt.Sprintf(`source %q`, sourceableFile),
		``,
		`# Useful function to easily regenerate all of your CLIs whenever your go code changes`,
		fmt.Sprintf(`function _regenerate_%s_CLIs() {`, targetName),
		`  pushd . > /dev/null`,
		fmt.Sprintf(`  cd %q`, userDir),
		fmt.Sprintf("  %s", goRunSourceCommand),
		`  popd . > /dev/null`,
		`}`,
	}
}

func (l *linux) FunctionWrap(name, fn string) string {
	return strings.Join([]string{
		"#!/bin/bash",
		fmt.Sprintf("function %s {", name),
		fn,
		"}",
		name,
		"",
	}, "\n")
}

func (l *linux) HandleAutocompleteSuccess(output command.Output, autocompletion *command.Autocompletion) {
	if len(autocompletion.Suggestions) == 1 && autocompletion.SpacelessCompletion {
		autocompletion.Suggestions = append(autocompletion.Suggestions, fmt.Sprintf("%s_", autocompletion.Suggestions[0]))
	}
	output.Stdoutf("%s\n", strings.Join(autocompletion.Suggestions, "\n"))
}

func (l *linux) HandleAutocompleteError(output command.Output, compType int, err error) {
	// Only display the error if the user is requesting completion via successive tabs (so distinct completions are guaranteed to be displayed)
	if compType == 63 { /* code 63 = '?' character */
		// Add newline so we're outputting stderr on a newline (and not line with cursor)
		output.Stderrf("\nAutocomplete Error: %v", err)
		// Suggest non-overlapping strings (one space and one tab) so COMP_LINE is reprinted
		output.Stdoutf("\t\n \n")
	}
}

func (l *linux) SourceFileCommand(sourcerersDir, targetName string) string {
	return fmt.Sprintf("source %q", filepath.Join(sourcerersDir, l.SourceableFile(targetName)))
}

func (l *linux) RecursiveCopyDir(src, dst string) string {
	// Need the `/*` outside of the quotes, otherwise bash thinks
	// the directory name simply contains a `*` character.
	return fmt.Sprintf("cp -a %q/* %q", src, dst)
}

func (l *linux) RegisterRunCLIAutocomplete(goExecutable, alias string) []string {
	targetName := fmt.Sprintf("RunCLI%s", alias)
	return append(
		l.autocompleteFunction(true, false, goExecutable, targetName),
		l.autocompleteRegistration(targetName, alias),
	)
}

func (l *linux) completeScriptCommand(functionName, alias string) string {
	// Use curly brackets (instead of parentheses) to ensure the code runs in the parent shell and not in a sub-shell
	return fmt.Sprintf(`{ { type complete > /dev/null 2>&1 ; } && complete -F %s %s %s ; } || { echo 'shell function "complete" either failed or does not exist; if using zsh, be sure to set up bashcompinit' 1>&2 ; }`, functionName, NosortString(), alias)
}

func (l *linux) autocompleteRegistration(targetName, alias string) string {
	return l.completeScriptCommand(l.autocompleteFunctionName(targetName, false), alias)
}

func (l *linux) RegisterCLIs(builtin bool, goExecutable, targetName string, clis []CLI) []string {
	// Generate the execute functions
	r := l.executeFileContents(builtin, goExecutable, targetName)
	// Generate the autocomplete function
	r = append(r, l.autocompleteFunction(false, builtin, goExecutable, targetName)...)

	sort.SliceStable(clis, func(i, j int) bool { return clis[i].Name() < clis[j].Name() })
	for _, cli := range clis {
		alias := cli.Name()

		aliasCommand := fmt.Sprintf(linuxRegisterCommandFormat, alias, targetName, alias)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			r = append(r, fmt.Sprintf(linuxSetupFunctionFormat, setupFunctionName, strings.Join(scs, "\n  ")))
			aliasCommand = fmt.Sprintf(linuxRegisterCommandWithSetupFormat, alias, setupFunctionName, targetName, alias)
		}

		r = append(r, aliasCommand)

		// We sort ourselves, hence the no sort.
		r = append(r, l.autocompleteRegistration(targetName, alias))
	}
	return r
}

func (*linux) getBranchString(builtin bool, branchName string) string {
	if builtin {
		return fmt.Sprintf("%s %s", BuiltInCommandParameter, branchName)
	}
	return branchName
}

func (l *linux) autocompleteFunctionName(targetName string, forAlias bool) string {
	var aliasSuffix string
	if forAlias {
		aliasSuffix = "_for_alias"
	}
	return fmt.Sprintf("_custom_autocomplete%s_%s", aliasSuffix, targetName)
}

func (l *linux) autocompleteFunction(runCLI bool, builtin bool, goExecutable, targetName string) []string {
	var cliRef string
	if !runCLI {
		cliRef = "${COMP_WORDS[0]}"
	}
	branchStr := l.getBranchString(builtin, AutocompleteBranchName)
	compType := "$COMP_TYPE"
	if _, ok := command.OSLookupEnv(zshEnvVar); ok {
		compType = "0"
	}
	return []string{
		fmt.Sprintf("function %s {", l.autocompleteFunctionName(targetName, false)),
		`  local tFile=$(mktemp)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		fmt.Sprintf(`  %s %s %s "%s" $COMP_POINT "$COMP_LINE" > $tFile`, goExecutable, branchStr, cliRef, compType),
		`  local IFS=$'\n'`,
		`  COMPREPLY=( $(cat $tFile) )`,
		`  rm $tFile`,
		"}",
		"",
	}
}

func (l *linux) executeFileContents(builtin bool, goExecutable, filename string) []string {
	branchStr := l.getBranchString(builtin, ExecuteBranchName)
	return []string{
		fmt.Sprintf(`function _custom_execute_%s {`, filename),
		`  # tmpFile is the file to which we write ExecuteData.Executable`,
		`  local tmpFile=$(mktemp)`,
		``,
		`  # Run the go-only code`,
		fmt.Sprintf(`  %s %s "$1" $tmpFile "${@:2}"`, goExecutable, branchStr),
		`  # Return the error code if go code terminated with an error`,
		`  local errorCode=$?`,
		`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
		``,
		`  # Otherwise, run the ExecuteData.Executable data`,
		`  source $tmpFile`,
		`  local errorCode=$?`,
		fmt.Sprintf(`  if [ -z "$%s" ]; then`, commander.DebugEnvVar),
		`    rm $tmpFile`,
		`  else`,
		`    echo $tmpFile`,
		`  fi`,
		`  return $errorCode`,
		`}`,
		``,
	}
}

func (l *linux) GlobalAliaserFunc(goExecutable string) []string {
	return l.aliaserGlobalAutocompleteFunction(goExecutable)
}

func (*linux) VerifyAliaser(aliaser *Aliaser) []string {
	return []string{
		FileStringFromCLI(aliaser.cli),
		`if [ -z "$file" ]; then`,
		FileStringFromCLIZSH(aliaser.cli),
		`  if [ -z "$file" ]; then`,
		fmt.Sprintf(`    echo Provided CLI %q is not a CLI generated with github.com/leep-frog/command`, aliaser.cli),
		`    return 1`,
		`  fi`,
		`fi`,
		``,
		``,
	}
}

func (l *linux) RegisterAliaser(goExecutable string, a *Aliaser) []string {
	// Output the bash alias and completion commands
	var qas []string
	for _, v := range a.values {
		qas = append(qas, fmt.Sprintf("%q", v))
	}
	quotedArgs := strings.Join(qas, " ")

	// The trailing space causes issues, so we need to make sure we remove that if necessary.
	aliasTo := strings.TrimSpace(fmt.Sprintf("%s %s", a.cli, quotedArgs))
	r := []string{
		fmt.Sprintf("alias -- %s=%q", a.alias, aliasTo),
	}

	r = append(r, l.aliaserAutocompleteFunction(a.alias, a.cli, quotedArgs)...)
	return append(r,
		l.completeScriptCommand(l.autocompleteFunctionName(a.alias, true), a.alias),
		``,
	)
}

func (*linux) aliaserGlobalAutocompleteFunction(goExecutable string) []string {
	return []string{
		`function _leep_frog_autocompleter {`,
		`  local tFile=$(mktemp)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		fmt.Sprintf(`  %s autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`, goExecutable),
		`  local IFS='`,
		`';`,
		`  COMPREPLY=( $(cat $tFile) )`,
		`  rm $tFile`,
		`}`,
		``,
	}
}

func (l *linux) aliaserAutocompleteFunction(alias string, cli string, quotedArg string) []string {
	return []string{
		fmt.Sprintf("function %s {", l.autocompleteFunctionName(alias, true)),
		fmt.Sprintf(`  _leep_frog_autocompleter %q %s`, cli, quotedArg),
		"}",
		"",
	}
}

func (*linux) SetEnvVar(envVar, value string) string {
	return fmt.Sprintf("export %q=%q", envVar, value)
}

func (*linux) UnsetEnvVar(envVar string) string {
	return fmt.Sprintf("unset %q", envVar)
}

func (*linux) ShellCommandFileRunner(file string) (string, []string) {
	return `bash`, []string{file}
}

// FileStringFromCLI returns a bash command that retrieves the binary file that
// is actually executed for a leep-frog-generated CLI.
func FileStringFromCLI(cli string) string {
	return fmt.Sprintf(`local file="$(type %s | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`, cli)
}

func FileStringFromCLIZSH(cli string) string {
	return fmt.Sprintf(`  local file="$(type %s | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`, cli)
}
