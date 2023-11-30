package spyinput

import "github.com/leep-frog/command/internal/spycommand"

type SpyInput[IB any] struct {
	Args          []*spycommand.InputArg
	Remaining     []int
	Delimiter     *rune
	Offset        int
	SnapshotCount spycommand.InputSnapshot
	// breakers are a set of `InputBreakers` that are required to pass for all `Pop` functions.
	Breakers []IB
}
