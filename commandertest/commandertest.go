// Package commandertest contains test functions for running execution and
// autocompletion tests.
package commandertest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
		TestInput: false,
	}, spycommander.Execute, spycommander.Use, commander.SetupArg, commander.SerialNodes)
}

// ChangeTest checks if the provided
func ChangeTest[T commandtest.Changeable](t *testing.T, want, got T, opts ...cmp.Option) {
	t.Helper()
	spycommandertest.ChangeTest[T](t, want, got, opts...)
}

// AutocompleteTest runs a command completion test against the provided command configuration.
func AutocompleteTest(t *testing.T, ctc *commandtest.CompleteTestCase) {
	t.Helper()
	spycommandertest.AutocompleteTest(t, ctc, &spycommandtest.CompleteTestCase{}, spycommander.Autocomplete)
}