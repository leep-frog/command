package commander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommander"
)

// SimpleNode implements the `command.Node` interface from a provided `command.Processor` and `Edge`.
type SimpleNode struct {
	Processor command.Processor
	Edge      command.Edge
}

func (sn *SimpleNode) Next(i *command.Input, d *command.Data) (command.Node, error) {
	if sn.Edge == nil {
		return nil, nil
	}
	return sn.Edge.Next(i, d)
}

func (sn *SimpleNode) UsageNext(input *command.Input, data *command.Data) (command.Node, error) {
	if sn.Edge == nil {
		return nil, nil
	}
	return sn.Edge.UsageNext(input, data)
}

func (sn *SimpleNode) Execute(input *command.Input, output command.Output, data *command.Data, exData *command.ExecuteData) error {
	if sn.Processor == nil {
		return nil
	}
	return spycommander.ProcessOrExecute(sn.Processor, input, output, data, exData)
}

func (sn *SimpleNode) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	if sn.Processor == nil {
		return nil, nil
	}
	return processOrComplete(sn.Processor, input, data)
}

func (sn *SimpleNode) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	if sn.Processor != nil {
		return spycommander.ProcessOrUsage(sn.Processor, i, d, u)
	}
	return nil
}

// SimpleEdge implements the `Edge` interface and points to the provided `command.Node`.
type SimpleEdge struct {
	// N is the next `command.Node` to visit.
	N command.Node
}

func (se *SimpleEdge) Next(*command.Input, *command.Data) (command.Node, error) {
	return se.N, nil
}

func (se *SimpleEdge) UsageNext(input *command.Input, data *command.Data) (command.Node, error) {
	return se.N, nil
}
