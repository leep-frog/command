package command

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	// SetupArg is an argument that points to the filename containing the output of the Setup command.
	SetupArg = FileNode("SETUP_FILE", "file used to run setup for command", HiddenArg[string]())
)

type SimpleEdge struct {
	n Node
}

func (se *SimpleEdge) Next(*Input, *Data) (Node, error) {
	return se.n, nil
}

func (se *SimpleEdge) UsageNext() Node {
	return se.n
}

// SerialNodes returns a graph that iterates serially over the provided `Processors`.
func SerialNodes(p Processor, ps ...Processor) Node {
	root := &SimpleNode{
		Processor: p,
	}
	n := root
	for _, newP := range ps {
		newN := &SimpleNode{
			Processor: newP,
		}
		n.Edge = &SimpleEdge{newN}
		n = newN
	}
	return root
}

// ExecuteErrNode creates a simple execution node from the provided error-able function.
type ExecutorProcessor struct {
	F func(Output, *Data) error
}

func (e *ExecutorProcessor) Execute(_ *Input, _ Output, _ *Data, eData *ExecuteData) error {
	eData.Executor = append(eData.Executor, e.F)
	return nil
}

func (e *ExecutorProcessor) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

func (e *ExecutorProcessor) Usage(u *Usage) {
	return
}

// BranchNode implements a node that branches on specific string arguments.
// If the argument does not match any branch, then the `Default` node is traversed.
type BranchNode struct {
	// Branches is a map from branching argument to `Node` that should be
	// executed if that branching argument is provided.
	Branches map[string]Node
	// Synonyms are synonyms for branching arguments.
	Synonyms map[string]string
	// Default is the `Node` that should be executed if the branching argument
	// does not match of any of the branches.
	Default Node
	// BranchCompletions is whether or not branch arguments should be completed
	// or if the completions from the Default `Node` should be used.
	// This is only relevant when the branching argument is the argument
	// being completed. Otherwise, this node is executed as normal.
	DefaultCompletion bool
	// HideUsage is wheter or not usage info for this node should be
	// hidden.
	HideUsage bool

	next Node
}

// BranchSynonyms converts a map from branching argument to synonyms to a
// branching synonym map.
func BranchSynonyms(m map[string][]string) map[string]string {
	r := map[string]string{}
	for branch, synonyms := range m {
		for _, syn := range synonyms {
			r[syn] = branch
		}
	}
	return r
}

func (bn *BranchNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	// The edge will figure out what needs to be done next.
	return output.Err(bn.getNext(input, data))
}

func (bn *BranchNode) Complete(input *Input, data *Data) (*Completion, error) {
	if len(input.remaining) > 1 {
		return nil, bn.getNext(input, data)
	}

	// Note: we don't try to merge completions between branches
	// and default node because the completion overlap can get
	// convoluted given the number of `Completion` options.
	if bn.DefaultCompletion {
		// Need to iterate over the remaining nodes in case the immediately next node
		// doesn't process any args and the one after it does.
		return processGraphCompletion(bn.Default, input, data, false)
	}

	var names []string
	for b := range bn.Branches {
		name, _ := bn.splitBranch(b)
		names = append(names, name)
	}

	return &Completion{
		Suggestions:     names,
		CaseInsensitive: true,
	}, nil
}

func (bn *BranchNode) getNext(input *Input, data *Data) error {
	s, ok := input.Peek()
	if !ok {
		if bn.Default == nil {
			return newBranchingErr(bn)
		}
		bn.next = bn.Default
		return nil
	}

	if bn.Synonyms != nil {
		if syn, ok := bn.Synonyms[s]; ok {
			s = syn
		}
	}

	for branch, n := range bn.Branches {
		name, syns := bn.splitBranch(branch)
		if s == name {
			input.Pop()
			bn.next = n
			return nil
		}

		for _, b := range syns {
			if s == b {
				input.Pop()
				bn.next = n
				return nil
			}
		}
	}

	if bn.Default != nil {
		bn.next = bn.Default
		return nil
	}

	return newBranchingErr(bn)
}

type branchingErr struct {
	bn *BranchNode
}

func (be *branchingErr) Error() string {
	choices := make([]string, 0, len(be.bn.Branches))
	for k := range be.bn.Branches {
		choices = append(choices, k)
	}
	sort.Strings(choices)
	return fmt.Sprintf("Branching argument must be one of %v", choices)
}

func newBranchingErr(bn *BranchNode) error {
	return &branchingErr{bn}
}

// IsBranchingError returns whether or not the provided error is a branching error.
func IsBranchingError(err error) bool {
	_, ok := err.(*branchingErr)
	return ok
}

func (bn *BranchNode) Next(input *Input, data *Data) (Node, error) {
	return bn.next, nil
}

