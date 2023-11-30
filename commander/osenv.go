package commander

import (
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/stubs"
)

// EnvArg loads the provided environment variable's value into `commondels.Data`.
// The provided `name` is also used as the `commondels.Data` key.
func EnvArg(name string) *GetProcessor[string] {
	return &GetProcessor[string]{
		SuperSimpleProcessor(func(i *commondels.Input, d *commondels.Data) error {
			if v, ok := stubs.OSLookupEnv(name); ok {
				d.Set(name, v)
			}
			return nil
		}),
		name,
	}
}

// SetEnvVarProcessor returns a `commondels.Processor` that sets the environment variable to the provided value.
func SetEnvVarProcessor(envVar, value string) commondels.Processor {
	return SimpleProcessor(func(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.SetEnvVar(envVar, value))
		return nil
	}, nil)
}

// UnsetEnvVarProcessor returns a `commondels.Processor` that unsets the environment variable.
func UnsetEnvVarProcessor(envVar string) commondels.Processor {
	return SimpleProcessor(func(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.UnsetEnvVar(envVar))
		return nil
	}, nil)
}
