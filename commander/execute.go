package commander

import (
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spycommander"
)

// Execute executes a node with the provided `commondels.Input` and `commondels.Output`.
func Execute(n commondels.Node, input *commondels.Input, output commondels.Output, os commondels.OS) (*commondels.ExecuteData, error) {
	return execute(n, input, output, &commondels.Data{OS: os})
}

// Separate method for testing purposes.
func execute(n commondels.Node, input *commondels.Input, output commondels.Output, data *commondels.Data) (eData *commondels.ExecuteData, retErr error) {
	eData = &commondels.ExecuteData{}
	return eData, spycommander.Execute(n, input, output, data, eData)
}

// processOrExecute checks if the provided processor is a `commondels.Node` or just a `commondels.Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrExecute(p commondels.Processor, input *commondels.Input, output commondels.Output, data *commondels.Data, eData *commondels.ExecuteData) error {
	return spycommander.ProcessOrExecute(p, input, output, data, eData)
}

// processGraphExecution processes the provided graph
func processGraphExecution(root commondels.Node, input *commondels.Input, output commondels.Output, data *commondels.Data, eData *commondels.ExecuteData) error {
	return spycommander.ProcessGraphExecution(root, input, output, data, eData)
}
