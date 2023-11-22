package command

// Execute executes a node with the provided `Input` and `Output`.
func Execute(n Node, input *Input, output Output, os OS) (*ExecuteData, error) {
	return execute(n, input, output, &Data{OS: os})
}

// Separate method for testing purposes.
func execute(n Node, input *Input, output Output, data *Data) (eData *ExecuteData, retErr error) {
	eData = &ExecuteData{}

	defer func() {
		r := recover()

		// No panic
		if r == nil {
			return
		}

		// Panicked due to terminate error
		if t, ok := r.(*terminator); ok && t.terminationError != nil {
			retErr = t.terminationError
			return
		}

		// Panicked for other reason
		panic(r)
	}()

	if retErr = processGraphExecution(n, input, output, data, eData); retErr != nil {
		return
	}

	if !input.FullyProcessed() {
		retErr = ExtraArgsErr(input)
		output.Stderrln(retErr)
		// TODO: Make this the last node we reached?
		ShowUsageAfterError(n, output)
		return
	}

	for _, ex := range eData.Executor {
		if retErr = ex(output, data); retErr != nil {
			return
		}
	}

	return
}

// processOrExecute checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrExecute(p Processor, input *Input, output Output, data *Data, eData *ExecuteData) error {
	if n, ok := p.(Node); ok {
		return processGraphExecution(n, input, output, data, eData)
	}
	return p.Execute(input, output, data, eData)
}

// processGraphExecution processes the provided graph
func processGraphExecution(root Node, input *Input, output Output, data *Data, eData *ExecuteData) error {
	for n := root; n != nil; {
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
