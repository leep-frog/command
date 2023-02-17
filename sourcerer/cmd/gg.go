package main

import (
	"fmt"

	"github.com/leep-frog/command"
)

var (
	RelevantPackages = []string{
		"cd",
		"command",
		"differ",
		"emacs",
		"gocli",
		"grep",
		"labelmaker",
		"notification",
		"pdf",
		"replace",
		"ssh",
		"sourcecontrol",
		"todo",
		"workspace",
	}
)

// UpdateLeepPackageCommand is a CLI for updating github.com/leep-frog packages
type UpdateLeepPackageCommand struct{}

func (*UpdateLeepPackageCommand) Setup() []string { return nil }
func (*UpdateLeepPackageCommand) Changed() bool   { return false }
func (*UpdateLeepPackageCommand) Name() string {
	// gg: "go get"
	return "gg"
}

var (
	packageArg = command.ListArg[string]("PACKAGE", "Package name", 1, command.UnboundedList, command.SimpleDistinctCompleter[[]string](RelevantPackages...))
)

func (*UpdateLeepPackageCommand) Node() command.Node {
	return command.SerialNodes(
		command.Description("gg updates go packages from the github.com/leep-frog repository"),
		packageArg,
		command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
			var r []string
			for _, p := range packageArg.Get(d) {
				r = append(r,
					fmt.Sprintf(`local commitSha="$(git ls-remote git@github.com:leep-frog/%s.git | grep ma[is][nt] | awk '{print $1}')"`, p),
					fmt.Sprintf(`go get -v "github.com/leep-frog/%s@$commitSha"`, p),
					// else:
					// fmt.Sprintf(`go get -u "github.com/leep-frog/%s"`, p),
				)
			}
			return r, nil
		}),
	)
}
