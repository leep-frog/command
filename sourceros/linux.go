package sourceros

import (
	"fmt"
	"strings"

	"github.com/leep-frog/command"
)

type linux struct{}

func Linux() OS {
	return &linux{}
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

func (l *linux) HandleAutocompleteSuccess(o command.Output, suggestions []string) {
	o.Stdoutf("%s\n", strings.Join(suggestions, "\n"))
}

func (l *linux) HandleAutocompleteError(o command.Output, compType int, err error) {
	// Only display the error if the user is requesting completion via successive tabs (so distinct completions are guaranteed to be displayed)
	if compType == 63 { /* code 63 = '?' character */
		// Add newline so we're outputting stderr on a newline (and not line with cursor)
		o.Stderrf("\n%v", err)
		// Suggest non-overlapping strings (one space and one tab) so COMP_LINE is reprinted
		o.Stdoutf("\t\n \n")
	}
}

func (l *linux) GenerateBinary(sl string, filename string) string {
	return strings.Join([]string{
		"pushd . > /dev/null",
		fmt.Sprintf(`cd "$(dirname %s)"`, sl),
		fmt.Sprintf("go build -o $GOPATH/bin/_%s_runner", filename),
		"popd > /dev/null",
		"",
	}, "\n")
}

func (*linux) AutocompleteFunction(filename string) string {
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

func (*linux) AliaserGlobalAutocompleteFunction() string {
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

func (*linux) AliaserVerify(cli string) string {
	return strings.Join([]string{
		FileStringFromCLI(cli),
		`if [ -z "$file" ]; then`,
		fmt.Sprintf(`  echo Provided CLI %q is not a CLI generated with github.com/leep-frog/command`, cli),
		`  return 1`,
		`fi`,
		``,
		``,
	}, "\n")
}

func (*linux) AliaserAutocompleteFunction(alias string, cli string, quotedArg string) string {
	return strings.Join([]string{
		fmt.Sprintf("function _custom_autocomplete_for_alias_%s {", alias),
		fmt.Sprintf(`  _leep_frog_autocompleter %q %s`, cli, quotedArg),
		"}",
		"",
	}, "\n")
}

func (*linux) ExecuteFileContents(filename string) string {
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

func (*linux) AliasFormat() string {
	return "alias %s='source $GOPATH/bin/_custom_execute_%s %s'"
}

func (*linux) AliasSetupFormat() string {
	return "alias %s='o=$(mktemp) && %s > $o && source $GOPATH/bin/_custom_execute_%s %s $o'"
}

func (*linux) SetupFunctionFormat() string {
	return strings.Join([]string{
		`function %s {`,
		`  %s`,
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
