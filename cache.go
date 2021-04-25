package command

/*
// CachableCLI
type CachableCLI interface {
	// GetCache returns a map from cache name to the last command run for the CLI.
	GetCache() map[string][]string
	// MarkChanged marks the CLI as changed.
	MarkChanged()
}

func Cache(name string, c CachableCLI, n *Node) *Node {
	return &Node{
		Processor: &commandCache{
			name: name,
			c:    c,
			n:    n,
		},
	}
}

type commandCache struct {
	name string
	c    CachableCLI
	n    *Node
}

func (cc *commandCache) Complete(input *Input, data *Data) *CompleteData {
	return getCompleteData(cc.n, input, data)
}

func (cc *commandCache) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	// TODO: [Part 1]: this is similar logic to alias. Can it be combined?
	// TODO: Additionally, this won't work if inputs are pushed so that can
	// be an issue when caching an aliased command.
	var remaining []int
	for _, r := range input.remaining {
		remaining = append(remaining, r)
	}

	// If it's fully processed, then populate inputs.
	if input.FullyProcessed() {
		if sl, ok := cc.c.GetCache()[name]; ok {
			input.PushFront(sl...)
		}
		return iterativeExecute(cc.n, input, output, data, eData, true)
	}

	err := iterativeExecute(cc.n, input, output, data, eData, true)

	// TODO: [Part 2]: see [Part 1] above
	var transformedArgs []string
	for _, r := range remaining {
		transformedArgs = append(transformedArgs, input.args[r])
	}
	// Remove the alias arg value.
	setAlias(aa.ac, aa.name, alias, transformedArgs)
	return nil
	err :=

	// Even if it resulted in an error, we want to add the command to the cache.
	cc.c.GetCache()[name] =

	return
}
*/
