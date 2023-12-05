package sourcerer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandtest"
)

// GoLeep is a CLI that runs command nodes that are defined in "main" packages.
type GoLeep struct{}

var (
	goDirectory = commander.Flag[string](
		"go-dir",
		'd',
		"Directory of package to run",
		commander.IsDir(),
		&commander.FileCompleter[string]{IgnoreFiles: true},
		commander.Default(""),
	)
	passAlongArgs = commander.ListArg[string]("PASSTHROUGH_ARGS", "Args to pass through to the command", 0, command.UnboundedList)
)

func (gl *GoLeep) Aliasers() Option {
	return NewAliaser("gl", gl.Name())
}

func (gl *GoLeep) Name() string {
	return "goleep"
}

func runCommand[T any](d *command.Data, subCmd, cli string, extraArgs []string) *commander.ShellCommand[T] {
	return &commander.ShellCommand[T]{
		CommandName: "go",
		Dir:         goDirectory.Get(d),
		Args: append([]string{
			"run",
			".", // directory is set by ShellCommand.Dir
			subCmd,
			cli,
		}, extraArgs...),
	}
}

// Separate method for testing
var (
	getTmpFile = func() (*os.File, error) {
		return ioutil.TempFile("", "goleep-node-runner")
	}
	goleepCLIArg = commander.Arg[string]("CLI", "CLI to use", commander.CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
		bc := &commander.ShellCommand[[]string]{
			CommandName: "go",
			Dir:         goDirectory.Get(d),
			Args: []string{
				"run",
				".",
				ListBranchName,
			},
			ForwardStdout: false,
			HideStderr:    true,
		}
		resp, err := bc.Run(nil, d)
		if err != nil {
			return nil, fmt.Errorf("failed to run shell script: %v\n", err)
		}
		return &command.Completion{
			Suggestions: resp,
		}, nil
	}))
)

func (gl *GoLeep) Changed() bool   { return false }
func (gl *GoLeep) Setup() []string { return nil }
func (gl *GoLeep) Node() command.Node {
	usageNode := commander.SerialNodes(
		commander.Description("Get the usage of the provided go files"),
		commander.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			sc := runCommand[[]string](d, UsageBranchName, fmt.Sprintf("%q", goleepCLIArg.Get(d)), nil)
			sc.ForwardStdout = true
			_, err := sc.Run(o, d)
			return o.Annotatef(err, "failed to run goleep usage command")
		}, nil),
	)

	passAlongArgs.AddOptions(gl.completer())

	dfltNode := commander.SerialNodes(
		commander.Description("Execute the provided go files"),
		passAlongArgs,
		commander.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			f, err := getTmpFile()
			if err != nil {
				return o.Stderrf("failed to create tmp file: %v\n", err)
			}

			// Run the command
			// Need to use ToSlash because mingw
			bc := runCommand[[]string](d, ExecuteBranchName, goleepCLIArg.Get(d), append([]string{filepath.ToSlash(f.Name())}, d.StringList(passAlongArgs.Name())...))
			bc.ArgName = "SHELL_OUTPUT"
			bc.ForwardStdout = true
			if _, err := bc.Run(o, d); err != nil {
				return o.Stderrf("failed to run shell script: %v\n", err)
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

	return commander.SerialNodes(
		commander.FlagProcessor(goDirectory),
		goleepCLIArg,
		&commander.BranchNode{
			Branches: map[string]command.Node{
				"usage": usageNode,
			},
			Default:           dfltNode,
			DefaultCompletion: true,
		},
	)
}

func (gl *GoLeep) completer() commander.Completer[[]string] {
	return commander.CompleterFromFunc(func(s []string, data *command.Data) (*command.Completion, error) {
		// Add a "dummyCommand" prefix to be removed by the commander.Autocomplete function.
		compLine := "dummyCommand " + strings.Join(passAlongArgs.Get(data), " ")
		compPoint := fmt.Sprintf("%d", len(compLine))

		extraArgs := []string{
			// COMP_TYPE: by setting to '?', we ensure that an error is always printed.
			"63",
			// COMP_POINT
			compPoint,
			// COMP_LINE
			compLine,
			// No passthrough args needed since that's only used for aliaser autocomplete
		}
		bc := runCommand[[]string](data, AutocompleteBranchName, goleepCLIArg.Get(data), extraArgs)
		bc.ArgName = "SHELL_OUTPUT"
		fo := commandtest.NewOutput()
		v, err := bc.Run(fo, data)
		fo.Close()
		if err != nil {
			stderr := fo.GetStderr()
			if stderr != "" {
				stderr = fmt.Sprintf("\n\nStderr:\n%s", stderr)
			}
			return nil, fmt.Errorf("failed to run goleep completion: %v%s", err, stderr)
		}
		return &command.Completion{
			Suggestions: v,
		}, nil
	})
}
