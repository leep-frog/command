package commander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommander"
)

// StringListListProcessor parses a two-dimensional slice of strings, with each slice being separated by `breakSymbol`
func StringListListProcessor(name, desc, breakSymbol string, minN, optionalN int, opts ...ArgumentOption[[]string]) command.Processor {
	lb := ListUntilSymbol(breakSymbol)
	lb.Discard = true

	n := &SimpleNode{
		Processor: ListArg(name, desc, 0, command.UnboundedList,
			append(opts,
				lb,
				&CustomSetter[[]string]{func(sl []string, d *command.Data) {
					if len(sl) > 0 {
						if !d.Has(name) {
							d.Set(name, [][]string{sl})
						} else {
							d.Set(name, append(command.GetData[[][]string](d, name), sl))
						}
					}
				}},
			)...,
		),
	}
	return NodeRepeater(n, minN, optionalN)
}

// NodeRepeater is a `command.Processor` that runs the provided command.Node at least `minN` times and up to `minN + optionalN` times.
// It should work with most node types, but hasn't been tested with branch nodes and flags really.
// Additionally, any argument nodes under it should probably use `CustomSetter` arg options.
func NodeRepeater(n command.Node, minN, optionalN int) command.Processor {
	return &nodeRepeater{minN, optionalN, n}
}

type nodeRepeater struct {
	minN      int
	optionalN int
	n         command.Node
}

func (nr *nodeRepeater) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	nu, err := spycommander.ProcessNewGraphUse(nr.n, i)
	if err != nil {
		return err
	}

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

	if nr.optionalN == command.UnboundedList {
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
	return nil
}

func (nr *nodeRepeater) proceedCondition(exCount int, i *command.Input) bool {
	// Keep going if...
	return (
	// we haven't run the minimum number of times
	exCount < nr.minN ||
		// there is more input AND there are optional cycles left
		(!i.FullyProcessed() && (nr.optionalN == command.UnboundedList || exCount < nr.minN+nr.optionalN)))
}

func (nr *nodeRepeater) Execute(i *command.Input, o command.Output, d *command.Data, e *command.ExecuteData) error {
	for exCount := 0; nr.proceedCondition(exCount, i); exCount++ {
		if err := processGraphExecution(nr.n, i, o, d, e); err != nil {
			return err
		}
	}
	// A not enough args error will, presumably, be returned by
	// one of the iterativeExecute functions if necessary
	return nil
}

func (nr *nodeRepeater) Complete(i *command.Input, d *command.Data) (*command.Completion, error) {
	for exCount := 0; nr.proceedCondition(exCount, i); exCount++ {
		c, err := processGraphCompletion(nr.n, i, d)
		if c != nil || (err != nil) {
			return c, err
		}
	}
	return nil, nil
}
