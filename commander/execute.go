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

// processOrExecute checks if the provided processor is a `command.Node` or just a `command.Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrExecute(p command.Processor, input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	return spycommander.ProcessOrExecute(p, input, output, data, eData)
}

// processGraphExecution processes the provided graph
func processGraphExecution(root command.Node, input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	return spycommander.ProcessGraphExecution(root, input, output, data, eData)
}
