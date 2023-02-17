package command

import "fmt"

// Autocomplete returns the completion suggestions for the provided node, `COMP_LINE`,
// and `passthroughArgs` (`passthroughArgs` are used for `Aliaser` statements).
// The returned slice is a list of autocompletion suggestions, and the returned error
// indicates if there was an issue. The error can be sent to stderr without
// causing any autocompletion issues.
func Autocomplete(n Node, compLine string, passthroughArgs []string) ([]string, error) {
	return autocomplete(n, compLine, passthroughArgs, &Data{})
}

// Separate method for testing purposes (and so Data doesn't need to be
// constructed by callers).
func autocomplete(n Node, compLine string, passthroughArgs []string, data *Data) ([]string, error) {
	input := ParseCompLine(compLine, passthroughArgs)
	c, err := processGraphCompletion(n, input, data, true)

	var r []string
	if c != nil {
		r = c.ProcessInput(input)
	}
	return r, err
}

// Separate method for use by modifiers (shortcut.go, cache.go, etc.)
func processGraphCompletion(n Node, input *Input, data *Data, checkInput bool) (*Completion, error) {
	for n != nil {
		c, err := n.Complete(input, data)
		if c != nil || err != nil {
			if c != nil && c.DeferredCompletion != nil {
				if err := processGraphExecution(c.DeferredCompletion.Graph, input, NewIgnoreAllOutput(), data, &ExecuteData{}); err != nil {
					return nil, fmt.Errorf("failed to execute DeferredCompletion graph: %v", err)
				}
				return c.DeferredCompletion.F(data)
			}
			return c, err
		}

		if n, err = n.Next(input, data); err != nil {
			return nil, err
		}
	}

	if checkInput {
		if !input.FullyProcessed() {
			return nil, ExtraArgsErr(input)
		}
	}
	return nil, nil
}

// processOrComplete checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or completes the processor accordingly.
func processOrComplete(p Processor, input *Input, data *Data) (*Completion, error) {
	if n, ok := p.(Node); ok {
		return processGraphCompletion(n, input, data, false)
	}
	return p.Complete(input, data)
}
