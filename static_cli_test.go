package command

import "testing"

func TestStaticCLIs(t *testing.T) {
	for _, test := range []struct {
		name string
		scli *staticCLI
		etc  *ExecuteTestCase
	}{
		{
			name: "static cli works",
			scli: StaticCLI("x", "exit"),
			etc: &ExecuteTestCase{
				WantExecuteData: &ExecuteData{
					Executable: []string{"exit"},
				},
			},
		},
		{
			name: "static cli works with multiple commands",
			scli: StaticCLI("xp", "exit", "please"),
			etc: &ExecuteTestCase{
				WantExecuteData: &ExecuteData{
					Executable: []string{"exit", "please"},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.etc.Node = test.scli.Node()
			ExecuteTest(t, test.etc)
		})
	}
}
