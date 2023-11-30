package sourcerer

import (
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commondels"
)

type Debugger struct{}

func (*Debugger) Setup() []string { return nil }
func (*Debugger) Changed() bool   { return false }
func (*Debugger) Name() string    { return "leep_debug" }

func (*Debugger) Node() commondels.Node {
	return commander.SerialNodes(
		// Get the environment variable
		commander.EnvArg(commander.DebugEnvVar),
		// Either set or unset the environment variable.
		commander.IfElseData(
			commander.DebugEnvVar,
			commander.SerialNodes(
				commander.UnsetEnvVarProcessor(commander.DebugEnvVar),
				commander.PrintlnProcessor("Exiting debug mode."),
			),
			commander.SerialNodes(
				commander.SetEnvVarProcessor(commander.DebugEnvVar, "1"),
				commander.PrintlnProcessor("Entering debug mode."),
			),
		),
	)
}
