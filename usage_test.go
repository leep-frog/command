package command

import (
	"testing"
)

func TestUsage(t *testing.T) {
	for _, test := range []struct {
		name string
		utc  *UsageTestCase
	}{
		{
			name: "works with empty node",
		},
		{
			name: "works with single arg",
			utc: &UsageTestCase{
				Node: SerialNodes(StringNode("SARG", "desc")),
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
				Node: SerialNodes(OptionalStringNode("SARG", "desc")),
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
					StringNode("SARG1", "desc"),
					StringNode("SARG2", "desc", HiddenArg()),
					StringNode("SARG3", "desc"),
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
					StringNode("SARG", "desc"),
					SetupArg,
					IntNode("IARG", "idesc"),
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
				Node: SerialNodes(StringNode("SARG", "desc"), Description("Does absolutely nothing")),
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
			name: "works with list arg",
			utc: &UsageTestCase{
				Node: SerialNodes(StringListNode("SARG", testDesc, 2, 3)),
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
				Node: SerialNodes(StringListNode("SARG", testDesc, 0, UnboundedList)),
				WantString: []string{
					"[ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
				},
			},
		},
		{
			name: "works with alias",
			utc: &UsageTestCase{
				Node: AliasNode("aliasName", nil, SerialNodes(
					Description("command desc"),
					StringListNode("SARG", testDesc, 0, UnboundedList),
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
					AliasDesc,
				},
			},
		},
		{
			name: "works with cache",
			utc: &UsageTestCase{
				Node: CacheNode("cacheName", nil, SerialNodes(
					Description("cmd desc"),
					StringListNode("SARG", testDesc, 0, UnboundedList),
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
			name: "works with branch node",
			utc: &UsageTestCase{
				Node: BranchNode(map[string]*Node{
					"alpha": nil,
					"beta":  SerialNodes(StringListNode("ROPES", "lots of strings", 2, 3)),
					"charlie": BranchNode(map[string]*Node{
						"brown":  SerialNodes(Description("learn about cartoons"), FloatNode("FLOATER", "something bouyant")),
						"yellow": SerialNodes(ExecutorNode(nil)),
					}, nil),
				}, SerialNodes(Description("the default command"), IntNode("INT_ARG", "an integer"), StringListNode("STRINGS", "unltd strings", 1, UnboundedList)), DontCompleteSubcommands()),
				WantString: []string{
					"the default command",
					"< INT_ARG STRINGS [ STRINGS ... ]",
					"",
					"  alpha",
					"",
					"  beta ROPES ROPES [ ROPES ROPES ROPES ]",
					"",
					"  charlie <",
					"",
					"    learn about cartoons",
					"    brown FLOATER",
					"",
					"    yellow",
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
			name: "works with flags",
			utc: &UsageTestCase{
				Node: SerialNodesTo(nil, NewFlagNode(
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
			name: "flags go at the end",
			utc: &UsageTestCase{
				Node: SerialNodesTo(nil,
					NewFlagNode(
						BoolFlag("new", 'n', "new files"),
						BoolFlag("debug", 'd', "debug stuff"),
					),
					StringNode("SN", "node for a string"),
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
				Node: SerialNodesTo(nil,
					NewFlagNode(
						BoolFlag("first", 'b', "un"),
						BoolFlag("second", 'a', "deux"),
					),
					StringNode("SN", "node for a string"),
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
			name: "NodeRepeater usage works for finite optional",
			utc: &UsageTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 1)),
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
				Node: SerialNodes(sampleRepeaterNode(2, 0)),
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
				Node: SerialNodes(sampleRepeaterNode(0, 1)),
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
				Node: SerialNodes(sampleRepeaterNode(3, UnboundedList)),
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
					StringListNode("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi")),
					StringListNode("SL2", testDesc, 0, UnboundedList),
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
		// StringListListNode
		{
			name: "StringListListNode",
			utc: &UsageTestCase{
				Node: SerialNodes(StringListListNode("SLL", "sl desc", ";", 1, 2)),
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
			name: "unbounded StringListListNode",
			utc: &UsageTestCase{
				Node: SerialNodes(StringListListNode("SLL", "sl desc", ";", 1, UnboundedList)),
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
