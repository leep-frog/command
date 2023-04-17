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
	passAlongArgs = command.ListArg[string]("PASSTHROUGH_ARGS", "Args to pass through to the command", 0, command.UnboundedList)
)

func (gl *GoLeep) Aliasers() sourcerer.Option {
	return sourcerer.NewAliaser("gl", gl.Name())
}

func (gl *GoLeep) Name() string {
	return "goleep"
}

func (gl *GoLeep) runCommand(d *command.Data, subCmd, cli string, extraArgs []string) (string, []string) {
	return "go", append([]string{
		"run",
		goDirectory.Get(d),
		subCmd,
		cli,
	}, extraArgs...)
}

// Separate method for testing
var (
	getTmpFile = func() (*os.File, error) {
		return ioutil.TempFile("", "goleep-node-runner")
	}
	goleepCLIArg = command.Arg[string]("CLI", "CLI to use", command.CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
		bc := &command.ShellCommand[[]string]{
			CommandName: "go",
			Args: []string{
				"run",
				goDirectory.Get(d),
				sourcerer.ListBranchName,
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
	usageNode := command.SerialNodes(
		command.Description("Get the usage of the provided go files"),
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			cmd, args := gl.runCommand(d, sourcerer.UsageBranchName, fmt.Sprintf("%q", goleepCLIArg.Get(d)), nil)
			ed.Executable = append(ed.Executable, fmt.Sprintf("%s %s", cmd, strings.Join(args, " ")))
			return nil
		}, nil),
	)

	passAlongArgs.AddOptions(gl.completer())

	dfltNode := command.SerialNodes(
		command.Description("Execute the provided go files"),
		passAlongArgs,
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			f, err := getTmpFile()
			if err != nil {
				return o.Stderrf("failed to create tmp file: %v\n", err)
			}

			// Run the command
			// Need to use ToSlash because mingw
			cmd, args := gl.runCommand(d, sourcerer.ExecuteBranchName, goleepCLIArg.Get(d), append([]string{filepath.ToSlash(f.Name())}, d.StringList(passAlongArgs.Name())...))
			bc := &command.ShellCommand[[]string]{ArgName: "SHELL_OUTPUT", CommandName: cmd, Args: args, ForwardStdout: true}
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

	return command.SerialNodes(
		command.FlagProcessor(goDirectory),
		goleepCLIArg,
		&command.BranchNode{
			Branches: map[string]command.Node{
				"usage": usageNode,
			},
			Default:           dfltNode,
			DefaultCompletion: true,
		},
	)
}

func (gl *GoLeep) completer() command.Completer[[]string] {
	return command.CompleterFromFunc(func(s []string, data *command.Data) (*command.Completion, error) {
		// Add a "dummyCommand" prefix to be removed by the command.Autocomplete function.
		compLine := "dummyCommand " + strings.Join(passAlongArgs.Get(data), " ")
		// TODO: This should also consider the quotes (before input processing). e.g. `abc "def"` should be 9 not 7
		compPoint := fmt.Sprintf("%d", len(compLine))

		extraArgs := []string{
			// COMP_TYPE: by setting to '?', we ensure that an error is always printed.
			// TODO: Get this from data.
			"63",
			// COMP_POINT (-2 for quotes)
			compPoint,
			// COMP_LINE
			compLine,
			// No passthrough args needed since that's only used for aliaser autocomplete
		}
		cmd, args := gl.runCommand(data, sourcerer.AutocompleteBranchName, goleepCLIArg.Get(data), extraArgs)
		bc := &command.ShellCommand[[]string]{ArgName: "SHELL_OUTPUT", CommandName: cmd, Args: args}
		fo := command.NewFakeOutput()
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
