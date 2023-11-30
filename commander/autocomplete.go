package commander

import (
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spycommander"
)

// Autocomplete returns the completion suggestions for the provided commondels.Node, `COMP_LINE`,
// and `passthroughArgs` (`passthroughArgs` are used for `Aliaser` statements).
// The returned slice is a list of autocompletion suggestions, and the returned error
// indicates if there was an issue. The error can be sent to stderr without
// causing any autocompletion issues.
func Autocomplete(n commondels.Node, compLine string, passthroughArgs []string, os commondels.OS) (*commondels.Autocompletion, error) {
	return autocomplete(n, compLine, passthroughArgs, &commondels.Data{OS: os})
}

// Separate method for testing purposes (and so commondels.Data doesn't need to be
// constructed by callers).
func autocomplete(n commondels.Node, compLine string, passthroughArgs []string, data *commondels.Data) (*commondels.Autocompletion, error) {
	return spycommander.Autocomplete(n, compLine, passthroughArgs, data)
}

func processGraphCompletion(n commondels.Node, input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	return spycommander.ProcessGraphCompletion(n, input, data)
}

func processOrComplete(p commondels.Processor, input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	return spycommander.ProcessOrComplete(p, input, data)
}
