package command

import (
	"strings"
	"testing"

	"github.com/leep-frog/command/internal/constants"
	"github.com/leep-frog/command/internal/testutil"
)

func TestUsage(t *testing.T) {
	for _, test := range []struct {
		name string
		uf   func() *Usage
		yuf  func(*Usage)
		want []string
	}{
		{
			name: "empty usage",
			yuf:  func(y *Usage) {},
		},
		{
			name: "Usage with description",
			yuf: func(y *Usage) {
				y.SetDescription("Does stuff")
			},
			want: []string{
				"Does stuff",
			},
		},
		{
			name: "Usage with args",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 1, 0)
				y.AddArg("ARG_2", "arg 2", 1, 0)
			},
			want: []string{
				"ARG_1 ARG_2",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
				"  ARG_2: arg 2",
			},
		},
		{
			name: "Arg with no description",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "", 1, 0)
			},
			want: []string{
				"ARG_1",
			},
		},
		{
			name: "Flag with no description",
			yuf: func(y *Usage) {
				y.AddFlag("flag", 'f', "FFF", "", 1, 0)
			},
			want: []string{
				"--flag|-f",
			},
		},
		{
			name: "Required, optional = 0, Unbounded",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 0, UnboundedList)
			},
			want: []string{
				"[ ARG_1 ... ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 1, Unbounded",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 1, UnboundedList)
			},
			want: []string{
				"ARG_1 [ ... ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 2, Unbounded",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 2, UnboundedList)
			},
			want: []string{
				"ARG_1 ARG_1 [ ... ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 0, 1",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 0, 1)
			},
			want: []string{
				"[ ARG_1 ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 1, 1",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 1, 1)
			},
			want: []string{
				"ARG_1 [ ARG_1 ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 2, 1",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 2, 1)
			},
			want: []string{
				"ARG_1 ARG_1 [ ARG_1 ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 0, 2",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 0, 2)
			},
			want: []string{
				"[ ARG_1 ARG_1 ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 1, 2",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 1, 2)
			},
			want: []string{
				"ARG_1 [ ARG_1 ARG_1 ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Required, optional = 2, 2",
			yuf: func(y *Usage) {
				y.AddArg("ARG_1", "arg 1", 2, 2)
			},
			want: []string{
				"ARG_1 ARG_1 [ ARG_1 ARG_1 ]",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
			},
		},
		{
			name: "Ignores nil branches",
			yuf: func(y *Usage) {
				y.SetDescription("Does stuff")
				y.InsertBranches(nil)
			},
			want: []string{
				"Does stuff",
			},
		},
		{
			name: "Ignores empty branches",
			yuf: func(y *Usage) {
				y.SetDescription("Does stuff")
				y.InsertBranches([]*BranchUsage{})
			},
			want: []string{
				"Does stuff",
			},
		},
		{
			name: "Usage with flags",
			yuf: func(y *Usage) {
				y.AddFlag("first-flag", 'f', "FFF", "1st", 1, 0)
				y.AddFlag("second-flag", constants.FlagNoShortName, "SS", "2nd", 1, 2)
			},
			want: []string{
				"--first-flag|-f --second-flag",
				"",
				"Flags:",
				"  [f] --first-flag: 1st",
				"      --second-flag: 2nd",
			},
		},
		{
			name: "Usage with desc, args, and flags",
			yuf: func(y *Usage) {
				y.SetDescription("Does stuff")
				// Intermix flag and args to verify flags go at end
				y.AddArg("ARG_1", "arg 1", 1, 0)
				y.AddFlag("first-flag", 'f', "FFF", "1st", 1, 0)
				y.AddArg("ARG_2", "arg 2", 1, 0)
				y.AddFlag("second-flag", constants.FlagNoShortName, "SS", "2nd", 1, 2)
			},
			want: []string{
				"Does stuff",
				"ARG_1 ARG_2 --first-flag|-f --second-flag",
				"",
				"Arguments:",
				"  ARG_1: arg 1",
				"  ARG_2: arg 2",
				"",
				"Flags:",
				"  [f] --first-flag: 1st",
				"      --second-flag: 2nd",
			},
		},
		{
			name: "SubSections",
			yuf: func(y *Usage) {
				b1 := &Usage{}
				b1.SetDescription("Branch 1")
				b1.AddFlag("branch-1", 'b', "B", "First branch", 2, 0)

				b2 := &Usage{}
				b2.SetDescription("Branch 2")
				b2.AddArg("ARG_B_2", "Second branch arg", 2, 1)

				y.SetDescription("Does stuff")
				// Intermix flag and args to verify flags go at end
				y.AddArg("ARG_1", "arg 1", 1, 0)

				y.InsertBranches([]*BranchUsage{}) // Confirm ignore `InsertBranches with empty is ignored
				y.InsertBranches([]*BranchUsage{
					{Usage: b1},
					{Usage: b2},
				})
				y.AddFlag("first-flag", constants.FlagNoShortName, "FFF", "1st", 1, 0)
				y.AddArg("ARG_2", "arg 2", 1, 0)
				y.AddFlag("second-flag", constants.FlagNoShortName, "SS", "2nd", 1, 2)
			},
			want: []string{
				`Does stuff`,
				`ARG_1 ┳ ARG_2 --first-flag --second-flag`,
				`┏━━━━━┛`,
				`┃   Branch 1`,
				`┣━━ --branch-1|-b`,
				`┃`,
				`┃   Branch 2`,
				`┗━━ ARG_B_2 ARG_B_2 [ ARG_B_2 ]`,
				``,
				`Arguments:`,
				`  ARG_1: arg 1`,
				`  ARG_2: arg 2`,
				`  ARG_B_2: Second branch arg`,
				``,
				`Flags:`,
				`  [b] --branch-1: First branch`,
				`      --first-flag: 1st`,
				`      --second-flag: 2nd`,
			},
		},
		{
			name: "Nested SubSections",
			yuf: func(y *Usage) {
				b1_1 := &Usage{}
				b1_1.SetDescription("Branch 1.1")
				b1_1.AddFlag("branch-1.1", constants.FlagNoShortName, "1_1", "one point one", 1, 1)

				b1_2 := &Usage{}
				b1_2.AddFlag("branch-1.2", constants.FlagNoShortName, "1_2", "one point two", 1, 2)

				b1_3 := &Usage{}
				b1_3.SetDescription("Branch 1.3")

				b1 := &Usage{}
				b1.SetDescription("Branch 1")
				b1.AddFlag("branch-1", 'b', "B", "First branch", 2, 0)

				b1.InsertBranches([]*BranchUsage{
					{Usage: b1_1},
					{Usage: b1_2},
					{Usage: b1_3},
				})

				b2_1 := &Usage{}
				b2_1.SetDescription("Branch 2.1")
				b2_1.AddFlag("ARG_B_2_1", constants.FlagNoShortName, "21", "two point one", 0, 2)

				b2 := &Usage{}
				b2.SetDescription("Branch 2")
				b2.AddArg("ARG_B_2", "Second branch arg", 2, 1)

				b2.InsertBranches([]*BranchUsage{
					{Usage: b2_1},
				})

				y.SetDescription("Does stuff")
				// Intermix flag and args to verify flags go at end
				y.AddArg("ARG_1", "arg 1", 1, 0)
				y.InsertBranches([]*BranchUsage{
					{Usage: b1},
					{Usage: b2},
				})
				y.AddFlag("first-flag", constants.FlagNoShortName, "FFF", "1st", 1, 0)
				y.AddArg("ARG_2", "arg 2", 1, 0)
				y.AddFlag("second-flag", constants.FlagNoShortName, "SS", "2nd", 1, 2)
			},
			want: []string{
				`Does stuff`,
				`ARG_1 ┳ ARG_2 --first-flag --second-flag`,
				`┏━━━━━┛`,
				`┃   Branch 1`,
				// TODO: fill in this space with a line
				`┣━━ ┳ --branch-1|-b`,
				`┃   ┃`,
				`┃   ┃   Branch 1.1`,
				`┃   ┣━━ --branch-1.1`,
				`┃   ┃`,
				`┃   ┣━━ --branch-1.2`,
				`┃   ┃`,
				`┃   ┃   Branch 1.3`,
				`┃   ┗━━ `,
				`┃`,
				`┃   Branch 2`,
				`┗━━ ARG_B_2 ARG_B_2 [ ARG_B_2 ] ┓`,
				`    ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━┛`,
				`    ┃   Branch 2.1`,
				`    ┗━━ --ARG_B_2_1`,
				``,
				`Arguments:`,
				`  ARG_1: arg 1`,
				`  ARG_2: arg 2`,
				`  ARG_B_2: Second branch arg`,
				``,
				`Flags:`,
				`      --ARG_B_2_1: two point one`,
				`  [b] --branch-1: First branch`,
				`      --branch-1.1: one point one`,
				`      --branch-1.2: one point two`,
				`      --first-flag: 1st`,
				`      --second-flag: 2nd`,
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			y := &Usage{}
			test.yuf(y)
			testutil.Cmp(t, "Usage.String() returned incorrect value", strings.Join(test.want, "\n"), y.String())
		})
	}
}

func TestInsertBranches(t *testing.T) {
	testutil.CmpPanic(t, "[InsertBranches() x 2]", func() bool {
		u := &Usage{}
		u.InsertBranches([]*BranchUsage{nil})
		u.InsertBranches([]*BranchUsage{nil})
		return false
	}, "Currently, only one branch point is supported per line")
}
