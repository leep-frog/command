package commander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommander"
)

// Autocomplete returns the completion suggestions for the provided command.Node, `COMP_LINE`,
// and `passthroughArgs` (`passthroughArgs` are used for `Aliaser` statements).
// The returned slice is a list of autocompletion suggestions, and the returned error
// indicates if there was an issue. The error can be sent to stderr without
// causing any autocompletion issues.
func Autocomplete(n command.Node, compLine string, passthroughArgs []string, os command.OS) (*command.Autocompletion, error) {
	return autocomplete(n, compLine, passthroughArgs, &command.Data{OS: os})
}

// Separate method for testing purposes (and so command.Data doesn't need to be
// constructed by callers).
func autocomplete(n command.Node, compLine string, passthroughArgs []string, data *command.Data) (*command.Autocompletion, error) {
	return spycommander.Autocomplete(n, compLine, passthroughArgs, data)
}

func processGraphCompletion(n command.Node, input *command.Input, data *command.Data) (*command.Completion, error) {
	return spycommander.ProcessGraphCompletion(n, input, data)
}

func processOrComplete(p command.Processor, input *command.Input, data *command.Data) (*command.Completion, error) {
	return spycommander.ProcessOrComplete(p, input, data)
}
