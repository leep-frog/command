package cache

import (
	"fmt"
	"os"

	"github.com/leep-frog/command"
)

const (
	// ShellOSEnvVar is an environment variable pointing to the
	// directory used for the shell-level cache.
	ShellOSEnvVar = "LEEP_CACHE_SHELL_DIR"
)

var (
	osMkdirTemp = os.MkdirTemp
)

// NewShell returns a cache specific to the current shell.
func NewShell() (*Cache, error) {
	if v, ok := command.OSLookupEnv(ShellOSEnvVar); !ok || v == "" {
		f, err := osMkdirTemp("", "leep-shell-cache")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary directory: %v", err)
		}
		if err := command.OSSetenv(ShellOSEnvVar, f); err != nil {
			return nil, fmt.Errorf("failed to set cache env var: %v", err)
		}
	}
	return New(ShellOSEnvVar)
}
