package sourcerer

import (
	"fmt"
	"os"
	"runtime"

	"github.com/leep-frog/command"
)

var (
	oses = []OS{
		Linux(),
		Windows(),
	}

	CurrentOS = func() OS {
		curOS, ok := os.LookupEnv("LEEP_FROG_CLI_OS_OVERRIDE")
		if !ok {
			curOS = runtime.GOOS
		}

		for _, os := range oses {
			if os.Name() == curOS {
				return os
			}
		}
		panic(fmt.Sprintf("Unsupported leep-frog/command os: %q", curOS))
	}()
)

type OS interface {
	command.OS

	// Name is the operating system as specified by runtime.GOOS
	Name() string

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
