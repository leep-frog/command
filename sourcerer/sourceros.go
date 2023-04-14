package sourcerer

import (
	"github.com/leep-frog/command"
)

var (
	CurrentOS = func() OS {
		// return Linux()
		return Windows()
	}()
)

type OS interface {
	// FunctionWrap wraps the provided commands in another function.
	FunctionWrap(string) string

	// HandleAutocompleteSuccess should output the suggestions for autocomplete consumption
	HandleAutocompleteSuccess(command.Output, []string)
	// HandleAutocompleteError should output error info on `Autocomplete` failure
	HandleAutocompleteError(output command.Output, compType int, err error)

	// CreateGoFiles builds the executable files needed for this script. Generally of the format:
	// `pushd . && cd ${sourceLocation} && go build -o /some/path/_${targetName}_runner && popd`
	CreateGoFiles(sourceLocation string, targetName string) string

	//
	SourcererGoCLI(dir string, targetName string, loadOnly string) []string

	// RegisterCLIs
	RegisterCLIs(output command.Output, targetName string, cli []CLI) error

	// RegisterAliasers
	GlobalAliaserFunc(command.Output)
	VerifyAliaser(command.Output, *Aliaser)
	RegisterAliaser(command.Output, *Aliaser)

	// Mancli returns shell commands that run the usage file
	Mancli(cli string) []string
}
