package commander

import (
	"fmt"
	"sort"
	"strings"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommander"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// BranchNode implements a node that branches on specific string arguments.
// If the argument does not match any branch, then the `Default` node is traversed.
type BranchNode struct {
	// Branches is a map from branching argument to `command.Node` that should be
	// executed if that branching argument is provided.
	Branches map[string]command.Node
	// Synonyms are synonyms for branching arguments.
	Synonyms map[string]string
	// Default is the `command.Node` that should be executed if the branching argument
	// does not match of any of the branches.
	Default command.Node
	// BranchCompletions is whether or not branch arguments should be completed
	// or if the completions from the Default `command.Node` should be used.
	// This is only relevant when the branching argument is the argument
	// being completed. Otherwise, this node is executed as normal.
	DefaultCompletion bool
	// BranchUsageOrder allows you to set the order for branch usage docs.
	// If this is nil, then branches are sorted in alphabetical order.
	// If this is an empty list, then no branch usage is shown.
	BranchUsageOrder []string

	next command.Node
}

func (bn *BranchNode) sortBranchSyns() ([]*branchSyn, error) {
	syns := bn.getSyns()
	if bn.BranchUsageOrder == nil {
		bss := maps.Values(syns)
		sort.Slice(bss, func(i, j int) bool {
			this, that := bss[i], bss[j]
			return this.name < that.name
		})
		return bss, nil
	}

	got := map[string]bool{}
	var order []*branchSyn
	for _, branch := range bn.BranchUsageOrder {
		if _, ok := bn.Branches[branch]; !ok {
			return nil, fmt.Errorf("provided branch (%s) isn't a valid branch (note branch synonyms aren't allowed in BranchUsageOrder)", branch)
		}
		if got[branch] {
			return nil, fmt.Errorf("BranchUsageOrder contains a duplicate entry (%s)", branch)
		}
		got[branch] = true
		order = append(order, syns[branch])
	}

	return order, nil
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

func (bn *BranchNode) Execute(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	// The edge will figure out what needs to be done next.
	return output.Err(bn.getNext(input, data))
}

func (bn *BranchNode) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	if input.NumRemaining() > 1 {
		return nil, bn.getNext(input, data)
	}

	// Note: we don't try to merge completions between branches
	// and default node because the completion overlap can get
	// convoluted given the number of `command.Completion` options.
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

	return &command.Completion{
		Suggestions:     names,
		CaseInsensitive: true,
	}, nil
}

func (bn *BranchNode) getNext(input *command.Input, data *command.Data) error {
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
			input.Pop(data)
			bn.next = n
			return nil
		}

		for _, b := range syns {
			if s == b {
				input.Pop(data)
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

func (bn *BranchNode) Next(input *command.Input, data *command.Data) (command.Node, error) {
	n := bn.next
	bn.next = nil
	return n, nil
}

func (bn *BranchNode) UsageNext(input *command.Input, data *command.Data) (command.Node, error) {
	return bn.Next(input, data)
}

func (bn *BranchNode) splitBranch(b string) (string, []string) {
	r := strings.Split(b, " ")
	return r[0], r[1:]
}

type branchSyn struct {
	name   string
	values []string
	n      command.Node
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

func (bn *BranchNode) Usage(input *command.Input, data *command.Data, u *command.Usage) error {
	// Don't display usage if a branching argument is provided
	// if !input.FullyProcessed() {
	if input.NumRemaining() > 0 {
		return bn.getNext(input, data)
	}

	bss, err := bn.sortBranchSyns()
	if err != nil {
		return err
	}

	var branchUsages []*command.BranchUsage
	for _, bs := range bss {
		// command.Input is empty at this point, so fine to pass the same input to all branches
		su := &command.Usage{}

		name := bs.name
		if len(bs.values) > 0 {
			name = fmt.Sprintf("[%s|%s]", name, strings.Join(bs.values, "|"))
		}
		su.AddArg(name, "", 1, 0)
		err := spycommander.ProcessGraphUse(bs.n, input, data, su)
		if err != nil {
			return err
		}
		branchUsages = append(branchUsages, &command.BranchUsage{
			Usage: su,
			// TODO: NoLines?
		})
	}
	u.SetBranches(branchUsages)

	if bn.Default != nil {
		if err := spycommander.ProcessGraphUse(bn.Default, input, data, u); err != nil {
			return err
		}
	}
	return nil
}
