package spycommander

import (
	"reflect"

	"github.com/leep-frog/command/commondels"
)

// Separate method for testing purposes.
func Execute(n commondels.Node, input *commondels.Input, output commondels.Output, data *commondels.Data, eData *commondels.ExecuteData) (retErr error) {
	defer func() {
		r := recover()

		// No panic
		if r == nil {
			return
		}

		// Panicked due to terminate error
		if ok, err := commondels.IsTerminationPanic(r); ok {
			retErr = err
			return
		}

		// Panicked for other reason
		panic(r)
	}()

	if retErr = ProcessGraphExecution(n, input, output, data, eData); retErr != nil {
		return
	}

	if !input.FullyProcessed() {
		retErr = commondels.ExtraArgsErr(input)
		output.Stderrln(retErr)
		ShowUsageAfterError(n, output)
		return retErr
	}

	for _, ex := range eData.Executor {
		if retErr = ex(output, data); retErr != nil {
			return
		}
	}

	return
}

// ProcessOrExecute checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func ProcessOrExecute(p commondels.Processor, input *commondels.Input, output commondels.Output, data *commondels.Data, eData *commondels.ExecuteData) error {
	if n, ok := p.(commondels.Node); ok {
		return ProcessGraphExecution(n, input, output, data, eData)
	}
	return p.Execute(input, output, data, eData)
}

// TODO: replace with pointer types
func isNil(o interface{}) bool {
	return o == nil || reflect.ValueOf(o).IsNil()
}

// ProcessGraphExecution processes the provided graph
func ProcessGraphExecution(root commondels.Node, input *commondels.Input, output commondels.Output, data *commondels.Data, eData *commondels.ExecuteData) error {
	for n := root; !isNil(n); {
		if err := n.Execute(input, output, data, eData); err != nil {
			return err
		}

		var err error
		if n, err = n.Next(input, data); err != nil {
			return err
		}
	}
	return nil
}
