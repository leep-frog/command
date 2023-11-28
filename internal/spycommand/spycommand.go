// Package spycommand is a package that contains types that are private in `commondels`, but
// needed for reference in other packages.
//
// All class methods should go in other packages to ensure they are tested properly
// For example `func (ia *InputArg) DoStuff(abc)` should be in another package
// as `func (DoInputArgStuff(ia *InputArg, abc)`.
package spycommand

type InputSnapshot int

type InputArg struct {
	Value     string
	Snapshots map[InputSnapshot]bool
}

// method to get coverage info (so I don't need to check whether there are no tests
// or nothing to cover).
func noop() {
	_ = 0
}
