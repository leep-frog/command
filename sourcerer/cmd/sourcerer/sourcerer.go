package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

var (
	RelevantPackages = []string{
		"cd",
		"command",
		"emacs",
		"gocli",
		"grep",
		"labelmaker",
		"pdf",
		"replace",
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

func (*UpdateLeepPackageCommand) Node() *command.Node {
	pkg := "PACKAGE"
	return command.SerialNodes(
		command.Description("gg updates go packages from the github.com/leep-frog repository"),
		command.ListArg[string](pkg, "Package name", 1, command.UnboundedList, command.SimpleDistinctCompletor[[]string](RelevantPackages...)),
		command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
			var r []string
			for _, p := range d.StringList(pkg) {
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

func getFile(cli string) string {
	return fmt.Sprintf(`local file="$(type %s | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`, cli)
}

// UsageCommand is a CLI for printing out usage info for a CLI.
type UsageCommand struct{}

func (*UsageCommand) Setup() []string { return nil }
func (*UsageCommand) Changed() bool   { return false }
func (*UsageCommand) Name() string    { return "mancli" }

func (*UsageCommand) Node() *command.Node {
	c := "CLI"
	return command.SerialNodes(
		command.Description("mancli prints out usage info for any leep-frog generated CLI"),
		command.Arg[string](c, "CLI for which usage should be fetched", command.SimpleDistinctCompletor[string](RelevantPackages...)),
		// TODO: This is run before all args are processed. That's confusing if extra args are provided.
		//       We'd expect an ExtraArgsErr, but instead get an error from this function.
		command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
			cli := d.String(c)
			return []string{
				// Extract the custom execute function so that this function
				// can work regardless of file name
				getFile(cli),
				`if [ -z "$file" ]; then`,
				fmt.Sprintf(`  echo %s is not a CLI generated via github.com/leep-frog/command`, cli),
				`  return 1`,
				`fi`,
				fmt.Sprintf(`  "$GOPATH/bin/_${file}_runner" usage %s`, cli),
			}, nil
		}),
	)
}

// AliaserCommand creates an alias for another arg
type AliaserCommand struct{}

func (*AliaserCommand) Setup() []string { return nil }
func (*AliaserCommand) Changed() bool   { return false }
func (*AliaserCommand) Name() string    { return "aliaser" }

func (*AliaserCommand) Node() *command.Node {
	a := "ALIAS"
	c := "CLI"
	pts := "PASSTHROUGH_ARG"
	return command.SerialNodes(
		command.Description("Alias a command to a cli with some args included"),
		command.Arg[string](a, "Alias of new command", command.MinLength(1)),
		command.Arg[string](c, "CLI of new command"),
		command.ListArg[string](pts, "Args to passthrough with alias", 0, command.UnboundedList),
		command.ExecutableNode(func(_ command.Output, d *command.Data) ([]string, error) {
			alias := d.String(a)
			cli := d.String(c)
			var qas []string
			for _, pt := range d.StringList(pts) {
				qas = append(qas, fmt.Sprintf("%q", pt))
			}
			quotedArgs := strings.Join(qas, " ")
			aliasTo := fmt.Sprintf("%s %s", cli, quotedArgs)
			return []string{
				// TODO: check that it's a leep-frog command
				getFile(cli),
				`if [ -z "$file" ]; then`,
				`  echo Provided CLI is not a CLI generated with github.com/leep-frog/command`,
				`  return 1`,
				`fi`,
				fmt.Sprintf("alias -- %s=%q", alias, aliasTo),
				fmt.Sprintf(sourcerer.AutocompleteForAliasFunction, alias, cli, cli, quotedArgs),
				fmt.Sprintf("complete -F _custom_autocomplete_for_alias_%s %s %s", alias, sourcerer.NosortString(), alias),
			}, nil
		}),
	)
}

type SourcererCommand struct{}

func (*SourcererCommand) Setup() []string { return nil }
func (*SourcererCommand) Changed() bool   { return false }

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
				`local tmpFile="$(mktemp)"`,
				fmt.Sprintf("go run . %s > $tmpFile && source $tmpFile ", d.String(bsName)),
				"popd > /dev/null",
			}, nil
		}),
	)
}

