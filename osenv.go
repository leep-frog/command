package command

import (
	"os"
	"testing"
)

var (
	// variables so it can be stubbed out in tests.
	OSLookupEnv = os.LookupEnv
	OSUnsetenv  = os.Unsetenv
	OSSetenv    = os.Setenv
)

func EnvArg(name string) Processor {
	return SimpleProcessor(
		func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			if v, ok := OSLookupEnv(name); ok {
				d.Set(name, v)
			}
			return nil
		},
		func(i *Input, d *Data) (*Completion, error) {
			if v, ok := OSLookupEnv(name); ok {
				d.Set(name, v)
			}
			return nil, nil
		},
	)
}

func StubEnv(t *testing.T, m map[string]string) {
	oldLookup := OSLookupEnv
	oldSet := OSSetenv
	oldUnset := OSUnsetenv

	OSLookupEnv = func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
	OSSetenv = func(key, value string) error {
		m[key] = value
		return nil
	}
	OSUnsetenv = func(key string) error {
		delete(m, key)
		return nil
	}

	t.Cleanup(func() {
		OSLookupEnv = oldLookup
		OSSetenv = oldSet
		OSUnsetenv = oldUnset
	})
}
