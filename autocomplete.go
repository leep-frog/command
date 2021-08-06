package command

func Autocomplete(n *Node, compLine string) []string {
	return autocomplete(n, compLine, &Data{})
}

// Separate method for testing purposes.
func autocomplete(n *Node, compLine string, data *Data) []string {
	input := ParseCompLine(compLine)
	cd := getCompleteData(n, input, data)
	if cd == nil {
		return nil
	}
	c := cd.Completion
	if c == nil {
		return nil
	}

	return append(c.Process(input))
}

// Separate method for testing purposes.
func getCompleteData(n *Node, input *Input, data *Data) *CompleteData {
	for n != nil {
		if n.Processor != nil {
			if c := n.Processor.Complete(input, data); c != nil {
				return c
			}
		}

		if n.Edge == nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, data); err != nil {
			return &CompleteData{
				Error: err,
			}
		}
	}

	return nil
}
