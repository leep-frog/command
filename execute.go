package command

type Node struct {
	Processor Processor
	Edge      Edge
}

type Processor interface {
	Execute(*Input, Output, *Data, *ExecuteData) error
	Complete(*Input, Output, *Data, *CompleteData) error
}

type Edge interface {
	Next(*Input, Output, *Data) (*Node, error)
}

func Execute(n *Node, input *Input, output Output) (*ExecuteData, error) {
	data := &Data{}
	return execute(n, input, output, data)
}

// Separate method for testing purposes.
func execute(n *Node, input *Input, output Output, data *Data) (*ExecuteData, error) {
	eData := &ExecuteData{}
	for n != nil {
		if n.Processor != nil {
			if err := n.Processor.Execute(input, output, data, eData); err != nil {
				return nil, err
			}
		}

		if n.Edge == nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, output, data); err != nil {
			return nil, err
		}
	}

	if !input.FullyProcessed() {
		return nil, output.Stderr("Unprocessed extra args: %v", input.Remaining())
	}

	if eData.Executor != nil {
		return eData, eData.Executor(output, data)
	}
	return eData, nil
}

func Complete(n *Node, input *Input, output Output) *CompleteData {
	cData := &CompleteData{}
	data := &Data{}

	for n != nil {
		if n.Processor != nil {
			if err := n.Processor.Complete(input, output, data, cData); err != nil {
				cData.Error = err
				break
			}
		}

		if n.Edge != nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, output, data); err != nil {
			cData.Error = err
			break
		}
	}

	return cData
}
