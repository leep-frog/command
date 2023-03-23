// Package colortest contains useful functions and logic for testing with the color package.
package colortest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color"
)

// StubTput will write to stdout with the provided format rather than actually
// running tput.
func StubTput(t *testing.T, format string) {
	command.StubValue(t, &color.TputCommand, func(output command.Output, args ...interface{}) error {
		var ss []string
		for _, a := range args {
			ss = append(ss, fmt.Sprintf("%v", a))
		}
		output.Stdoutf("__tput_%s__", strings.Join(ss, "_"))
		return nil
	})
}
