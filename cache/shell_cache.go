package cache

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/stubs"
	"github.com/leep-frog/command/internal/testutil"
)

const (
	// ShellOSEnvVar is an environment variable pointing to the
	// directory used for the shell-level cache.
	ShellOSEnvVar = "LEEP_CACHE_SHELL_DIR"
	// ShellDataKey is the data key used to store the shell-level cache.
	// Callers should use the `ShellProcessor` and `ShellFromData` functions
	// rather than using this key.
	ShellDataKey = "LEEP_CACHE_SHELL"
)

var (
	getShellCache = func(d *commondels.Data, ed *commondels.ExecuteData) error {
		v, ok := stubs.OSLookupEnv(ShellOSEnvVar)
		if !ok || v == "" {
			var err error
			v, err = osMkdirTemp("", "leep-shell-cache")
			if err != nil {
				return fmt.Errorf("failed to create temporary directory: %v", err)
			}
			ed.Executable = append(ed.Executable, d.OS.SetEnvVar(ShellOSEnvVar, v))
		}
		c, err := FromDir(v)
		if err != nil {
			return fmt.Errorf("failed to create shell-level cache: %v", err)
		}
		d.Set(ShellDataKey, c)
		return nil
	}
)

// ShellProcessor returns a processor that creates a shell-level `Cache`.
// This needs to be done at the processor level so we can update an environment
// variable via `ExecuteData`.
func ShellProcessor() commondels.Processor {
	return commander.SimpleProcessor(func(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
		return o.Err(getShellCache(d, ed))
	}, func(i *commondels.Input, d *commondels.Data) (*commondels.Completion, error) {
		return nil, getShellCache(d, &commondels.ExecuteData{})
	})
}

// ShellFromData retrieves the shell-level `Cache` that was set by `ShellProcessor`.
func ShellFromData(d *commondels.Data) *Cache {
	i := d.Get(ShellDataKey)
	return i.(*Cache)
}

// StubShellCache stubs the cache created and set by `ShellProcessor`.
func StubShellCache(t *testing.T, c *Cache) {
	testutil.StubValue(t, &getShellCache, func(d *commondels.Data, ed *commondels.ExecuteData) error {
		d.Set(ShellDataKey, c)
		return nil
	})
}
