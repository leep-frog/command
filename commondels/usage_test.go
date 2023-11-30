package commondels

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
		want []string
	}{
		{
			name: "empty usage",
			uf: func() *Usage {
				return &Usage{
					UsageSection: &UsageSection{},
				}
			},
		},
		{
			name: "Usage with description",
			uf: func() *Usage {
				return &Usage{
					UsageSection: &UsageSection{},
					Description:  "Does stuff",
				}
			},
			want: []string{
				"Does stuff",
			},
		},
		{
			name: "Usage with args",
			uf: func() *Usage {
				return &Usage{
					UsageSection: &UsageSection{},
					Usage:        []string{"ARG_1", "ARG_2"},
				}
			},
			want: []string{
				"ARG_1 ARG_2",
			},
		},
		{
			name: "Usage with flags",
			uf: func() *Usage {
				return &Usage{
					UsageSection: &UsageSection{},
					Flags:        []string{"--first-flag", "--second-flag"},
				}
			},
			want: []string{
				"--first-flag --second-flag",
			},
		},
		{
			name: "Usage with desc, args, and flags",
			uf: func() *Usage {
				return &Usage{
					UsageSection: &UsageSection{},
					Description:  "Does stuff",
					Usage:        []string{"ARG_1", "ARG_2"},
					Flags:        []string{"--first-flag", "--second-flag"},
				}
			},
			want: []string{
				"Does stuff",
				"ARG_1 ARG_2 --first-flag --second-flag",
			},
		},
		{
			name: "Usage with non-empty UsageSection",
			uf: func() *Usage {
				u := &Usage{
					UsageSection: &UsageSection{},
					Description:  "Does stuff",
					Usage:        []string{"ARG_1", "ARG_2"},
					Flags:        []string{"--first-flag", "--second-flag"},
				}

				u.UsageSection.Add(ArgSection, "ARG_1", "required")
				u.UsageSection.Add(FlagSection, "[f] --first-flag", "1st")
				u.UsageSection.Add(FlagSection, "[s] --second-flag", "2nd")

				return u
			},
			want: []string{
				"Does stuff",
				"ARG_1 ARG_2 --first-flag --second-flag",
				"",
				`Arguments:`,
				`  ARG_1: required`,
				``,
				`Flags:`,
				`  [f] --first-flag: 1st`,
				`  [s] --second-flag: 2nd`,
			},
		},
		{
			name: "UsageSection.Set",
			uf: func() *Usage {
				u := &Usage{
					UsageSection: &UsageSection{},
					Description:  "Does stuff",
					Usage:        []string{"ARG_1", "ARG_2"},
					Flags:        []string{"--first-flag", "--second-flag"},
				}

				u.UsageSection.Add(ArgSection, "ARG_1", "required")
				u.UsageSection.Set(FlagSection, "[f] --first-flag", "1st")
				u.UsageSection.Set(FlagSection, "[s] --second-flag", "2nd")

				u.UsageSection.Set(ArgSection, "ARG_1", "required (updated)")
				u.UsageSection.Set(FlagSection, "[f] --first-flag", "1st (updated)")

				return u
			},
			want: []string{
				"Does stuff",
				"ARG_1 ARG_2 --first-flag --second-flag",
				"",
				`Arguments:`,
				`  ARG_1: required (updated)`,
				``,
				`Flags:`,
				`  [f] --first-flag: 1st (updated)`,
				`  [s] --second-flag: 2nd`,
			},
		},
		{
			name: "Multi-line keys",
			uf: func() *Usage {
				u := &Usage{
					UsageSection: &UsageSection{},
					Description:  "Does stuff",
					Usage:        []string{"ARG_1", "ARG_2"},
					Flags:        []string{"--first-flag", "--second-flag"},
				}

				u.UsageSection.Add(ArgSection, "ARG_1", "req", "uired")
				u.UsageSection.Add(FlagSection, "[f] --first-flag", "1st", "flag")

				return u
			},
			want: []string{
				"Does stuff",
				"ARG_1 ARG_2 --first-flag --second-flag",
				"",
				`Arguments:`,
				`  ARG_1: req`,
				`    uired`,
				``,
				`Flags:`,
				`  [f] --first-flag: 1st`,
				`    flag`,
			},
		},
		{
			name: "SubSections",
			uf: func() *Usage {
				return &Usage{
					UsageSection:    &UsageSection{},
					Description:     "Does stuff",
					Usage:           []string{"ARG_1", constants.UsageBoxLeftRightDown, "ARG_2"},
					Flags:           []string{"--first-flag", "--second-flag"},
					SubSectionLines: true,
					SubSections: []*Usage{
						{
							UsageSection: &UsageSection{
								"Branch": {
									FlagSection: {
										"branch-1",
									},
								},
							},
							Description:     "Branch 1",
							Flags:           []string{"--branch-1"},
							SubSectionLines: true,
						},
						{
							UsageSection: &UsageSection{
								"Branch": {
									ArgSection: {
										"branch-2",
									},
								},
							},
							Description:     "Branch 2",
							Usage:           []string{"ARG_B_2"},
							SubSectionLines: true,
						},
					},
				}
			},
			want: []string{
				`Does stuff`,
				`ARG_1 ┳ ARG_2 --first-flag --second-flag`,
				`┏━━━━━┛`,
				`┃   Branch 1`,
				`┣━━ --branch-1`,
				`┃`,
				`┃   Branch 2`,
				`┗━━ ARG_B_2`,
				``,
				`Branch:`,
				`  Arguments: branch-2`,
				`  Flags: branch-1`,
			},
		},
		{
			name: "Nested SubSections",
			uf: func() *Usage {
				return &Usage{
					UsageSection:    &UsageSection{},
					Description:     "Does stuff",
					Usage:           []string{"ARG_1", constants.UsageBoxLeftRightDown, "ARG_2"},
					Flags:           []string{"--first-flag", "--second-flag"},
					SubSectionLines: true,
					SubSections: []*Usage{
						{
							UsageSection: &UsageSection{
								"Branch": {
									FlagSection: {
										"branch-1",
									},
								},
							},
							Description:     "Branch 1",
							Usage:           []string{constants.UsageBoxLeftDown},
							Flags:           []string{"--branch-1"},
							SubSectionLines: true,
							SubSections: []*Usage{
								{
									UsageSection: &UsageSection{
										"Branch": {
											FlagSection: {
												"branch-1.1",
											},
										},
									},
									Description:     "Branch 1.1",
									Flags:           []string{"--branch-1.1"},
									SubSectionLines: true,
								},
								{
									UsageSection: &UsageSection{
										"Branch": {
											FlagSection: {
												"branch-1.2",
											},
										},
									},
									Description:     "Branch 1.2",
									Flags:           []string{"--branch-1.2"},
									SubSectionLines: true,
								},
							},
						},
						{
							UsageSection: &UsageSection{
								"Branch": {
									ArgSection: {
										"branch-2",
									},
								},
							},
							Description:     "Branch 2",
							Usage:           []string{constants.UsageBoxLeftRightDown, "ARG_B_2"},
							SubSectionLines: true,
							SubSections: []*Usage{
								{
									UsageSection: &UsageSection{
										"Branch": {
											FlagSection: {
												"branch-2.1",
											},
										},
									},
									Description:     "Branch 2.1",
									Flags:           []string{"--branch-2.1"},
									SubSectionLines: true,
								},
							},
						},
					},
				}
			},
			want: []string{
				`Does stuff`,
				`ARG_1 ┳ ARG_2 --first-flag --second-flag`,
				`┏━━━━━┛`,
				`┃   Branch 1`,
				`┣━━ ┓ --branch-1`,
				`┃   ┃`,
				`┃   ┃   Branch 1.1`,
				`┃   ┣━━ --branch-1.1`,
				`┃   ┃`,
				`┃   ┃   Branch 1.2`,
				`┃   ┗━━ --branch-1.2`,
				`┃`,
				`┃   Branch 2`,
				`┗━━ ┳ ARG_B_2`,
				`    ┃`,
				`    ┃   Branch 2.1`,
				`    ┗━━ --branch-2.1`,
				``,
				`Branch:`,
				`  Arguments: branch-2`,
				`  Flags: branch-2.1`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testutil.Cmp(t, "Usage.String() returned incorrect value", strings.Join(test.want, "\n"), test.uf().String())
		})
	}
}
