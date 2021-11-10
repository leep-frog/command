package command

func Autocomplete(n *Node, compLine string) []string {
	// Printing things out in autocomplete mode isn't feasible, so the error
	// is only really used for testing purposes, hence why it is ignored here.
	sl, _ := autocomplete(n, compLine, &Data{})
	return sl
}

// Separate method for testing purposes.
func autocomplete(n *Node, compLine string, data *Data) ([]string, error) {
	input := ParseCompLine(compLine)
	c, err := getCompleteData(n, input, data)

	var r []string
	if c != nil {
		r = c.Process(input)
	}
	return r, err
}

// Separate method for testing purposes.
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

	// TODO: return error if not fully processed (extra args err)
	return nil, nil
}
