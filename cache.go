package command

import (
	"fmt"
	"strings"
)

var (
	cacheHistoryFlag     = NewFlag("cache-len", 'n', "Number of historical elements to display from the cache", Default(1))
	cachePrintPrefixFlag = BoolFlag("cache-prefix", 'p', "Include prefix arguments in print statement")
	cachePrefixData      = "CACHE_PREFIX_DATA"
	defaultHistory       = 100
)

// CachableCLI
type CachableCLI interface {
	// GetCache returns a map from cache name to the last commands run for the CLI.
	Cache() map[string][][]string
	// MarkChanged marks the CLI as changed.
	MarkChanged()
}

func CacheNode(name string, c CachableCLI, n *Node, opts ...CacheOption) *Node {
	cc := &commandCache{
		name: name,
		c:    c,
		n:    n,
		ch:   &cacheHistory{defaultHistory},
	}
	for _, opt := range opts {
		opt.modifyCache(cc)
	}
	ccN := &Node{
		Processor: cc,
		Edge:      &cacheUsageNode{n},
	}
	return BranchNode(map[string]*Node{
		"history": SerialNodes(
			SimpleProcessor(func(input *Input, _ Output, data *Data, _ *ExecuteData) error {
				used := input.Used()
				if len(used) <= 1 {
					// If only history arg is provided, then no prefix
					data.Set(cachePrefixData, "")
					return nil
				}
				// Remove "history" arg
				used = used[:len(used)-1]
				data.Set(cachePrefixData, fmt.Sprintf("%s ", strings.Join(used, " ")))
				return nil
			}, nil),
			NewFlagNode(
				cacheHistoryFlag,
				cachePrintPrefixFlag,
			),
			SimpleProcessor(cc.history, nil),
		),
	}, ccN, HideBranchUsage(), DontCompleteSubcommands(), BranchSynonyms(map[string][]string{"history": {"h"}}))
}

type CacheOption interface {
	modifyCache(*commandCache)
}

func CacheHistory(n int) CacheOption {
	return &cacheHistory{n}
}

type cacheHistory struct {
	number int
}

func (ch *cacheHistory) modifyCache(cc *commandCache) {
	cc.ch = ch
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
	ch   *cacheHistory
}

func (cc *commandCache) history(input *Input, output Output, data *Data, _ *ExecuteData) error {
	sls := cc.c.Cache()[cc.name]
	start := len(sls) - data.Int(cacheHistoryFlag.Name())
	if start < 0 {
		start = 0
	}

	var prefix string
	if data.Bool(cachePrintPrefixFlag.Name()) {
		prefix = data.String(cachePrefixData)
	}
	for i := start; i < len(sls); i++ {
		// TODO: Stdoutf shouldn't add newline?
		output.Stdoutf("%s%s", prefix, strings.Join(sls[i], " "))
	}
	return nil
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
		if sls, ok := cc.c.Cache()[cc.name]; ok {
			input.PushFront(sls[len(sls)-1]...)
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
	sls := cc.c.Cache()[cc.name]
	if len(sls) == 0 || !sliceEquals(sls[len(sls)-1], s) {
		sls = append(sls, s)
		cut := 1
		if cc.ch != nil {
			cut = cc.ch.number
			if cut > len(sls) {
				cut = len(sls)
			}
		}
		cc.c.Cache()[cc.name] = sls[len(sls)-cut:]
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
