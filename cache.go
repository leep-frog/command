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
		Edge: &cacheUsageNode{n},
	}
}

type cacheUsageNode struct {
	n *Node
}

func (cun *cacheUsageNode) Next(i *Input, d *Data) (*Node, error) {
	return nil, nil
}

func (cun *cacheUsageNode) UsageNext() *Node {
	return cun.n
}

type commandCache struct {
	name string
	c    CachableCLI
	n    *Node
}

func (cc *commandCache) Usage(u *Usage) {
	u.UsageSection.Add(SymbolSection, "^", "Start of new cachable section")
	u.Usage = append(u.Usage, "^")
}

func (cc *commandCache) Complete(input *Input, data *Data) (*Completion, error) {
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

	// Don't cache if retrying will never fix the issue (outside of a change
	// to the code for the specific CLI).
	if IsExtraArgsError(err) || IsNotEnoughArgsError(err) {
		return err
	}

	// Even if it resulted in an error, we want to add the command to the cache.
	s := input.GetSnapshot(snapshot)
	if existing := cc.c.Cache()[cc.name]; !sliceEquals(existing, s) {
		cc.c.Cache()[cc.name] = s
		cc.c.MarkChanged()
	}
	return err
}

func sliceEquals(this, that []string) bool {
	if len(this) != len(that) {
		return false
	}
	for i := range this {
		if this[i] != that[i] {
			return false
		}
	}
	return true
}
