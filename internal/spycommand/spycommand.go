// Package spycommand is a package that contains types that are private in `command`, but
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

func SnapshotsMap(iss ...InputSnapshot) map[InputSnapshot]bool {
	if len(iss) == 0 {
		return nil
	}
	m := map[InputSnapshot]bool{}
	for _, is := range iss {
		m[is] = true
	}
	return m
}
