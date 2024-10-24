package command

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/leep-frog/command/internal/constants"
)

type UsageSection string

const (
	// ArgSection is the title of the arguments usage section.
	ArgSection UsageSection = "Arguments"
	// FlagSection is the title of the flags usage section.
	FlagSection UsageSection = "Flags"
	// SymbolSection is the title of the symbols usage section.
	SymbolSection UsageSection = "Symbols"
)

var (
	trailingNestedUsageRegex = regexp.MustCompile(fmt.Sprintf(`%s(\s*)$`, constants.UsageBoxUpDown))
	trailingWhitspaceRegex   = regexp.MustCompile(`\s*$`)
	branchStartRegex         = regexp.MustCompile(fmt.Sprintf(`[%s%s]`, constants.UsageBoxLeftRightDown, constants.UsageBoxLeftDown))

	PRE_ITEM_PREFIX = "┃   "
	ITEM_PREFIX     = map[bool]string{
		// Middle item
		false: "┣━━ ",
		// Final item
		true: "┗━━ ",
	}
	POST_ITEM_PREFIX = map[bool]string{
		// Middle item
		false: "┃   ",
		// Final item
		true: "    ",
	}
)

type argumentUsage struct {
	usageStringPrefix *string
	usageString       *string
	section           UsageSection
	sectionKey        string
	description       string

	required int
	optional int
}

type BranchUsage struct {
	// Usage is the usage object for the branched node.
	Usage *Usage
}

type Usage struct {
	description *string
	args        []*argumentUsage
	flags       []*argumentUsage

	// This is the argument index where the branch/cache thing should be added
	branchArgIdx *int
	cacheArgIdx  int

	branches []*BranchUsage

	symbols map[string]string
}

func (u *Usage) SetDescription(desc string) {
	u.description = &desc
}

func (u *Usage) AddArg(name, description string, required, optional int) {
	u.args = append(u.args, &argumentUsage{
		usageString: &name,
		description: description,
		section:     ArgSection,
		sectionKey:  name,
		required:    required,
		optional:    optional,
	})
}

func (u *Usage) AddSymbol(symbol, description string) {
	if u.symbols == nil {
		u.symbols = map[string]string{}
	}
	u.symbols[symbol] = description

	u.AddArg(symbol, "", 1, 0)
}

func (u *Usage) AddFlag(fullFlag string, shortFlag rune, argName string, description string, required, optional int) {
	usageStringPrefix := fmt.Sprintf("--%s", fullFlag)
	sectionKey := fmt.Sprintf("    %s", fullFlag)

	if shortFlag != constants.FlagNoShortName {
		sectionKey = fmt.Sprintf("[%c] %s", shortFlag, fullFlag)
		usageStringPrefix = fmt.Sprintf("%s|-%c", usageStringPrefix, shortFlag)
	}
	u.flags = append(u.flags, &argumentUsage{
		usageStringPrefix: &usageStringPrefix,
		description:       description,
		section:           FlagSection,
		sectionKey:        sectionKey,

		// Flag arguments
		usageString: &argName,
		required:    required,
		optional:    optional,
	})
}

func (u *Usage) SetBranches(branches []*BranchUsage) {
	if u.branchArgIdx != nil {
		panic("Currently, only one branch point is supported per line")
	}
	if len(branches) == 0 {
		return
	}
	numArgs := len(u.args)
	u.branchArgIdx = &numArgs
	u.branches = branches
}

