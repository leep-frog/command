package main

import (
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

type SourcererCommand struct{}

func (*SourcererCommand) Load(string) error { return nil }
func (*SourcererCommand) Setup() []string   { return nil }
func (*SourcererCommand) Changed() bool     { return false }

func (*SourcererCommand) Name() string {
	return "sourcerer"
}

func (*SourcererCommand) Node() *command.Node {
	return command.SerialNodes(
		command.FileNode("DIRECTORY", "Directory in which to create CLI"),
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			command.ExecuteData
		}, nil),
		command.ExecutorNode(func(o command.Output, d *command.Data) error {
			command.ExecuteData
		})
	)
}

func main() {
	sourcerer.Source(&SourcererCommand{})
}
