package command

import (
	"fmt"
	"sort"
)

const (
	SetupArgName = "SETUP_FILE"
)

var (
	SetupArg = FileNode(SetupArgName, "file used to run setup for command", HiddenArg())
)

type simpleEdge struct {
	n *Node
}

func (se *simpleEdge) Next(*Input, *Data) (*Node, error) {
	return se.n, nil
}

func (se *simpleEdge) UsageNext() *Node {
	return se.n
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
	eData.Executor = append(eData.Executor, e.executor)
	return nil
}

func (e *executor) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

func (e *executor) Usage(u *Usage) {
	return
}

func ExecutorNode(f func(Output, *Data) error) Processor {
	return &executor{
		executor: f,
	}
}

func SafeExecutorNode(f func(Output, *Data)) Processor {
	return ExecutorNode(func(o Output, d *Data) error {
		f(o, d)
		return nil
	})
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
		return output.Err(err)
	}
	return nil
}

func (bn *branchNode) Complete(input *Input, data *Data) (*Completion, error) {
	if len(input.remaining) > 1 {
		return nil, bn.getNext(input, data)
	}

	c := &Completion{}
	var defaultNodeErr error
	if bn.def != nil {
		// Need to iterate over the remaining nodes in case the immediately next node
		// doesn't process any args and the one after it does.
		var newC *Completion
		newC, defaultNodeErr = getCompleteData(bn.def, input, data)
		if newC != nil {
			c = newC
		}
	}

	if !bn.scCompletion {
		return c, defaultNodeErr
	}

	for k := range bn.branches {
		c.Suggestions = append(c.Suggestions, k)
	}
	return c, defaultNodeErr
}

