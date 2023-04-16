package command

import (
	"os"
	"testing"
)

var (
	// variables so it can be stubbed out in tests.
	OSLookupEnv = os.LookupEnv
)

// EnvArg loads the provided environment variable's value into `Data`.
// The provided `name` is also used as the `Data` key.
func EnvArg(name string) Processor {
	return SuperSimpleProcessor(func(i *Input, d *Data) error {
		if v, ok := OSLookupEnv(name); ok {
			d.Set(name, v)
		}
		return nil
	})
}

// SetEnvVarProcessor returns a `Processor` that sets the environment variable to the provided value.
func SetEnvVarProcessor(envVar, value string) Processor {
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.SetEnvVar(envVar, value))
		return nil
	}, nil)
}

// UnsetEnvVarProcessor returns a `Processor` that unsets the environment variable.
func UnsetEnvVarProcessor(envVar string) Processor {
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		ed.Executable = append(ed.Executable, d.OS.UnsetEnvVar(envVar))
		return nil
	}, nil)
}

// StubEnv uses the provided map as the OS environment.
func StubEnv(t *testing.T, m map[string]string) {
	StubValue(t, &OSLookupEnv, func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	})
}
