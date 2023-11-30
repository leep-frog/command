package commander

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/commondels"
)

const (
	ShortcutDesc             = "  *: Start of new shortcut-able section"
	CacheDesc                = "  ^: Start of new cachable section"
	BranchDescWithDefault    = "  ┳: Start of subcommand branches (with default node)"
	BranchDescWithoutDefault = "  ┓: Start of subcommand branches (without default node)"
)

type usageNode struct {
	usageErr     error
	usageNextErr error
}

func (un *usageNode) Usage(*commondels.Input, *commondels.Data, *commondels.Usage) error {
	return un.usageErr
}

func (un *usageNode) UsageNext(*commondels.Input, *commondels.Data) (commondels.Node, error) {
	return nil, un.usageNextErr
}

func (un *usageNode) Execute(*commondels.Input, commondels.Output, *commondels.Data, *commondels.ExecuteData) error {
	return nil
}
func (un *usageNode) Complete(*commondels.Input, *commondels.Data) (*commondels.Completion, error) {
	return nil, nil
}
func (un *usageNode) Next(*commondels.Input, *commondels.Data) (commondels.Node, error) {
	return nil, nil
}

func TestUsage(t *testing.T) {

	branchesForSorting := map[string]commondels.Node{
		"alpha": nil,
		"beta":  SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
		"charlie": &BranchNode{
			Branches: map[string]commondels.Node{
				"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
				"yellow": SerialNodes(&ExecutorProcessor{}),
			},
		},
		"delta": nil,
		"echo":  nil,
	}

	for _, test := range []struct {
		name string
		etc  *commandtest.ExecuteTestCase
	}{
		{
			name: "works with empty node",
		},
		{
			name: "fails if node.Usage() returns error",
			etc: &commandtest.ExecuteTestCase{
				Node:    &usageNode{fmt.Errorf("oops"), nil},
				WantErr: fmt.Errorf("oops"),
			},
		},
		{
			name: "fails if node.UsageNext() returns error",
			etc: &commandtest.ExecuteTestCase{
				Node:    &usageNode{nil, fmt.Errorf("whoops")},
				WantErr: fmt.Errorf("whoops"),
			},
		},
		{
			name: "works with basic Description node",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Description("hello %s")),
				WantStdout: strings.Join([]string{
					"hello %s",
				}, "\n"),
			},
		},
		{
			name: "works with basic Descriptionf node",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Descriptionf("hello %s", "there")),
				WantStdout: strings.Join([]string{
					"hello there",
				}, "\n"),
			},
		},
		{
			name: "works with single arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("SARG", "desc")),
				WantStdout: strings.Join([]string{
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
				}, "\n"),
			},
		},
		{
			name: "works with optional arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(OptionalArg[string]("SARG", "desc")),
				WantStdout: strings.Join([]string{
					"[ SARG ]",
					"",
					"Arguments:",
					"  SARG: desc",
				}, "\n"),
			},
		},
		{
			name: "works with hidden arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("SARG1", "desc"),
					Arg("SARG2", "desc", HiddenArg[string]()),
					Arg[string]("SARG3", "desc"),
				),
				WantStdout: strings.Join([]string{
					"SARG1 SARG3",
					"",
					"Arguments:",
					"  SARG1: desc",
					"  SARG3: desc",
				}, "\n"),
			},
		},
		{
			name: "setup arg is hidden",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("SARG", "desc"),
					SetupArg,
					Arg[int]("IARG", "idesc"),
				),
				WantStdout: strings.Join([]string{
					"SARG IARG",
					"",
					"Arguments:",
					"  IARG: idesc",
					"  SARG: desc",
				}, "\n"),
			},
		},
		{
			name: "works with single arg and description node",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("SARG", "desc"), Description("Does absolutely nothing")),
				WantStdout: strings.Join([]string{
					"Does absolutely nothing",
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
				}, "\n"),
			},
		},
		{
			name: "works with validators and description node",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("SARG", "desc",
						MinLength[string, string](3),
						Contains("X"),
						FileExists(),
					),
					Description("Does absolutely nothing"),
				),
				WantStdout: strings.Join([]string{
					"Does absolutely nothing",
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
					"    MinLength(3)",
					`    Contains("X")`,
					`    FileExists()`,
				}, "\n"),
			},
		},
		{
			name: "works with multiple args with validators",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("SARG", "desc",
						MinLength[string, string](3),
						Contains("X"),
						FileExists(),
					),
					Arg[int]("IARG", "iDesc",
						NonNegative[int](),
						GTE(-2),
						LTE(39),
					),
					Description("Does absolutely nothing"),
				),
				WantStdout: strings.Join([]string{
					"Does absolutely nothing",
					"SARG IARG",
					"",
					"Arguments:",
					"  IARG: iDesc",
					"    NonNegative()",
					"    GTE(-2)",
					"    LTE(39)",
					"  SARG: desc",
					"    MinLength(3)",
					`    Contains("X")`,
					`    FileExists()`,
				}, "\n"),
			},
		},
		{
			name: "fails if validator returns error with some args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"12"},
				Node: SerialNodes(
					Arg[string]("SARG", "desc",
						MinLength[string, string](3),
						Contains("X"),
						FileExists(),
					),
					Description("Does absolutely nothing"),
				),
				WantErr: fmt.Errorf(`validation for "SARG" failed: [MinLength] length must be at least 3`),
			},
		},
		{
			name: "works with list arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("SARG", testDesc, 2, 3)),
				WantStdout: strings.Join([]string{
					"SARG SARG [ SARG SARG SARG ]",
					"",
					"Arguments:",
					"  SARG: test desc",
				}, "\n"),
			},
		},
		{
			name: "works with unbounded list arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("SARG", testDesc, 0, commondels.UnboundedList)),
				WantStdout: strings.Join([]string{
					"[ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
				}, "\n"),
			},
		},
		{
			name: "works with shortcut",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("shortcutName", nil, SerialNodes(
					Description("command desc"),
					ListArg[string]("SARG", testDesc, 0, commondels.UnboundedList),
					SimpleProcessor(nil, nil),
				)),
				WantStdout: strings.Join([]string{
					"command desc",
					"* [ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
					"",
					"Symbols:",
					ShortcutDesc,
				}, "\n"),
			},
		},
		{
			name: "works with cache",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("cacheName", nil, SerialNodes(
					Description("cmd desc"),
					ListArg[string]("SARG", testDesc, 0, commondels.UnboundedList),
					SimpleProcessor(nil, nil),
				)),
				WantStdout: strings.Join([]string{
					"cmd desc",
					"^ [ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
					"",
					"Symbols:",
					CacheDesc,
				}, "\n"),
			},
		},
		{
			name: "works with simple branch node",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha": nil,
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┗━━ alpha",
					"",
					"Arguments:",
					"  INT_ARG: an integer",
					"  STRINGS: unltd strings",
					"",
					"Symbols:",
					BranchDescWithDefault,
				}, "\n"),
			},
		},
		{
			name: "works with simple branch node with no default",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha": nil,
					},
				},
				WantStdout: strings.Join([]string{
					"┓",
					"┃",
					"┗━━ alpha",
					"",
					"Symbols:",
					BranchDescWithoutDefault,
				}, "\n"),
			},
		},
		{
			name: "BranchNode usage doesn't display if default node traversed",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"123"},
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha": nil,
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"STRINGS [ STRINGS ... ]",
					"",
					"Arguments:",
					"  STRINGS: unltd strings",
				}, "\n"),
			},
		},
		{
			name: "BranchNode usage doesn't display if branch traversed",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"beta"},
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha": nil,
						"beta":  SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"ROPES ROPES [ ROPES ROPES ROPES ]",
					"",
					"Arguments:",
					"  ROPES: lots of strings",
				}, "\n"),
			},
		},
		{
			name: "branch node with HideUsage and default",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Description("command start"),
					Arg[string]("STRING", "A string"),
					&BranchNode{
						Branches: map[string]commondels.Node{
							"alpha": nil,
						},
						HideUsage: true,
						Default:   SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
					},
				),
				WantStdout: strings.Join([]string{
					"the default command",
					"STRING INT_ARG STRINGS [ STRINGS ... ]",
					"",
					"Arguments:",
					"  INT_ARG: an integer",
					"  STRING: A string",
					"  STRINGS: unltd strings",
				}, "\n"),
			},
		},
		{
			name: "branch node with HideUsage and no default",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Description("command start"),
					Arg[string]("STRING", "A string"),
					&BranchNode{
						Branches: map[string]commondels.Node{
							"alpha": nil,
						},
						HideUsage: true,
					},
				),
				WantStdout: strings.Join([]string{
					"command start",
					"STRING",
					"",
					"Arguments:",
					"  STRING: A string",
				}, "\n"),
			},
		},
		{
			name: "works with branch node",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha": nil,
						"beta":  SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]commondels.Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┣━━ alpha",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					"┗━━ charlie ┓",
					"    ┏━━━━━━━┛",
					"    ┃   learn about cartoons",
					"    ┣━━ brown FLOATER",
					"    ┃",
					"    ┗━━ yellow",
					"",
					"Arguments:",
					"  FLOATER: something bouyant",
					"  INT_ARG: an integer",
					"  ROPES: lots of strings",
					"  STRINGS: unltd strings",
					"",
					"Symbols:",
					BranchDescWithoutDefault,
					BranchDescWithDefault,
				}, "\n"),
			},
		},
		{
			name: "works with branch node shortcut option",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha": SerialNodes(
							Description("The first"),
						),
						"beta": SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]commondels.Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Synonyms: BranchSynonyms(map[string][]string{
						"charlie": {"charles", "chuck"},
						"alpha":   {"omega"},
					}),
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┃   The first",
					"┣━━ [alpha|omega]",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					"┗━━ [charlie|charles|chuck] ┓",
					"    ┏━━━━━━━━━━━━━━━━━━━━━━━┛",
					"    ┃   learn about cartoons",
					"    ┣━━ brown FLOATER",
					"    ┃",
					"    ┗━━ yellow",
					"",
					"Arguments:",
					"  FLOATER: something bouyant",
					"  INT_ARG: an integer",
					"  ROPES: lots of strings",
					"  STRINGS: unltd strings",
					"",
					"Symbols:",
					BranchDescWithoutDefault,
					BranchDescWithDefault,
				}, "\n"),
			},
		},
		{
			name: "works with branch node shortcut option via spaces",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha omega1": SerialNodes(
							Description("The first"),
						),
						"beta": SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]commondels.Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Synonyms: BranchSynonyms(map[string][]string{
						"charlie": {"charles", "chuck"},
						"alpha":   {"omega2"},
					}),
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┃   The first",
					"┣━━ [alpha|omega1|omega2]",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					"┗━━ [charlie|charles|chuck] ┓",
					"    ┏━━━━━━━━━━━━━━━━━━━━━━━┛",
					"    ┃   learn about cartoons",
					"    ┣━━ brown FLOATER",
					"    ┃",
					"    ┗━━ yellow",
					"",
					"Arguments:",
					"  FLOATER: something bouyant",
					"  INT_ARG: an integer",
					"  ROPES: lots of strings",
					"  STRINGS: unltd strings",
					"",
					"Symbols:",
					BranchDescWithoutDefault,
					BranchDescWithDefault,
				}, "\n"),
			},
		},
		{
			name: "works with multiple node",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]commondels.Node{
						"alpha": nil,
						"beta": &BranchNode{
							Branches: map[string]commondels.Node{
								"one":   SerialNodes(Description("First"), Arg[int]("ONE", "A number")),
								"two":   SerialNodes(Arg[int]("TWO", "Another number")),
								"three": SerialNodes(&ExecutorProcessor{}),
							},
						},
						"charlie": &BranchNode{
							Branches: map[string]commondels.Node{
								"delta": SerialNodes(SerialNodes(Description("Something else"), Arg[string]("DELTA", "delta description"))),
								"brown": &BranchNode{
									Branches: map[string]commondels.Node{
										"movie":      SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
										"comic":      SerialNodes(Description("Comic strip")),
										"characters": SerialNodes(ListArg[string]("CHARACTERS", "Character names", 2, 1)),
									},
								},
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┣━━ alpha",
					"┃",
					"┣━━ beta ┓",
					"┃   ┏━━━━┛",
					"┃   ┃   First",
					"┃   ┣━━ one ONE",
					"┃   ┃",
					"┃   ┣━━ three",
					"┃   ┃",
					"┃   ┗━━ two TWO",
					"┃",
					"┗━━ charlie ┓",
					"    ┏━━━━━━━┛",
					"    ┣━━ brown ┓",
					"    ┃   ┏━━━━━┛",
					"    ┃   ┣━━ characters CHARACTERS CHARACTERS [ CHARACTERS ]",
					"    ┃   ┃",
					"    ┃   ┃   Comic strip",
					"    ┃   ┣━━ comic",
					"    ┃   ┃",
					"    ┃   ┃   learn about cartoons",
					"    ┃   ┗━━ movie FLOATER",
					"    ┃",
					"    ┃   Something else",
					"    ┣━━ delta DELTA",
					"    ┃",
					"    ┗━━ yellow",
					"",
					"Arguments:",
					"  CHARACTERS: Character names",
					"  DELTA: delta description",
					"  FLOATER: something bouyant",
					"  INT_ARG: an integer",
					"  ONE: A number",
					"  STRINGS: unltd strings",
					"  TWO: Another number",
					"",
					"Symbols:",
					BranchDescWithoutDefault,
					BranchDescWithDefault,
				}, "\n"),
			},
		},
		// BranchUsageOrderFunc tests
		{
			name: "BranchUsageOrderFunc fails if extra strings",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "charlie", "delta", "echo", "foxtrot"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantErr: fmt.Errorf("BranchUsageOrder includes an incorrect set of branches: expected [alpha beta charlie delta echo]; got [alpha beta charlie delta echo foxtrot]"),
			},
		},
		{
			name: "BranchUsageOrderFunc fails if fewer strings",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantErr: fmt.Errorf("BranchUsageOrder includes an incorrect set of branches: expected [alpha beta charlie delta echo]; got [alpha beta delta echo]"),
			},
		},
		{
			name: "BranchUsageOrderFunc fails if duplicate strings",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "beta", "charlie", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantErr: fmt.Errorf("BranchUsageOrder includes an incorrect set of branches: expected [alpha beta charlie delta echo]; got [alpha beta beta charlie delta echo]"),
			},
		},
		{
			name: "BranchUsageOrderFunc fails if duplicate strings but right number",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "beta", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantErr: fmt.Errorf("BranchUsageOrder includes an incorrect set of branches: expected [alpha beta charlie delta echo]; got [alpha beta beta delta echo]"),
			},
		},
		{
			name: "BranchUsageOrderFunc works",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "charlie", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, commondels.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┣━━ alpha",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					"┣━━ charlie ┓",
					"┃   ┏━━━━━━━┛",
					"┃   ┃   learn about cartoons",
					"┃   ┣━━ brown FLOATER",
					"┃   ┃",
					"┃   ┗━━ yellow",
					"┃",
					"┣━━ delta",
					"┃",
					"┗━━ echo",
					"",
					"Arguments:",
					"  FLOATER: something bouyant",
					"  INT_ARG: an integer",
					"  ROPES: lots of strings",
					"  STRINGS: unltd strings",
					"",
					"Symbols:",
					BranchDescWithoutDefault,
					BranchDescWithDefault,
				}, "\n"),
			},
		},
		// Flag tests
		{
			name: "works with flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FlagProcessor(
					BoolFlag("new", 'n', "new files"),
					BoolFlag("debug", 'd', "debug stuff"),
				)),
				WantStdout: strings.Join([]string{
					"--debug|-d --new|-n",
					"",
					"Flags:",
					"  [d] debug: debug stuff",
					"  [n] new: new files",
				}, "\n"),
			},
		},
		{
			name: "Works with input flag and no args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantStdout: strings.Join([]string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				}, "\n"),
			},
		},
		{
			name: "Works with input flag and some args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "un"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantStdout: strings.Join([]string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				}, "\n"),
			},
		},
		{
			name: "Fails if validation error",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "nope"},
				Node: SerialNodes(FlagProcessor(
					Flag[string]("str", 's', "desc", Contains("t")),
				)),
				WantErr: fmt.Errorf(`validation for "str" failed: [Contains] value doesn't contain substring "t"`),
			},
		},
		{
			name: "Works with input flag and required args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "un", "deux"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantStdout: strings.Join([]string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				}, "\n"),
			},
		},
		{
			name: "Works with input flag and all args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "un", "deux", "trois"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantStdout: strings.Join([]string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				}, "\n"),
			},
		},
		{
			name: "Works with input flag and extra args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "un", "deux", "trois", "quatre"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantStdout: strings.Join([]string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				}, "\n"),
			},
		},
		{
			name: "flags go at the end",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("new", 'n', "new files"),
						BoolFlag("debug", 'd', "debug stuff"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantStdout: strings.Join([]string{
					"SN --debug|-d --new|-n",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"  [d] debug: debug stuff",
					"  [n] new: new files",
				}, "\n"),
			},
		},
		{
			name: "flags are sorted by full name, not short flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", 'b', "un"),
						BoolFlag("second", 'a', "deux"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantStdout: strings.Join([]string{
					"SN --first|-b --second|-a",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"  [b] first: un",
					"  [a] second: deux",
				}, "\n"),
			},
		},
		// TODO:
		// {
		// 	name: "flags with values",
		// 	etc: &commandtest.ExecuteTestCase{
		// 		Args: []string{"--second", "2nd"},
		// 		Node: SerialNodes(
		// 			FlagProcessor(
		// 				BoolFlag("first", 'b', "un"),
		// 				BoolFlag("second", 'a', "deux"),
		// 			),
		// 			Arg[string]("SN", "node for a string"),
		// 		),
		// 		WantStdout: []string{
		// 			"SN --first|-b --second|-a",
		// 			"",
		// 			"Arguments:",
		// 			"  SN: node for a string",
		// 			"",
		// 			"Flags:",
		// 			"  [b] first: un",
		// 			"  [a] second: deux",
		// 		},
		// 	},
		// },
		{
			name: "flags without short names work",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", FlagNoShortName, "un"),
						BoolFlag("second", '2', "deux"),
						Flag[string]("third", '3', "trois"),
						Flag[string]("fourth", FlagNoShortName, "quatre"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantStdout: strings.Join([]string{
					"SN --first --fourth --second|-2 --third|-3",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"      first: un",
					"      fourth: quatre",
					"  [2] second: deux",
					"  [3] third: trois",
				}, "\n"),
			},
		},
		{
			name: "NodeRepeater usage works for finite optional",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 1)),
				WantStdout: strings.Join([]string{
					"KEY VALUE KEY VALUE { KEY VALUE }",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				}, "\n"),
			},
		},
		{
			name: "NodeRepeater usage works for no optional",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 0)),
				WantStdout: strings.Join([]string{
					"KEY VALUE KEY VALUE",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				}, "\n"),
			},
		},
		{
			name: "NodeRepeater usage works for no required",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(0, 1)),
				WantStdout: strings.Join([]string{
					"{ KEY VALUE }",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				}, "\n"),
			},
		},
		{
			name: "NodeRepeater usage works for unbounded",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(3, commondels.UnboundedList)),
				WantStdout: strings.Join([]string{
					"KEY VALUE KEY VALUE KEY VALUE { KEY VALUE } ...",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				}, "\n"),
			},
		},
		// ListBreaker tests
		{
			name: "NodeRepeater usage works for unbounded",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, commondels.UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 0, commondels.UnboundedList),
				),
				WantStdout: strings.Join([]string{
					"SL [ SL ... ] ghi [ SL2 ... ]",
					"",
					"Arguments:",
					"  SL: test desc",
					"  SL2: test desc",
					"",
					"Symbols:",
					"  ghi: List breaker",
				}, "\n"),
			},
		},
		// StringListListProcessor
		{
			name: "StringListListProcessor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(StringListListProcessor("SLL", "sl desc", ";", 1, 2)),
				WantStdout: strings.Join([]string{
					"[ SLL ... ] ; { [ SLL ... ] ; [ SLL ... ] ; }",
					"",
					"Arguments:",
					"  SLL: sl desc",
					"",
					"Symbols:",
					"  ;: List breaker",
				}, "\n"),
			},
		},
		{
			name: "unbounded StringListListProcessor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(StringListListProcessor("SLL", "sl desc", ";", 1, commondels.UnboundedList)),
				WantStdout: strings.Join([]string{
					"[ SLL ... ] ; { [ SLL ... ] ; } ...",
					"",
					"Arguments:",
					"  SLL: sl desc",
					"",
					"Symbols:",
					"  ;: List breaker",
				}, "\n"),
			},
		},
		// MapArg tests
		{
			name: "MapArg usage as arg and as flag",
			etc: func() *commandtest.ExecuteTestCase {
				type vType struct {
					A int
					B float64
				}
				flagMap := MapFlag("m1", 'm', "un", map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)
				otherflagMap := MapFlag("m2", FlagNoShortName, "deux", map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)
				argMap := MapArg("m3", "trois", map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						FlagProcessor(
							flagMap,
							otherflagMap,
						),
						argMap,
					),
					WantStdout: strings.Join([]string{
						"m3 --m1|-m --m2",
						"",
						"Arguments:",
						"  m3: trois",
						"",
						"Flags:",
						"  [m] m1: un",
						"      m2: deux",
					}, "\n"),
				}
			}(),
		},
		/* Useful comment for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			executeTest(t, test.etc, nil)
		})
	}
}
