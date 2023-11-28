// Package spycommandtest contains additional test parameters that
// are only relevant for tests run inside of the `command` project.
package spycommandtest

import "github.com/leep-frog/command/commondels"

type ExecuteTestCase struct {
	// Whether or not to test actual input against wantInput.
	TestInput bool
	WantInput *commondels.Input
}

type CompleteTestCase struct{}
