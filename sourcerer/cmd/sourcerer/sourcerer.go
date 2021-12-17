package main

import (
	"fmt"
	"path/filepath"
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
		command.FileNode(dName, "Directory in which to create CLI"),
		command.StringNode(bsName, "Suffix for the name", command.MinLength(1)),
		command.ExecutableNode(func(_ command.Output, d *command.Data) ([]string, error) {
			f := strings.ReplaceAll(filepath.Join(d.String(dName), "*.go"), `\`, "/")
			return []string{
				fmt.Sprintf("source $(go run %s %s)", f, d.String(bsName)),
			}, nil
		}),
	)
}

func main() {
	sourcerer.Source(&SourcererCommand{})
}
