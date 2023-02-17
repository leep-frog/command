package command

import (
	"sync"
)

// Execute executes a node with the provided `Input` and `Output`.
func Execute(n Node, input *Input, output Output) (*ExecuteData, error) {
	return execute(n, input, output, &Data{})
}

// Separate method for testing purposes.
func execute(n Node, input *Input, output Output, data *Data) (*ExecuteData, error) {
	eData := &ExecuteData{}

	// This threading logic is needed in case the underlying process calls an output.Terminate command.
	var wg sync.WaitGroup
	wg.Add(1)

	var termErr error
	go func() {
		defer func() {
			if termErr == nil {
				termErr = output.terminateError()
			}
			wg.Done()
		}()
		if err := processGraphExecution(n, input, output, data, eData, true); err != nil {
			termErr = err
			return
		}

		if err := input.CheckForExtraArgsError(); err != nil {
			output.Stderrln(err)
			// TODO: Make this the last node we reached?
			ShowUsageAfterError(n, output)
			termErr = err
			return
		}

		for _, ex := range eData.Executor {
			if err := ex(output, data); err != nil {
				termErr = err
				return
			}
		}
	}()
	wg.Wait()
	return eData, termErr
}

// processOrExecute checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrExecute(p Processor, input *Input, output Output, data *Data, eData *ExecuteData) error {
	if n, ok := p.(Node); ok {
		return processGraphExecution(n, input, output, data, eData, false)
	}
	return p.Execute(input, output, data, eData)
}

// processGraphExecution processes the provided graph
func processGraphExecution(root Node, input *Input, output Output, data *Data, eData *ExecuteData, checkInput bool) error {
	for n := root; n != nil; {
		if err := n.Execute(input, output, data, eData); err != nil {
			return err
		}

		var err error
		if n, err = n.Next(input, data); err != nil {
			return err
		}
	}

	if checkInput {
		return output.Err(input.CheckForExtraArgsError())
	}
	return nil
}
