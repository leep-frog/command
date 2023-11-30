package commander

import "github.com/leep-frog/command/commondels"

// SerialNodes returns a graph that iterates serially over nodes with the provided `commondels.Processor` objects.
func SerialNodes(ps ...commondels.Processor) commondels.Node {
	if len(ps) == 0 {
		return &SimpleNode{}
	}

	root := &SimpleNode{
		Processor: ps[0],
	}
	n := root
	for _, newP := range ps[1:] {
		newN := &SimpleNode{
			Processor: newP,
		}
		n.Edge = &SimpleEdge{newN}
		n = newN
	}
	return root
}
