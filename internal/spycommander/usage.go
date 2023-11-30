package spycommander

import (
	"strings"

	"github.com/leep-frog/command/command"
)

// Use constructs a `command.Usage` object from the root `command.Node` of a command graph.
func Use(root command.Node, input *command.Input) (*command.Usage, error) {
	u, err := ProcessNewGraphUse(root, input)
	if err != nil {
		return nil, err
	}

	// Note, we ignore ExtraArgsErr (by not checking input.FullyProcessed()
	return u, nil
}

// ProcessNewGraphUse processes the usage for provided graph
func ProcessNewGraphUse(root command.Node, input *command.Input) (*command.Usage, error) {
	u := &command.Usage{
		UsageSection: &command.UsageSection{},
	}
	// TODO: Add OS
	return u, ProcessGraphUse(root, input, &command.Data{}, u)
}

// ProcessGraphUse processes the usage for provided graph
func ProcessGraphUse(root command.Node, input *command.Input, data *command.Data, usage *command.Usage) error {
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

// ProcessOrUsage checks if the provided `Processor` is a `command.Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func ProcessOrUsage(p command.Processor, i *command.Input, d *command.Data, usage *command.Usage) error {
	if n, ok := p.(command.Node); ok {
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
func ShowUsageAfterError(n command.Node, o command.Output) {
	if u, err := ProcessNewGraphUse(n, command.ParseExecuteArgs(nil)); err != nil {
		o.Stderrf("\n%s\nfailed to get command usage: %v\n", UsageErrorSectionStart, err)
	} else if usageDoc := u.String(); len(strings.TrimSpace(usageDoc)) != 0 {
		o.Stderrf("\n%s\n%v\n", UsageErrorSectionStart, u)
	}
}
