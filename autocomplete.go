package command

func Autocomplete(n *Node, args []string) []string {
	input := ParseArgs(args)
	cd := getCompleteData(n, input, &Data{})
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
