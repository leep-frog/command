package sourcerer

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

var (
	sourcererSourceDirArg = commander.FileArgument("SOURCE_DIRECTORY", "Directory in which to create CLI", commander.IsDir())
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
		sourcererSourceDirArg,
		targetNameArg,
		outputFolderArg,
		commander.ExecutableProcessor(func(_ command.Output, d *command.Data) ([]string, error) {
			return CurrentOS.SourcererGoCLI(sourcererSourceDirArg.Get(d), targetNameArg.Get(d), outputFolderArg.Get(d)), nil
		}),
	)
}
