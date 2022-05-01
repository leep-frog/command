package command

// Execute executes a node with the provided `Input` and `Output`.
// Autocomplete returns the completion suggestions for the provided node, `COMP_LINE`,
// and `passthroughArgs` (`passthroughArgs` are used for `Aliaser` statements).
func Autocomplete(n *Node, compLine string, passthroughArgs []string) []string {
	// Printing things out in autocomplete mode isn't feasible, so the error
	// is only really used for testing purposes, hence why it is ignored here.
	sl, _ := autocomplete(n, compLine, passthroughArgs, &Data{})
	return sl
}

// Separate method for testing purposes.
func autocomplete(n *Node, compLine string, passthroughArgs []string, data *Data) ([]string, error) {
	input := ParseCompLine(compLine, passthroughArgs)
	c, err := getCompleteData(n, input, data)

	var r []string
	if c != nil {
		r = c.Process(input)
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
