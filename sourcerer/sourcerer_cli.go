package sourcerer

import (
	"fmt"

	"github.com/leep-frog/command"
)

var (
	sourcererDirArg      = command.FileArgument("DIRECTORY", "Directory in which to create CLI", command.IsDir())
	sourcererSuffixArg   = command.Arg[string]("BINARY_SUFFIX", "Suffix for the name", command.MinLength[string, string](1))
	externalLoadOnlyFlag = command.BoolValueFlag(loadOnlyFlag.Name(), loadOnlyFlag.ShortName(), loadOnlyFlag.Desc(), fmt.Sprintf("--%s", loadOnlyFlag.Name()))
)

func SourcererCLI() CLI {
	return &SourcererCommand{}
}

// SourcererCommand is a command that creates CLIs from main files. Use the `SourcererCLI` function to initialize.
type SourcererCommand struct{}

func (*SourcererCommand) Setup() []string { return nil }
func (*SourcererCommand) Changed() bool   { return false }

func (*SourcererCommand) Name() string {
	return "sourcerer"
}

func (*SourcererCommand) Node() command.Node {
	return command.SerialNodes(
		command.FlagProcessor(
			externalLoadOnlyFlag,
		),
		sourcererDirArg,
		sourcererSuffixArg,
		command.ExecutableProcessor(func(_ command.Output, d *command.Data) ([]string, error) {
			return []string{
				"pushd . > /dev/null",
				fmt.Sprintf("cd %q", sourcererDirArg.Get(d)),
				`local tmpFile="$(mktemp)"`,
				fmt.Sprintf("go run . source %q %s > $tmpFile && source $tmpFile ", sourcererSuffixArg.Get(d), externalLoadOnlyFlag.Get(d)),
				"popd > /dev/null",
			}, nil
		}),
	)
}
