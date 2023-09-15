package command

import (
	"fmt"
	"runtime"
	"testing"
)

const (
	// RuntimeCallerKey is the `Data` key used by `RuntimeCaller`.
	RuntimeCallerKey = "GETWD"
)

var (
	runtimeCaller = runtime.Caller
)

// RuntimeCaller is a `GetProcessor` that retrieves the filepath of the file that
func RuntimeCaller() *GetProcessor[string] {
	f := func() (string, bool) {
		_, filename, _, ok := runtimeCaller(1)
		return filename, ok
	}

	return &GetProcessor[string]{
		SuperSimpleProcessor(func(i *Input, d *Data) error {
			filename, ok := f()
			if !ok {
				return fmt.Errorf("runtime.Caller failed to retrieve filepath info")
			}
			d.Set(GetwdKey, filename)
			return nil
		}),
		func(d *Data) string {
			return d.String(GetwdKey)
		},
	}
}

func StubRuntimeCaller(t *testing.T, s string, ok bool) {
	StubValue(t, &runtimeCaller, func(int) (uintptr, string, int, bool) {
		return 0, s, 0, ok
	})
}
