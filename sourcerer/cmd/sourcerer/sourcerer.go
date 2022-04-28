package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

// UpdateLeepPackageCommand is a CLI for updating github.com/leep-frog packages
type UpdateLeepPackageCommand struct{}

func (*UpdateLeepPackageCommand) UnmarshalJSON([]byte) error { return nil }
func (*UpdateLeepPackageCommand) Setup() []string            { return nil }
func (*UpdateLeepPackageCommand) Changed() bool              { return false }

func (*UpdateLeepPackageCommand) Name() string {
	// gg: "go get"
	return "gg"
}

func (*UpdateLeepPackageCommand) Node() *command.Node {
	pkg := "PACKAGE"
	return command.SerialNodes(
		command.Description("gg updates go packages from the github.com/leep-frog repository"),
		command.ListArg[string](pkg, "Package name", 1, command.UnboundedList),
		command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
			var r []string
			for _, p := range d.StringList(pkg) {
				r = append(r,
					fmt.Sprintf(`commitSha="$(git ls-remote git@github.com:leep-frog/%s.git | grep ma[is][nt] | awk '{print $1}')"`, p),
					fmt.Sprintf(`go get -v "github.com/leep-frog/%s@$commitSha"`, p),
					// else:
					// fmt.Sprintf(`go get -u "github.com/leep-frog/%s"`, p),
				)
			}
			return r, nil
		}),
	)
}

/*type AliaserCommand struct{}

func (*AliaserCommand) UnmarshalJSON([]byte) error { return nil }
func (*AliaserCommand) Setup() []string            { return nil }
func (*AliaserCommand) Changed() bool              { return false }

func (*AliaserCommand) Name() string {
	return "aliaser"
}

func (*AliaserCommand) Node() *command.Node {
	dName := ""
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
				fmt.Sprintf("go run . %s > $tmpFile && source $tmpFile ", d.String(bsName)),
				"popd > /dev/null",
			}, nil
		}),
	)
}*/

type SourcererCommand struct{}

func (*SourcererCommand) UnmarshalJSON([]byte) error { return nil }
func (*SourcererCommand) Setup() []string            { return nil }
func (*SourcererCommand) Changed() bool              { return false }

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
				fmt.Sprintf("go run . %s > $tmpFile && source $tmpFile ", d.String(bsName)),
				"popd > /dev/null",
			}, nil
		}),
	)
}

func main() {
	os.Exit(sourcerer.Source(
		&SourcererCommand{},
		&UpdateLeepPackageCommand{},
	))
}
