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
	ShortcutArg = Arg[string]("SHORTCUT", "Name of the shortcut", MinLength[string, string](1))
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

// ShortcutNode wraps the provided node with a shortcut node.
// It is important to note that any argument transformers should be idempotent
// (e.g. for some string variable `s`, `transformer.F(s) == transformer.F(transformer.F(s))`)
// as the transformer will run when the shortcut values are added to a shortcut
// *and* after the shortcut values are popped when using the shortcut with those
// values. The transformation when adding the shortcut was intentionally
// implemented mostly for file arguments, so that if you use a relative path for
// a file argument when creating your shortcut, it will still work with the same
// file when executing your shortcut from another directory.
// The second transformation wasn't intentionally added, but removing it
// requires multiple isolated types to know the inner workings of each other
// (mainly `ShorcutNode`, `Argument`, `Flag`, and `Input`). That makes all of
// those implementations more closely tied together and less independent and
// robust which is why there will not be work done to remove/fix the second
// transformation occurrence.
func ShortcutNode(name string, sc ShortcutCLI, n command.Node) command.Node {
	as := &addShortcut{node: n, sc: sc, name: name}
	shortcutBn := &BranchNode{
		Branches: map[string]command.Node{
			"add a":    SerialNodes(ShortcutArg, as),
			"delete d": shortcutDeleter(name, sc, n),
			"get g":    shortcutGetter(name, sc, n),
			"list l":   shortcutLister(name, sc, n),
			"search s": shortcutSearcher(name, sc, n),
		},
	}
	as.branchNode = shortcutBn

	executor := SerialNodes(&executeShortcut{node: n, sc: sc, name: name}, n)
	return &BranchNode{
		Branches: map[string]command.Node{
			"shortcuts": shortcutBn,
		},
		// This hides the `shortcuts` branch, but the help doc for it
		// can still be obtained by running `cmd ... shortcuts --help`.
		BranchUsageOrder:  []string{},
		Default:           executor,
		DefaultCompletion: true,
	}
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

func shortcutListArg(name string, sc ShortcutCLI) command.Processor {
	return ListArg[string](ShortcutArg.Name(), ShortcutArg.usageDescription(), 1, command.UnboundedList, CompleterList(shortcutCompleter(name, sc)))
}

func shortcutSearcher(name string, sc ShortcutCLI, n command.Node) command.Node {
	regexArg := ListArg[string]("REGEXP", "Regexp values with which shortcut names will be searched", 1, command.UnboundedList, ListifyValidatorOption(IsRegex()))
	return SerialNodes(regexArg, &ExecutorProcessor{func(output command.Output, data *command.Data) error {
		var rs []*regexp.Regexp
		for _, s := range regexArg.Get(data) {
			// MustCompile can be used safely because of the `IsRegex()` validator used in the regexpArg definition.
			rs = append(rs, regexp.MustCompile(s))
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
	u.AddSymbol("{ shortcuts }", "Start of new shortcut-able section. This is usable by providing the `shortcuts` keyword in this position. Run `cmd ... shortcuts --help` for more details")
	return nil
}

func (ea *executeShortcut) Execute(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	return output.Err(processOrExecute(shortcutInputTransformer(ea.sc, ea.name, 0), input, output, data, eData))
}

func (ea *executeShortcut) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	return processOrComplete(shortcutInputTransformer(ea.sc, ea.name, 0), input, data)
}

type addShortcut struct {
	node       command.Node
	sc         ShortcutCLI
	name       string
	branchNode *BranchNode
}

func (as *addShortcut) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	u.AddArg("SHORTCUT_VALUE", strings.Join([]string{
		"These are values that will be added to the shortcut.",
		"They must follow the same usage pattern as the command.Node passed to the ShortcutNode function.",
	}, " "), 1, command.UnboundedList)
	return nil
}

func (as *addShortcut) Execute(input *command.Input, output command.Output, data *command.Data, _ *command.ExecuteData) error {
	shortcut := data.String(ShortcutArg.Name())
	if as.branchNode.IsBranch(shortcut) {
		return output.Stderrf("cannot create shortcut for reserved value (%s)\n", shortcut)
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
	if err := processGraphExecution(as.node, input, ieo, data, fakeEData, IsNotEnoughArgsError); err != nil && !IsNotEnoughArgsError(err) {
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
