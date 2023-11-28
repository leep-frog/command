package stubs

import (
	"os"
	"os/exec"
	"testing"

	"github.com/leep-frog/command/internal/testutil"
)

var (
	// OSLookupEnv is the env lookup to use that will be stubbed by `commandtest.*TestCase` objects.
	OSLookupEnv = os.LookupEnv

	// Run is a wrapper `exec.Cmd` used for stubbing purposes.
	Run = func(cmd *exec.Cmd) error {
		return cmd.Run()
	}
)

// StubEnv stubs the environment variable used throughout this package.
func StubEnv(t *testing.T, m map[string]string) {
	testutil.StubValue(t, &OSLookupEnv, func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	})
}