func (bn *BranchNode) UsageNext() Node {
	return bn.Default
}

func (bn *BranchNode) splitBranch(b string) (string, []string) {
	r := strings.Split(b, " ")
	return r[0], r[1:]
}

type branchSyn struct {
	name   string
	values []string
	n      Node
}

func (bn *BranchNode) getSyns() map[string]*branchSyn {
	nameToSyns := map[string]*branchSyn{}

	for bs, n := range bn.Branches {
		name, syns := bn.splitBranch(bs)
		nameToSyns[name] = &branchSyn{
			name:   name,
			values: syns,
			n:      n,
		}
	}

	for syn, name := range bn.Synonyms {
		nameToSyns[name].values = append(nameToSyns[name].values, syn)
	}

	for _, bs := range nameToSyns {
		slices.Sort(bs.values)
	}
	return nameToSyns
}

func (bn *BranchNode) Usage(u *Usage) {
	if bn.HideUsage {
		return
	}

	u.UsageSection.Set(SymbolSection, "<", "Start of subcommand branches")
	u.Usage = append(u.Usage, "<")

	bss := maps.Values(bn.getSyns())
	slices.SortFunc(bss, func(this, that *branchSyn) bool {
		return this.name < that.name
	})

	for _, bs := range bss {
		su := GetUsage(bs.n)
		v := bs.name
		if len(bs.values) > 0 {
			v = fmt.Sprintf("[%s|%s]", bs.name, strings.Join(bs.values, "|"))
		}
		su.Usage = append([]string{v}, su.Usage...)
		u.SubSections = append(u.SubSections, su)
	}
}

// DataValue is an interface for types that can be stored in `Data`.
type DataValue[T any] interface {
	Get(*Data) T
	Has(*Data) bool
	Set(T, *Data)
}

// ArgFilter filters out elements in an `ArgNode` or `Flag` slice.
func ArgFilter[T any](arg DataValue[[]T], f func(T, *Data) (bool, error)) Processor {
	filterFunc := func(d *Data) error {
		if !arg.Has(d) {
			return nil
		}
		var filtered []T
		for _, t := range arg.Get(d) {
			include, err := f(t, d)
			if err != nil {
				return err
			}
			if include {
				filtered = append(filtered, t)
			}
		}
		arg.Set(filtered, d)
		return nil
	}
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		return o.Err(filterFunc(d))
	}, func(i *Input, d *Data) (*Completion, error) {
		return nil, filterFunc(d)
	})
}

// EmptyArgFilter is an `ArgFilter` that filters out empty elements.
func EmptyArgFilter[T comparable](arg DataValue[[]T]) Processor {
	return ArgFilter(arg, func(t T, d *Data) (bool, error) {
		var nill T
		return t != nill, nil
	})
}

// SimpleProcessor creates a `Processor` from execution and completion functions.
func SimpleProcessor(e func(*Input, Output, *Data, *ExecuteData) error, c func(*Input, *Data) (*Completion, error)) Processor {
	return &simpleProcessor{
		e: e,
		c: c,
	}
}

// SuperSimpleProcessor returns a processor from a single function that is run in both
// the execution and completion contexts.
func SuperSimpleProcessor(f func(*Input, *Data) error) Processor {
	return &simpleProcessor{
		e: func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			return o.Err(f(i, d))
		},
		c: func(i *Input, d *Data) (*Completion, error) {
			return nil, f(i, d)
		},
	}
}

// NodeRepeater is a `Processor` that runs the provided Node at least `minN` times and up to `minN + optionalN` times.
// It should work with most node types, but hasn't been tested with branch nodes and flags really.
// Additionally, any argument nodes under it should probably use `CustomSetter` arg options.
func NodeRepeater(n Node, minN, optionalN int) Processor {
	return &nodeRepeater{minN, optionalN, n}
}

type nodeRepeater struct {
	minN      int
	optionalN int
	n         Node
}

