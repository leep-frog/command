package command

import "testing"

func TestHasArg(t *testing.T) {
	d := &Data{}
	d.Set("yes", "hello")

	if !d.Has("yes") {
		t.Errorf("data.HasArg('yes') returned false; want true")
	}

	if d.Has("no") {
		t.Errorf("data.HasArg('no') returned true; want false")
	}
}
