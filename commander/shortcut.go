package commander

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/leep-frog/command/command"
)

var (
	// ShortcutArg is the `Arg` used to check for shortcuts.
	ShortcutArg = Arg[string]("SHORTCUT", "TODO shortcut desc", MinLength[string, string](1))
)

// ShortcutCLI is the interface required for integrating with shortcut nodes.
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

func shortcutMap(name string, sc ShortcutCLI, n command.Node) map[string]command.Node {
	adder := SerialNodes(ShortcutArg, &addShortcut{node: n, sc: sc, name: name})
	return map[string]command.Node{
		// TODO: Trigger this section by a nested branch `shortcut add; shortcut delete; etc.`
		"a": adder,
		"d": shortcutDeleter(name, sc, n),
		"g": shortcutGetter(name, sc, n),
		"l": shortcutLister(name, sc, n),
		"s": shortcutSearcher(name, sc, n),
	}
}

// ShortcutNode wraps the provided node with a shortcut node.
func ShortcutNode(name string, sc ShortcutCLI, n command.Node) command.Node {
	executor := SerialNodes(&executeShortcut{node: n, sc: sc, name: name}, n)
	return &BranchNode{Branches: shortcutMap(name, sc, n), Default: executor, BranchUsageOrder: []string{}, DefaultCompletion: true}
}

func shortcutCompleter(name string, sc ShortcutCLI) Completer[string] {
	return CompleterFromFunc(func(string, *command.Data) (*command.Completion, error) {
		s := []string{}
		for k := range getShortcutMap(sc, name) {
			s = append(s, k)
		}
		return &command.Completion{
			Suggestions: s,
			Distinct:    true,
		}, nil
	})

}

const (
	hiddenNodeDesc = "hidden_node"
)

func shortcutListArg(name string, sc ShortcutCLI) command.Processor {
	return ListArg[string](ShortcutArg.Name(), hiddenNodeDesc, 1, command.UnboundedList, CompleterList(shortcutCompleter(name, sc)))
}

func shortcutSearcher(name string, sc ShortcutCLI, n command.Node) command.Node {
	regexArg := ListArg[string]("regexp", hiddenNodeDesc, 1, command.UnboundedList, ListifyValidatorOption(IsRegex()))
	return SerialNodes(regexArg, &ExecutorProcessor{func(output command.Output, data *command.Data) error {
		var rs []*regexp.Regexp
		for _, s := range data.StringList("regexp") {
			r, err := regexp.Compile(s)
			if err != nil {
				return output.Annotate(err, "failed to compile shortcut regexp")
			}
			rs = append(rs, r)
		}
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
				output.Stdoutln(a)
			}
		}
		return nil
	}})
}

func shortcutLister(name string, sc ShortcutCLI, n command.Node) command.Node {
	return SerialNodes(&ExecutorProcessor{func(output command.Output, data *command.Data) error {
		var r []string
		for k, v := range getShortcutMap(sc, name) {
			r = append(r, shortcutStr(k, v))
		}
		sort.Strings(r)
		for _, v := range r {
			output.Stdoutln(v)
		}
		return nil
	}})
}

func shortcutDeleter(name string, sc ShortcutCLI, n command.Node) command.Node {
	return SerialNodes(shortcutListArg(name, sc), &ExecutorProcessor{func(output command.Output, data *command.Data) error {
		if len(getShortcutMap(sc, name)) == 0 {
			return output.Stderrln("Shortcut group has no shortcuts yet.")
		}
		for _, a := range data.StringList(ShortcutArg.Name()) {
			if err := deleteShortcut(sc, name, a); err != nil {
				output.Err(err)
			}
		}
		return nil
	}})
}

func shortcutStr(shortcut string, values []string) string {
	return fmt.Sprintf("%s: %s", shortcut, strings.Join(values, " "))
}

func shortcutGetter(name string, sc ShortcutCLI, n command.Node) command.Node {
	return SerialNodes(shortcutListArg(name, sc), &ExecutorProcessor{func(output command.Output, data *command.Data) error {
		if getShortcutMap(sc, name) == nil {
			return output.Stderrf("No shortcuts exist for shortcut type %q\n", name)
		}

		for _, shortcut := range data.StringList(ShortcutArg.Name()) {
			if v, ok := getShortcut(sc, name, shortcut); ok {
				output.Stdoutln(shortcutStr(shortcut, v))
			} else {
				output.Stderrf("Shortcut %q does not exist\n", shortcut)
			}
		}
		return nil
	}})
}

type executeShortcut struct {
	node command.Node
	sc   ShortcutCLI
	name string
}

func (ea *executeShortcut) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	u.AddSymbol("*", "Start of new shortcut-able section")
	// TODO: show shortcut subcommands on --help
	// u.Usage = append(u.Usage, "*")
	return nil
}

func (ea *executeShortcut) Execute(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	return output.Err(processOrExecute(shortcutInputTransformer(ea.sc, ea.name, 0), input, output, data, eData))
}

func (ea *executeShortcut) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	return processOrComplete(shortcutInputTransformer(ea.sc, ea.name, 0), input, data)
}

type addShortcut struct {
	node command.Node
	sc   ShortcutCLI
	name string
}

func (as *addShortcut) Usage(*command.Input, *command.Data, *command.Usage) error {
	return nil
}

func (as *addShortcut) Execute(input *command.Input, output command.Output, data *command.Data, _ *command.ExecuteData) error {
	shortcut := data.String(ShortcutArg.Name())
	sm := shortcutMap(as.name, as.sc, as.node)
	if _, ok := sm[shortcut]; ok {
		return output.Stderrln("cannot create shortcut for reserved value")
	}
	if _, ok := getShortcut(as.sc, as.name, shortcut); ok {
		return output.Stderrf("Shortcut %q already exists\n", shortcut)
	}

	snapshot := input.Snapshot()

	// We don't want the executor to run, so we pass fakeEData to children nodes.
	fakeEData := &command.ExecuteData{}
	// We don't want to output not enough args error, because we actually
	// don't mind those when adding shortcuts.
	ieo := command.NewIgnoreErrOutput(output, IsNotEnoughArgsError)
	if err := processGraphExecution(as.node, input, ieo, data, fakeEData); err != nil && !IsNotEnoughArgsError(err) {
		return err
	}

	// Don't create the shortcut since it will always result in an error (the input check
	// will be done by the outer calls).
	if !input.FullyProcessed() {
		return nil
	}

	sl := input.GetSnapshot(snapshot)
	if len(sl) == 0 {
		return output.Err(NotEnoughArgs(ShortcutArg.Name(), 1, 0))
	}
	setShortcut(as.sc, as.name, shortcut, sl)
	return nil
}

func (as *addShortcut) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	return processGraphCompletion(as.node, input, data)
}
