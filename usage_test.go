package command

import "testing"

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
	} {
		t.Run(test.name, func(t *testing.T) {
			UsageTest(t, test.utc)
		})
	}
}
