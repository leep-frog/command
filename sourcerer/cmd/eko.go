package main

import (
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

// Eko is just a simple command to test out windows argument passing
type Eko struct{}

func (*Eko) Setup() []string { return nil }
func (*Eko) Changed() bool   { return false }
func (*Eko) Name() string    { return "eko" }

func EkoAliasers() sourcerer.Option {
	return sourcerer.Aliasers((map[string][]string{
		"ekoOne": {"eko", "uny"},
	}))
}

func (*Eko) Node() command.Node {
	la := command.ListArg[string]("LIST", "", 0, command.UnboundedList,
		command.SimpleDistinctCompleter[[]string]("uny", "deuxy", "troisy"),
	)
	return command.SerialNodes(
		la,
		&command.ExecutorProcessor{func(o command.Output, d *command.Data) error {
			for i, s := range la.Get(d) {
				o.Stdoutf("%2d: %s|\n", i, s)
			}
			return nil
		}},
	)
}
