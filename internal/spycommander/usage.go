package spycommander

import (
	"strings"

	"github.com/leep-frog/command/commondels"
)

// Use constructs a `commondels.Usage` object from the root `commondels.Node` of a command graph.
func Use(root commondels.Node, input *commondels.Input) (*commondels.Usage, error) {
	u, err := ProcessNewGraphUse(root, input)
	if err != nil {
		return nil, err
	}

	// Note, we ignore ExtraArgsErr (by not checking input.FullyProcessed()
	return u, nil
}

// ProcessNewGraphUse processes the usage for provided graph
func ProcessNewGraphUse(root commondels.Node, input *commondels.Input) (*commondels.Usage, error) {
	u := &commondels.Usage{
		UsageSection: &commondels.UsageSection{},
	}
	// TODO: Add OS
	return u, ProcessGraphUse(root, input, &commondels.Data{}, u)
}

// ProcessGraphUse processes the usage for provided graph
func ProcessGraphUse(root commondels.Node, input *commondels.Input, data *commondels.Data, usage *commondels.Usage) error {
	for n := root; n != nil; {
		if err := n.Usage(input, data, usage); err != nil {
			return err
		}

		var err error
		if n, err = n.UsageNext(input, data); err != nil {
			return err
		}
	}

	return nil
}

// ProcessOrUsage checks if the provided `Processor` is a `commondels.Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func ProcessOrUsage(p commondels.Processor, i *commondels.Input, d *commondels.Data, usage *commondels.Usage) error {
	if n, ok := p.(commondels.Node); ok {
		return ProcessGraphUse(n, i, d, usage)
	} else {
		return p.Usage(i, d, usage)
	}
}

const (
	UsageErrorSectionStart = "======= Command Usage ======="
)

// ShowUsageAfterError generates the usage doc for the provided `Node`. If there
// is no error generating the usage doc, then the doc is sent to stderr; otherwise,
// no output is sent.
func ShowUsageAfterError(n commondels.Node, o commondels.Output) {
	if u, err := ProcessNewGraphUse(n, commondels.ParseExecuteArgs(nil)); err != nil {
		o.Stderrf("\n%s\nfailed to get command usage: %v\n", UsageErrorSectionStart, err)
	} else if usageDoc := u.String(); len(strings.TrimSpace(usageDoc)) != 0 {
		o.Stderrf("\n%s\n%v\n", UsageErrorSectionStart, u)
	}
}
