package commander

import (
	"fmt"

	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/stubs"
)

const (
	// GetwdKey is the `commondels.Data` key used by `Getwd`.
	GetwdKey = "GETWD"
)

var (
	// Getwd is a `GetProcessor` that retrieves the current working directory.
	Getwd = &GetProcessor[string]{
		SuperSimpleProcessor(func(i *commondels.Input, d *commondels.Data) error {
			s, err := stubs.OSGetwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}
			d.Set(GetwdKey, s)
			return nil
		}),
		GetwdKey,
	}
)
