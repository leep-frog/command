package command

import (
	"fmt"
)

type Node struct {
	Processor Processor
	Edge      Edge
}

type Processor interface {
	Execute(*Input, Output, *Data, *ExecuteData) error
	// Complete should return complete data if there was an error or a completion can be made.
	Complete(*Input, *Data) *CompleteData
}

type Edge interface {
	Next(*Input, *Data) (*Node, error)
}

func Execute(n *Node, input *Input, output Output) (*ExecuteData, error) {
	return execute(n, input, output, &Data{})
}

// Separate method for testing purposes.
func execute(n *Node, input *Input, output Output, data *Data) (*ExecuteData, error) {
	eData := &ExecuteData{}
	if err := iterativeExecute(n, input, output, data, eData); err != nil {
		return eData, err
	}

	if eData.Executor != nil {
		return eData, eData.Executor(output, data)
	}
	return eData, nil
}

func iterativeExecute(n *Node, input *Input, output Output, data *Data, eData *ExecuteData) error {
	for n != nil {
		if n.Processor != nil {
			if err := n.Processor.Execute(input, output, data, eData); err != nil {
				return err
			}
		}

		if n.Edge == nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, data); err != nil {
			return err
		}
	}

	if !input.FullyProcessed() {
		return output.Err(ExtraArgsErr(input))
	}

	return nil
}

func ExtraArgsErr(input *Input) error {
	return &extraArgsErr{input}
}

type extraArgsErr struct {
	input *Input
}

func (eae *extraArgsErr) Error() string {
	return fmt.Sprintf("Unprocessed extra args: %v", eae.input.Remaining())
}

func IsExtraArgsError(err error) bool {
	_, ok := err.(*extraArgsErr)
	return ok
}