// GoLeep is a CLI that runs command nodes that are defined in "main" packages.
type GoLeep struct{}

var (
	goDirectory = command.NewFlag[string](
		"go-dir",
		'd',
		"Directory of package to run",
		command.IsDir(),
		&command.FileCompletor[string]{IgnoreFiles: true},
		command.Default("."),
	)
	passAlongArgs = command.ListArg[string](command.PassthroughArgs, "Args to pass through to the command", 0, command.UnboundedList)
)

func (gl *GoLeep) Name() string {
	return "goleep"
}

func (gl *GoLeep) runCommand(d *command.Data, subCmd string, extraArgs []string) []string {
	var ea string
	if len(extraArgs) > 0 {
		ea = fmt.Sprintf(" %s", strings.Join(extraArgs, " "))
	}

	return []string{
		fmt.Sprintf("go run %s %s%s", d.String(goDirectory.Name()), subCmd, ea),
	}
}

// Separate method for testing
var (
	getTmpFile = func() (*os.File, error) {
		return ioutil.TempFile("", "goleep-node-runner")
	}
)

func (gl *GoLeep) Load(json string) error { return nil }
func (gl *GoLeep) Changed() bool          { return false }
func (gl *GoLeep) Setup() []string        { return nil }
func (gl *GoLeep) Node() *command.Node {
	usageNode := command.SerialNodes(
		command.Description("Get the usage of the provided go files"),
		command.NewFlagNode(goDirectory),
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			ed.Executable = gl.runCommand(d, "usage", nil)
			return nil
		}, nil),
	)

	passAlongArgs.AddOptions(gl.completor())

	exNode := command.SerialNodes(
		command.Description("Execute the provided go files"),
		command.NewFlagNode(goDirectory),
		passAlongArgs,
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			f, err := getTmpFile()
			if err != nil {
				return o.Stderrf("failed to create tmp file: %v", err)
			}

			// Run the command
			// Need to use ToSlash because mingw
			cmd := gl.runCommand(d, "execute", append([]string{filepath.ToSlash(f.Name())}, d.StringList(passAlongArgs.Name())...))
			bc := command.NewBashCommand("BASH_OUTPUT", cmd, command.ForwardStdout[[]string]())
			if _, err := bc.Run(o); err != nil {
				return o.Stderrf("failed to run bash script: %v", err)
			}

			b, err := ioutil.ReadFile(f.Name())
			f.Close()
			if err != nil {
				return o.Stderrf("failed to read temporary file: %v", err)
			}

			// Add the eData from the previous file to this one's
			for _, line := range strings.Split(string(b), "\n") {
				if line != "" {
					ed.Executable = append(ed.Executable, line)
				}
			}

			if err := os.Remove(f.Name()); err != nil {
				o.Stderrf("failed to delete temporary file: %v", err)
			}

			return nil
		}, nil),
	)

	return command.BranchNode(map[string]*command.Node{
		"usage": usageNode,
	}, exNode, command.DontCompleteSubcommands())
}

func (gl *GoLeep) completor() command.Completor[[]string] {
	return command.CompletorFromFunc(func(s []string, data *command.Data) (*command.Completion, error) {
		extraArgs := []string{
			fmt.Sprintf("%q", strings.Join(passAlongArgs.Get(data), " ")),
		}
		bc := command.NewBashCommand("BASH_OUTPUT", gl.runCommand(data, "autocomplete", extraArgs), command.HideStderr[[]string]())
		v, err := bc.Run(nil)
		if err != nil {
			return nil, err
		}
		return &command.Completion{
			Suggestions: v,
		}, nil
	})
}

func main() {
	os.Exit(sourcerer.Source([]sourcerer.CLI{
		&SourcererCommand{},
		&UpdateLeepPackageCommand{},
		&UsageCommand{},
		&AliaserCommand{},
		&GoLeep{},
	}))
}