func (nr *nodeRepeater) Usage(u *Usage) {
	nu := GetUsage(nr.n)

	// Merge UsageSection
	for k1, m := range *nu.UsageSection {
		for k2, v := range m {
			u.UsageSection.Add(k1, k2, v...)
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
		if err := processGraphExecution(nr.n, i, ieo, d, e, false); err != nil && !IsExtraArgsError(err) {
			return err
		}
	}
	// A not enough args error will, presumably, be returned by
	// one of the iterativeExecute functions if necessary
	return nil
}

func (nr *nodeRepeater) Complete(i *Input, d *Data) (*Completion, error) {
	for exCount := 0; nr.proceedCondition(exCount, i); exCount++ {
		c, err := processGraphCompletion(nr.n, i, d, false)
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

// MenuFlag returns an `Arg` that is required to be one of the provided choices.
func MenuFlag[T comparable](name string, shortName rune, desc string, choices ...T) FlagWithType[T] {
	var strChoices []string
	op := getOperator[T]()
	for _, c := range choices {
		strChoices = append(strChoices, op.toArgs(c)...)
	}
	return Flag[T](name, shortName, desc, SimpleCompleter[T](strChoices...), InList(choices...))
}

// MenuArg returns an `Arg` that is required to be one of the provided choices.
func MenuArg[T comparable](name, desc string, choices ...T) *ArgNode[T] {
	var strChoices []string
	op := getOperator[T]()
	for _, c := range choices {
		strChoices = append(strChoices, op.toArgs(c)...)
	}
	return Arg[T](name, desc, SimpleCompleter[T](strChoices...), InList(choices...))
}

// TODO: Remove this (or hide this) in favor of ItemizedListFlag?
// ListBreakerOption is an option type for the `ListBreaker` type.
type ListBreakerOption[T any] func(*ListBreaker[T])

func newBreakerOpt[T any](f func(*ListBreaker[T])) ListBreakerOption[T] {
	return f
}

// DiscardBreaker is a `ListBreakerOption` that removes the breaker argument from the input (rather than keeping it for the next node to parse).
func DiscardBreaker[T any]() ListBreakerOption[T] {
	return newBreakerOpt(func(lb *ListBreaker[T]) {
		lb.discard = true
	})
}

// ListBreakerUsage is a `ListBreakerOption` that inlcudes usage info in the command's usage text.
func ListBreakerUsage[T any](uf func(*Usage)) ListBreakerOption[T] {
	return newBreakerOpt(func(lb *ListBreaker[T]) {
		lb.u = uf
	})
}

// ListUntilSymbol returns an unbounded list node that ends when a specific symbol is parsed.
func ListUntilSymbol[T any](symbol string, opts ...ListBreakerOption[T]) *ListBreaker[T] {
	return ListUntil[T](NEQ(symbol)).AddOptions(append(opts, ListBreakerUsage[T](func(u *Usage) {
		u.Usage = append(u.Usage, symbol)
		u.UsageSection.Add(SymbolSection, symbol, "List breaker")
	}))...)
}

// AddOptions adds `ListBreakerOptions` to a `ListBreaker` object.
func (lb *ListBreaker[T]) AddOptions(opts ...ListBreakerOption[T]) *ListBreaker[T] {
	for _, opt := range opts {
		opt(lb)
	}
	return lb
}

// ListUntil returns a `ListBreaker` node that breaks when any of the provided `ValidatorOptions` are not satisfied.
func ListUntil[T any](validators ...*ValidatorOption[string]) *ListBreaker[T] {
	return &ListBreaker[T]{
		validators: validators,
	}
}

// ListBreaker is an `ArgOpt` for breaking out of lists with an optional number of arguments.
type ListBreaker[T any] struct {
	validators []*ValidatorOption[string]
	discard    bool
	u          func(*Usage)
}

func (lb *ListBreaker[T]) Validate(s string) error {
	for _, v := range lb.validators {
		if err := v.Validate(s); err != nil {
			return err
		}
	}
	return nil
}

func (lb *ListBreaker[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.breakers = append(ao.breakers, lb)
}

// Validators returns the `ListBreaker`'s validators.
func (lb *ListBreaker[T]) Validators() []*ValidatorOption[string] {
	return lb.validators
}

// DiscardBreak indicates whether the `ListBreaker` discards the argument that breaks the list.
func (lb *ListBreaker[T]) DiscardBreak() bool {
	return lb.discard
}

// Usage updates the provided `Usage` object.
func (lb *ListBreaker[T]) Usage(u *Usage) {
	if lb.u != nil {
		lb.u(u)
	}
}

// StringListListNode parses a two-dimensional slice of strings, with each slice being separated by `breakSymbol`
func StringListListNode(name, desc, breakSymbol string, minN, optionalN int, opts ...ArgOpt[[]string]) Processor {
	n := &SimpleNode{
		Processor: ListArg(name, desc, 0, UnboundedList,
			append(opts,
				ListUntilSymbol(breakSymbol, DiscardBreaker[[]string]()),
				&CustomSetter[[]string]{func(sl []string, d *Data) {
					if len(sl) > 0 {
						if !d.Has(name) {
							d.Set(name, [][]string{sl})
						} else {
							d.Set(name, append(GetData[[][]string](d, name), sl))
						}
					}
				}},
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

// FunctionWrap sets ExecuteData.FunctionWrap to true.
func FunctionWrap() Processor {
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		ed.FunctionWrap = true
		return nil
	}, nil)
}

// FileContents converts a filename into the file's contents.
func FileContents(name, desc string, opts ...ArgOpt[string]) Processor {
	fc := FileNode(name, desc, opts...)
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		if err := processOrExecute(fc, i, o, d, ed); err != nil {
			return err
		}
		b, err := os.ReadFile(d.String(name))
		if err != nil {
			return o.Annotatef(err, "failed to read fileee")
		}
		d.Set(name, strings.Split(strings.TrimSpace(string(b)), "\n"))
		return nil
	}, func(i *Input, d *Data) (*Completion, error) {
		return processOrComplete(fc, i, d)
	})
}

// EchoExecuteDataProcessor is a `Processor` that outputs the current ExecuteData contents.
type EchoExecuteDataProcessor struct {
	// Stderr is whether or not the output should be written to Stderr instead.
	Stderr bool
	// Format
	Format string
}

func (e *EchoExecuteDataProcessor) Execute(_ *Input, o Output, _ *Data, ed *ExecuteData) error {
	if e.Format != "" && len(ed.Executable) > 0 {
		if e.Stderr {
			o.Stderrf(e.Format, strings.Join(ed.Executable, "\n"))
		} else {
			o.Stdoutf(e.Format, strings.Join(ed.Executable, "\n"))
		}
		return nil
	}

	for _, s := range ed.Executable {
		if e.Stderr {
			o.Stderrln(s)
		} else {
			o.Stdoutln(s)
		}
	}
	return nil
}

func (e *EchoExecuteDataProcessor) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

func (e *EchoExecuteDataProcessor) Usage(*Usage) {}

// EchoExecuteData returns a `Processor` that sends the `ExecuteData` contents
// to stdout.
func EchoExecuteData() Processor {
	return &EchoExecuteDataProcessor{}
}

// EchoExecuteDataf returns a `Processor` that sends the `ExecuteData` contents
// to stdout with the provided format
func EchoExecuteDataf(format string) Processor {
	return &EchoExecuteDataProcessor{Format: format}
}

const (
	// GetwdKey is the `Data` key used by `GetwdProcessor` and `Getwd`.
	GetwdKey = "GETWD"
)

// Getwd retrieves the current directory from `Data` (as set by
// `GetwdProcessor`).
func Getwd(d *Data) string {
	return d.String(GetwdKey)
}

var (
	osGetwd     = os.Getwd
	filepathRel = filepath.Rel
)

// StubGetwdProcessor uses the provided string and error when calling command.GetwdProcessor.
// TODO: Change to StubGetwd because this actually stubs getwd (used by cd)
func StubGetwdProcessor(t *testing.T, wd string, err error) {
	StubValue(t, &osGetwd, func() (string, error) {
		return wd, err
	})
}

// MapArg returns a `Processor` that converts an input key into it's value.
func MapArg[K constraints.Ordered, V any](name, desc string, m map[K]V, allowMissing bool) *MapArgNode[K, V] {
	var keys []string
	for _, k := range maps.Keys(m) {
		keys = append(keys, fmt.Sprintf("%v", k))
	}
	ma := &MapArgNode[K, V]{}
	opts := []ArgOpt[K]{
		SimpleCompleter[K](keys...),
		&CustomSetter[K]{F: func(key K, d *Data) {
			d.Set(name, m[key])
			ma.key = key
		}},
	}

	if !allowMissing {
		opts = append(opts, &ValidatorOption[K]{
			func(k K) error {
				if _, ok := m[k]; !ok {
					return fmt.Errorf("[MapArg] key is not in map")
				}
				return nil
			},
			"MapArg",
		})
	}
	ma.ArgNode = Arg(name, desc, opts...)
	return ma
}

type MapArgNode[K constraints.Ordered, V any] struct {
	*ArgNode[K]
	key K
}

// Get overrides the Arg.Get function to return V (rather than type K).
func (man *MapArgNode[K, V]) Get(d *Data) V {
	return GetData[V](d, man.name)
}

// GetKey returns the key that was set by the am
func (man *MapArgNode[K, V]) GetKey() K {
	return man.key
}

// GetOrDefault overrides the Arg.GetOrDefault function to return V (rather than type K).
func (man *MapArgNode[K, V]) GetOrDefault(d *Data, dflt V) V {
	if d.Has(man.name) {
		return GetData[V](d, man.name)
	}
	return dflt
}

// GetwdProcessor returns a processor that stores the present directory in `Data`.
// Use the `Getwd` function to retrieve its value.
func GetwdProcessor() Processor {
	return SuperSimpleProcessor(func(i *Input, d *Data) error {
		s, err := osGetwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %v", err)
		}
		d.Set(GetwdKey, s)
		return nil
	})
}

func getwd() (string, error) {
	return os.Getwd()
}

// PrintlnProcessor returns a `Processor` that runs `output.Stdoutln(v)`.
func PrintlnProcessor(v string) Processor {
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		o.Stdoutln(v)
		return nil
	}, nil)
}
