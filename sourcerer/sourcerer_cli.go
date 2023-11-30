package sourcerer

import (
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commondels"
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

func (*SourcererCommand) Node() commondels.Node {
	return commander.SerialNodes(
		sourcererDirArg,
		sourcererSuffixArg,
		commander.ExecutableProcessor(func(_ commondels.Output, d *commondels.Data) ([]string, error) {
			return CurrentOS.SourcererGoCLI(sourcererDirArg.Get(d), sourcererSuffixArg.Get(d)), nil
		}),
	)
}
