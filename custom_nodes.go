package command

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// SetupArgName is the argument name for `SetupArg`
	SetupArgName = "SETUP_FILE"
)

var (
	// SetupArg is an argument that points to the filename containing the output of the Setup command.
	SetupArg = FileNode(SetupArgName, "file used to run setup for command", HiddenArg[string]())
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

// SimpleEdge returns an edge that points to the provided node.
func SimpleEdge(n *Node) Edge {
	if n == nil {
		return nil
	}
	return &simpleEdge{
		n: n,
	}
}

// SerialNodes returns a graph that iterates serially over the provided `Processors`.
func SerialNodes(p Processor, ps ...Processor) *Node {
	return SerialNodesTo(nil, p, ps...)
}

// SerialNodesTo returns a graph that iterates serially over the provided `Processors`.
// The last `Processor` then has an edge to the provided `to` node.
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

// ExecuteErrNode creates a simple execution node from the provided error-able function.
func ExecuteErrNode(f func(Output, *Data) error) Processor {
	return &executor{
		executor: f,
	}
}

// ExecturoNode creates a simple execution node from the provided no-error function.
func ExecutorNode(f func(Output, *Data)) Processor {
	return ExecuteErrNode(func(o Output, d *Data) error {
		f(o, d)
		return nil
	})
}

type branchNode struct {
	branches map[string]*Node
	// Map from branch code to synonyms
	synonyms     map[string]string
	def          *Node
	next         *Node
	nextErr      error
	scCompletion bool
	hideUsage    bool
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

	if n, ok := bn.branches[bn.synonyms[s]]; ok {
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

// IsBranchingError returns whether or not the provided error is a branching error.
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
	if bn.hideUsage {
		return
	}

	u.UsageSection.Add(SymbolSection, "<", "Start of subcommand branches")
	u.Usage = append(u.Usage, "<")

	var names []string
	for name := range bn.branches {
		names = append(names, name)
	}
	sort.Strings(names)

	branchToSynonyms := map[string][]string{}
	for k, v := range bn.synonyms {
		branchToSynonyms[v] = append(branchToSynonyms[v], k)
	}

	for _, name := range names {
		su := GetUsage(bn.branches[name])
		if as := branchToSynonyms[name]; len(as) > 0 {
			var r []string
			for _, a := range as {
				r = append(r, a)
			}
			sort.Strings(r)
			name = fmt.Sprintf("[%s|%v]", name, strings.Join(r, "|"))
		}
		su.Usage = append([]string{name}, su.Usage...)
		u.SubSections = append(u.SubSections, su)
	}
}

// BranchNodeOption is an option type for modifying a `BranchNode`.
type BranchNodeOption func(*branchNode)

// DontCompleteSubcommands is a `BranchNodeOption` that prevents
// subcommands from being included in autocompletion.
func DontCompleteSubcommands() BranchNodeOption {
	return func(bn *branchNode) {
		bn.scCompletion = false
	}
}

// HideBranchUsage is a `BranchNodeOption` that prevents `BranchNode` usage
// from showing up in the command's usage text.
func HideBranchUsage() BranchNodeOption {
	return func(bn *branchNode) {
		bn.hideUsage = true
	}
}

// BranchSynonyms is a `BranchNodeOption` to specify synonyms for branches in a
// `BranchNode`.
func BranchSynonyms(synonyms map[string][]string) BranchNodeOption {
	m := map[string]string{}
	for k, vs := range synonyms {
		for _, v := range vs {
			m[v] = k
		}
	}
	return func(bn *branchNode) {
		bn.synonyms = m
	}
}

// BranchNode returns a node that branches on specific string arguments.
// If the argument does not match any branch, then the `dflt` node is traversed.
func BranchNode(branches map[string]*Node, dflt *Node, opts ...BranchNodeOption) *Node {
	if branches == nil {
		branches = map[string]*Node{}
	}
	synonyms := map[string]string{}
	ob := map[string]*Node{}
	for str, v := range branches {
		ks := strings.Split(str, " ")
		ob[ks[0]] = v
		for _, k := range ks[1:] {
			synonyms[k] = ks[0]
		}
	}
	bn := &branchNode{
		branches:     ob,
		synonyms:     synonyms,
		def:          dflt,
		scCompletion: true,
	}
	for _, opt := range opts {
		opt(bn)
	}
	return &Node{
		Processor: bn,
		Edge:      bn,
	}
}

// SimpleProcessor creates a `Processor` from execution and completion functions.
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

// StringMenu returns an `Arg` that is required to be one of the provided choices.
func StringMenu(name, desc string, choices ...string) *ArgNode[string] {
	return Arg[string](name, desc, SimpleCompletor[string](choices...), InList(choices...))
}

// ListBreakerOption is an option type for the `ListBreaker` type.
type ListBreakerOption func(*ListBreaker)

func newBreakerOpt(f func(*ListBreaker)) ListBreakerOption {
	return f
}

// DiscardBreaker is a `ListBreakerOption` that removes the breaker argument from the input (rather than keeping it for the next node to parse).
func DiscardBreaker() ListBreakerOption {
	return newBreakerOpt(func(lb *ListBreaker) {
		lb.discard = true
	})
}

// ListBreakerUsage is a `ListBreakerOption` that inlcudes usage info in the command's usage text.
func ListBreakerUsage(uf func(*Usage)) ListBreakerOption {
	return newBreakerOpt(func(lb *ListBreaker) {
		lb.u = uf
	})
}

// ListUntilSymbol returns an unbounded list node that ends when a specific symbol is parsed.
func ListUntilSymbol(symbol string, opts ...ListBreakerOption) *ListBreaker {
	return ListUntil(NEQ(symbol)).AddOptions(append(opts, ListBreakerUsage(func(u *Usage) {
		u.Usage = append(u.Usage, symbol)
		u.UsageSection.Add(SymbolSection, symbol, "List breaker")
	}))...)
}

// AddOptions adds `ListBreakerOptions` to a `ListBreaker` object.
func (lb *ListBreaker) AddOptions(opts ...ListBreakerOption) *ListBreaker {
	for _, opt := range opts {
		opt(lb)
	}
	return lb
}

// ListUntil returns a `ListBreaker` node that breaks when any of the provided `ValidatorOptions` are not satisfied.
func ListUntil(validators ...*ValidatorOption[string]) *ListBreaker {
	return &ListBreaker{
		validators: validators,
	}
}

// ListBreaker is an `ArgOpt` for breaking out of lists with an optional number of arguments.
// TODO: this should be ListBreaker[T any, ST []T]
type ListBreaker struct {
	validators []*ValidatorOption[string]
	discard    bool
	u          func(*Usage)
}

func (lb *ListBreaker) modifyArgOpt(ao *argOpt[[]string]) {
	ao.breaker = lb
}

// Validators returns the `ListBreaker`'s validators.
func (lb *ListBreaker) Validators() []*ValidatorOption[string] {
	return lb.validators
}

// DiscardBreak indicates whether the `ListBreaker` discards the argument that breaks the list.
func (lb *ListBreaker) DiscardBreak() bool {
	return lb.discard
}

// Usage updates the provided `Usage` object.
func (lb *ListBreaker) Usage(u *Usage) {
	if lb.u != nil {
		lb.u(u)
	}
}

// StringListListNode parses a two-dimensional slice of strings, with each slice being separated by `breakSymbol`
func StringListListNode(name, desc, breakSymbol string, minN, optionalN int, opts ...ArgOpt[[]string]) Processor {
	n := &Node{
		Processor: ListArg(name, desc, 0, UnboundedList,
			append(opts,
				ListUntilSymbol(breakSymbol, DiscardBreaker()),
				CustomSetter(func(sl []string, d *Data) {
					if len(sl) > 0 {
						if !d.Has(name) {
							d.Set(name, [][]string{sl})
						} else {
							d.Set(name, append(GetData[[][]string](d, name), sl))
						}
					}
				}),
			)...,
		),
	}
	return NodeRepeater(n, minN, optionalN)
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

// SimpleExecutableNode returns a `Processor` that adds to the command's `Executable`.
func SimpleExecutableNode(sl ...string) Processor {
	return ExecutableNode(func(_ Output, d *Data) ([]string, error) { return sl, nil })
}

// ExecutableNode returns a `Processor` that adds to the command's `Executable`.
// Below are some tips when writing bash outputs for this:
// 1. Be sure to initialize variables with `local` to avoid overriding variables used in
// sourcerer scripts.
// 2. Use `return` rather than `exit` when terminating a session early.
func ExecutableNode(f func(Output, *Data) ([]string, error)) Processor {
	return &executableAppender{f}
}

// FileContents converts a filename into the file's contents.
func FileContents(name, desc string, opts ...ArgOpt[string]) Processor {
	fc := FileNode(name, desc, opts...)
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		if err := fc.Execute(i, o, d, ed); err != nil {
			return err
		}
		c, err := ReadFile(d.String(name))
		if err != nil {
			return o.Annotatef(err, "failed to read file")
		}
		d.Set(name, c)
		return nil
	}, func(i *Input, d *Data) (*Completion, error) {
		return fc.Complete(i, d)
	})
}
