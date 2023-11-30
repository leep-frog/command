package spycommander

import (
	"reflect"

	"github.com/leep-frog/command/command"
)

// Separate method for testing purposes.
func Execute(n command.Node, input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) (retErr error) {
	defer func() {
		r := recover()

		// No panic
		if r == nil {
			return
		}

		// Panicked due to terminate error
		if ok, err := command.IsTerminationPanic(r); ok {
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
		retErr = command.ExtraArgsErr(input)
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
func ProcessOrExecute(p command.Processor, input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	if n, ok := p.(command.Node); ok {
		return ProcessGraphExecution(n, input, output, data, eData)
	}
	return p.Execute(input, output, data, eData)
}

// TODO: replace with pointer types
func isNil(o interface{}) bool {
	return o == nil || reflect.ValueOf(o).IsNil()
}

// ProcessGraphExecution processes the provided graph
func ProcessGraphExecution(root command.Node, input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
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
