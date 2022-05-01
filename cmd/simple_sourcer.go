package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

const (
	primaryArg   = "primary"
	secondaryArg = "secondary"
)

func main() {
	os.Exit(sourcerer.Source(&Todo{}))
}

type Todo struct {
	Items map[string]map[string]bool

	changed bool
}

func (tl *Todo) Load(jsn string) error {
	if err := json.Unmarshal([]byte(jsn), tl); err != nil {
		return fmt.Errorf("failed to unmarshal todo json: %v", err)
	}
	return nil
}

func (tl *Todo) ListItems(output command.Output, data *command.Data) {
	ps := make([]string, 0, len(tl.Items))
	count := 0
	for k, v := range tl.Items {
		ps = append(ps, k)
		count += len(v)
	}
	sort.Strings(ps)

	for _, p := range ps {
		output.Stdout(p)
		ss := make([]string, 0, len(tl.Items[p]))
		for s := range tl.Items[p] {
			ss = append(ss, s)
		}
		sort.Strings(ss)
		for _, s := range ss {
			output.Stdout(fmt.Sprintf("  %s", s))
		}
	}
}

func (tl *Todo) Setup() []string { return nil }

func (tl *Todo) Changed() bool {
	return tl.changed
}

func (tl *Todo) AddItem(output command.Output, data *command.Data) error {
	if tl.Items == nil {
		tl.Items = map[string]map[string]bool{}
		tl.changed = true
	}

	p := data.String(primaryArg)
	if _, ok := tl.Items[p]; !ok {
		tl.Items[p] = map[string]bool{}
		tl.changed = true
	}

	if data.Has(secondaryArg) {
		s := data.String(secondaryArg)
		if tl.Items[p][s] {
			return output.Stderrf("item %q, %q already exists", p, s)
		}
		tl.Items[p][s] = true
		tl.changed = true
	} else if !tl.changed {
		return output.Stderrf("primary item %q already exists", p)
	}
	return nil
}

func (tl *Todo) DeleteItem(output command.Output, data *command.Data) error {
	if tl.Items == nil {
		return output.Stderr("can't delete from empty list")
	}

	p := data.String(primaryArg)
	if _, ok := tl.Items[p]; !ok {
		return output.Stderrf("Primary item %q does not exist", p)
	}

	// Delete secondary if provided
	if data.Has(secondaryArg) {
		s := data.String(secondaryArg)
		if tl.Items[p][s] {
			delete(tl.Items[p], s)
			tl.changed = true
			return nil
		} else {
			return output.Stderrf("Secondary item %q does not exist", s)
		}
	}

	if len(tl.Items[p]) != 0 {
		return output.Stderr("Can't delete primary item that still has secondary items")
	}

	delete(tl.Items, p)
	tl.changed = true
	return nil
}

// Name returns the name of the CLI.
func (tl *Todo) Name() string {
	return "todo"
}

type fetcher struct {
	List *Todo
	// Primary is whether or not to complete primary or secondary result.
	Primary bool
}

func (f *fetcher) Fetch(value string, data *command.Data) (*command.Completion, error) {
	if f.Primary {
		primaries := make([]string, 0, len(f.List.Items))
		for p := range f.List.Items {
			primaries = append(primaries, p)
		}
		return &command.Completion{
			Suggestions: primaries,
		}, nil
	}

	p := data.String(primaryArg)
	sMap := f.List.Items[p]
	secondaries := make([]string, 0, len(sMap))
	for s := range sMap {
		secondaries = append(secondaries, s)
	}
	return &command.Completion{
		Suggestions: secondaries,
	}, nil
}

func (tl *Todo) Node() *command.Node {
	pf := &command.Completor[string]{
		Fetcher: &fetcher{
			List:    tl,
			Primary: true,
		},
	}
	sf := &command.Completor[string]{
		Fetcher: &fetcher{List: tl},
	}
	return command.BranchNode(
		map[string]*command.Node{
			"a": command.SerialNodes(
				command.Arg[string](primaryArg, "primary", pf),
				command.OptionalArg[string](secondaryArg, "secondary"),
				command.ExecuteErrNode(tl.AddItem),
			),
			"d": command.SerialNodes(
				command.Arg[string](primaryArg, "primary", pf),
				command.OptionalArg[string](secondaryArg, "secondary", sf),
				command.ExecuteErrNode(tl.DeleteItem),
			),
		},
		command.SerialNodes(command.ExecutorNode(tl.ListItems)),
	)
}
