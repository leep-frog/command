package sourcerer

import "github.com/leep-frog/command"

type Debugger struct{}

func (*Debugger) Setup() []string { return nil }
func (*Debugger) Changed() bool   { return false }
func (*Debugger) Name() string    { return "leep_debug" }

func (*Debugger) Node() command.Node {
	return command.SerialNodes(
		// Get the environment variable
		command.EnvArg(command.DebugEnvVar),
		// Either set or unset the environment variable.
		command.IfElseData(
			command.DebugEnvVar,
			command.SerialNodes(
				command.UnsetEnvVarProcessor(command.DebugEnvVar),
				command.PrintlnProcessor("Exiting debug mode."),
			),
			command.SerialNodes(
				command.SetEnvVarProcessor(command.DebugEnvVar, "1"),
				command.PrintlnProcessor("Entering debug mode."),
			),
		),
	)
}
