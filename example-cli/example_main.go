package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/sourcerer"
)

func main() {
	// sourcerer.Source returns 0 if the command resulted in success and 1 otherwise. Using `os.Exit` in this way ensures that your Go errors result in the appropriate command exit status in bash.
	// os.Exit(sourcerer.Source([]sourcerer.CLI{
	// &myFirstCommand{},
	// &mySecondCommand,
	// &myThirdCommand,
	// ...
	// }, sourcerer.NewAliaser("jj", "mfc", "--formal")))
	os.Exit(sourcerer.RunCLI(&myFirstCommand{}))
}

func tputStuff() {
	fmt.Println("Before tput")
	cmd := exec.Command("tput", "setaf", "4")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("tput error: %v\n", err)
	}
	fmt.Println("After tput")
}

type myFirstCommand struct{}

// Name is the bash alias that will be created for this CLI.
func (mfc *myFirstCommand) Name() string {
	return "mfc"
}

// Changed is whether or not the persistent data for a command
// has changed (in which case the object will be saved).
// See the persistent data feature doc for more info on this.
func (mfc *myFirstCommand) Changed() bool {
	return false
}

// Setup returns some bash setup commands. See the [Setup feature doc](TODO) for more info on this.
func (mfc *myFirstCommand) Setup() []string {
	return nil
}

// Node returns the logic of your new command!
func (mfc *myFirstCommand) Node() command.Node {

	fc := &commander.FileCompleter[string]{
		Directory:   filepath.Join(".."),
		IgnoreFiles: true,
		ExcludePwd:  true,
	}

	ff := commander.FileArgument("FILE", "desc", fc)
	// A boolean flag (set by passing `--formal` or `-f` to your command in bash).
	formalFlag := commander.BoolFlag("formal", 'f', "Whether or not the response should be formal")
	// A required string argument that can be autocompleted!
	nameArg := commander.Arg[string]("NAME", "Your name", commander.SimpleCompleter[string]("Alice", "Bob", "Bruno", "Charlie", "World"))
	// An optional integer argument that must be a positive number and defaults to 1.
	nArg := commander.OptionalArg[int](
		"N", "Number of times to say hello",
		commander.Positive[int](),
		commander.Default(1),
	)

	// SerialNodes runs a list of processors in sequence.
	return commander.SerialNodes(
		sourcerer.ExecutableFileGetProcessor(),
		// Description adds a description field to your commands usage doc.
		commander.Description("My very first command!"),
		ff,
		// This node defines all of the flags for your command.
		commander.FlagProcessor(
			formalFlag,
			commander.BoolFlag("blop", 'b', "desc"),
		),
		nameArg,
		nArg,
		// The logic of your function!
		// ExecutorNode doesn't deal with errors. If your command involves potential
		// errors, use ExecuteErrNode instead.
		&commander.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
			name := nameArg.Get(d)
			n := nArg.Get(d)
			for i := 0; i < n; i++ {
				if formalFlag.Get(d) {
					o.Stdoutf("Greetings, %s.\n", name)
				} else {
					o.Stdoutf("Hello, %s.\n", name)
				}
			}
			o.Stdoutln("GET", sourcerer.ExecutableFileGetProcessor().Get(d))
			return nil
		}},
	)
}
