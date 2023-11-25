package command

import "github.com/leep-frog/command/internal/spycommander"

// Autocompletion is a subset of the `Completion` type and contains only
// data relevant for the OS package to handle autocompletion logic.
type Autocompletion struct {
	// Suggestions is the set of autocomplete suggestions.
	Suggestions []string
	// SpacelessCompletion indicates that a space should *not* be added (which happens
	// automatically if there is only one completion suggestion).
	SpacelessCompletion bool
}

// Autocomplete returns the completion suggestions for the provided node, `COMP_LINE`,
// and `passthroughArgs` (`passthroughArgs` are used for `Aliaser` statements).
// The returned slice is a list of autocompletion suggestions, and the returned error
// indicates if there was an issue. The error can be sent to stderr without
// causing any autocompletion issues.
func Autocomplete(n Node, compLine string, passthroughArgs []string, os OS) (*Autocompletion, error) {
	return autocomplete(n, compLine, passthroughArgs, &Data{OS: os})
}

// Separate method for testing purposes (and so Data doesn't need to be
// constructed by callers).
func autocomplete(n Node, compLine string, passthroughArgs []string, data *Data) (*Autocompletion, error) {
	return spycommander.Autocomplete[*Input, Output, *Data, *ExecuteData, *Completion, *Usage, *Autocompletion, Node](n, compLine, passthroughArgs, data, afb)
}

var (
	afb = &autocompleteFunctionBag{}
)

type autocompleteFunctionBag struct{}

func (afb *autocompleteFunctionBag) ParseCompLine(compLine string, passthroughArgs []string) *Input {
	return ParseCompLine(compLine, passthroughArgs)
}

func (afb *autocompleteFunctionBag) ExtraArgsErr(i *Input) error {
	return i.extraArgsErr()
}

func (afb *autocompleteFunctionBag) IgnoreAllOutput() Output {
	return NewIgnoreAllOutput()
}

func (afb *autocompleteFunctionBag) MakeAutocompletion(c *Completion, input *Input) *Autocompletion {
	return &Autocompletion{
		c.ProcessInput(input),
		c.SpacelessCompletion,
	}
}

func (afb *autocompleteFunctionBag) DeferredCompletionIsNil(c *Completion) bool {
	return c.DeferredCompletion == nil
}

func (afb *autocompleteFunctionBag) DeferredCompletionGraph(c *Completion) Node {
	return c.DeferredCompletion.Graph
}

func (afb *autocompleteFunctionBag) DeferredCompletionFunc(c *Completion) func(*Data) (*Completion, error) {
	return c.DeferredCompletion.F
}

func (afb *autocompleteFunctionBag) MakeE() *ExecuteData {
	return &ExecuteData{}
}

func processGraphCompletion(n Node, input *Input, data *Data) (*Completion, error) {
	return spycommander.ProcessGraphCompletion[*Input, Output, *Data, *ExecuteData, *Completion, *Usage, *Autocompletion, Node](n, input, data, afb)
}

func processOrComplete(p Processor, input *Input, data *Data) (*Completion, error) {
	return spycommander.ProcessOrComplete[*Input, Output, *Data, *ExecuteData, *Completion, *Usage, *Autocompletion, Node](p, input, data, afb)
}
