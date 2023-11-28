package spycommander

import (
	"fmt"

	"github.com/leep-frog/command/commondels"
)

// Separate method for testing purposes (and so Data doesn't need to be
// constructed by callers).
func Autocomplete(n commondels.Node, compLine string, passthroughArgs []string, data *commondels.Data) (*commondels.Autocompletion, error) {
	input := commondels.ParseCompLine(compLine, passthroughArgs...)
	c, err := ProcessGraphCompletion(n, input, data)

	if c != nil {
		return &commondels.Autocompletion{
			c.ProcessInput(input),
			c.SpacelessCompletion,
		}, err
	}

	if c == nil && err == nil && !input.FullyProcessed() {
		err = commondels.ExtraArgsErr(input)
	}
	return nil, err
}

// Separate method for use by modifiers (shortcut.go, cache.go, etc.)
func ProcessGraphCompletion(n commondels.Node, input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	for !isNil(n) {
		c, err := n.Complete(input, data)

		// Proceed to next node if no completion and no error
		if c == nil && err == nil {
			if n, err = n.Next(input, data); err != nil {
				return nil, err
			}
			continue
		}

		// If completion or error, try to do DeferredCompletion if relevant
		if c != nil && c.DeferredCompletion != nil {
			return processDeferredCompletion(c, err, input, data)
		}

		// Otherwise, we are at the end, so just return
		return c, err
	}

	return nil, nil
}

func processDeferredCompletion(c *commondels.Completion, err error, input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	if err := ProcessGraphExecution(c.DeferredCompletion.Graph, input, commondels.NewIgnoreAllOutput(), data, &commondels.ExecuteData{}); err != nil {
		return nil, fmt.Errorf("failed to execute DeferredCompletion graph: %v", err)
	}

	if c.DeferredCompletion.F != nil {
		return c.DeferredCompletion.F(c, data)
	}

	return c, nil
}

// ProcessOrComplete checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or completes the processor accordingly.
func ProcessOrComplete(p commondels.Processor, input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	if n, ok := p.(commondels.Node); ok {
		return ProcessGraphCompletion(n, input, data)
	}
	return p.Complete(input, data)
}
