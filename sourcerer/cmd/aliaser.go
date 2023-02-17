package main

import (
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

// AliaserCommand creates an alias for another arg
type AliaserCommand struct{}

var (
	aliasArg    = command.Arg[string]("ALIAS", "Alias of new command", command.MinLength[string, string](1))
	aliasCLIArg = command.Arg[string]("CLI", "CLI of new command")
	aliasPTArg  = command.ListArg[string]("PASSTHROUGH_ARGS", "Args to passthrough with alias", 0, command.UnboundedList)
)

func (*AliaserCommand) Setup() []string { return nil }
func (*AliaserCommand) Changed() bool   { return false }
func (*AliaserCommand) Name() string    { return "aliaser" }

func (*AliaserCommand) Node() command.Node {
	return command.SerialNodes(
		command.Description("Alias a command to a cli with some args included"),
		aliasArg,
		aliasCLIArg,
		aliasPTArg,
		command.ExecutableProcessor(func(_ command.Output, d *command.Data) ([]string, error) {
			aliaser := sourcerer.NewAliaser(aliasArg.Get(d), aliasCLIArg.Get(d), aliasPTArg.Get(d)...)
			fo := command.NewFakeOutput()
			sourcerer.AliasSourcery(fo, aliaser)
			fo.Close()
			return strings.Split(fo.GetStdout(), "\n"), nil
		}),
	)
}
