package spycommander

import (
	"fmt"

	"github.com/leep-frog/command/commondels"
)

/*type autocompleteFunctionBag[*commondels.Input input, O output, *commondels.Data any, E any, *commondels.Completion completion[*commondels.Input], U, *commondels.Autocompletion any, commondels.Node node[*commondels.Input, O, *commondels.Data, E, *commondels.Completion, U, commondels.Node]] interface {
	ParseCompLine(string, []string) *commondels.Input
	ExtraArgsErr(*commondels.Input) error
	MakeAutocompletion(*commondels.Completion, *commondels.Input) *commondels.Autocompletion
	IgnoreAllOutput() O
	DeferredCompletionIsNil(*commondels.Completion) bool
	DeferredCompletionGraph(*commondels.Completion) commondels.Node
	DeferredCompletionFunc(*commondels.Completion) func(*commondels.Data) (*commondels.Completion, error)
	MakeE() E
}*/

// Separate method for testing purposes (and so Data doesn't need to be
// constructed by callers).
func Autocomplete(n commondels.Node, compLine string, passthroughArgs []string, data *commondels.Data) (*commondels.Autocompletion, error) {
	input := commondels.ParseCompLine(compLine, passthroughArgs...)
	c, err := ProcessGraphCompletion(n, input, data)

	if c != nil {
		return &commondels.Autocompletion{
			c.ProcessInput(input),
			c.SpacelessCompletion,
		}, nil
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
		if c != nil || err != nil {
			if c != nil && c.DeferredCompletion != nil {
				if err := ProcessGraphExecution(c.DeferredCompletion.Graph, input, commondels.NewIgnoreAllOutput(), data, &commondels.ExecuteData{}); err != nil {
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

	return nil, nil
}

// ProcessOrComplete checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or completes the processor accordingly.
func ProcessOrComplete(p commondels.Processor, input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	if n, ok := p.(commondels.Node); ok {
		return ProcessGraphCompletion(n, input, data)
	}
	return p.Complete(input, data)
}
