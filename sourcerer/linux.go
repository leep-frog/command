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
	// getExecuteFile returns the name of the file to which execute file logic is written.
	// It is a separte function so it can be stubbed out for testing.
	getExecuteFile = func(filename string) string {
		return fmt.Sprintf("%s/bin/_custom_execute_%s", os.Getenv("GOPATH"), filename)
	}

	linuxRegisterCommandFormat          = "alias %s='source $GOPATH/bin/_custom_execute_%s %s'"
	linuxRegisterCommandWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && source $GOPATH/bin/_custom_execute_%s %s $o'"
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

func (l *linux) CreateGoFiles(sourceLocation string, targetName string) string {
	return strings.Join([]string{
		"pushd . > /dev/null",
		fmt.Sprintf(`cd "$(dirname %s)"`, sourceLocation),
		fmt.Sprintf("go build -o $GOPATH/bin/_%s_runner", targetName),
		"popd > /dev/null",
		"",
	}, "\n")
}

func (l *linux) SourcererGoCLI(dir string, targetName string, loadFlag string) []string {
	return []string{
		"pushd . > /dev/null",
		fmt.Sprintf("cd %q", dir),
		`local tmpFile="$(mktemp)"`,
		fmt.Sprintf("go run . source %q %s > $tmpFile && source $tmpFile ", targetName, loadFlag),
		"popd > /dev/null",
	}
}

func (l *linux) RegisterCLIs(output command.Output, targetName string, clis []CLI) error {
	// Generate the autocomplete function
	output.Stdoutln(l.autocompleteFunction(targetName))

	// The execute logic is put in an actual file so it can be used by other
	// bash environments that don't actually source sourcerer-related commands.
	efc := l.executeFileContents(targetName)

	f, err := os.OpenFile(getExecuteFile(targetName), os.O_WRONLY|os.O_CREATE, command.CmdOS.DefaultFilePerm())
	if err != nil {
		return output.Stderrf("failed to open execute function file: %v\n", err)
	}

	if _, err := f.WriteString(efc); err != nil {
		return output.Stderrf("failed to write to execute function file: %v\n", err)
	}

	sort.SliceStable(clis, func(i, j int) bool { return clis[i].Name() < clis[j].Name() })
	for _, cli := range clis {
		alias := cli.Name()

		aliasCommand := fmt.Sprintf(linuxRegisterCommandFormat, alias, targetName, alias)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			output.Stdoutf(linuxSetupFunctionFormat, setupFunctionName, strings.Join(scs, "\n  "))
			aliasCommand = fmt.Sprintf(linuxRegisterCommandWithSetupFormat, alias, setupFunctionName, targetName, alias)
		}

		output.Stdoutln(aliasCommand)

		// We sort ourselves, hence the no sort.
		output.Stdoutf("complete -F _custom_autocomplete_%s %s %s\n", targetName, NosortString(), alias)
	}
	return nil
}

func (*linux) autocompleteFunction(filename string) string {
	return strings.Join([]string{
		fmt.Sprintf("function _custom_autocomplete_%s {", filename),
		`  local tFile=$(mktemp)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		fmt.Sprintf(`  $GOPATH/bin/_%s_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`, filename),
		`  local IFS=$'\n'`,
		`  COMPREPLY=( $(cat $tFile) )`,
		`  rm $tFile`,
		"}",
		"",
	}, "\n")
}

func (*linux) executeFileContents(filename string) string {
	return strings.Join([]string{
		fmt.Sprintf(`function _custom_execute_%s {`, filename),
		`  # tmpFile is the file to which we write ExecuteData.Executable`,
		`  local tmpFile=$(mktemp)`,
		``,
		`  # Run the go-only code`,
		fmt.Sprintf(`  $GOPATH/bin/_%s_runner execute "$1" $tmpFile "${@:2}"`, filename),
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
		fmt.Sprintf(`_custom_execute_%s "$@"`, filename),
		``,
	}, "\n")
}

func (l *linux) GlobalAliaserFunc(output command.Output) {
	output.Stdoutln(l.aliaserGlobalAutocompleteFunction())
}

func (*linux) VerifyAliaser(output command.Output, aliaser *Aliaser) {
	output.Stdoutln(strings.Join([]string{
		FileStringFromCLI(aliaser.cli),
		`if [ -z "$file" ]; then`,
		fmt.Sprintf(`  echo Provided CLI %q is not a CLI generated with github.com/leep-frog/command`, aliaser.cli),
		`  return 1`,
		`fi`,
		``,
		``,
	}, "\n"))
}

func (l *linux) RegisterAliaser(output command.Output, a *Aliaser) {
	// Output the bash alias and completion commands
	var qas []string
	for _, v := range a.values {
		qas = append(qas, fmt.Sprintf("%q", v))
	}
	quotedArgs := strings.Join(qas, " ")

	// The trailing space causes issues, so we need to make sure we remove that if necessary.
	aliasTo := strings.TrimSpace(fmt.Sprintf("%s %s", a.cli, quotedArgs))
	output.Stdoutf(strings.Join([]string{
		fmt.Sprintf("alias -- %s=%q", a.alias, aliasTo),
		l.aliaserAutocompleteFunction(a.alias, a.cli, quotedArgs),
		fmt.Sprintf("complete -F _custom_autocomplete_for_alias_%s %s %s", a.alias, NosortString(), a.alias),
		``,
		``,
	}, "\n"))
}

func (*linux) aliaserGlobalAutocompleteFunction() string {
	return strings.Join([]string{
		`function _leep_frog_autocompleter {`,
		fmt.Sprintf("  %s", FileStringFromCLI(`"$1"`)),
		`  local tFile=$(mktemp)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		`  $GOPATH/bin/_${file}_runner autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`,
		`  local IFS='`,
		`';`,
		`  COMPREPLY=( $(cat $tFile) )`,
		`  rm $tFile`,
		`}`,
		``,
	}, "\n")
}

func (*linux) aliaserAutocompleteFunction(alias string, cli string, quotedArg string) string {
	return strings.Join([]string{
		fmt.Sprintf("function _custom_autocomplete_for_alias_%s {", alias),
		fmt.Sprintf(`  _leep_frog_autocompleter %q %s`, cli, quotedArg),
		"}",
		"",
	}, "\n")
}

func (*linux) Mancli(cli string) []string {
	return []string{
		// Extract the custom execute function so that this function
		// can work regardless of file name
		FileStringFromCLI(cli),
		`if [ -z "$file" ]; then`,
		fmt.Sprintf(`  echo %s is not a CLI generated via github.com/leep-frog/command`, cli),
		`  return 1`,
		`fi`,
		fmt.Sprintf(`  "$GOPATH/bin/_${file}_runner" usage %s`, cli),
	}
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
