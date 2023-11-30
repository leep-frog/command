package commander

import (
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spycommander"
)

// SimpleNode implements the `commondels.Node` interface from a provided `commondels.Processor` and `Edge`.
type SimpleNode struct {
	Processor commondels.Processor
	Edge      commondels.Edge
}

func (sn *SimpleNode) Next(i *commondels.Input, d *commondels.Data) (commondels.Node, error) {
	if sn.Edge == nil {
		return nil, nil
	}
	return sn.Edge.Next(i, d)
}

func (sn *SimpleNode) UsageNext(input *commondels.Input, data *commondels.Data) (commondels.Node, error) {
	if sn.Edge == nil {
		return nil, nil
	}
	return sn.Edge.UsageNext(input, data)
}

func (sn *SimpleNode) Execute(input *commondels.Input, output commondels.Output, data *commondels.Data, exData *commondels.ExecuteData) error {
	if sn.Processor == nil {
		return nil
	}
	return processOrExecute(sn.Processor, input, output, data, exData)
}

func (sn *SimpleNode) Complete(input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	if sn.Processor == nil {
		return nil, nil
	}
	return processOrComplete(sn.Processor, input, data)
}

func (sn *SimpleNode) Usage(i *commondels.Input, d *commondels.Data, u *commondels.Usage) error {
	if sn.Processor != nil {
		return spycommander.ProcessOrUsage(sn.Processor, i, d, u)
	}
	return nil
}

// SimpleEdge implements the `Edge` interface and points to the provided `commondels.Node`.
type SimpleEdge struct {
	// N is the next `commondels.Node` to visit.
	N commondels.Node
}

func (se *SimpleEdge) Next(*commondels.Input, *commondels.Data) (commondels.Node, error) {
	return se.N, nil
}

func (se *SimpleEdge) UsageNext(input *commondels.Input, data *commondels.Data) (commondels.Node, error) {
	return se.N, nil
}
