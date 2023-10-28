package sourcerer

import (
	"github.com/leep-frog/command"
)

// UsageCommand is a CLI for printing out usage info for a CLI.
type UsageCommand struct{}

var (
	usageCLIArg     = command.Arg[string]("CLI", "CLI for which usage should be fetched", command.SimpleDistinctCompleter[string](RelevantPackages...))
	extraMancliArgs = command.ListArg[string]("ARGS", "Additional args to consider and traverse through when generating the usage doc", 0, command.UnboundedList)
)

func (*UsageCommand) Setup() []string { return nil }
func (*UsageCommand) Changed() bool   { return false }
func (*UsageCommand) Name() string    { return "mancli" }

func (*UsageCommand) Node() command.Node {
	return command.SerialNodes(
		command.Description("mancli prints out usage info for any leep-frog generated CLI"),
		usageCLIArg,
		extraMancliArgs,
		command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
			cli := usageCLIArg.Get(d)
			return CurrentOS.Mancli(cli, extraMancliArgs.Get(d)...), nil
		}),
	)
}