func (bn *branchNode) getNext(input *Input, data *Data) error {
	s, ok := input.Peek()
	if !ok {
		if bn.def == nil {
			return newBranchingErr(bn)
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

	return newBranchingErr(bn)
}

type branchingErr struct {
	bn *branchNode
}

func (be *branchingErr) Error() string {
	choices := make([]string, 0, len(be.bn.branches))
	for k := range be.bn.branches {
		choices = append(choices, k)
	}
	sort.Strings(choices)
	return fmt.Sprintf("Branching argument must be one of %v", choices)
}

func newBranchingErr(bn *branchNode) error {
	return &branchingErr{bn}
}

func IsBranchingError(err error) bool {
	_, ok := err.(*branchingErr)
	return ok
}

func (bn *branchNode) Next(input *Input, data *Data) (*Node, error) {
	return bn.next, nil
}

func (bn *branchNode) UsageNext() *Node {
	return bn.def
}

func (bn *branchNode) Usage(u *Usage) {
	u.UsageSection.Add(SymbolSection, "<", "Start of subcommand branches")
	u.Usage = append(u.Usage, "<")

	var names []string
	for name := range bn.branches {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		su := GetUsage(bn.branches[name])
		su.Usage = append([]string{name}, su.Usage...)
		u.SubSections = append(u.SubSections, su)
	}
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

func SimpleProcessor(e func(*Input, Output, *Data, *ExecuteData) error, c func(*Input, *Data) (*Completion, error)) Processor {
	return &simpleProcessor{
		e: e,
		c: c,
	}
}

// NodeRepeater is a `Processor` that runs the provided Node at least `minN` times and up to `minN + optionalN` times.
// It should work with most node types, but hasn't been tested with branch nodes and flags really.
// Additionally, any argument nodes under it should probably use `CustomSetter` arg options.
func NodeRepeater(n *Node, minN, optionalN int) Processor {
	return &nodeRepeater{minN, optionalN, n}
}

type nodeRepeater struct {
	minN      int
	optionalN int
	n         *Node
}

func (nr *nodeRepeater) Usage(u *Usage) {
	nu := GetUsage(nr.n)

	// Merge UsageSection
	for k1, m := range *nu.UsageSection {
		for k2, v := range m {
			u.UsageSection.Add(k1, k2, v)
		}
	}

	// Merge Description
	if nu.Description != "" {
		u.Description = nu.Description
	}

	// Add Arguments
	for i := 0; i < nr.minN; i++ {
		u.Usage = append(u.Usage, nu.Usage...)
	}

	if nr.optionalN == UnboundedList {
		u.Usage = append(u.Usage, "{")
		u.Usage = append(u.Usage, nu.Usage...)
		u.Usage = append(u.Usage, "}")
		u.Usage = append(u.Usage, "...")
	} else if nr.optionalN > 0 {
		u.Usage = append(u.Usage, "{")
		for i := 0; i < nr.optionalN; i++ {
			u.Usage = append(u.Usage, nu.Usage...)
		}
		u.Usage = append(u.Usage, "}")
	}

	// We don't add flags because those are, presumably, done all at once at the beginning.
	// Additionally, SubSections are only used by BranchNodes, and I can't imagine those being used inside of NodeRepeater
	// If I am ever proven wrong on either of those claims, that person can implement usage updating in that case.
}

func (nr *nodeRepeater) proceedCondition(exCount int, i *Input) bool {
	// Keep going if...
	return (
	// we haven't run the minimum number of times
	exCount < nr.minN ||
		// there is more input AND there are optional cycles left
		(!i.FullyProcessed() && (nr.optionalN == UnboundedList || exCount < nr.minN+nr.optionalN)))
}

func (nr *nodeRepeater) Execute(i *Input, o Output, d *Data, e *ExecuteData) error {
	ieo := NewIgnoreErrOutput(o, IsExtraArgsError)
	for exCount := 0; nr.proceedCondition(exCount, i); exCount++ {
		if err := iterativeExecute(nr.n, i, ieo, d, e); err != nil && !IsExtraArgsError(err) {
			return err
		}
	}
	// A not enough args error will, presumably, be returned by
	// one of the iterativeExecute functions if necessary
	return nil
}

func (nr *nodeRepeater) Complete(i *Input, d *Data) (*Completion, error) {
	for exCount := 0; nr.proceedCondition(exCount, i); exCount++ {
		c, err := getCompleteData(nr.n, i, d)
		if c != nil || (err != nil && !IsExtraArgsError(err)) {
			return c, err
		}
	}
	return nil, nil
}

type simpleProcessor struct {
	e    func(*Input, Output, *Data, *ExecuteData) error
	c    func(*Input, *Data) (*Completion, error)
	desc string
}

func (sp *simpleProcessor) Usage(u *Usage) {
	if sp.desc != "" {
		u.Description = sp.desc
	}
}

func (sp *simpleProcessor) Execute(i *Input, o Output, d *Data, e *ExecuteData) error {
	if sp.e == nil {
		return nil
	}
	return sp.e(i, o, d, e)
}

func (sp *simpleProcessor) Complete(i *Input, d *Data) (*Completion, error) {
	if sp.c == nil {
		return nil, nil
	}
	return sp.c(i, d)
}

func StringMenu(name, desc string, choices ...string) *ArgNode {
	return StringNode(name, desc, SimpleCompletor(choices...), InList(choices...))
}

type ListBreakerOption func(*ListBreaker)

func newBreakerOpt(f func(*ListBreaker)) ListBreakerOption {
	return f
}

func DiscardBreaker() ListBreakerOption {
	return newBreakerOpt(func(lb *ListBreaker) {
		lb.discard = true
	})
}

func ListBreakerUsage(uf func(*Usage)) ListBreakerOption {
	return newBreakerOpt(func(lb *ListBreaker) {
		lb.u = uf
	})
}

func ListUntilSymbol(symbol string, opts ...ListBreakerOption) *ListBreaker {
	return ListUntil(StringDoesNotEqual(symbol)).AddOptions(append(opts, ListBreakerUsage(func(u *Usage) {
		u.Usage = append(u.Usage, symbol)
		u.UsageSection.Add(SymbolSection, symbol, "List breaker")
	}))...)
}

func (lb *ListBreaker) AddOptions(opts ...ListBreakerOption) *ListBreaker {
	for _, opt := range opts {
		opt(lb)
	}
	return lb
}

func ListUntil(validators ...*ValidatorOption) *ListBreaker {
	lb := &ListBreaker{
		validators: validators,
	}
	return lb
}

type ListBreaker struct {
	validators []*ValidatorOption
	discard    bool
	u          func(*Usage)
}

func (lb *ListBreaker) modifyArgOpt(ao *argOpt) {
	ao.breaker = lb
}

func (lb *ListBreaker) Validators() []*ValidatorOption {
	return lb.validators
}

func (lb *ListBreaker) DiscardBreak() bool {
	return lb.discard
}

func (lb *ListBreaker) Usage(u *Usage) {
	if lb.u != nil {
		lb.u(u)
	}
}

// StringListListNode parses a two-dimensional slice of strings, with each slice being separated by `breakSymbol`
func StringListListNode(name, desc, breakSymbol string, minN, optionalN int, opts ...ArgOpt) Processor {
	n := &Node{
		Processor: StringListNode(name, desc, 0, UnboundedList,
			append(opts,
				ListUntilSymbol(breakSymbol, DiscardBreaker()),
				CustomSetter(func(v *Value, d *Data) {
					if v.Length() > 0 {
						if !d.HasArgI(name) {
							d.SetI(name, [][]string{v.ToStringList()})
						} else {
							sl := d.GetI(name).([][]string)
							d.SetI(name, append(sl, v.ToStringList()))
						}
					}
				}),
			)...,
		),
	}
	return NodeRepeater(n, minN, optionalN)
}

type hiddenArg struct{}

func (ha *hiddenArg) modifyArgOpt(ao *argOpt) {
	ao.hiddenUsage = true
}

func HiddenArg() ArgOpt {
	return &hiddenArg{}
}

type executableAppender struct {
	f func(Output, *Data) ([]string, error)
}

func (ea *executableAppender) Execute(i *Input, o Output, d *Data, ed *ExecuteData) error {
	sl, err := ea.f(o, d)
	if err != nil {
		return err
	}
	ed.Executable = append(ed.Executable, sl...)
	return nil
}

func (ea *executableAppender) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

func (ea *executableAppender) Usage(*Usage) {}

func SimpleExecutableNode(sl ...string) Processor {
	return ExecutableNode(func(_ Output, d *Data) ([]string, error) { return sl, nil })
}

func ExecutableNode(f func(Output, *Data) ([]string, error)) Processor {
	return &executableAppender{f}
}
