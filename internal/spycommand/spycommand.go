// Package spycommand is a package that contains types that are private in `command`, but
// needed for reference in other packages.
//
// All class methods should go in other packages to ensure they are tested properly
// For example `func (ia *InputArg) DoStuff(abc)` should be in another package
// as `func (DoInputArgStuff(ia *InputArg, abc)`.
package spycommand

import "github.com/google/go-cmp/cmp"

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

// Terminator is a custom type that is passed to panic
// when running `o.Terminate`
type Terminator struct {
	TerminationError error
}

// Terminate terminates the execution (via panic) with a `Terminator` error.
func Terminate(err error) {
	if err != nil {
		panic(TerminationErr(err))
	}
}

// IsTerminationPanic determines if the panic was caused by a termination error.
func IsTerminationPanic(recovered any) (bool, error) {
	t, ok := recovered.(*Terminator)
	if !ok {
		return false, nil
	}
	return ok && t.TerminationError != nil, t.TerminationError
}

func TerminationErr(err error) *Terminator {
	return &Terminator{err}
}

// TerminationCmpopts returns a `cmp.Option` for comparing `Terminator` objects.
func TerminationCmpopts() cmp.Option {
	return cmp.Options([]cmp.Option{
		cmp.Comparer(func(this, that *Terminator) bool {
			if this == nil || that == nil {
				return (this == nil) == (that == nil)
			}
			if this.TerminationError == nil || that.TerminationError == nil {
				return (this.TerminationError == nil) == (that.TerminationError == nil)
			}
			return this.TerminationError.Error() == that.TerminationError.Error()
		}),
	})
}
