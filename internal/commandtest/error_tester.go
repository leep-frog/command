package commandtest

import (
	"testing"

	"github.com/leep-frog/command/internal/testutil"
)

type errorTester struct {
	want error
}

func (*errorTester) setup(*testing.T, *testContext) {}
func (et *errorTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	testutil.CmpError(t, tc.prefix, et.want, tc.err)
}
