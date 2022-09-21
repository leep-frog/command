package command

import (
	"fmt"
	"os"
	"testing"
)

var (
	// variables so it can be stubbed out in tests.
	OSLookupEnv = os.LookupEnv
	OSUnsetenv  = os.Unsetenv
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

// SetEnvVar updates the provided ExecuteData to set `envVar` to `value`.
// This can't and shouldn't be done by os.Setenv because the go CLI executable
// is run in a sub-shell.
func SetEnvVar(envVar, value string, ed *ExecuteData) {
	ed.Executable = append(ed.Executable, fmt.Sprintf("export %q=%q", envVar, value))
}

// SetEnvVarProcessor returns a `Processor` that sets the environment variable to the provided value.
func SetEnvVarProcessor(envVar, value string) Processor {
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		SetEnvVar(envVar, value, ed)
		return nil
	}, nil)
}

// StubEnv uses the provided map as the OS environment.
func StubEnv(t *testing.T, m map[string]string) {
	StubValue(t, &OSLookupEnv, func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	})
	StubValue(t, &OSUnsetenv, func(key string) error {
		delete(m, key)
		return nil
	})
}
