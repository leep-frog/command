package command

import (
	"fmt"
	"os"
	"testing"
)

const (
	// GetwdKey is the `Data` key used by `GetwdProcessor` and `Getwd`.
	GetwdKey = "GETWD"
)

// Getwd retrieves the current directory from `Data` (as set by
// `GetwdProcessor`).
func Getwd(d *Data) string {
	return d.String(GetwdKey)
}

var (
	osGetwd = os.Getwd
)

// StubGetwdProcessor uses the provided string and error when calling command.GetwdProcessor.
// TODO: Change to StubGetwd because this actually stubs getwd (used by cd)
func StubGetwdProcessor(t *testing.T, wd string, err error) {
	StubValue(t, &osGetwd, func() (string, error) {
		return wd, err
	})
}

// GetwdProcessor returns a processor that stores the present directory in `Data`.
// Use the `Getwd` function to retrieve its value.
func GetwdProcessor() Processor {
	return SuperSimpleProcessor(func(i *Input, d *Data) error {
		s, err := osGetwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %v", err)
		}
		d.Set(GetwdKey, s)
		return nil
	})
}

func getwd() (string, error) {
	return os.Getwd()
}
