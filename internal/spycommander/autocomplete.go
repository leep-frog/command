package spycommander

import (
	"fmt"

	"github.com/leep-frog/command/command"
)

// Separate method for testing purposes (and so Data doesn't need to be
// constructed by callers).
func Autocomplete(n command.Node, compLine string, passthroughArgs []string, data *command.Data) (*command.Autocompletion, error) {
	input := command.ParseCompLine(compLine, passthroughArgs...)
	c, err := ProcessGraphCompletion(n, input, data)

	if c != nil {
		return &command.Autocompletion{
			c.ProcessInput(input),
			c.SpacelessCompletion,
		}, err
	}

	if c == nil && err == nil && !input.FullyProcessed() {
		err = command.ExtraArgsErr(input)
	}
	return nil, err
}

// Separate method for use by modifiers (shortcut.go, cache.go, etc.)
func ProcessGraphCompletion(n command.Node, input *command.Input, data *command.Data) (*command.Completion, error) {
	for n != nil {
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

func processDeferredCompletion(c *command.Completion, err error, input *command.Input, data *command.Data) (*command.Completion, error) {
	if err := ProcessGraphExecution(c.DeferredCompletion.Graph, input, command.NewIgnoreAllOutput(), data, &command.ExecuteData{}); err != nil {
		return nil, fmt.Errorf("failed to execute DeferredCompletion graph: %v", err)
	}

	if c.DeferredCompletion.F != nil {
		return c.DeferredCompletion.F(c, data)
	}

	return c, nil
}

// ProcessOrComplete checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or completes the processor accordingly.
func ProcessOrComplete(p command.Processor, input *command.Input, data *command.Data) (*command.Completion, error) {
	if n, ok := p.(command.Node); ok {
		return ProcessGraphCompletion(n, input, data)
	}
	return p.Complete(input, data)
}
