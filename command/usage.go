package command

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/leep-frog/command/internal/constants"
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

var (
	trailingNestedUsageRegex = regexp.MustCompile(fmt.Sprintf(`%s(\s*)$`, constants.UsageBoxUpDown))
	trailingWhitspaceRegex   = regexp.MustCompile(`\s*$`)
	branchStartRegex         = regexp.MustCompile(fmt.Sprintf(`[%s%s]`, constants.UsageBoxLeftRightDown, constants.UsageBoxLeftDown))
	MIDDLE_ITEM_PREFIX       = map[bool]string{
		true:  "┣━━ ",
		false: constants.NoDrawLinePrefix,
	}
	NO_ITEM_PREFIX = map[bool]string{
		true:  "┃   ",
		false: constants.NoDrawLinePrefix,
	}
	END_ITEM_PREFIX = map[bool]string{
		true:  "┗━━ ",
		false: constants.NoDrawLinePrefix,
	}
)

// TODO: Update Usage to be an interface (public methods instead of modifying fields)
// There are a lot of required configuration caveats that aren't actually enforced anywhere

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
		if finalSubSection && len(prefixStartParts) > 0 && prefixStartParts[len(prefixStartParts)-1] != constants.NoDrawLinePrefix {
			prefixStartParts[len(prefixStartParts)-1] = "    "
		}

		prefix := strings.Join(prefixStartParts, "")
		index := branchStartRegex.FindStringIndex(usageString)
		if index[0] == 0 {
			r = append(r, prefix+constants.UsageBoxUpDown)
		} else {
			r = append(r, prefix+constants.UsageBoxRightDown+strings.Repeat(constants.UsageBoxLeftRight, index[0]-1)+constants.UsageBoxLeftUp)
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

func trimRightSpace(s string) string {
	return trailingWhitspaceRegex.ReplaceAllString(s, "")
}
