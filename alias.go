package command

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	aliasArgName = "ALIAS"
	// TODO: completor
	aliasArg   = StringNode(aliasArgName, &ArgOpt{Validators: []ArgValidator{MinLength(1)}})
	aliasesArg = StringListNode(aliasArgName, 1, UnboundedList, nil)
)

type AliasCLI interface {
	// AliasMap returns a map from "alias type" to "alias name" to an array of "aliased values".
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

func AliasNode(name string, ac AliasCLI, n *Node) *Node {
	// TODO: the default node should check for aliases.
	adder := SerialNodes(aliasArg, &addAlias{node: n, ac: ac, name: name})
	executor := SerialNodesTo(n, &executeAlias{node: n, ac: ac, name: name})
	return BranchNode(map[string]*Node{
		"a": adder,
		"d": aliasDeleter(name, ac, n),
		"g": aliasGetter(name, ac, n),
		"l": aliasLister(name, ac, n),
		"s": aliasSearcher(name, ac, n),
	}, executor)
}

func aliasSearcher(name string, ac AliasCLI, n *Node) *Node {
	// TODO: make regexp arg type.
	regexArg := StringListNode("regexp", 1, UnboundedList, nil)
	return SerialNodes(regexArg, ExecutorNode(func(output Output, data *Data) error {
		rs := []*regexp.Regexp{}
		for _, r := range data.Values["regexp"].StringList() {
			rx, err := regexp.Compile(r)
			if err != nil {
				return output.Stderr("Invalid regexp: %v", err)
			}
			rs = append(rs, rx)
		}

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
		return nil
	}))
}

func aliasLister(name string, ac AliasCLI, n *Node) *Node {
	return SerialNodes(ExecutorNode(func(output Output, data *Data) error {
		var r []string
		for k, v := range getAliasMap(ac, name) {
			r = append(r, aliasStr(k, v))
		}
		sort.Strings(r)
		for _, v := range r {
			output.Stdout(v)
		}
		return nil
	}))
}

func aliasDeleter(name string, ac AliasCLI, n *Node) *Node {
	aliasListArg := StringListNode(aliasArgName, 1, UnboundedList, nil /* TODO */)
	return SerialNodes(aliasListArg, ExecutorNode(func(output Output, data *Data) error {
		if len(getAliasMap(ac, name)) == 0 {
			return output.Stderr("Alias group has no aliases yet.")
		}
		for _, a := range data.Values[aliasArgName].StringList() {
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
	// TODO: merge with delete definition and add completor.
	aliasListArg := StringListNode(aliasArgName, 1, UnboundedList, nil)
	return SerialNodes(aliasListArg, ExecutorNode(func(output Output, data *Data) error {
		if getAliasMap(ac, name) == nil {
			return output.Stderr("No aliases exist for alias type %q", name)
		}

		for _, alias := range data.Values[aliasArgName].StringList() {
			if v, ok := getAlias(ac, name, alias); ok {
				output.Stdout(aliasStr(alias, v))
			} else {
				output.Stderr("Alias %q does not exist", alias)
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

func (ea *executeAlias) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	ea.checkForAlias(input, false)
	return nil
}

func (ea *executeAlias) checkForAlias(input *Input, complete bool) {
	// We don't care if the arg being completed is an alias so we shouldn't replace
	if complete && len(input.remaining) == 1 {
		return
	}

	nxt, ok := input.Peek()
	if !ok {
		return
	}
	sl, ok := getAlias(ea.ac, ea.name, nxt)
	if !ok {
		return
	}
	// TODO: verify that sl isn't 0?  Or only allow non-zero alias values.
	// Guaranteed to have at least one since we already peeked.
	end := len(sl) - 1
	input.Replace(sl[end])
	input.PushFront(sl[:end]...)
}

func (ea *executeAlias) Complete(input *Input, data *Data) *CompleteData {
	ea.checkForAlias(input, true)
	return nil
}

type addAlias struct {
	node *Node
	ac   AliasCLI
	name string
}

func (aa *addAlias) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	alias := data.Values[aliasArgName].String()
	if _, ok := getAlias(aa.ac, aa.name, alias); ok {
		return output.Stderr("Alias %q already exists", alias)
	}
	var remaining []int
	for _, r := range input.remaining {
		remaining = append(remaining, r)
	}
	n := aa.node
	// TODO: make function for this (used here and in execute.go)
	for n != nil {
		if n.Processor != nil {
			err := n.Processor.Execute(input, output, data, eData)
			if IsNotEnoughArgsErr(err) {
				break
			}

			if err != nil {
				// Check for not enough args error
				return err
			}
		}

		if n.Edge == nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, data); err != nil {
			return err
		}
	}

	if !input.FullyProcessed() {
		return output.Err(ExtraArgsErr(input))
	}

	var transformedArgs []string
	for _, r := range remaining {
		transformedArgs = append(transformedArgs, input.args[r])
	}
	// Remove the alias arg value.
	setAlias(aa.ac, aa.name, alias, transformedArgs)
	return nil
}

func (aa *addAlias) Complete(input *Input, data *Data) *CompleteData {
	return getCompleteData(aa.node, input, data)
}
