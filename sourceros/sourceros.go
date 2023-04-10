package sourceros

import "github.com/leep-frog/command"

var (
	Current = func() OS {
		return Linux()
	}()
)

type OS interface {
	// FunctionWrap wraps the provided commands in another function.
	FunctionWrap(string) string

	// HandleAutocompleteSuccess should output the suggestions for autocomplete consumption
	HandleAutocompleteSuccess(command.Output, []string)
	// HandleAutocompleteError should output error info on `Autocomplete` failure
	HandleAutocompleteError(output command.Output, compType int, err error)

	// GenerateBinary should cd into the directory of the provided file,
	// build the go file with the provided target name.
	GenerateBinary(sl string, filename string) string
	// AutocompleteFunction defines a function for CLI autocompletion.
	AutocompleteFunction(filename string) string
	// AliaserGlobalAutocompleteFunction
	AliaserGlobalAutocompleteFunction() string
	// AliaserVerify verifies the provided alias is actually a groog alias
	AliaserVerify(cli string) string
	// AliaserAutocompleteFunction defines a function for CLI autocompletion for aliased commands.
	AliaserAutocompleteFunction(alias string, cli string, quotedArg string) string
	// ExecuteFileContents defines a function for CLI execution.
	ExecuteFileContents(filename string) string
	// SetupFunction is used to run setup functions prior to a CLI command execution.
	SetupFunctionFormat() string
	// AliasFormat is an alias definition template for commands that don't require a setup function.
	AliasFormat() string
	// AliasSetupFormat is an alias definition template for commands that require a setup function.
	AliasSetupFormat() string
	// Mancli returns shell commands that run the usage file
	Mancli(cli string) []string
}
