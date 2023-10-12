package command

import (
	"fmt"
	"testing"
)

type usageNode struct {
	usageErr     error
	usageNextErr error
}

func (un *usageNode) Usage(*Input, *Data, *Usage) error {
	return un.usageErr
}

func (un *usageNode) UsageNext(*Input, *Data) (Node, error) {
	return nil, un.usageNextErr
}

func (un *usageNode) Execute(*Input, Output, *Data, *ExecuteData) error { return nil }
func (un *usageNode) Complete(*Input, *Data) (*Completion, error)       { return nil, nil }
func (un *usageNode) Next(*Input, *Data) (Node, error)                  { return nil, nil }

func TestUsage(t *testing.T) {
	for _, test := range []struct {
		name string
		utc  *UsageTestCase
	}{
		{
			name: "works with empty node",
		},
		{
			name: "fails if node.Usage() returns error",
			utc: &UsageTestCase{
				Node:    &usageNode{fmt.Errorf("oops"), nil},
				WantErr: fmt.Errorf("oops"),
			},
		},
		{
			name: "fails if node.UsageNext() returns error",
			utc: &UsageTestCase{
				Node:    &usageNode{nil, fmt.Errorf("whoops")},
				WantErr: fmt.Errorf("whoops"),
			},
		},
		{
			name: "works with basic Description node",
			utc: &UsageTestCase{
				Node: SerialNodes(Description("hello %s")),
				WantString: []string{
					"hello %s",
				},
			},
		},
		{
			name: "works with basic Descriptionf node",
			utc: &UsageTestCase{
				Node: SerialNodes(Descriptionf("hello %s", "there")),
				WantString: []string{
					"hello there",
				},
			},
		},
		{
			name: "works with single arg",
			utc: &UsageTestCase{
				Node: SerialNodes(Arg[string]("SARG", "desc")),
				WantString: []string{
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
				},
			},
		},
		{
			name: "works with optional arg",
			utc: &UsageTestCase{
				Node: SerialNodes(OptionalArg[string]("SARG", "desc")),
				WantString: []string{
					"[ SARG ]",
					"",
					"Arguments:",
					"  SARG: desc",
				},
			},
		},
		{
			name: "works with hidden arg",
			utc: &UsageTestCase{
				Node: SerialNodes(
					Arg[string]("SARG1", "desc"),
					Arg("SARG2", "desc", HiddenArg[string]()),
					Arg[string]("SARG3", "desc"),
				),
				WantString: []string{
					"SARG1 SARG3",
					"",
					"Arguments:",
					"  SARG1: desc",
					"  SARG3: desc",
				},
			},
		},
		{
			name: "setup arg is hidden",
			utc: &UsageTestCase{
				Node: SerialNodes(
					Arg[string]("SARG", "desc"),
					SetupArg,
					Arg[int]("IARG", "idesc"),
				),
				WantString: []string{
					"SARG IARG",
					"",
					"Arguments:",
					"  IARG: idesc",
					"  SARG: desc",
				},
			},
		},
		{
			name: "works with single arg and description node",
			utc: &UsageTestCase{
				Node: SerialNodes(Arg[string]("SARG", "desc"), Description("Does absolutely nothing")),
				WantString: []string{
					"Does absolutely nothing",
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
				},
			},
		},
		{
			name: "works with validators and description node",
			utc: &UsageTestCase{
				Node: SerialNodes(
					Arg[string]("SARG", "desc",
						MinLength[string, string](3),
						Contains("X"),
						FileExists(),
					),
					Description("Does absolutely nothing"),
				),
				WantString: []string{
					"Does absolutely nothing",
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
					"    MinLength(3)",
					`    Contains("X")`,
					`    FileExists()`,
				},
			},
		},
		{
			name: "works with multiple args with validators",
			utc: &UsageTestCase{
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
				WantString: []string{
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
				},
			},
		},
		{
			name: "works with list arg",
			utc: &UsageTestCase{
				Node: SerialNodes(ListArg[string]("SARG", testDesc, 2, 3)),
				WantString: []string{
					"SARG SARG [ SARG SARG SARG ]",
					"",
					"Arguments:",
					"  SARG: test desc",
				},
			},
		},
		{
			name: "works with unbounded list arg",
			utc: &UsageTestCase{
				Node: SerialNodes(ListArg[string]("SARG", testDesc, 0, UnboundedList)),
				WantString: []string{
					"[ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
				},
			},
		},
		{
			name: "works with shortcut",
			utc: &UsageTestCase{
				Node: ShortcutNode("shortcutName", nil, SerialNodes(
					Description("command desc"),
					ListArg[string]("SARG", testDesc, 0, UnboundedList),
					SimpleProcessor(nil, nil),
				)),
				WantString: []string{
					"command desc",
					"* [ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
					"",
					"Symbols:",
					ShortcutDesc,
				},
			},
		},
		{
			name: "works with cache",
			utc: &UsageTestCase{
				Node: CacheNode("cacheName", nil, SerialNodes(
					Description("cmd desc"),
					ListArg[string]("SARG", testDesc, 0, UnboundedList),
					SimpleProcessor(nil, nil),
				)),
				WantString: []string{
					"cmd desc",
					"^ [ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
					"",
					"Symbols:",
					CacheDesc,
				},
			},
		},
		{
			name: "works with simple branch node",
			utc: &UsageTestCase{
				Node: &BranchNode{
					Branches: map[string]Node{
						"alpha": nil,
					},
					Default:           SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, UnboundedList)),
					DefaultCompletion: true,
				},
				WantString: []string{
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
					BranchDesc,
				},
			},
		},
		{
			name: "works with branch node",
			utc: &UsageTestCase{
				Node: &BranchNode{
					Branches: map[string]Node{
						"alpha": nil,
						"beta":  SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Default:           SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, UnboundedList)),
					DefaultCompletion: true,
				},
				WantString: []string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┣━━ alpha",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					"┗━━ charlie ┳",
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
					BranchDesc,
				},
			},
		},
		{
			name: "works with branch node shortcut option",
			utc: &UsageTestCase{
				Node: &BranchNode{
					Branches: map[string]Node{
						"alpha": SerialNodes(
							Description("The first"),
						),
						"beta": SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Synonyms: BranchSynonyms(map[string][]string{
						"charlie": {"charles", "chuck"},
						"alpha":   {"omega"},
					}),
					Default:           SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, UnboundedList)),
					DefaultCompletion: true,
				},
				WantString: []string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┃   The first",
					"┣━━ [alpha|omega]",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					"┗━━ [charlie|charles|chuck] ┳",
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
					BranchDesc,
				},
			},
		},
		{
			name: "works with branch node shortcut option via spaces",
			utc: &UsageTestCase{
				Node: &BranchNode{
					Branches: map[string]Node{
						"alpha omega1": SerialNodes(
							Description("The first"),
						),
						"beta": SerialNodes(ListArg[string]("ROPES", "lots of strings", 2, 3)),
						"charlie": &BranchNode{
							Branches: map[string]Node{
								"brown":  SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Synonyms: BranchSynonyms(map[string][]string{
						"charlie": {"charles", "chuck"},
						"alpha":   {"omega2"},
					}),
					Default:           SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, UnboundedList)),
					DefaultCompletion: true,
				},
				WantString: []string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┃   The first",
					"┣━━ [alpha|omega1|omega2]",
					"┃",
					"┣━━ beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"┃",
					"┗━━ [charlie|charles|chuck] ┳",
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
					BranchDesc,
				},
			},
		},
		{
			name: "works with multiple node",
			utc: &UsageTestCase{
				Node: &BranchNode{
					Branches: map[string]Node{
						"alpha": nil,
						"beta": &BranchNode{
							Branches: map[string]Node{
								"one":   SerialNodes(Description("First"), Arg[int]("ONE", "A number")),
								"two":   SerialNodes(Arg[int]("TWO", "Another number")),
								"three": SerialNodes(&ExecutorProcessor{}),
							},
						},
						"charlie": &BranchNode{
							Branches: map[string]Node{
								"delta": SerialNodes(SerialNodes(Description("Something else"), Arg[string]("DELTA", "delta description"))),
								"brown": &BranchNode{
									Branches: map[string]Node{
										"movie":      SerialNodes(Description("learn about cartoons"), Arg[float64]("FLOATER", "something bouyant")),
										"comic":      SerialNodes(Description("Comic strip")),
										"characters": SerialNodes(ListArg[string]("CHARACTERS", "Character names", 2, 1)),
									},
								},
								"yellow": SerialNodes(&ExecutorProcessor{}),
							},
						},
					},
					Default:           SerialNodes(Description("the default command"), Arg[int]("INT_ARG", "an integer"), ListArg[string]("STRINGS", "unltd strings", 1, UnboundedList)),
					DefaultCompletion: true,
				},
				WantString: []string{
					"the default command",
					"┳ INT_ARG STRINGS [ STRINGS ... ]",
					"┃",
					"┣━━ alpha",
					"┃",
					"┣━━ beta ┳",
					"┃   ┏━━━━┛",
					"┃   ┃   First",
					"┃   ┣━━ one ONE",
					"┃   ┃",
					"┃   ┣━━ three",
					"┃   ┃",
					"┃   ┗━━ two TWO",
					"┃",
					"┗━━ charlie ┳",
					"    ┏━━━━━━━┛",
					"    ┣━━ brown ┳",
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
					BranchDesc,
				},
			},
		},
		{
			name: "works with flags",
			utc: &UsageTestCase{
				Node: SerialNodes(FlagProcessor(
					BoolFlag("new", 'n', "new files"),
					BoolFlag("debug", 'd', "debug stuff"),
				)),
				WantString: []string{
					"--debug|-d --new|-n",
					"",
					"Flags:",
					"  [d] debug: debug stuff",
					"  [n] new: new files",
				},
			},
		},
		{
			name: "Works with input flag and no args",
			utc: &UsageTestCase{
				Args: []string{"--str"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantString: []string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				},
			},
		},
		{
			name: "Works with input flag and some args",
			utc: &UsageTestCase{
				Args: []string{"--str", "un"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantString: []string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				},
			},
		},
		{
			name: "Works with input flag and required args",
			utc: &UsageTestCase{
				Args: []string{"--str", "un", "deux"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantString: []string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				},
			},
		},
		{
			name: "Works with input flag and all args",
			utc: &UsageTestCase{
				Args: []string{"--str", "un", "deux", "trois"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantString: []string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				},
			},
		},
		{
			name: "Works with input flag and extra args",
			utc: &UsageTestCase{
				Args: []string{"--str", "un", "deux", "trois", "quatre"},
				Node: SerialNodes(FlagProcessor(
					ListFlag[string]("str", 's', "strings", 2, 1),
				)),
				WantString: []string{
					"--str|-s",
					"",
					"Flags:",
					"  [s] str: strings",
				},
			},
		},
		{
			name: "flags go at the end",
			utc: &UsageTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("new", 'n', "new files"),
						BoolFlag("debug", 'd', "debug stuff"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantString: []string{
					"SN --debug|-d --new|-n",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"  [d] debug: debug stuff",
					"  [n] new: new files",
				},
			},
		},
		{
			name: "flags are sorted by full name, not short flag",
			utc: &UsageTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", 'b', "un"),
						BoolFlag("second", 'a', "deux"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantString: []string{
					"SN --first|-b --second|-a",
					"",
					"Arguments:",
					"  SN: node for a string",
					"",
					"Flags:",
					"  [b] first: un",
					"  [a] second: deux",
				},
			},
		},
		{
			name: "flags without short names work",
			utc: &UsageTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("first", FlagNoShortName, "un"),
						BoolFlag("second", '2', "deux"),
						Flag[string]("third", '3', "trois"),
						Flag[string]("fourth", FlagNoShortName, "quatre"),
					),
					Arg[string]("SN", "node for a string"),
				),
				WantString: []string{
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
				},
			},
		},
		{
			name: "NodeRepeater usage works for finite optional",
			utc: &UsageTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 1)),
				WantString: []string{
					"KEY VALUE KEY VALUE { KEY VALUE }",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				},
			},
		},
		{
			name: "NodeRepeater usage works for no optional",
			utc: &UsageTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 0)),
				WantString: []string{
					"KEY VALUE KEY VALUE",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				},
			},
		},
		{
			name: "NodeRepeater usage works for no required",
			utc: &UsageTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(0, 1)),
				WantString: []string{
					"{ KEY VALUE }",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				},
			},
		},
		{
			name: "NodeRepeater usage works for unbounded",
			utc: &UsageTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(3, UnboundedList)),
				WantString: []string{
					"KEY VALUE KEY VALUE KEY VALUE { KEY VALUE } ...",
					"",
					"Arguments:",
					"  KEY: test desc",
					"  VALUE: test desc",
				},
			},
		},
		// ListBreaker tests
		{
			name: "NodeRepeater usage works for unbounded",
			utc: &UsageTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol[[]string]("ghi")),
					ListArg[string]("SL2", testDesc, 0, UnboundedList),
				),
				WantString: []string{
					"SL [ SL ... ] ghi [ SL2 ... ]",
					"",
					"Arguments:",
					"  SL: test desc",
					"  SL2: test desc",
					"",
					"Symbols:",
					"  ghi: List breaker",
				},
			},
		},
		// StringListListProcessor
		{
			name: "StringListListProcessor",
			utc: &UsageTestCase{
				Node: SerialNodes(StringListListProcessor("SLL", "sl desc", ";", 1, 2)),
				WantString: []string{
					"[ SLL ... ] ; { [ SLL ... ] ; [ SLL ... ] ; }",
					"",
					"Arguments:",
					"  SLL: sl desc",
					"",
					"Symbols:",
					"  ;: List breaker",
				},
			},
		},
		{
			name: "unbounded StringListListProcessor",
			utc: &UsageTestCase{
				Node: SerialNodes(StringListListProcessor("SLL", "sl desc", ";", 1, UnboundedList)),
				WantString: []string{
					"[ SLL ... ] ; { [ SLL ... ] ; } ...",
					"",
					"Arguments:",
					"  SLL: sl desc",
					"",
					"Symbols:",
					"  ;: List breaker",
				},
			},
		},
		/* Useful comment for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			UsageTest(t, test.utc)
		})
	}
}
