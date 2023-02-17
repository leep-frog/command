package main

import (
	"fmt"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
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
			return []string{
				// Extract the custom execute function so that this function
				// can work regardless of file name
				sourcerer.FileStringFromCLI(cli),
				`if [ -z "$file" ]; then`,
				fmt.Sprintf(`  echo %s is not a CLI generated via github.com/leep-frog/command`, cli),
				`  return 1`,
				`fi`,
				fmt.Sprintf(`  "$GOPATH/bin/_${file}_runner" usage %s`, cli),
			}, nil
		}),
	)
}
