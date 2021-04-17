package command

type simpleEdge struct {
	n *Node
}

func (se *simpleEdge) Next(*Input, Output, *Data) (*Node, error) {
	return se.n, nil
}

func SimpleEdge(n *Node) Edge {
	return &simpleEdge{
		n: n,
	}
}

// SerialNodes returns a graph that iterates serially over the provided processors.
func SerialNodes(ps ...Processor) *Node {
	if len(ps) == 0 {
		return nil
	}

	n := &Node{
		Processor: ps[len(ps)-1],
	}
	for j := len(ps) - 2; j >= 0; j-- {
		newN := &Node{
			Processor: ps[j],
			Edge:      SimpleEdge(n),
		}
		n = newN
	}
	return n
}
