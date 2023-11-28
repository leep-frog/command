package spycommand

import "testing"

func TestInputTypes(t *testing.T) {
	_ = &InputArg{
		Value: "This test was added to get coverage info for this package",
		Snapshots: map[InputSnapshot]bool{
			1: true,
			2: false,
			4: true,
		},
	}

	noop()
}
