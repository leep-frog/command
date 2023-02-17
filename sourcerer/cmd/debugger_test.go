package main

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestDebugger(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *command.ExecuteTestCase
	}{
		{
			name: "Activates debug mode",
			etc: &command.ExecuteTestCase{
				WantStdout: "Entering debug mode.\n",
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("export %q=%q", command.DebugEnvVar, "1"),
					},
				},
			},
		},
		{
			name: "Deactivates debug mode",
			etc: &command.ExecuteTestCase{
				WantStdout: "Exiting debug mode.\n",
				Env: map[string]string{
					command.DebugEnvVar: "1",
				},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fmt.Sprintf("unset %q", command.DebugEnvVar),
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					command.DebugEnvVar: "1",
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run(test.name, func(t *testing.T) {
				cli := &Debugger{}
				test.etc.Node = cli.Node()
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, cli)
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
