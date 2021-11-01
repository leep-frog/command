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
			name: "works with list arg",
			utc: &UsageTestCase{
				Node: SerialNodes(StringListNode("SARG", testDesc, 2, 3)),
				WantString: []string{
					"SARG",
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
