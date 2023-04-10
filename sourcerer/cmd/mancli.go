package main

import (
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourceros"
)

// UsageCommand is a CLI for printing out usage info for a CLI.
type UsageCommand struct{}

var (
	usageCLIArg = command.Arg[string]("CLI", "CLI for which usage should be fetched", command.SimpleDistinctCompleter[string](RelevantPackages...))
)

func (*UsageCommand) Setup() []string { return nil }
func (*UsageCommand) Changed() bool   { return false }
func (*UsageCommand) Name() string    { return "mancli" }

func (*UsageCommand) Node() command.Node {
	return command.SerialNodes(
		command.Description("mancli prints out usage info for any leep-frog generated CLI"),
		usageCLIArg,
		command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
			cli := usageCLIArg.Get(d)
			return sourceros.Current.Mancli(cli), nil
		}),
	)
}
