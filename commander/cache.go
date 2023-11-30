package commander

import (
	"fmt"
	"strings"

	"github.com/leep-frog/command/commondels"
)

var (
	cacheHistoryFlag     = Flag("cache-len", 'n', "Number of historical elements to display from the cache", Default(1))
	cachePrintPrefixFlag = BoolFlag("cache-prefix", 'p', "Include prefix arguments in print statement")
	cachePrefixData      = "CACHE_PREFIX_DATA"
	CacheDefaultHistory  = 100
)

// CachableCLI is an interface for CLIs that can store cached executions.
type CachableCLI interface {
	// GetCache returns a map from cache name to the last commands run for the CLI.
	Cache() map[string][][]string
	// MarkChanged marks the CLI as changed.
	MarkChanged()
}

// CacheNode returns a node that caches any execution of downstream commands.
// A `CacheNode` introduces new branches, hence the requirement for it to be a `commondels.Node`
// and not just a `commondels.Processor`.
func CacheNode(name string, c CachableCLI, n commondels.Node, opts ...CacheOption) commondels.Node {
	cc := &commandCache{
		name: name,
		c:    c,
		n:    n,
		ch:   &cacheHistory{CacheDefaultHistory},
	}
	for _, opt := range opts {
		opt.modifyCache(cc)
	}
	ccN := &SimpleNode{
		Processor: cc,
		Edge:      &cacheUsageNode{n},
	}
	return &BranchNode{
		Branches: map[string]commondels.Node{
			"history": SerialNodes(
				SimpleProcessor(func(input *commondels.Input, _ commondels.Output, data *commondels.Data, _ *commondels.ExecuteData) error {
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
				FlagProcessor(
					cacheHistoryFlag,
					cachePrintPrefixFlag,
				),
				SimpleProcessor(cc.history, nil),
			),
		},
		Default:           ccN,
		HideUsage:         true,
		DefaultCompletion: true,
		Synonyms:          BranchSynonyms(map[string][]string{"history": {"h"}}),
	}
}

// CacheOption is an option interface for modifying `CacheNode` objects.
type CacheOption interface {
	modifyCache(*commandCache)
}

// CacheHistory is a `CacheOption` for specifying the number of command executions that should be saved.
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
	n commondels.Node
}

func (cun *cacheUsageNode) Next(i *commondels.Input, d *commondels.Data) (commondels.Node, error) {
	return nil, nil
}

func (cun *cacheUsageNode) UsageNext(input *commondels.Input, data *commondels.Data) (commondels.Node, error) {
	return cun.n, nil
}

type commandCache struct {
	name string
	c    CachableCLI
	n    commondels.Node
	ch   *cacheHistory
}

func (cc *commandCache) history(input *commondels.Input, output commondels.Output, data *commondels.Data, _ *commondels.ExecuteData) error {
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
		output.Stdoutf("%s%s\n", prefix, strings.Join(sls[i], " "))
	}
	return nil
}

func (cc *commandCache) Usage(input *commondels.Input, data *commondels.Data, u *commondels.Usage) error {
	u.UsageSection.Add(commondels.SymbolSection, "^", "Start of new cachable section")
	u.Usage = append(u.Usage, "^")
	return nil
}

func (cc *commandCache) Complete(input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	return processGraphCompletion(cc.n, input, data)
}

func (cc *commandCache) Execute(input *commondels.Input, output commondels.Output, data *commondels.Data, eData *commondels.ExecuteData) error {
	// If it's fully processed, then populate inputs.
	if input.FullyProcessed() {
		if sls, ok := cc.c.Cache()[cc.name]; ok {
			input.PushFront(sls[len(sls)-1]...)
		}
		return processGraphExecution(cc.n, input, output, data, eData)
	}

	snapshot := input.Snapshot()
	err := processGraphExecution(cc.n, input, output, data, eData)

	// Don't cache if retrying will never fix the issue (outside of a change
	// to the code for the specific CLI).
	if IsNotEnoughArgsError(err) {
		return err
	}
	if !input.FullyProcessed() {
		// Extra args error will happen upstream
		return nil
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
