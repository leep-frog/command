package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func main() {
	os.Exit(sourcerer.Source(&SimpleCLI{}))
}

type SimpleCLI struct {
	changed bool
}

func (*SimpleCLI) Name() string {
	return "lf"
}

func (ss *SimpleCLI) Changed() bool {
	return ss.changed
}

func (ss *SimpleCLI) Load(jsn string) error {
	if err := json.Unmarshal([]byte(jsn), ss); err != nil {
		return fmt.Errorf("failed to unmarshal emacs json: %v", err)
	}
	return nil
}
func (ss *SimpleCLI) Setup() []string {
	return nil
}

func (ss *SimpleCLI) Node() *command.Node {
	return command.SerialNodes(
		command.Arg[string]("firstName", "first name", command.SimpleCompletor[string]("Greg", "Groog", "Gregory", "Groooooooooooooooooog")),
		command.OptionalArg[string]("lastName", "last name"),
		command.ExecuteErrNode(func(output command.Output, data *command.Data) error {
			output.Stdoutf("Hello, %s", data.String("firstName"))
			if data.Has("lastName") {
				return output.Stderrf("or should I say, Professor %s!!", data.String("lastName"))
			}
			return nil
		}),
	)
}
