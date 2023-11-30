// Package commandertest contains test functions for running execution and
// autocompletion tests.
package commandertest

import (
	"testing"

	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/spycommander"
	"github.com/leep-frog/command/internal/spycommandertest"
)

// ExecuteTest runs a command exectuion test against the provided command configuration.
func ExecuteTest(t *testing.T, etc *commandtest.ExecuteTestCase) {
	spycommandertest.ExecuteTest(t, etc, nil, spycommander.Execute, spycommander.Use)
}

// TODO: Rename to AutocompleteTest
// CompleteTest runs a command completion test against the provided command configuration.
func CompleteTest(t *testing.T, ctc *commandtest.CompleteTestCase) {
	spycommandertest.CompleteTest(t, ctc, nil, spycommander.Autocomplete)
}
