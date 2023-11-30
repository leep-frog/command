package command

import (
	"testing"

	"github.com/leep-frog/command/internal/testutil"
)

func TestClone(t *testing.T) {
	c := &Completion{
		[]string{"un", "deux"},
		true,
		true,
		true,
		true,
		true,
		true,
		&DeferredCompletion{},
	}

	d := c.Clone()
	testutil.Cmp(t, "Completion.Clone() returned incorrect value", c, d)

	c.Distinct = false
	if !d.Distinct {
		t.Fatalf("Completion.Clone() resulted in objects that point to same Distinct value")
	}
	d.DontComplete = false
	if !c.DontComplete {
		t.Fatalf("Completion.Clone() resulted in objects that point to same DontComplete value")
	}
}
