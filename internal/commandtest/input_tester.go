package commandtest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spycommand"
)

type inputTester struct {
	want *commondels.Input
}

func (*inputTester) setup(*testing.T, *testContext) {}
func (it *inputTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if it.want == nil {
		it.want = &commondels.Input{}
	}
	if diff := cmp.Diff(it.want, tc.input, cmpopts.EquateEmpty(), cmp.AllowUnexported(commondels.Input{}, spycommand.InputArg{})); diff != "" {
		t.Errorf("%s incorrectly modified input (-want, +got):\n%s", tc.prefix, diff)
	}
}
