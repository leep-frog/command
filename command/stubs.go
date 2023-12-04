package command

import "os"

var (
	// OSLookupEnv is the env lookup command used internally by the entire `command` project.
	// It's value can be stubbed in tests by using the `commandtest.*TestCase.Env` fields.
	OSLookupEnv = os.LookupEnv
)
