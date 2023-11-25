package command

import "github.com/leep-frog/command/internal/spycommander"

// Execute executes a node with the provided `Input` and `Output`.
func Execute(n Node, input *Input, output Output, os OS) (*ExecuteData, error) {
	return execute(n, input, output, &Data{OS: os})
}

// Separate method for testing purposes.
func execute(n Node, input *Input, output Output, data *Data) (eData *ExecuteData, retErr error) {
	eData = &ExecuteData{}
	return eData, spycommander.Execute[*Input, Output, *Data, *ExecuteData, *Completion, *Usage, Node](n, input, output, data, eData, efb)
}

// processOrExecute checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrExecute(p Processor, input *Input, output Output, data *Data, eData *ExecuteData) error {
	return spycommander.ProcessOrExecute[*Input, Output, *Data, *ExecuteData, *Completion, *Usage, Node](p, input, output, data, eData)
}

// processGraphExecution processes the provided graph
func processGraphExecution(root Node, input *Input, output Output, data *Data, eData *ExecuteData) error {
	return spycommander.ProcessGraphExecution[*Input, Output, *Data, *ExecuteData, *Completion, *Usage, Node](root, input, output, data, eData)
}

var (
	efb = &executeFunctionBag{}
)

type executeFunctionBag struct{}

func (efb *executeFunctionBag) ShowUsageAfterError(n Node, o Output) {
	ShowUsageAfterError(n, o)
}

func (efb *executeFunctionBag) ExtraArgsErr(i *Input) error {
	return i.extraArgsErr()
}

func (efb *executeFunctionBag) GetExecutor(ed *ExecuteData) []func(Output, *Data) error {
	return ed.Executor
}
