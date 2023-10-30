package sourcerer

import (
	"github.com/leep-frog/command"
)

var (
	sourcererDirArg    = command.FileArgument("DIRECTORY", "Directory in which to create CLI", command.IsDir())
	sourcererSuffixArg = command.Arg[string]("BINARY_SUFFIX", "Suffix for the name", command.MinLength[string, string](1))
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
		sourcererDirArg,
		sourcererSuffixArg,
		command.ExecutableProcessor(func(_ command.Output, d *command.Data) ([]string, error) {
			return CurrentOS.SourcererGoCLI(sourcererDirArg.Get(d), sourcererSuffixArg.Get(d)), nil
		}),
	)
}
