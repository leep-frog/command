package command

import (
	"testing"
)

const (
	aliasDesc = "  *: Start of new aliasable section"
	cacheDesc = "  ^: Start of new cachable section"
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
					"\n",
					"Symbols:",
					aliasDesc,
				},
			},
		},
		{
			name: "works with cache",
			utc: &UsageTestCase{
				Node: CacheNode("cacheName", nil, SerialNodes(
					Description("command desc"),
					StringListNode("SARG", testDesc, 0, UnboundedList),
					SimpleProcessor(nil, nil),
				)),
				WantString: []string{
					"command desc",
					"^ [ SARG ... ]",
					"",
					"Arguments:",
					"  SARG: test desc",
					"\n",
					"Symbols:",
					cacheDesc,
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
					}, nil, true),
				}, SerialNodes(Description("the default command"), IntNode("INT_ARG", "an integer"), StringListNode("STRINGS", "unltd strings", 1, UnboundedList)), false),
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
					"",
					"Symbols:",
					"  <: start of subcommand branches",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			UsageTest(t, test.utc)
		})
	}
}
