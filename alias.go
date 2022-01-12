package command

import (
	"fmt"
	"sort"
	"strings"
)

var (
	aliasArgName = "ALIAS"
	aliasArg     = StringNode(aliasArgName, "TODO alias desc", MinLength(1))
)

type aliasedArg struct {
	Processor
}

type AliasCLI interface {
	// AliasMap returns a map from "alias type" to "alias name" to an array of "aliased".
	// This structure easily allows for one CLI to have multiple alias types.
	AliasMap() map[string]map[string][]string
	// MarkChanged is called when a change has been made to the alias map.
	MarkChanged()
}

func getAliasMap(ac AliasCLI, name string) map[string][]string {
	got, ok := ac.AliasMap()[name]
	if !ok {
		return nil
	}
	return got
}

func getAlias(ac AliasCLI, name, alias string) ([]string, bool) {
	got := getAliasMap(ac, name)
	if got == nil {
		return nil, false
	}
	v, ok := got[alias]
	return v, ok
}

func deleteAlias(ac AliasCLI, name, alias string) error {
	m, _ := ac.AliasMap()[name]
	if _, ok := m[alias]; !ok {
		return fmt.Errorf("Alias %q does not exist", alias)
	}
	ac.MarkChanged()
	delete(m, alias)
	return nil
}

func setAlias(ac AliasCLI, name, alias string, value []string) {
	ac.MarkChanged()
	m, ok := ac.AliasMap()[name]
	if !ok {
		m = map[string][]string{}
		ac.AliasMap()[name] = m
	}
	m[alias] = value
}

func aliasMap(name string, ac AliasCLI, n *Node) map[string]*Node {
	adder := SerialNodes(aliasArg, &addAlias{node: n, ac: ac, name: name})
	return map[string]*Node{
		"a": adder,
		"d": aliasDeleter(name, ac, n),
		"g": aliasGetter(name, ac, n),
		"l": aliasLister(name, ac, n),
		"s": aliasSearcher(name, ac, n),
	}
}

type aliasUsageNode struct {
	opNode    *Node
	usageNode *Node
}

func (un *aliasUsageNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	return un.opNode.Processor.Execute(input, output, data, eData)
}

func (un *aliasUsageNode) Complete(input *Input, data *Data) (*Completion, error) {
	return un.opNode.Processor.Complete(input, data)
}

func (un *aliasUsageNode) Usage(u *Usage) {
	u.UsageSection.Add(SymbolSection, "*", "Start of new aliasable section")
	// TODO: show alias subcommands on --help
	u.Usage = append(u.Usage, "*")

	if un.usageNode == nil || un.usageNode.Processor == nil {
		return
	}
	un.usageNode.Processor.Usage(u)
}

func (un *aliasUsageNode) Next(i *Input, d *Data) (*Node, error) {
	return un.opNode.Edge.Next(i, d)
}

func (un *aliasUsageNode) UsageNext() *Node {
	if un.usageNode == nil || un.usageNode.Edge == nil {
		return nil
	}
	return un.usageNode.Edge.UsageNext()
}

func AliasNode(name string, ac AliasCLI, n *Node) *Node {
	executor := SerialNodesTo(n, &executeAlias{node: n, ac: ac, name: name})
	uw := &aliasUsageNode{
		opNode:    BranchNode(aliasMap(name, ac, n), executor, DontCompleteSubcommands()),
		usageNode: n,
	}
	return &Node{
		Processor: uw,
		Edge:      uw,
	}
}

func aliasCompletor(name string, ac AliasCLI) *Completor {
	return &Completor{
		Distinct: true,
		SuggestionFetcher: SimpleFetcher(func(v *Value, d *Data) (*Completion, error) {
			s := []string{}
			for k := range getAliasMap(ac, name) {
				s = append(s, k)
			}
			return &Completion{
				Suggestions: s,
			}, nil
		}),
	}
}

const (
	hiddenNodeDesc = "hidden_node"
)

func aliasListArg(name string, ac AliasCLI) Processor {
	return StringListNode(aliasArgName, hiddenNodeDesc, 1, UnboundedList, aliasCompletor(name, ac))
}

