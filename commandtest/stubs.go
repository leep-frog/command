package commandtest

import (
	"testing"

	"github.com/leep-frog/command/internal/stubs"
)

// StubEnv stubs the environment variable used throughout this package.
func StubEnv(t *testing.T, m map[string]string) {
	stubs.StubEnv(t, m)
}

// StubGetwd uses the provided string and error when calling command.GetwdProcessor.
func StubGetwd(t *testing.T, wd string, err error) {
	stubs.StubGetwd(t, wd, err)
}
