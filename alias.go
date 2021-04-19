package command

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

func setAlias(ac AliasCLI, name, alias string, value []string) {
	ac.MarkChanged()
	m, ok := ac.AliasMap()[name]
	if !ok {
		m = map[string][]string{}
		ac.AliasMap()[name] = m
	}
	m[alias] = value
}

func AliasNode(n *Node, ac AliasCLI, name string) *Node {
	// TODO: the default node should check for aliases.
	adder := SerialNodes(aliasArg, &addAlias{node: n, ac: ac, name: name})
	executor := SerialNodesTo(n, &executeAlias{node: n, ac: ac, name: name})
	return BranchNode(map[string]*Node{
		"a": adder,
	}, executor)
}

type executeAlias struct {
	node *Node
	ac   AliasCLI
	name string
}

func (ea *executeAlias) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	/*nxt, ok := input.Peek()
	if !ok {
		return nil
	}
	/sl, _ := getAlias(ea.ac, ea.name, nxt)
	input.*/
	return nil
}

func (ea *executeAlias) Complete(input *Input, data *Data) *CompleteData {
	return ea.node.Processor.Complete(input, data)
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
	return aa.node.Processor.Complete(input, data)
}
