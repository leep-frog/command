package commandtest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type outputTester struct {
	wantStdout string
	wantStderr string
}

func (*outputTester) setup(*testing.T, *testContext) {}

func (ot *outputTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if diff := cmp.Diff(ot.wantStdout, tc.fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s sent wrong data to stdout (-want, +got):\n%s", tc.prefix, diff)
	}
	if diff := cmp.Diff(ot.wantStderr, tc.fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s sent wrong data to stderr (-want, +got):\n%s", tc.prefix, diff)
	}
}