func (u *Usage) String() string {
	usageSection := &usageSectionMap{}
	r := u.string([]string{}, "", "", "", usageSection)

	var sections []UsageSection
	if len(*usageSection) > 0 {
		r = append(r, "")

		// Sort section titles
		for s := range *usageSection {
			sections = append(sections, s)
		}
		slices.Sort(sections)

		// Iterate over sections
		for _, sk := range sections {
			r = append(r, fmt.Sprintf("%s:", sk))
			kvs := (*usageSection)[sk]
			var keys []string
			for k := range kvs {
				keys = append(keys, k)
			}

			// Sort by flag name or by key name
			if sk == FlagSection {
				// We want to sort flags by full name, not short flags.
				// So, we trim "  [c] " from each flag description.
				sort.SliceStable(keys, func(i, j int) bool {
					return keys[i][4:] < keys[j][4:]
				})
			} else {
				sort.Strings(keys)
			}

			// Iterate over keys
			for _, k := range keys {
				r = append(r, fmt.Sprintf("  %s: %s", k, kvs[k]))
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

func (u *Usage) string(r []string, preItemPrefix, itemPrefix, postItemPrefix string, sections *usageSectionMap) []string {
	for sym, desc := range u.symbols {
		sections.add(SymbolSection, sym, desc)
	}
	rappend := func(prefix, s string) {
		r = append(r, prefix+s)
	}

	if u.description != nil {
		rappend(preItemPrefix, *u.description)
	}

	// Construct usage line
	var usageLine []string
	var branchStringIdx *int
	// Append nil to ensure branchStringIdx gets added if at end
	for idx, ui := range append(append(u.args, nil), u.flags...) {
		if u.branchArgIdx != nil && idx == *u.branchArgIdx {
			lenIdx := len(usageLine)
			branchStringIdx = &lenIdx
		}

		if ui == nil {
			continue
		}

		// Add usage strings
		if ui.usageStringPrefix != nil {
			usageLine = append(usageLine, *ui.usageStringPrefix)
		}
		if ui.usageString != nil {
			for i := 0; i < ui.required; i++ {
				usageLine = append(usageLine, *ui.usageString)
			}
			if ui.optional > 0 {
				usageLine = append(usageLine, "[")
				for i := 0; i < ui.optional; i++ {
					usageLine = append(usageLine, *ui.usageString)
				}
				usageLine = append(usageLine, "]")
			} else if ui.optional == UnboundedList {
				usageLine = append(usageLine, "[", *ui.usageString, "...", "]")
			}
		}
		if ui.description != "" {
			sections.add(ui.section, ui.sectionKey, ui.description)
		}
	}

	// Add branch character if relevant
	var hasPostBranchUsage bool
	if branchStringIdx != nil {
		pre := strings.Join(usageLine[:*branchStringIdx], " ")
		post := strings.Join(usageLine[*branchStringIdx:], " ")

		usageLine = []string{}
		if len(pre) > 0 {
			usageLine = append(usageLine, pre)
		}

		if len(post) > 0 {
			hasPostBranchUsage = true
			usageLine = append(usageLine, constants.UsageBoxLeftRightDown)
			usageLine = append(usageLine, post)
		} else {
			usageLine = append(usageLine, constants.UsageBoxLeftDown)
		}
		newIdx := len(pre)
		branchStringIdx = &newIdx
	}

	if branchStringIdx != nil && *branchStringIdx == 0 && strings.HasSuffix(itemPrefix, " ") {
		r = append(r, fmt.Sprintf("%s━%s", strings.TrimSuffix(itemPrefix, " "), strings.Join(usageLine, " ")))
	} else {
		rappend(itemPrefix, strings.Join(usageLine, " "))
	}

	if len(u.branches) == 0 {
		return r
	}

	// The previous check implies that branchStringIdx is not nil
	if *branchStringIdx != 0 {
		rappend(postItemPrefix, constants.UsageBoxRightDown+strings.Repeat(constants.UsageBoxLeftRight, *branchStringIdx)+constants.UsageBoxLeftUp)
		rappend(postItemPrefix, constants.UsageBoxUpDown)
	} else if hasPostBranchUsage {
		rappend(postItemPrefix, constants.UsageBoxUpDown)
	}

	for i, subUsage := range u.branches {
		isFinal := i == (len(u.branches) - 1)

		subPreItemPrefix := postItemPrefix + PRE_ITEM_PREFIX
		subItemPrefix := postItemPrefix + ITEM_PREFIX[isFinal]
		subPostItemPrefix := postItemPrefix + POST_ITEM_PREFIX[isFinal]

		r = subUsage.Usage.string(r, subPreItemPrefix, subItemPrefix, subPostItemPrefix, sections)

		if !isFinal {
			rappend(trimRightSpace(subPostItemPrefix), "")
		}
	}

	return r
}

// usageSectionMap is a map from section name to key phrase for that section to description for that key.
type usageSectionMap map[UsageSection]map[string]string

// Add adds a usage section.
func (us *usageSectionMap) add(section UsageSection, key string, value string) {
	if (*us)[section] == nil {
		(*us)[section] = map[string]string{}
	}
	(*us)[section][key] = value
}

func trimRightSpace(s string) string {
	return trailingWhitspaceRegex.ReplaceAllString(s, "")
}
