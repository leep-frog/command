// Package commandertest contains test functions for running execution and
// autocompletion tests.
package commandertest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/constants"
	"github.com/leep-frog/command/internal/spycommander"
	"github.com/leep-frog/command/internal/spycommandertest"
	"github.com/leep-frog/command/internal/spycommandtest"
)

const (
	ShortcutDesc = constants.ShortcutDesc
	CacheDesc    = constants.CacheDesc
)

// ExecuteTest runs a command exectuion test against the provided command configuration.
func ExecuteTest(t *testing.T, etc *commandtest.ExecuteTestCase) {
	t.Helper()
	spycommandertest.ExecuteTest(t, etc, &spycommandtest.ExecuteTestCase{
		SkipInputCheck:     true,
		SkipErrorTypeCheck: true,
	}, &spycommandertest.ExecuteTestFunctionBag{
		spycommander.Execute,
		spycommander.Use,
		commander.SetupArg,
		commander.SerialNodes,
		spycommander.HelpBehavior,
		commander.IsBranchingError,
		commander.IsUsageError,
		commander.IsNotEnoughArgsError,
		command.IsExtraArgsError,
		commander.IsValidationError,
	})
}

// ChangeTest tests if a command object has changed properly. If `want != nil`,
// then `original.Changed()` should return `true` and `original` should equal `want`.
// If `want == nil`, then `original.Changed()` should return `false`.
func ChangeTest[T commandtest.Changeable](t *testing.T, want, got T, opts ...cmp.Option) {
	t.Helper()
	spycommandertest.ChangeTest[T](t, want, got, opts...)
}

// AutocompleteTest runs a command completion test against the provided command configuration.
func AutocompleteTest(t *testing.T, ctc *commandtest.CompleteTestCase) {
	t.Helper()
	spycommandertest.AutocompleteTest(t, ctc, &spycommandtest.CompleteTestCase{
		SkipErrorTypeCheck: true,
	}, &spycommandertest.CompleteTestFunctionBag{
		spycommander.Autocomplete,
		commander.IsBranchingError,
		commander.IsUsageError,
		commander.IsNotEnoughArgsError,
		command.IsExtraArgsError,
		commander.IsValidationError,
	})
}
