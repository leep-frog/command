package main

import (
	"encoding/json"
	"fmt"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func main() {
	sourcerer.Source(&SimpleCLI{})
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
	if jsn == "" {
		ss = &SimpleCLI{}
		return nil
	}

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
		command.StringNode("firstName", &command.ArgOpt{
			Completor: command.SimpleCompletor("Greg", "Groog", "Gregory", "Groooooooooooooooooog"),
		}),
		command.OptionalStringNode("lastName", nil),
		command.ExecutorNode(func(output command.Output, data *command.Data) error {
			output.Stdout("Hello, %s", data.Values["firstName"].String())
			if ln := data.Values["lastName"]; ln.Provided() {
				return output.Stderr("or should I say, Professor %s!!", ln.String())
			}
			return nil
		}),
	)
}