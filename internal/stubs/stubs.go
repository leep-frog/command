package stubs

import (
	"os"
	"os/exec"
	"testing"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/testutil"
)

var (
	// OSGetwd is a stub for os.Getwd
	OSGetwd = os.Getwd

	// Run is a wrapper `exec.Cmd` used for stubbing purposes.
	Run = func(cmd *exec.Cmd) error {
		return cmd.Run()
	}
)

// StubEnv stubs the environment variable used throughout this package.
func StubEnv(t *testing.T, m map[string]string) {
	testutil.StubValue(t, &command.OSLookupEnv, func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	})
}

// StubGetwd uses the provided string and error when calling command.GetwdProcessor.
func StubGetwd(t *testing.T, wd string, err error) {
	testutil.StubValue(t, &OSGetwd, func() (string, error) {
		return wd, err
	})
}
