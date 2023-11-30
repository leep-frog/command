package commander

import (
	"testing"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
)

func TestStaticCLIs(t *testing.T) {
	for _, test := range []struct {
		name string
		scli *staticCLI
		etc  *commandtest.ExecuteTestCase
	}{
		{
			name: "static cli works",
			scli: StaticCLI("x", "exit"),
			etc: &commandtest.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"exit"},
				},
			},
		},
		{
			name: "static cli works with multiple commands",
			scli: StaticCLI("xp", "exit", "please"),
			etc: &commandtest.ExecuteTestCase{
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"exit", "please"},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.etc.Node = test.scli.Node()
			executeTest(t, test.etc, nil)
		})
	}
}
