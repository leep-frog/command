package sourcerer

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

var (
	sourcererDirArg    = commander.FileArgument("DIRECTORY", "Directory in which to create CLI", commander.IsDir())
	sourcererSuffixArg = commander.Arg[string]("BINARY_SUFFIX", "Suffix for the name", commander.MinLength[string, string](1))
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
	return commander.SerialNodes(
		sourcererDirArg,
		sourcererSuffixArg,
		commander.ExecutableProcessor(func(_ command.Output, d *command.Data) ([]string, error) {
			return CurrentOS.SourcererGoCLI(sourcererDirArg.Get(d), sourcererSuffixArg.Get(d)), nil
		}),
	)
}
