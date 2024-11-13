package stubs

import (
	"io"
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

	// StubStdinPipe stubs the cmd.StdinPipe method (used for testing purposes)
	StubStdinPipe = func(cmd *exec.Cmd) (io.WriteCloser, error) {
		return cmd.StdinPipe()
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

// StubRun stubs the cmd.Run() method with the provided function.
func StubRun(t *testing.T, f func(cmd *exec.Cmd) error) {
	testutil.StubValue(t, &Run, f)
}
