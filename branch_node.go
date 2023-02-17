package command

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

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
	// TODO: Change to HideBranchUsage and HideDefaultUsage
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
		return processGraphCompletion(bn.Default, input, data)
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
	// Set next to nil in case used multiple times
	bn.next = nil

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
	n := bn.next
	bn.next = nil
	return n, nil
}

func (bn *BranchNode) UsageNext(input *Input, data *Data) (Node, error) {
	return bn.Next(input, data)
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

func (bn *BranchNode) Usage(input *Input, data *Data, u *Usage) error {
	// Don't display usage if a branching argument is provided
	// if !input.FullyProcessed() {
	if input.NumRemaining() > 0 {
		return bn.getNext(input, data)
	}

	if bn.HideUsage {
		// Proceed to default
		if bn.Default != nil {
			return bn.getNext(input, data)
		}
		return nil
	}

	u.UsageSection.Set(SymbolSection, "<", "Start of subcommand branches")
	u.Usage = append(u.Usage, "<")

	bss := maps.Values(bn.getSyns())
	slices.SortFunc(bss, func(this, that *branchSyn) bool {
		return this.name < that.name
	})

	if bn.Default != nil {
		if err := processGraphUse(bn.Default, input, data, u); err != nil {
			return err
		}
	}

	for _, bs := range bss {
		// Input is empty at this point, so fine to pass the same input to all branches
		su, err := processNewGraphUse(bs.n, input)
		if err != nil {
			return err
		}
		v := bs.name
		if len(bs.values) > 0 {
			v = fmt.Sprintf("[%s|%s]", bs.name, strings.Join(bs.values, "|"))
		}
		su.Usage = append([]string{v}, su.Usage...)
		u.SubSections = append(u.SubSections, su)
	}
	return nil
}
