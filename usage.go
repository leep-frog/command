package command

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// ArgSection is the title of the arguments usage section.
	ArgSection = "Arguments"
	// FlagSection is the title of the flags usage section.
	FlagSection = "Flags"
	// SymbolSection is the title of the symbols usage section.
	SymbolSection = "Symbols"
)

// Use constructs a `Usage` object from the root `Node` of a command graph.
func Use(root Node, input *Input) (*Usage, error) {
	u, err := processNewGraphUse(root, input)
	if err != nil {
		return nil, err
	}

	if !input.FullyProcessed() {
		return nil, ExtraArgsErr(input)
	}
	return u, nil
}

// processNewGraphUse processes the usage for provided graph
func processNewGraphUse(root Node, input *Input) (*Usage, error) {
	u := &Usage{
		UsageSection: &UsageSection{},
	}
	// TODO: Add OS
	return u, processGraphUse(root, input, &Data{}, u)
}

// processGraphUse processes the usage for provided graph
func processGraphUse(root Node, input *Input, data *Data, usage *Usage) error {
	for n := root; n != nil; {
		if err := n.Usage(input, data, usage); err != nil {
			return err
		}

		var err error
		if n, err = n.UsageNext(input, data); err != nil {
			return err
		}
	}

	return nil
}

// processOrUsage checks if the provided `Processor` is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrUsage(p Processor, i *Input, d *Data, usage *Usage) error {
	if n, ok := p.(Node); ok {
		return processGraphUse(n, i, d, usage)
	} else {
		return p.Usage(i, d, usage)
	}
}

// Usage contains all data needed for constructing a command's usage text.
type Usage struct {
	// UsageSection is a map from section name to key phrase for that section to description for that key.
	// TODO: Only displayed when --help flag is provided
	UsageSection *UsageSection
	// Description is the usage doc for the command.
	Description string
	// Usage is arg usage string.
	Usage []string
	// Flags is the flag usage string.
	Flags []string

	// Subsections is a set of `Usage` objects that are indented from the current `Usage` object.
	SubSections []*Usage
}

// UsageSection is a map from section name to key phrase for that section to description for that key.
type UsageSection map[string]map[string][]string

// Add adds a usage section.
func (us *UsageSection) Add(section, key string, value ...string) {
	if (*us)[section] == nil {
		(*us)[section] = map[string][]string{}
	}
	(*us)[section][key] = append((*us)[section][key], value...)
}

func (us *UsageSection) Set(section, key string, value ...string) {
	if (*us)[section] == nil {
		(*us)[section] = map[string][]string{}
	}
	(*us)[section][key] = value
}

// String crates the full usage text as a single string.
func (u *Usage) String() string {
	var r []string
	r = u.string(r, 0)

	var sections []string
	if u.UsageSection != nil {
		for s := range *u.UsageSection {
			sections = append(sections, s)
		}
		sort.Strings(sections)
		for _, sk := range sections {
			r = append(r, fmt.Sprintf("%s:", sk))
			kvs := (*u.UsageSection)[sk]
			var keys []string
			for k := range kvs {
				keys = append(keys, k)
			}
			if sk == FlagSection {
				// We want to sort flags by full name, not short flags.
				// So, we trim "  [c]" from each flag description.
				sort.SliceStable(keys, func(i, j int) bool {
					return keys[i][4:] < keys[j][4:]
				})
			} else {
				sort.Strings(keys)
			}
			for _, k := range keys {
				for idx, kv := range kvs[k] {
					if idx == 0 {
						r = append(r, fmt.Sprintf("  %s: %s", k, kv))
					} else {
						r = append(r, fmt.Sprintf("    %s", kv))
					}
				}

			}

			// Since already split by newlines, this statement actually adds one newline.
			r = append(r, "")
		}
	}

	i := len(r) - 1
	for ; i > 0 && r[i-1] == ""; i-- {
	}
	r = r[:i]

	return strings.Join(r, "\n")
}

func (u *Usage) string(r []string, depth int) []string {
	prefix := strings.Repeat(" ", depth*2)
	if u.Description != "" {
		r = append(r, prefix+u.Description)
	}
	r = append(r, prefix+strings.Join(append(u.Usage, u.Flags...), " "))
	r = append(r, "") // since we join with newlines, this just adds one extra newline

	for _, su := range u.SubSections {
		r = su.string(r, depth+1)

		if su.UsageSection != nil {
			for section, m := range *su.UsageSection {
				for k, v := range m {
					// Subsections override the higher-level section
					// Mostly needed for duplicate sections (like nested branch nodes).
					u.UsageSection.Set(section, k, v...)
				}
			}
		}
	}
	return r
}

const (
	UsageErrorSectionStart = "======= Command Usage ======="
)

// ShowUsageAfterError generates the usage doc for the provided `Node`. If there
// is no error generating the usage doc, then the doc is sent to stderr; otherwise,
// no output is sent.
func ShowUsageAfterError(n Node, o Output) {
	if u, err := processNewGraphUse(n, ParseExecuteArgs(nil)); err != nil {
		o.Stderrf("\n%s\nfailed to get command usage: %v\n", UsageErrorSectionStart, err)
	} else if usageDoc := u.String(); len(strings.TrimSpace(usageDoc)) != 0 {
		o.Stderrf("\n%s\n%v\n", UsageErrorSectionStart, u)
	}
}
