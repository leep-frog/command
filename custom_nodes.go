package command

import (
	"fmt"
	"sort"
)

type simpleEdge struct {
	n *Node
}

func (se *simpleEdge) Next(*Input, *Data) (*Node, error) {
	return se.n, nil
}

func SimpleEdge(n *Node) Edge {
	if n == nil {
		return nil
	}
	return &simpleEdge{
		n: n,
	}
}

// SerialNodes returns a graph that iterates serially over the provided processors.
func SerialNodes(p Processor, ps ...Processor) *Node {
	return SerialNodesTo(nil, p, ps...)
}

func SerialNodesTo(to *Node, p Processor, ps ...Processor) *Node {
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
	n.Edge = SimpleEdge(to)
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

type branchNode struct {
	branches     map[string]*Node
	def          *Node
	next         *Node
	nextErr      error
	scCompletion bool
}

func (bn *branchNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	// The edge will figure out what needs to be done next.
	if err := bn.getNext(input, data); err != nil {
		return output.Stderr(err.Error())
	}
	return nil
}

func (bn *branchNode) Complete(input *Input, data *Data) *CompleteData {
	if len(input.remaining) > 1 {
		bn.getNext(input, data)
		return nil
	}

	cd := &CompleteData{}
	if bn.def != nil {
		// Need to iterate over the remaining nodes in case the immediately next node
		// doesn't process any args and the one after it does.
		if newCD := getCompleteData(bn.def, input, data); newCD != nil {
			cd = newCD
		}
	}

	if !bn.scCompletion {
		return cd
	}

	if cd.Completion == nil {
		cd.Completion = &Completion{}
	}

	for k := range bn.branches {
		cd.Completion.Suggestions = append(cd.Completion.Suggestions, k)
	}
	return cd
}

func (bn *branchNode) getNext(input *Input, data *Data) error {
	s, ok := input.Peek()
	if !ok {
		if bn.def == nil {
			return fmt.Errorf("branching argument required")
		}
		bn.next = bn.def
		return nil
	}

	if n, ok := bn.branches[s]; ok {
		input.Pop()
		bn.next = n
		return nil
	}

	if bn.def != nil {
		bn.next = bn.def
		return nil
	}

	choices := make([]string, 0, len(bn.branches))
	for k := range bn.branches {
		choices = append(choices, k)
	}
	sort.Strings(choices)
	return fmt.Errorf("argument must be one of %v", choices)
}

func (bn *branchNode) Next(input *Input, data *Data) (*Node, error) {
	return bn.next, nil
}

func BranchNode(branches map[string]*Node, dflt *Node, completeSubcommands bool) *Node {
	if branches == nil {
		branches = map[string]*Node{}
	}
	bn := &branchNode{
		branches:     branches,
		def:          dflt,
		scCompletion: completeSubcommands,
	}
	return &Node{
		Processor: bn,
		Edge:      bn,
	}
}

func SimpleProcessor(e func(*Input, Output, *Data, *ExecuteData) error, c func(*Input, *Data) *CompleteData) Processor {
	return &simpleProcessor{
		e: e,
		c: c,
	}
}

type simpleProcessor struct {
	e func(*Input, Output, *Data, *ExecuteData) error
	c func(*Input, *Data) *CompleteData
}

func (sp *simpleProcessor) Execute(i *Input, o Output, d *Data, e *ExecuteData) error {
	if sp.e == nil {
		return nil
	}
	return sp.e(i, o, d, e)
}

func (sp *simpleProcessor) Complete(i *Input, d *Data) *CompleteData {
	if sp.c == nil {
		return nil
	}
	return sp.c(i, d)
}