func aliasSearcher(name string, ac AliasCLI, n *Node) *Node {
	regexArg := StringListNode("regexp", hiddenNodeDesc, 1, UnboundedList, ListIsRegex())
	return SerialNodes(regexArg, ExecutorNode(func(output Output, data *Data) {
		rs := data.RegexpList("regexp")
		var as []string
		for k, v := range getAliasMap(ac, name) {
			as = append(as, aliasStr(k, v))
		}
		sort.Strings(as)
		for _, a := range as {
			matches := true
			for _, r := range rs {
				if !r.MatchString(a) {
					matches = false
					break
				}
			}
			if matches {
				output.Stdout(a)
			}
		}
	}))
}

func aliasLister(name string, ac AliasCLI, n *Node) *Node {
	return SerialNodes(ExecutorNode(func(output Output, data *Data) {
		var r []string
		for k, v := range getAliasMap(ac, name) {
			r = append(r, aliasStr(k, v))
		}
		sort.Strings(r)
		for _, v := range r {
			output.Stdout(v)
		}
	}))
}

func aliasDeleter(name string, ac AliasCLI, n *Node) *Node {
	return SerialNodes(aliasListArg(name, ac), ExecuteErrNode(func(output Output, data *Data) error {
		if len(getAliasMap(ac, name)) == 0 {
			return output.Stderr("Alias group has no aliases yet.")
		}
		for _, a := range data.StringList(aliasArgName) {
			if err := deleteAlias(ac, name, a); err != nil {
				output.Err(err)
			}
		}
		return nil
	}))
}

func aliasStr(alias string, values []string) string {
	return fmt.Sprintf("%s: %s", alias, strings.Join(values, " "))
}

func aliasGetter(name string, ac AliasCLI, n *Node) *Node {
	return SerialNodes(aliasListArg(name, ac), ExecuteErrNode(func(output Output, data *Data) error {
		if getAliasMap(ac, name) == nil {
			return output.Stderrf("No aliases exist for alias type %q", name)
		}

		for _, alias := range data.StringList(aliasArgName) {
			if v, ok := getAlias(ac, name, alias); ok {
				output.Stdout(aliasStr(alias, v))
			} else {
				output.Stderrf("Alias %q does not exist", alias)
			}
		}
		return nil
	}))
}

type executeAlias struct {
	node *Node
	ac   AliasCLI
	name string
}

func (ea *executeAlias) Usage(u *Usage) {}

func (ea *executeAlias) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	return output.Err(input.CheckAliases(1, ea.ac, ea.name, false))
}

func (ea *executeAlias) Complete(input *Input, data *Data) (*Completion, error) {
	if err := input.CheckAliases(1, ea.ac, ea.name, true); err != nil {
		return nil, err
	}
	return nil, nil
}

type addAlias struct {
	node *Node
	ac   AliasCLI
	name string
}

func (aa *addAlias) Usage(*Usage) {
	return
}

func (aa *addAlias) Execute(input *Input, output Output, data *Data, _ *ExecuteData) error {
	alias := data.String(aliasArgName)
	am := aliasMap(aa.name, aa.ac, aa.node)
	if _, ok := am[alias]; ok {
		return output.Stderr("cannot create alias for reserved value")
	}
	if _, ok := getAlias(aa.ac, aa.name, alias); ok {
		return output.Stderrf("Alias %q already exists", alias)
	}

	snapshot := input.Snapshot()

	// We don't want the executor to run, so we pass fakeEData to children nodes.
	fakeEData := &ExecuteData{}
	// We don't want to output not enough args error, because we actually
	// don't mind those when adding aliases.
	ieo := NewIgnoreErrOutput(output, IsNotEnoughArgsError)
	err := iterativeExecute(aa.node, input, ieo, data, fakeEData)
	if err != nil && !IsNotEnoughArgsError(err) {
		return err
	}

	sl := input.GetSnapshot(snapshot)
	if len(sl) == 0 {
		return output.Err(NotEnoughArgs(aliasArgName, 1, 0))
	}
	setAlias(aa.ac, aa.name, alias, sl)
	return nil
}

func (aa *addAlias) Complete(input *Input, data *Data) (*Completion, error) {
	return getCompleteData(aa.node, input, data)
}
