package spycommander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommand"
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
		if ok, err := spycommand.IsTerminationPanic(r); ok {
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

// ProcessGraphExecution processes the provided graph
func ProcessGraphExecution(n command.Node, input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData, ignoreErrFuncs ...func(error) bool) error {
	for n != nil {
		if err := n.Execute(input, output, data, eData); err != nil {
			for _, f := range ignoreErrFuncs {
				if f(err) {
					goto IGNORE_ERR
				}
			}
			return err
		IGNORE_ERR:
		}

		var err error
		if n, err = n.Next(input, data); err != nil {
			return err
		}
	}
	return nil
}
