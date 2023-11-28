package spycommandertest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type panicTester struct {
	want interface{}
}

func (*panicTester) setup(*testing.T, *testContext) {}
func (pt *panicTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	if diff := cmp.Diff(pt.want, tc.panic); diff != "" {
		t.Errorf("%s panicked with unexpected value (-want, +got):\n%s", tc.prefix, diff)
	}
}
