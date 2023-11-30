package commander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/stubs"
)

// EnvArg loads the provided environment variable's value into `command.Data`.
// The provided `name` is also used as the `command.Data` key.
func EnvArg(name string) *GetProcessor[string] {
	return &GetProcessor[string]{
		SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
			if v, ok := stubs.OSLookupEnv(name); ok {
				d.Set(name, v)
			}
			return nil
		}),
		name,
	}
}

// SetEnvVarProcessor returns a `command.Processor` that sets the environment variable to the provided value.
func SetEnvVarProcessor(envVar, value string) command.Processor {
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.SetEnvVar(envVar, value))
		return nil
	}, nil)
}

// UnsetEnvVarProcessor returns a `command.Processor` that unsets the environment variable.
func UnsetEnvVarProcessor(envVar string) command.Processor {
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.UnsetEnvVar(envVar))
		return nil
	}, nil)
}
