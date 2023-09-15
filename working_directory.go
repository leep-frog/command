package command

import (
	"fmt"
	"os"
	"testing"
)

const (
	// GetwdKey is the `Data` key used by `Getwd`.
	GetwdKey = "GETWD"
)

var (
	osGetwd = os.Getwd

	// Getwd is a `GetProcessor` that retrieves the current working directory.
	Getwd = &GetProcessor[string]{
		SuperSimpleProcessor(func(i *Input, d *Data) error {
			s, err := osGetwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}
			d.Set(GetwdKey, s)
			return nil
		}),
		func(d *Data) string {
			return d.String(GetwdKey)
		},
	}
)

// StubGetwd uses the provided string and error when calling command.GetwdProcessor.
func StubGetwd(t *testing.T, wd string, err error) {
	StubValue(t, &osGetwd, func() (string, error) {
		return wd, err
	})
}
