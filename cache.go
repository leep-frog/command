package command

// CachableCLI
type CachableCLI interface {
	// GetCache returns a map from cache name to the last command run for the CLI.
	Cache() map[string][]string
	// MarkChanged marks the CLI as changed.
	MarkChanged()
}

func CacheNode(name string, c CachableCLI, n *Node) *Node {
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
	// If it's fully processed, then populate inputs.
	if input.FullyProcessed() {
		if sl, ok := cc.c.Cache()[cc.name]; ok {
			input.PushFront(sl...)
		}
		return iterativeExecute(cc.n, input, output, data, eData)
	}

	snapshot := input.Snapshot()
	err := iterativeExecute(cc.n, input, output, data, eData)
	// Even if it resulted in an error, we want to add the command to the cache.
	cc.c.Cache()[cc.name] = input.GetSnapshot(snapshot)
	return err
}
