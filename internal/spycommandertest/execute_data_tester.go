package spycommandertest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/command"
)

type executeDataTester struct {
	want *command.ExecuteData
}

func (*executeDataTester) setup(*testing.T, *testContext) {}
func (et *executeDataTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	if et.want == nil {
		et.want = &command.ExecuteData{}
	}
	if tc.eData == nil {
		tc.eData = &command.ExecuteData{}
	}
	if diff := cmp.Diff(et.want, tc.eData, cmpopts.IgnoreFields(command.ExecuteData{}, "Executor")); diff != "" {
		t.Errorf("%s returned unexpected ExecuteData (-want, +got):\n%s", tc.prefix, diff)
	}
}
