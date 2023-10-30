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

	// InitializationLogic generates the shell commands to run
	// to initialize the builtin commands. If `lazyLoad` is set
	// to true, then the executables will only be generated if they
	// don't already exist; otherwise, if `lazyLoad` is false,
	// then all executables will be regenerated.
	InitializationLogic(lazyLoad bool, sourceLocationDir string) string

	// FunctionWrap wraps the provided commands in another function.
	FunctionWrap(string) string

	// HandleAutocompleteSuccess should output the suggestions for autocomplete consumption
	HandleAutocompleteSuccess(command.Output, []string)
	// HandleAutocompleteError should output error info on `Autocomplete` failure
	HandleAutocompleteError(output command.Output, compType int, err error)

	//
	SourcererGoCLI(dir string, targetName string) []string

	// RegisterCLIs generates the code for
	RegisterCLIs(builtin bool, goExecutable, targetName string, cli []CLI) ([]string, error)

	// RegisterAliasers
	GlobalAliaserFunc(goExecutable string) []string
	VerifyAliaser(*Aliaser) []string
	RegisterAliaser(goExecutable string, a *Aliaser) []string

	// Mancli returns shell commands that run the usage file
	Mancli(builtin bool, goExecutable, cli string, args ...string) []string
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
