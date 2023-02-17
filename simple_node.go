package command

// SimpleNode implements the `Node` interface from a provided `Processor` and `Edge`.
type SimpleNode struct {
	Processor Processor
	Edge      Edge
}

func (sn *SimpleNode) Next(i *Input, d *Data) (Node, error) {
	if sn.Edge == nil {
		return nil, nil
	}
	return sn.Edge.Next(i, d)
}

func (sn *SimpleNode) UsageNext(input *Input, data *Data) (Node, error) {
	if sn.Edge == nil {
		return nil, nil
	}
	return sn.Edge.UsageNext(input, data)
}

func (sn *SimpleNode) Execute(input *Input, output Output, data *Data, exData *ExecuteData) error {
	if sn.Processor == nil {
		return nil
	}
	return processOrExecute(sn.Processor, input, output, data, exData)
}

func (sn *SimpleNode) Complete(input *Input, data *Data) (*Completion, error) {
	if sn.Processor == nil {
		return nil, nil
	}
	return processOrComplete(sn.Processor, input, data)
}

func (sn *SimpleNode) Usage(i *Input, d *Data, u *Usage) error {
	if sn.Processor != nil {
		return processOrUsage(sn.Processor, i, d, u)
	}
	return nil
}

// SimpleEdge implements the `Edge` interface and points to the provided `Node`.
type SimpleEdge struct {
	// N is the next `Node` to visit.
	N Node
}

func (se *SimpleEdge) Next(*Input, *Data) (Node, error) {
	return se.N, nil
}

func (se *SimpleEdge) UsageNext(input *Input, data *Data) (Node, error) {
	return se.N, nil
}
