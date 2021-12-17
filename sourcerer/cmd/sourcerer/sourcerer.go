package main

import (
	"fmt"
	"strings"

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
	dName := "DIRECTORY"
	bsName := "BINARY_SUFFIX"
	return command.SerialNodes(
		command.FileNode(dName, "Directory in which to create CLI", command.IsDir()),
		command.StringNode(bsName, "Suffix for the name", command.MinLength(1)),
		command.ExecutableNode(func(_ command.Output, d *command.Data) ([]string, error) {
			dir := strings.ReplaceAll(d.String(dName), `\`, "/")
			return []string{
				fmt.Sprintf("source $(pushd . > /dev/null ; cd %s && go run *.go %s ; popd > /dev/null)", dir, d.String(bsName)),
			}, nil
		}),
	)
}

func main() {
	sourcerer.Source(&SourcererCommand{})
}
