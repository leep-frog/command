package sourcerer

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/leep-frog/command"
)

type GoLeep struct{}

var (
	goFileRegex = regexp.MustCompile(`\.go$`)
	goFilesFlag = command.StringListFlag("GO_FILES", 'f', "Go files to run", 1, command.UnboundedList, command.ListUntilNotRegex(goFileRegex), &command.Completor{
		SuggestionFetcher: &command.FileFetcher{
			Distinct: true,
			Regexp:   goFileRegex,
		},
	})
	passAlongArgs = command.StringListNode("PASSTHROUGH_ARGS", "Args to pass through to the command", 0, command.UnboundedList)
)

func (gl *GoLeep) Name() string {
	return "goleep"
}

func (gl *GoLeep) goFiles(d *command.Data) []string {
	if d.HasArg(goFilesFlag.Name()) {
		return d.StringList(goFilesFlag.Name())
	}
	return []string{"*.go"}
}

func (gl *GoLeep) runCommand(d *command.Data, subCmd string, extraArgs []string) []string {
	args := append([]string{"go", "run"}, gl.goFiles(d)...)
	args = append(args, subCmd)
	return []string{strings.Join(append(args, extraArgs...), " ")}
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
		command.NewFlagNode(goFilesFlag),
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			ed.Executable = gl.runCommand(d, "usage", nil)
			return nil
		}, nil),
	)

	passAlongArgs.AddOptions(&command.Completor{
		SuggestionFetcher: &goleepFetcher{gl},
	})

	exNode := command.SerialNodes(
		command.Description("Execute the provided go files"),
		command.NewFlagNode(goFilesFlag),
		passAlongArgs,
		command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			f, err := getTmpFile()
			if err != nil {
				return o.Stderrf("failed to create tmp file: %v", err)
			}

			// Run the command
			cmd := gl.runCommand(d, "execute", append([]string{f.Name()}, d.StringList(passAlongArgs.Name())...))
			bc := command.BashCommand(command.StringListType, "BASH_OUTPUT", cmd)
			v, err := bc.Run(o)
			if err != nil {
				return nil
			}
			// Just pass input through
			// TODO: add option to do this in bc.Run
			for _, line := range v.ToStringList() {
				o.Stdout(line)
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
	}, exNode, false)
}

type goleepFetcher struct {
	gl *GoLeep
}

// TODO: change fetch interface function to return error as well so it can be tested
func (glf *goleepFetcher) Fetch(v *command.Value, data *command.Data) *command.Completion {
	extraArgs := []string{
		// Need the extra "unusedCmd" arg because autocompletion throws away the first arg (because it assumes it's the command)
		fmt.Sprintf("%q", strings.Join(data.StringList(passAlongArgs.Name()), " ")),
	}
	bc := command.BashCommand(command.StringListType, "BASH_OUTPUT", glf.gl.runCommand(data, "autocomplete", extraArgs), command.HideStderr())
	o := command.NewFakeOutput()
	v, err := bc.Run(o)
	o.Close()
	if err != nil {
		return nil
	}
	return &command.Completion{
		Suggestions: v.ToStringList(),
	}
}
