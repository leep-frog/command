package main

import (
	"github.com/leep-frog/command"
)

/* To test, cd into this directory and then run the following commands:
# Load the goleep command
source "$(go run ../goleeper/goleeper.go goleeper)"

# Test usage and wildcards
goleep usage main.go
goleep usage *.go
goleep usage *

# Test autocompletion
goleep m[TAB]
goleep main.go [TAB][TAB]
goleep * d[TAB]
goleep *.go deux t[TAB]

# Test execution
go run main.go un deux
goleep main.go
*/

func main() {
	command.RunNodes(command.SerialNodes(
		command.ListArg[string]("SL", "", 1, 2, command.SimpleCompleter[[]string]("un", "deux", "trois")),
		&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
			o.Stdoutf("%v\n", d.StringList("SL"))
			return nil
		}},
	))
}
