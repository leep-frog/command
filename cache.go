package command

/*// CachableCLI
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
	return cc.n.Processor.Complete(input, data)
}

func (cc *commandCache) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	return cc.n.Processor.Execute(input, output, data, eData)
}
*/
