package main

import (
	"os"

	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/sourcerer"
)

/* To test, cd into this directory and then run the following commands:
# Load the goleep command
source "$(go run source ../goleeper/goleeper.go goleeper)"

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
	// fmt.Println(runtime.Caller(0))
	os.Exit(sourcerer.Source([]sourcerer.CLI{
		sourcerer.ToCLI("simple", commander.SerialNodes(
			commander.ListArg[string]("SL", "", 1, 2, commander.SimpleCompleter[[]string]("un", "deux", "trois")),
			&commander.ExecutorProcessor{F: func(o commondels.Output, d *commondels.Data) error {
				o.Stdoutf("%v\n", d.StringList("SL"))
				return nil
			}},
		)),
	}))
}
