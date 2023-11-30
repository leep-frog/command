// Package spycommandtest contains additional test parameters that
// are only relevant for tests run inside of the `command` project.
package spycommandtest

import (
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spyinput"
)

type SpyInput spyinput.SpyInput[commondels.InputBreaker]

type ExecuteTestCase struct {
	// Whether or not to test actual input against wantInput.
	TestInput bool
	WantInput *SpyInput
}

type CompleteTestCase struct{}

func convertSpyInput(si SpyInput) spyinput.SpyInput[commondels.InputBreaker] {
	return spyinput.SpyInput[commondels.InputBreaker](si)
}
