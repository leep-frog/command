package commander

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/constants"
	"github.com/leep-frog/command/internal/spycommand"
	"github.com/leep-frog/command/internal/spycommandtest"
)

type usageNode struct {
	usageErr     error
	usageNextErr error
}

func (un *usageNode) Usage(*command.Input, *command.Data, *command.Usage) error {
	return un.usageErr
}

func (un *usageNode) UsageNext(*command.Input, *command.Data) (command.Node, error) {
	return nil, un.usageNextErr
}

func (un *usageNode) Execute(*command.Input, command.Output, *command.Data, *command.ExecuteData) error {
	return nil
}
func (un *usageNode) Complete(*command.Input, *command.Data) (*command.Completion, error) {
	return nil, nil
}
func (un *usageNode) Next(*command.Input, *command.Data) (command.Node, error) {
	return nil, nil
}

func TestUsage(t *testing.T) {

	branchesForSorting := map[string]command.Node{
		"alpha": nil,
		"beta":  SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
		"charlie": &BranchNode{
			Branches: map[string]command.Node{
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
		ietc *spycommandtest.ExecuteTestCase
	}{
		{
			name: "works with empty node",
			etc: &commandtest.ExecuteTestCase{
				WantStdout: "\n",
			},
		},
		{
			name: "fails if node.Usage() returns error",
			etc: &commandtest.ExecuteTestCase{
				Node:       &usageNode{fmt.Errorf("oops"), nil},
				WantErr:    fmt.Errorf("oops"),
				WantStderr: "oops\n",
			},
		},
		{
			name: "fails if node.UsageNext() returns error",
			etc: &commandtest.ExecuteTestCase{
				Node:       &usageNode{nil, fmt.Errorf("whoops")},
				WantErr:    fmt.Errorf("whoops"),
				WantStderr: "whoops\n",
			},
		},
		{
			name: "works with basic Description node",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Description("hello %s")),
				WantStdout: strings.Join([]string{
					"hello %s",
					"",
				}, "\n"),
			},
		},
		{
			name: "works with basic Descriptionf node",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Descriptionf("hello %s", "there")),
				WantStdout: strings.Join([]string{
					"hello there",
					"",
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
					"",
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
					"",
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
					"",
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
					"",
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
					"",
				}, "\n"),
			},
		},
		{
			name: "works with default, validators, and description node",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("SARG", "desc",
						MinLength[string, string](3),
						Contains("X"),
						FileExists(),
						Default("dflt"),
					),
					Description("Does absolutely nothing"),
				),
				WantStdout: strings.Join([]string{
					"Does absolutely nothing",
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
					"    Default: dflt",
					"    MinLength(3)",
					`    Contains("X")`,
					`    FileExists()`,
					"",
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
					"",
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
				WantErr:    fmt.Errorf(`validation for "SARG" failed: [MinLength] length must be at least 3`),
				WantStderr: "validation for \"SARG\" failed: [MinLength] length must be at least 3\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "12"},
					},
				},
				WantIsValidationError: true,
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
					"",
				}, "\n"),
			},
		},
		{
			name: "works with unbounded list arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("SARG", testDesc, 0, command.UnboundedList)),
				WantStdout: strings.Join([]string{
					"[ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
					"",
				}, "\n"),
			},
		},
		{
			name: "works with shortcut",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("shortcutName", nil, SerialNodes(
					Description("command desc"),
					ListArg[string]("SARG", testDesc, 0, command.UnboundedList),
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
					constants.ShortcutDesc,
					"",
				}, "\n"),
			},
		},
		{
			name: "works with cache",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("cacheName", nil, SerialNodes(
					Description("cmd desc"),
					ListArg[string]("SARG", testDesc, 0, command.UnboundedList),
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
					constants.CacheDesc,
					"",
				}, "\n"),
			},
		},
		{
			name: "works with simple branch node",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha": nil,
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
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
				}, "\n"),
			},
		},
		{
			name: "works with simple branch node with no default",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha": nil,
					},
				},
				WantStdout: strings.Join([]string{
					"┓",
					"┗━━ alpha",
					"",
				}, "\n"),
			},
		},
		{
			name: "BranchNode usage doesn't display if default node traversed",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"123"},
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha": nil,
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"STRINGS [ STRINGS ... ]",
					"",
					"Arguments:",
					"  STRINGS: unltd strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "123"},
					},
				},
			},
		},
		{
			name: "BranchNode usage doesn't display if branch traversed",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"beta"},
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha": nil,
						"beta":  SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"ROPES ROPES [ ROPES ROPES ROPES ]",
					"",
					"Arguments:",
					"  ROPES: lots of strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "beta"},
					},
				},
			},
		},
		{
			name: "branch node fails if usage error on branch",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Description("command start"),
					Arg[string]("STRING", "A string"),
					&BranchNode{
						Branches: map[string]command.Node{
							"alpha": SerialNodes(SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
								return fmt.Errorf("usage oops")
							})),
						},
						Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
					},
				),
				WantErr:    fmt.Errorf("failed to get usage for branch alpha: usage oops"),
				WantStderr: "failed to get usage for branch alpha: usage oops\n",
			},
		},
		{
			name: "branch node fails if usage error on default",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Description("command start"),
					Arg[string]("STRING", "A string"),
					&BranchNode{
						Branches: map[string]command.Node{
							"alpha": SerialNodes(Description("first")),
						},
						Default: SerialNodes(SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							return fmt.Errorf("default usage oops")
						})),
					},
				),
				WantErr:    fmt.Errorf("failed to get usage for BranchNode default: default usage oops"),
				WantStderr: "failed to get usage for BranchNode default: default usage oops\n",
			},
		},
		{
			name: "branch node with empty BranchUsageOrder and default",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Description("command start"),
					Arg[string]("STRING", "A string"),
					&BranchNode{
						Branches: map[string]command.Node{
							"alpha": nil,
						},
						BranchUsageOrder: []string{},
						Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
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
					"",
				}, "\n"),
			},
		},
		{
			name: "branch node with empty BranchUsageOrder and no default",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Description("command start"),
					Arg[string]("STRING", "A string"),
					&BranchNode{
						Branches: map[string]command.Node{
							"alpha": nil,
						},
						BranchUsageOrder: []string{},
					},
				),
				WantStdout: strings.Join([]string{
					"command start",
					"STRING",
					"",
					"Arguments:",
					"  STRING: A string",
					"",
				}, "\n"),
			},
		},
		{
			name: "works with branch node",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha": nil,
						"beta":  SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]command.Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
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
					"    ┃",
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
				}, "\n"),
			},
		},
		{
			name: "works with branch node shortcut option",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha": SerialNodes(
							Description("The first"),
						),
						"beta": SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]command.Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Synonyms: BranchSynonyms(map[string][]string{
						"charlie": {"charles", "chuck"},
						"alpha":   {"omega"},
					}),
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
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
					"    ┃",
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
				}, "\n"),
			},
		},
		{
			name: "works with branch node shortcut option via spaces",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha omega1": SerialNodes(
							Description("The first"),
						),
						"beta": SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]command.Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Synonyms: BranchSynonyms(map[string][]string{
						"charlie": {"charles", "chuck"},
						"alpha":   {"omega2"},
					}),
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
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
					"    ┃",
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
				}, "\n"),
			},
		},
		{
			name: "works with multiple node",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"alpha": nil,
						"beta": &BranchNode{
							Branches: map[string]command.Node{
								"one":   SerialNodes(Description("First"), Arg[int]("ONE", "A number")),
								"two":   SerialNodes(Arg[int]("TWO", "Another number")),
								"three": SerialNodes(&ExecutorProcessor{}),
							},
						},
						"charlie": &BranchNode{
							Branches: map[string]command.Node{
								"delta": SerialNodes(SerialNodes(Description("Something else"), Arg[string]("DELTA", "delta description"))),
								"brown": &BranchNode{
									Branches: map[string]command.Node{
										"movie":      SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
										"comic":      SerialNodes(Description("Comic strip")),
										"characters": SerialNodes(ListArg[string]("CHARACTERS", "Character names", 2, 1)),
									},
								},
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Default: SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┣━━ alpha",
					"┃",
					"┣━━ beta ┓",
					"┃   ┏━━━━┛",
					"┃   ┃",
					"┃   ┃   First",
					"┃   ┣━━ one ONE",
					"┃   ┃",
					"┃   ┣━━ three",
					"┃   ┃",
					"┃   ┗━━ two TWO",
					"┃",
					"┗━━ charlie ┓",
					"    ┏━━━━━━━┛",
					"    ┃",
					"    ┣━━ brown ┓",
					"    ┃   ┏━━━━━┛",
					"    ┃   ┃",
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
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
				},
				WantErr:    fmt.Errorf("provided branch (foxtrot) isn't a valid branch (note branch synonyms aren't allowed in BranchUsageOrder)"),
				WantStderr: "provided branch (foxtrot) isn't a valid branch (note branch synonyms aren't allowed in BranchUsageOrder)\n",
			},
		},
		{
			name: "BranchUsageOrderFunc works if fewer strings",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
				},
				WantStdout: strings.Join([]string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┣━━ alpha",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					// `charlie` isn't included in list, hence why it's omitted here.
					"┣━━ delta",
					"┃",
					"┗━━ echo",
					"",
					"Arguments:",
					"  INT_ARG: an integer",
					"  ROPES: lots of strings",
					"  STRINGS: unltd strings",
					"",
				}, "\n"),
			},
		},
		{
			name: "BranchUsageOrderFunc fails if duplicate strings",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "beta", "charlie", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
				},
				WantErr:    fmt.Errorf("BranchUsageOrder contains a duplicate entry (beta)"),
				WantStderr: "BranchUsageOrder contains a duplicate entry (beta)\n",
			},
		},
		{
			name: "BranchUsageOrderFunc fails if duplicate strings but right number",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "beta", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
				},
				WantErr:    fmt.Errorf("BranchUsageOrder contains a duplicate entry (beta)"),
				WantStderr: "BranchUsageOrder contains a duplicate entry (beta)\n",
			},
		},
		{
			name: "BranchUsageOrderFunc works",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					BranchUsageOrder: []string{"alpha", "beta", "charlie", "delta", "echo"},
					Branches:         branchesForSorting,
					Default:          SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, command.UnboundedList)),
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
					"┃   ┃",
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
				}, "\n"),
			},
		},
		// Flag tests
		{
			name: "works with flags (in provided order 1)",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FlagProcessor(
					BoolFlag("debug", 'd', "debug stuff"),
					BoolFlag("new", 'n', "new files"),
				)),
				WantStdout: strings.Join([]string{
					"--debug|-d --new|-n",
					"",
					"Flags:",
					"  [d] debug: debug stuff",
					"  [n] new: new files",
					"",
				}, "\n"),
			},
		},
		{
			name: "works with flags (in provided order 2)",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FlagProcessor(
					BoolFlag("new", 'n', "new files"),
					BoolFlag("debug", 'd', "debug stuff"),
				)),
				WantStdout: strings.Join([]string{
					"--new|-n --debug|-d",
					"",
					"Flags:",
					"  [d] debug: debug stuff",
					"  [n] new: new files",
					"",
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
				WantErr:    fmt.Errorf(`validation for "str" failed: [Contains] value doesn't contain substring "t"`),
				WantStderr: "validation for \"str\" failed: [Contains] value doesn't contain substring \"t\"\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "nope"},
					},
				},
				WantIsValidationError: true,
			},
		},
		// ListFlag(1, 0) usage tests
		{
			name: "ListFlag(1, 0) and no flag prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("BEFORE", ""),
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, 0),
					),
					Arg[string]("AFTER", ""),
				),
				WantStdout: strings.Join([]string{
					"BEFORE AFTER --str|-s STR",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
		},
		{
			name: "ListFlag(1, 0) and flag with no args prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, 0),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND --str|-s STR",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
					},
				},
			},
		},
		{
			name: "ListFlag(1, 0) and flag with arg does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, 0),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
					},
				},
			},
		},
		// ListFlag(2, 0) usage tests
		{
			name: "ListFlag(2, 0) and no flag prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("BEFORE", ""),
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 0),
					),
					Arg[string]("AFTER", ""),
				),
				WantStdout: strings.Join([]string{
					"BEFORE AFTER --str|-s STR STR",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
		},
		{
			name: "ListFlag(2, 0) and flag with no args prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 0),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND --str|-s STR STR",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
					},
				},
			},
		},
		{
			name: "ListFlag(2, 0) and flag with not enough args prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 0),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND --str|-s STR STR",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
					},
				},
			},
		},
		{
			name: "ListFlag(2, 0) and flag with enough args does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v", "w"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 0),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
						{Value: "w"},
					},
				},
			},
		},
		// ListFlag(1, 1) usage tests
		{
			name: "ListFlag(1, 1) and no flag prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("BEFORE", ""),
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, 1),
					),
					Arg[string]("AFTER", ""),
				),
				WantStdout: strings.Join([]string{
					"BEFORE AFTER --str|-s STR [ STR ]",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
		},
		{
			name: "ListFlag(1, 1) and flag with no args prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, 1),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND --str|-s STR [ STR ]",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
					},
				},
			},
		},
		{
			name: "ListFlag(1, 1) and flag with arg does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, 1),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
					},
				},
			},
		},
		{
			name: "ListFlag(1, 1) and flag with optional args does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v", "w"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, 1),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
						{Value: "w"},
					},
				},
			},
		},
		// ListFlag(2, 2) usage tests
		{
			name: "ListFlag(2, 2) and no flag prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("BEFORE", ""),
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 2),
					),
					Arg[string]("AFTER", ""),
				),
				WantStdout: strings.Join([]string{
					"BEFORE AFTER --str|-s STR STR [ STR STR ]",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
		},
		{
			name: "ListFlag(2, 2) and flag with no args prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 2),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND --str|-s STR STR [ STR STR ]",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
					},
				},
			},
		},
		{
			name: "ListFlag(2, 2) and flag with not enough args prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 2),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND --str|-s STR STR [ STR STR ]",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
					},
				},
			},
		},
		{
			name: "ListFlag(2, 2) and flag with enough args does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v", "w"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 2),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
						{Value: "w"},
					},
				},
			},
		},
		{
			name: "ListFlag(2, 2) and flag with some optional args does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v", "w", "vw"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 2),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
						{Value: "w"},
						{Value: "vw"},
					},
				},
			},
		},
		{
			name: "ListFlag(2, 2) and flag with all optional args does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v", "w", "vw", "ww"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 2, 2),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
						{Value: "w"},
						{Value: "vw"},
						{Value: "ww"},
					},
				},
			},
		},
		// ListFlag(1, UnboundedList) usage tests
		{
			name: "ListFlag(1, UnboundedList) and no flag prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("BEFORE", ""),
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, command.UnboundedList),
					),
					Arg[string]("AFTER", ""),
				),
				WantStdout: strings.Join([]string{
					"BEFORE AFTER --str|-s STR [ STR ... ]",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
		},
		{
			name: "ListFlag(1, UnboundedList) and flag with no args prints flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, command.UnboundedList),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND --str|-s STR [ STR ... ]",
					"",
					"Flags:",
					"  [s] str: strings",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
					},
				},
			},
		},
		{
			name: "ListFlag(1, UnboundedList) and flag with arg does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, command.UnboundedList),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
					},
				},
			},
		},
		{
			name: "ListFlag(1, UnboundedList) and flag with optional args does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v", "w"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, command.UnboundedList),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
						{Value: "w"},
					},
				},
			},
		},
		{
			name: "ListFlag(1, UnboundedList) and flag with lots of optional args does not print flag usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--str", "v", "w", "vw", "ww"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("str", 's', "strings", 1, command.UnboundedList),
					),
					Arg[string]("FIRST", ""),
					Arg[string]("SECOND", ""),
				),
				WantStdout: strings.Join([]string{
					"FIRST SECOND",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--str"},
						{Value: "v"},
						{Value: "w"},
						{Value: "vw"},
						{Value: "ww"},
					},
				},
			},
		},
		{
			name: "flags go at the end",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("debug", 'd', "debug stuff"),
						BoolFlag("new", 'n', "new files"),
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
					"",
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
					"",
				}, "\n"),
			},
		},
		{
			name: "flags with values",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--second", "2nd"},
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", 'b', "un"),
						Flag[string]("second", 'a', "deux"),
					),
					Arg[string]("SN", "node for a string"),
				),

				WantStdout: strings.Join([]string{
					"SN --first|-b",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"  [b] first: un",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--second"},
						{Value: "2nd"},
					},
				},
			},
		},
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
					"SN --first --second|-2 --third|-3 THIRD --fourth FOURTH",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"      first: un",
					"      fourth: quatre",
					"  [2] second: deux",
					"  [3] third: trois",
					"",
				}, "\n"),
			},
		},
		// Flag order on not enough args error
		{
			name: "Flag order doesn't change if not enough args error for first flag",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--before"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("before", 'b', "before flag", 1, command.UnboundedList),
						ListFlag[string]("after", 'a', "after flag", 1, 0),
					),
					Arg[string]("ARG_1", ""),
					Arg[string]("ARG_2", ""),
				),
				WantStdout: strings.Join([]string{
					"ARG_1 ARG_2 --before|-b BEFORE [ BEFORE ... ] --after|-a AFTER",
					"",
					"Flags:",
					"  [a] after: after flag",
					"  [b] before: before flag",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--before"},
					},
				},
			},
		},
		{
			name: "Flag order doesn't change if not enough args error for second flag",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--after"},
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("before", 'b', "before flag", 1, command.UnboundedList),
						ListFlag[string]("after", 'a', "after flag", 1, 0),
					),
					Arg[string]("ARG_1", ""),
					Arg[string]("ARG_2", ""),
				),
				WantStdout: strings.Join([]string{
					"ARG_1 ARG_2 --before|-b BEFORE [ BEFORE ... ] --after|-a AFTER",
					"",
					"Flags:",
					"  [a] after: after flag",
					"  [b] before: before flag",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--after"},
					},
				},
			},
		},
		// ItemizedListFlag tests
		{
			name: "ItemizedListFlag is included in usage",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", FlagNoShortName, "un"),
						ItemizedListFlag[string]("ilf", 'i', "itemized"),
						Flag[string]("third", '3', "trois"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantStdout: strings.Join([]string{
					"SN --first --ilf|-i ILF --third|-3 THIRD",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"      first: un",
					"  [i] ilf: itemized",
					"  [3] third: trois",
					"",
				}, "\n"),
			},
		},
		{
			name: "ItemizedListFlag is included in usage if flag provided",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--ilf"},
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", FlagNoShortName, "un"),
						ItemizedListFlag[string]("ilf", 'i', "itemized"),
						Flag[string]("third", '3', "trois"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantStdout: strings.Join([]string{
					"SN --first --ilf|-i ILF --third|-3 THIRD",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"      first: un",
					"  [i] ilf: itemized",
					"  [3] third: trois",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--ilf"},
					},
				},
			},
		},
		{
			name: "ItemizedListFlag is not included in usage if flag provided with arg",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--ilf", "v"},
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", FlagNoShortName, "un"),
						ItemizedListFlag[string]("ilf", 'i', "itemized"),
						Flag[string]("third", '3', "trois"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantStdout: strings.Join([]string{
					"SN --first --third|-3 THIRD",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"      first: un",
					"  [3] third: trois",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--ilf"},
						{Value: "v"},
					},
				},
			},
		},
		// NodeRepeater tests
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
					"",
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
					"",
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
					"",
				}, "\n"),
			},
		},
		{
			name: "NodeRepeater usage works for unbounded",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(3, command.UnboundedList)),
				WantStdout: strings.Join([]string{
					"KEY VALUE KEY VALUE KEY VALUE { KEY VALUE } ...",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
					"",
				}, "\n"),
			},
		},
		// ListBreaker tests
		{
			name: "NodeRepeater usage works for unbounded",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 0, command.UnboundedList),
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
					"",
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
					"",
				}, "\n"),
			},
		},
		{
			name: "unbounded StringListListProcessor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(StringListListProcessor("SLL", "sl desc", ";", 1, command.UnboundedList)),
				WantStdout: strings.Join([]string{
					"[ SLL ... ] ; { [ SLL ... ] ; } ...",
					"",
					"Arguments:",
					"  SLL: sl desc",
					"",
					"Symbols:",
					"  ;: List breaker",
					"",
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
						"m3 --m1|-m MAP_KEY --m2 MAP_KEY",
						"",
						"Arguments:",
						"  m3: trois",
						"",
						"Flags:",
						"  [m] m1: un",
						"      m2: deux",
						"",
					}, "\n"),
				}
			}(),
		},
		// MutableProcessor tests
		{
			name: "Reference does not update underlying processor",
			etc: func() *commandtest.ExecuteTestCase {
				hi := &simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
					u.SetDescription("hi")
					return nil
				}}
				hello := &simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
					u.SetDescription("hello")
					return nil
				}}

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						&simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							hi = hello
							return nil
						}},
						hi,
						&simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							su := &command.Usage{}
							err := hi.Usage(nil, nil, su)
							if err != nil {
								return err
							}
							u.AddSymbol("hi value", u.String())
							return nil
						}},
					),
					WantStdout: strings.Join([]string{
						// From shallow reference usage
						"hi",
						// From referenced change usage
						"hi value",
						"",
						"Symbols:",
						"  hi value: hi",
						"",
					}, "\n"),
				}
			}(),
		},
		{
			name: "MutableProcessor DOES update underlying processor",
			etc: func() *commandtest.ExecuteTestCase {
				hi := NewMutableProcessor(&simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
					u.SetDescription("hi")
					return nil
				}})
				hello := &simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
					u.SetDescription("hello")
					return nil
				}}

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						&simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							hi.Processor = &hello
							return nil
						}},
						hi,
						&simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							su := &command.Usage{}
							err := hi.Usage(nil, nil, su)
							if err != nil {
								return err
							}
							u.AddSymbol("hi value", u.String())
							return nil
						}},
					),
					WantStdout: strings.Join([]string{
						// From shallow reference usage
						"hello",
						// From referenced change usage
						"hi value",
						"",
						"Symbols:",
						"  hi value: hello",
						"",
					}, "\n"),
				}
			}(),
		},
		/* Useful comment for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.etc == nil {
				test.etc = &commandtest.ExecuteTestCase{}
			}
			test.etc.Args = append(test.etc.Args, "--help")
			if test.ietc == nil {
				test.ietc = &spycommandtest.ExecuteTestCase{}
			}
			test.ietc.SkipErrorTypeCheck = false
			test.ietc.SkipInputCheck = false
			executeTest(t, test.etc, test.ietc)
		})
	}
}
