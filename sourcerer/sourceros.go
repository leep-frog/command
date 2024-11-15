package sourcerer

import (
	"fmt"
	"os"
	"runtime"

	"github.com/leep-frog/command/command"
)

var (
	oses = []OS{
		Linux(),
		Windows(),
	}

	CurrentOS = func() OS {
		curOS, ok := os.LookupEnv("COMMAND_CLI_OS_OVERRIDE")
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

	// ExecutableFileSuffix is the suffix to add for executable files (e.g. `.exe` in Windows)
	ExecutableFileSuffix() string

	// SourceableFile is the file basename to use for sourceable files (e.g. `myCLI_loader.sh` in linux)
	SourceableFile(target string) string

	// SourceSetup returns the code that should be run and/or added to a user's terminal profile
	// after successfully running `go run . source ...`
	SourceSetup(sourceableFile, targetName, goRunSourceCommand, userDir string) []string

	// FunctionWrap wraps the provided commands in another function.
	FunctionWrap(name, fn string) string

	// HandleAutocompleteSuccess should output the suggestions for autocomplete consumption
	HandleAutocompleteSuccess(command.Output, *command.Autocompletion)
	// HandleAutocompleteError should output error info on `Autocomplete` failure
	HandleAutocompleteError(output command.Output, compType int, err error)

	// SourceFileCommand should return the command to source the provided file.
	SourceFileCommand(sourcerersDir, targetName string) string

	// RecursiveCopyDir should return the command to recusrively copy contents from src to dst.
	RecursiveCopyDir(src, dst string) string

	// RegisterCLIs generates the code for
	RegisterCLIs(builtin bool, goExecutable, targetName string, cli []CLI) []string

	// RegisterRunCLIAutocomplete generates the shell code to source to set up
	// autocomplete for the provided alias.
	RegisterRunCLIAutocomplete(goExecutable, alias string) []string

	// RegisterAliasers
	GlobalAliaserFunc(goExecutable string) []string
	VerifyAliaser(*Aliaser) []string
	RegisterAliaser(goExecutable string, a *Aliaser) []string
}

// ValueByOS will return the value that is associated
// with the current OS. If there is no match, then the
// function will panic.
func ValueByOS[T any](values map[string]T) T {
	if v, ok := values[CurrentOS.Name()]; ok {
		return v
	}
	panic(fmt.Sprintf("No value provided for the current OS (%s)", CurrentOS.Name()))
}
