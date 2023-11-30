package commander

import "github.com/leep-frog/command/command"

// SerialNodes returns a graph that iterates serially over nodes with the provided `command.Processor` objects.
func SerialNodes(ps ...command.Processor) command.Node {
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
