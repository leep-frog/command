package commander

import "fmt"

// type autocompleteFunctionBar[I, O, D, E, C, U, AC, N any] struct {
// parseCompLine func(string, []string) I
// extraAr
// }

type autocompleteFunctionBag[I input, O output, D any, E any, C completion[I], U, AC any, N node[I, O, D, E, C, U, N]] interface {
	ParseCompLine(string, []string) I
	ExtraArgsErr(I) error
	MakeAutocompletion(C, I) AC
	IgnoreAllOutput() O
	DeferredCompletionIsNil(C) bool
	DeferredCompletionGraph(C) N
	DeferredCompletionFunc(C) func(D) (C, error)
	MakeE() E
}

type completion[I any] interface {
	ProcessInput(I) []string
}

// Separate method for testing purposes (and so Data doesn't need to be
// constructed by callers).
func Autocomplete[I input, O output, D any, E any, C completion[I], U, AC any, N node[I, O, D, E, C, U, N]](n N, compLine string, passthroughArgs []string, data D, fb autocompleteFunctionBag[I, O, D, E, C, U, AC, N]) (AC, error) {
	input := fb.ParseCompLine(compLine, passthroughArgs)
	c, err := ProcessGraphCompletion[I, O, D, E, C, U, AC, N](n, input, data, fb)

	var ac AC
	if !isNil(c) {
		ac = fb.MakeAutocompletion(c, input)
	}

	if isNil(c) && err == nil && !input.FullyProcessed() {
		err = fb.ExtraArgsErr(input)
	}
	return ac, err
}

// Separate method for use by modifiers (shortcut.go, cache.go, etc.)
func ProcessGraphCompletion[I input, O output, D any, E any, C completion[I], U, AC any, N node[I, O, D, E, C, U, N]](n N, input I, data D, fb autocompleteFunctionBag[I, O, D, E, C, U, AC, N]) (C, error) {
	var nill C
	for !isNil(n) {
		c, err := n.Complete(input, data)
		if !isNil(c) || err != nil {
			if !isNil(c) && !fb.DeferredCompletionIsNil(c) {
				if err := ProcessGraphExecution[I, O, D, E, C, U, N](fb.DeferredCompletionGraph(c), input, fb.IgnoreAllOutput(), data, fb.MakeE()); err != nil {
					return nill, fmt.Errorf("failed to execute DeferredCompletion graph: %v", err)
				}
				return fb.DeferredCompletionFunc(c)(data)
			}
			return c, err
		}

		if n, err = n.Next(input, data); err != nil {
			return nill, err
		}
	}

	return nill, nil
}

// ProcessOrComplete checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or completes the processor accordingly.
func ProcessOrComplete[I input, O output, D, E any, C completion[I], U, AC any, N node[I, O, D, E, C, U, N]](p processor[I, O, D, E, C, U], input I, data D, fb autocompleteFunctionBag[I, O, D, E, C, U, AC, N]) (C, error) {
	if n, ok := p.(N); ok {
		return ProcessGraphCompletion[I, O, D, E, C, U, AC, N](n, input, data, fb)
	}
	return p.Complete(input, data)
}
