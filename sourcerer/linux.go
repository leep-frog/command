package sourcerer

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/leep-frog/command"
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
	linuxRegisterCommandFormat          = "alias %s='source _custom_execute_%s %s'"
	linuxRegisterCommandWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && source _custom_execute_%s %s $o'"
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

func (l *linux) FunctionWrap(fn string) string {
	return strings.Join([]string{
		"function _leep_execute_data_function_wrap {",
		fn,
		"}",
		"_leep_execute_data_function_wrap",
		"",
	}, "\n")
}

func (l *linux) HandleAutocompleteSuccess(output command.Output, suggestions []string) {
	output.Stdoutf("%s\n", strings.Join(suggestions, "\n"))
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

func (l *linux) SourcererGoCLI(dir string, targetName string) []string {
	return []string{
		"pushd . > /dev/null",
		fmt.Sprintf("cd %q", dir),
		`local tmpFile="$(mktemp)"`,
		fmt.Sprintf("go run . source %q > $tmpFile && source $tmpFile ", targetName),
		"popd > /dev/null",
	}
}

func (l *linux) RegisterCLIs(builtin bool, goExecutable, targetName string, clis []CLI) ([]string, error) {
	// Generate the execute functions
	r := l.executeFileContents(builtin, goExecutable, targetName)
	// Generate the autocomplete function
	r = append(r, l.autocompleteFunction(builtin, goExecutable, targetName)...)

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
		r = append(r, fmt.Sprintf("complete -F _custom_autocomplete_%s %s %s", targetName, NosortString(), alias))
	}
	return r, nil
}

func (*linux) getBranchString(builtin bool, branchName string) string {
	if builtin {
		return fmt.Sprintf("%s %s", BuiltInCommandParameter, branchName)
	}
	return branchName
}

func (l *linux) autocompleteFunction(builtin bool, goExecutable, filename string) []string {
	branchStr := l.getBranchString(builtin, AutocompleteBranchName)
	return []string{
		fmt.Sprintf("function _custom_autocomplete_%s {", filename),
		`  local tFile=$(mktemp)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		fmt.Sprintf(`  %s %s ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, goExecutable, branchStr),
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
		fmt.Sprintf(`  if [ -z "$%s" ]; then`, command.DebugEnvVar),
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
		fmt.Sprintf(`  echo Provided CLI %q is not a CLI generated with github.com/leep-frog/command`, aliaser.cli),
		`  return 1`,
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
		fmt.Sprintf("complete -F _custom_autocomplete_for_alias_%s %s %s", a.alias, NosortString(), a.alias),
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

func (*linux) aliaserAutocompleteFunction(alias string, cli string, quotedArg string) []string {
	return []string{
		fmt.Sprintf("function _custom_autocomplete_for_alias_%s {", alias),
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

func (*linux) AddsSpaceToSingleAutocompletion() bool {
	return true
}

func (*linux) ShellCommandFileRunner(file string) (string, []string) {
	return `bash`, []string{file}
}

/*
// autocompleteForAliasFunction
	// See AliaserCommand.
	autocompleteForAliasFunction =
/**/

// FileStringFromCLI returns a bash command that retrieves the binary file that
// is actually executed for a leep-frog-generated CLI.
func FileStringFromCLI(cli string) string {
	return fmt.Sprintf(`local file="$(type %s | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`, cli)
}
