package commander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommander"
)

// Execute executes a node with the provided `command.Input` and `command.Output`.
func Execute(n command.Node, input *command.Input, output command.Output, os command.OS) (*command.ExecuteData, error) {
	return execute(n, input, output, &command.Data{OS: os})
}

// Separate method for testing purposes.
func execute(n command.Node, input *command.Input, output command.Output, data *command.Data) (eData *command.ExecuteData, retErr error) {
	eData = &command.ExecuteData{}
	return eData, spycommander.Execute(n, input, output, data, eData)
}
