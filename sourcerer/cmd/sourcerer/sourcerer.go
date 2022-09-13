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
		"notification",
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

var (
	packageArg = command.ListArg[string]("PACKAGE", "Package name", 1, command.UnboundedList, command.SimpleDistinctCompleter[[]string](RelevantPackages...))
)

func (*UpdateLeepPackageCommand) Node() *command.Node {
	return command.SerialNodes(
		command.Description("gg updates go packages from the github.com/leep-frog repository"),
		packageArg,
		command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
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

// UsageCommand is a CLI for printing out usage info for a CLI.
type UsageCommand struct{}

var (
	usageCLIArg = command.Arg[string]("CLI", "CLI for which usage should be fetched", command.SimpleDistinctCompleter[string](RelevantPackages...))
)

func (*UsageCommand) Setup() []string { return nil }
func (*UsageCommand) Changed() bool   { return false }
func (*UsageCommand) Name() string    { return "mancli" }

func (*UsageCommand) Node() *command.Node {
	return command.SerialNodes(
		command.Description("mancli prints out usage info for any leep-frog generated CLI"),
		usageCLIArg,
		command.ExecutableNode(func(o command.Output, d *command.Data) ([]string, error) {
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

// AliaserCommand creates an alias for another arg
type AliaserCommand struct{}

var (
	aliasArg    = command.Arg[string]("ALIAS", "Alias of new command", command.MinLength(1))
	aliasCLIArg = command.Arg[string]("CLI", "CLI of new command")
	aliasPTArg  = command.ListArg[string]("PASSTHROUGH_ARGS", "Args to passthrough with alias", 0, command.UnboundedList)
)

func (*AliaserCommand) Setup() []string { return nil }
func (*AliaserCommand) Changed() bool   { return false }
func (*AliaserCommand) Name() string    { return "aliaser" }

func (*AliaserCommand) Node() *command.Node {
	return command.SerialNodes(
		command.Description("Alias a command to a cli with some args included"),
		aliasArg,
		aliasCLIArg,
		aliasPTArg,
		command.ExecutableNode(func(_ command.Output, d *command.Data) ([]string, error) {
			aliaser := sourcerer.NewAliaser(aliasArg.Get(d), aliasCLIArg.Get(d), aliasPTArg.Get(d)...)
			fo := command.NewFakeOutput()
			sourcerer.AliasSourcery(fo, aliaser)
			fo.Close()
			return strings.Split(fo.GetStdout(), "\n"), nil
		}),
	)
}

type SourcererCommand struct{}

func (*SourcererCommand) Setup() []string { return nil }
func (*SourcererCommand) Changed() bool   { return false }

func (*SourcererCommand) Name() string {
	return "sourcerer"
}

var (
	sourcererDirArg    = command.FileNode("DIRECTORY", "Directory in which to create CLI", command.IsDir())
	sourcererSuffixArg = command.Arg[string]("BINARY_SUFFIX", "Suffix for the name", command.MinLength(1))
)

func (*SourcererCommand) Node() *command.Node {
	return command.SerialNodes(
		sourcererDirArg,
		sourcererSuffixArg,
		command.ExecutableNode(func(_ command.Output, d *command.Data) ([]string, error) {
			return []string{
				"pushd . > /dev/null",
				fmt.Sprintf("cd %q", sourcererDirArg.Get(d)),
				`local tmpFile="$(mktemp)"`,
				fmt.Sprintf("go run . %q > $tmpFile && source $tmpFile ", sourcererSuffixArg.Get(d)),
				"popd > /dev/null",
			}, nil
		}),
	)
}

// GoLeep is a CLI that runs command nodes that are defined in "main" packages.
type GoLeep struct{}

var (
	goDirectory = command.Flag[string](
		"go-dir",
		'd',
		"Directory of package to run",
		command.IsDir(),
		&command.FileCompleter[string]{IgnoreFiles: true},
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
		command.FlagNode(goDirectory),
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			ed.Executable = gl.runCommand(d, "usage", nil)
			return nil
		}, nil),
	)

	passAlongArgs.AddOptions(gl.completer())

	exNode := command.SerialNodes(
		command.Description("Execute the provided go files"),
		command.FlagNode(goDirectory),
		passAlongArgs,
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			f, err := getTmpFile()
			if err != nil {
				return o.Stderrf("failed to create tmp file: %v\n", err)
			}

			// Run the command
			// Need to use ToSlash because mingw
			cmd := gl.runCommand(d, "execute", append([]string{filepath.ToSlash(f.Name())}, d.StringList(passAlongArgs.Name())...))
			bc := command.NewBashCommand("BASH_OUTPUT", cmd, command.ForwardStdout[[]string]())
			if _, err := bc.Run(o, d); err != nil {
				return o.Stderrf("failed to run bash script: %v\n", err)
			}

			b, err := ioutil.ReadFile(f.Name())
			f.Close()
			if err != nil {
				return o.Stderrf("failed to read temporary file: %v\n", err)
			}

			// Add the eData from the previous file to this one's
			for _, line := range strings.Split(string(b), "\n") {
				if line != "" {
					ed.Executable = append(ed.Executable, line)
				}
			}

			if err := os.Remove(f.Name()); err != nil {
				o.Stderrf("failed to delete temporary file: %v\n", err)
			}

			return nil
		}, nil),
	)

	return command.AsNode(&command.BranchNode{
		Branches: map[string]*command.Node{
			"usage": usageNode,
		},
		Default: exNode, DefaultCompletion: true,
	})
}

func (gl *GoLeep) completer() command.Completer[[]string] {
	return command.CompleterFromFunc(func(s []string, data *command.Data) (*command.Completion, error) {
		extraArgs := []string{
			fmt.Sprintf("%q", strings.Join(passAlongArgs.Get(data), " ")),
		}
		bc := command.NewBashCommand("BASH_OUTPUT", gl.runCommand(data, "autocomplete", extraArgs), command.HideStderr[[]string]())
		v, err := bc.Run(nil, data)
		if err != nil {
			return nil, err
		}
		return &command.Completion{
			Suggestions: v,
		}, nil
	})
}

type Debugger struct{}

func (*Debugger) Setup() []string { return nil }
func (*Debugger) Changed() bool   { return false }
func (*Debugger) Name() string    { return "leep_debug" }

func (*Debugger) Node() *command.Node {
	return command.SerialNodes(
		// Get the environment variable
		command.EnvArg(command.DebugEnvVar),
		// Either set or unset the environment variable.
		command.IfElseData(
			command.DebugEnvVar,
			&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
				command.OSUnsetenv(command.DebugEnvVar)
				o.Stdoutln("Exiting debug mode.")
				return nil
			}},
			&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
				command.OSSetenv(command.DebugEnvVar, "1")
				o.Stdoutln("Entering debug mode.")
				return nil
			}},
		),
	)
}

func main() {
	os.Exit(sourcerer.Source([]sourcerer.CLI{
		&SourcererCommand{},
		&UpdateLeepPackageCommand{},
		&UsageCommand{},
		&AliaserCommand{},
		&GoLeep{},
		&Debugger{},
	}))
}
