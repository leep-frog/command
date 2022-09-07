package command

// Autocomplete returns the completion suggestions for the provided node, `COMP_LINE`,
// and `passthroughArgs` (`passthroughArgs` are used for `Aliaser` statements).
// The returned slice is a list of autocompletion suggestions, and the returned error
// indicates if there was an issue. The error can be sent to stderr without
// causing any autocompletion issues.
func Autocomplete(n *Node, compLine string, passthroughArgs []string) ([]string, error) {
	return autocomplete(n, compLine, passthroughArgs, &Data{})
}

// Separate method for testing purposes (and so Data doesn't need to be
// constructed by callers).
func autocomplete(n *Node, compLine string, passthroughArgs []string, data *Data) ([]string, error) {
	input := ParseCompLine(compLine, passthroughArgs)
	c, err := getCompleteData(n, input, data)

	var r []string
	if c != nil {
		r = c.ProcessInput(input)
	}
	return r, err
}

// Separate method for use by modifiers (shortcut.go, cache.go, etc.)
func getCompleteData(n *Node, input *Input, data *Data) (*Completion, error) {
	for n != nil {
		if n.Processor != nil {
			c, err := n.Processor.Complete(input, data)
			if c != nil || err != nil {
				return c, err
			}
		}

		if n.Edge == nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, data); err != nil {
			return nil, err
		}
	}

	if !input.FullyProcessed() {
		return nil, ExtraArgsErr(input)
	}
	return nil, nil
}
