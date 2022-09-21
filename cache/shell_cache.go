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
	// ShellDataKey is the data key used to store the shell-level cache.
	// Callers should use the `ShellProcessor` and `ShellFromData` functions
	// rather than using this key.
	ShellDataKey = "LEEP_CACHE_SHELL"
)

var (
	osMkdirTemp = os.MkdirTemp
)

func getShellCache(d *command.Data, ed *command.ExecuteData) error {
	v, ok := command.OSLookupEnv(ShellOSEnvVar)
	fmt.Println(v, ok)
	if !ok || v == "" {
		var err error
		v, err = osMkdirTemp("", "leep-shell-cache")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}
		// We can't us os.Setenv because the go executable runs in
		// a separate shell.
		ed.Executable = append(ed.Executable,
			fmt.Sprintf("export %s=%q", ShellOSEnvVar, v),
		)
	}
	c, err := ForDir(v)
	if err != nil {
		return fmt.Errorf("failed to create shell-level cache: %v", err)
	}
	d.Set(ShellDataKey, c)
	return nil
}

// ShellProcessor returns a processor that creates a shell-level `Cache`.
// This needs to be done at the processor level so we can update an environment
// variable via `ExecuteData`.
func ShellProcessor() command.Processor {
	return command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		return o.Err(getShellCache(d, ed))
	}, func(i *command.Input, d *command.Data) (*command.Completion, error) {
		return nil, getShellCache(d, &command.ExecuteData{})
	})
}

// ShellFromData retrieves the shell-level `Cache` that was set by `ShellProcessor`.
func ShellFromData(d *command.Data) *Cache {
	i := d.Get(ShellDataKey)
	return i.(*Cache)
}
