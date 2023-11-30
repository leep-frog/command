package sourcerer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
)

func TestDebugger(t *testing.T) {
	fos := &commandtest.FakeOS{}
	for _, test := range []struct {
		name string
		etc  *commandtest.ExecuteTestCase
	}{
		{
			name: "Activates debug mode",
			etc: &commandtest.ExecuteTestCase{
				WantStdout: "Entering debug mode.\n",
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fos.SetEnvVar(commander.DebugEnvVar, "1"),
					},
				},
			},
		},
		{
			name: "Deactivates debug mode",
			etc: &commandtest.ExecuteTestCase{
				WantStdout: "Exiting debug mode.\n",
				Env: map[string]string{
					commander.DebugEnvVar: "1",
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fos.UnsetEnvVar(commander.DebugEnvVar),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					commander.DebugEnvVar: "1",
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run(test.name, func(t *testing.T) {
				cli := &Debugger{}
				test.etc.Node = cli.Node()
				test.etc.OS = fos
				commandertest.ExecuteTest(t, test.etc)
				commandertest.ChangeTest(t, nil, cli)
			})
		})
	}
}

func TestDebuggerMetadata(t *testing.T) {
	cli := &Debugger{}
	if diff := cmp.Diff("leep_debug", cli.Name()); diff != "" {
		t.Errorf("Unexpected cli name (-want, +got):\n%s", diff)
	}

	if setup := cli.Setup(); setup != nil {
		t.Errorf("Expected cli.Setup() to be nil; got %v", setup)
	}
}
