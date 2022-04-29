package command

import (
	"fmt"
	"sort"
	"strings"
)

var (
	shortcutArgName = "SHORTCUT"
	shortcutArg     = Arg[string](shortcutArgName, "TODO shortcut desc", MinLength(1))
)

type ShortcutCLI interface {
	// ShortcutMap returns a map from "shortcut type" to "shortcut name" to an array of "shortcut expansion".
	// This structure easily allows for one CLI to have multiple shortcut commands.
	ShortcutMap() map[string]map[string][]string
	// MarkChanged is called when a change has been made to the shortcut map.
	MarkChanged()
}

func getShortcutMap(sc ShortcutCLI, name string) map[string][]string {
	got, ok := sc.ShortcutMap()[name]
	if !ok {
		return nil
	}
	return got
}

func getShortcut(sc ShortcutCLI, name, shortcut string) ([]string, bool) {
	got := getShortcutMap(sc, name)
	if got == nil {
		return nil, false
	}
	v, ok := got[shortcut]
	return v, ok
}

func deleteShortcut(sc ShortcutCLI, name, shortcut string) error {
	m, _ := sc.ShortcutMap()[name]
	if _, ok := m[shortcut]; !ok {
		return fmt.Errorf("Shortcut %q does not exist", shortcut)
	}
	sc.MarkChanged()
	delete(m, shortcut)
	return nil
}

func setShortcut(sc ShortcutCLI, name, shortcut string, value []string) {
	sc.MarkChanged()
	m, ok := sc.ShortcutMap()[name]
	if !ok {
		m = map[string][]string{}
		sc.ShortcutMap()[name] = m
	}
	m[shortcut] = value
}

func shortcutMap(name string, sc ShortcutCLI, n *Node) map[string]*Node {
	adder := SerialNodes(shortcutArg, &addShortcut{node: n, sc: sc, name: name})
	return map[string]*Node{
		"a": adder,
		"d": shortcutDeleter(name, sc, n),
		"g": shortcutGetter(name, sc, n),
		"l": shortcutLister(name, sc, n),
		"s": shortcutSearcher(name, sc, n),
	}
}

func ShortcutNode(name string, sc ShortcutCLI, n *Node) *Node {
	executor := SerialNodesTo(n, &executeShortcut{node: n, sc: sc, name: name})
	return BranchNode(shortcutMap(name, sc, n), executor, HideBranchUsage(), DontCompleteSubcommands())
}

func shortcutCompletor(name string, sc ShortcutCLI) *Completor[string] {
	return &Completor[string]{
		Distinct: true,
		SuggestionFetcher: SimpleFetcher(func(v string, d *Data) (*Completion, error) {
			s := []string{}
			for k := range getShortcutMap(sc, name) {
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

func shortcutListArg(name string, sc ShortcutCLI) Processor {
	return ListArg[string](shortcutArgName, hiddenNodeDesc, 1, UnboundedList, CompletorList(shortcutCompletor(name, sc)))
}

func shortcutSearcher(name string, sc ShortcutCLI, n *Node) *Node {
	regexArg := ListArg[string]("regexp", hiddenNodeDesc, 1, UnboundedList, ValidatorList(IsRegex()))
	return SerialNodes(regexArg, ExecutorNode(func(output Output, data *Data) {
		rs := data.RegexpList("regexp")
		var as []string
		for k, v := range getShortcutMap(sc, name) {
			as = append(as, shortcutStr(k, v))
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

func shortcutLister(name string, sc ShortcutCLI, n *Node) *Node {
	return SerialNodes(ExecutorNode(func(output Output, data *Data) {
		var r []string
		for k, v := range getShortcutMap(sc, name) {
			r = append(r, shortcutStr(k, v))
		}
		sort.Strings(r)
		for _, v := range r {
			output.Stdout(v)
		}
	}))
}

func shortcutDeleter(name string, sc ShortcutCLI, n *Node) *Node {
	return SerialNodes(shortcutListArg(name, sc), ExecuteErrNode(func(output Output, data *Data) error {
		if len(getShortcutMap(sc, name)) == 0 {
			return output.Stderr("Shortcut group has no shortcuts yet.")
		}
		for _, a := range data.StringList(shortcutArgName) {
			if err := deleteShortcut(sc, name, a); err != nil {
				output.Err(err)
			}
		}
		return nil
	}))
}

func shortcutStr(shortcut string, values []string) string {
	return fmt.Sprintf("%s: %s", shortcut, strings.Join(values, " "))
}

func shortcutGetter(name string, sc ShortcutCLI, n *Node) *Node {
	return SerialNodes(shortcutListArg(name, sc), ExecuteErrNode(func(output Output, data *Data) error {
		if getShortcutMap(sc, name) == nil {
			return output.Stderrf("No shortcuts exist for shortcut type %q", name)
		}

		for _, shortcut := range data.StringList(shortcutArgName) {
			if v, ok := getShortcut(sc, name, shortcut); ok {
				output.Stdout(shortcutStr(shortcut, v))
			} else {
				output.Stderrf("Shortcut %q does not exist", shortcut)
			}
		}
		return nil
	}))
}

type executeShortcut struct {
	node *Node
	sc   ShortcutCLI
	name string
}

func (ea *executeShortcut) Usage(u *Usage) {
	u.UsageSection.Add(SymbolSection, "*", "Start of new shortcut-able section")
	// TODO: show shortcut subcommands on --help
	u.Usage = append(u.Usage, "*")
}

func (ea *executeShortcut) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	return output.Err(input.CheckShortcuts(1, ea.sc, ea.name, false))
}

func (ea *executeShortcut) Complete(input *Input, data *Data) (*Completion, error) {
	if err := input.CheckShortcuts(1, ea.sc, ea.name, true); err != nil {
		return nil, err
	}
	return nil, nil
}

type addShortcut struct {
	node *Node
	sc   ShortcutCLI
	name string
}

func (as *addShortcut) Usage(*Usage) {
	return
}

func (as *addShortcut) Execute(input *Input, output Output, data *Data, _ *ExecuteData) error {
	shortcut := data.String(shortcutArgName)
	sm := shortcutMap(as.name, as.sc, as.node)
	if _, ok := sm[shortcut]; ok {
		return output.Stderr("cannot create shortcut for reserved value")
	}
	if _, ok := getShortcut(as.sc, as.name, shortcut); ok {
		return output.Stderrf("Shortcut %q already exists", shortcut)
	}

	snapshot := input.Snapshot()

	// We don't want the executor to run, so we pass fakeEData to children nodes.
	fakeEData := &ExecuteData{}
	// We don't want to output not enough args error, because we actually
	// don't mind those when adding shortcuts.
	ieo := NewIgnoreErrOutput(output, IsNotEnoughArgsError)
	err := iterativeExecute(as.node, input, ieo, data, fakeEData)
	if err != nil && !IsNotEnoughArgsError(err) {
		return err
	}

	sl := input.GetSnapshot(snapshot)
	if len(sl) == 0 {
		return output.Err(NotEnoughArgs(shortcutArgName, 1, 0))
	}
	setShortcut(as.sc, as.name, shortcut, sl)
	return nil
}

func (as *addShortcut) Complete(input *Input, data *Data) (*Completion, error) {
	return getCompleteData(as.node, input, data)
}
