// Package spycommandtest contains additional test parameters that
// are only relevant for tests run inside of the `command` project.
package spycommandtest

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spyinput"
)

type SpyInput spyinput.SpyInput[command.InputBreaker]

type ExecuteTestCase struct {
	// Whether or not to test actual input against wantInput.
	SkipInputCheck bool
	WantInput      *SpyInput

	SkipErrorTypeCheck       bool
	WantIsBranchingError     bool
	WantIsUsageError         bool
	WantIsNotEnoughArgsError bool
	WantIsExtraArgsError     bool
	WantIsValidationError    bool
}

type CompleteTestCase struct {
	SkipErrorTypeCheck       bool
	WantIsBranchingError     bool
	WantIsUsageError         bool
	WantIsNotEnoughArgsError bool
	WantIsExtraArgsError     bool
	WantIsValidationError    bool
}

func convertSpyInput(si SpyInput) spyinput.SpyInput[command.InputBreaker] {
	return spyinput.SpyInput[command.InputBreaker](si)
}
