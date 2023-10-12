package command

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/exp/slices"
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

	// Note, we ignore ExtraArgsErr (by not checking input.FullyProcessed()
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
	// SubSectionLines indicates whether or not to draw lines to the usage sub sections.
	SubSectionLines bool
}

// SubSection is a sub-usage section that will be indented accordingly.
type SubSection struct {
	// Usage is the usage info for the sub-section
	Usage *Usage

	// Lines is whether or not to draw lines to the nested elements
	DrawLines bool
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
	r = u.string(r, nil, nil, nil, true, true)

	var sections []string
	if u.UsageSection != nil && len(*u.UsageSection) > 0 {
		r = append(r, "")
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

			// Since already split by newlines, this statement simply adds one ore newline.
			r = append(r, "")
		}
	}

	// Remove all trailing newlines
	for ; len(r) > 0 && r[len(r)-1] == ""; r = r[:len(r)-1] {
	}

	return strings.Join(r, "\n")
}

func trimRightSpace(s string) string {
	return trailingWhitspaceRegex.ReplaceAllString(s, "")
}

const (
	noDrawLinePrefix = "  "
)

var (
	trailingNestedUsageRegex = regexp.MustCompile("\u2503(\\s*)$")
	trailingWhitspaceRegex   = regexp.MustCompile(`\s*$`)
	MIDDLE_ITEM_PREFIX       = map[bool]string{
		true:  "┣━━ ",
		false: noDrawLinePrefix,
	}
	NO_ITEM_PREFIX = map[bool]string{
		true:  "┃   ",
		false: noDrawLinePrefix,
	}
	END_ITEM_PREFIX = map[bool]string{
		true:  "┗━━ ",
		false: noDrawLinePrefix,
	}
)

func (u *Usage) string(r, noItemPrefixParts, middleItemPrefixParts, finalItemPrefixParts []string, rootSection, finalSubSection bool) []string {
	noItemPrefix := strings.Join(noItemPrefixParts, "")
	middleItemPrefix := strings.Join(middleItemPrefixParts, "")
	finalItemPrefix := strings.Join(finalItemPrefixParts, "")

	emptyPrefix := trimRightSpace(noItemPrefix)
	if !rootSection {
		r = append(r, emptyPrefix)
	}

	if u.Description != "" {
		r = append(r, noItemPrefix+u.Description)
	}

	usageString := strings.Join(append(u.Usage, u.Flags...), " ")
	if finalSubSection {
		r = append(r, finalItemPrefix+usageString)
	} else {
		r = append(r, middleItemPrefix+usageString)
	}

	if len(u.SubSections) > 0 {
		prefixStartParts := slices.Clone(noItemPrefixParts)
		if finalSubSection && len(prefixStartParts) > 0 && prefixStartParts[len(prefixStartParts)-1] != noDrawLinePrefix {
			prefixStartParts[len(prefixStartParts)-1] = "    "
		}

		prefix := strings.Join(prefixStartParts, "")
		index := strings.Index(usageString, "\u2533")
		if index == 0 {
			r = append(r, prefix+"\u2503")
		} else {
			r = append(r, prefix+"\u250f"+strings.Repeat("\u2501", index-1)+"\u251b")
		}

		for i, su := range u.SubSections {
			isFinal := i == (len(u.SubSections) - 1)

			drawLines := su.SubSectionLines
			r = su.string(r,
				append(slices.Clone(prefixStartParts), NO_ITEM_PREFIX[drawLines]),
				append(slices.Clone(prefixStartParts), MIDDLE_ITEM_PREFIX[drawLines]),
				append(slices.Clone(prefixStartParts), END_ITEM_PREFIX[drawLines]),
				i == 0, isFinal,
			)

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
