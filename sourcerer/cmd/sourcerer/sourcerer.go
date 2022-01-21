package main

import (
	"fmt"
	"os"
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
		command.Arg[string](bsName, "Suffix for the name", command.MinLength(1)),
		command.ExecutableNode(func(_ command.Output, d *command.Data) ([]string, error) {
			dir := strings.ReplaceAll(d.String(dName), `\`, "/")
			// TODO: try using this? filepath.FromSlash()
			return []string{
				"pushd . > /dev/null",
				fmt.Sprintf("cd %s", dir),
				`tmpFile="$(mktemp)"`,
				fmt.Sprintf("go run *.go %s > $tmpFile && source $tmpFile ", d.String(bsName)),
				"popd > /dev/null",
			}, nil
		}),
	)
}

func main() {
	os.Exit(sourcerer.Source(&SourcererCommand{}))
}
