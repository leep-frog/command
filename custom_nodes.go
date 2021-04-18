package command

type simpleEdge struct {
	n *Node
}

func (se *simpleEdge) Next(*Input, *Data) (*Node, error) {
	return se.n, nil
}

func SimpleEdge(n *Node) Edge {
	return &simpleEdge{
		n: n,
	}
}

// SerialNodes returns a graph that iterates serially over the provided processors.
func SerialNodes(p Processor, ps ...Processor) *Node {
	root := &Node{
		Processor: p,
	}
	n := root
	for _, newP := range ps {
		newN := &Node{
			Processor: newP,
		}
		n.Edge = SimpleEdge(newN)
		n = newN
	}
	return root
}

type executor struct {
	executor func(Output, *Data) error
}

func (e *executor) Execute(_ *Input, _ Output, _ *Data, eData *ExecuteData) error {
	eData.Executor = e.executor
	return nil
}

func (e *executor) Complete(*Input, *Data) *CompleteData {
	return nil
}

func ExecutorNode(f func(Output, *Data) error) Processor {
	return &executor{
		executor: f,
	}
}

type branchEdge struct {
	branches map[string]*Node
	def      *Node
}

func (be *branchEdge) Next(input *Input, data *Data) (*Node, error) {
	s, ok := input.Peek()
	if !ok {
		return be.def, nil
	}

	if n, ok := be.branches[s]; ok {
		return n, nil
	}

	return nil, nil
}

func BranchNode(branches map[string]*Node, dflt *Node) *Node {
	return &Node{
		Edge: &branchEdge{
			branches: branches,
			def:      dflt,
		},
	}
}
