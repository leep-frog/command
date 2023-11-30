package spycommandertest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/command"
)

type dataTester struct {
	want *command.Data
	opts cmp.Options
}

func (*dataTester) setup(*testing.T, *testContext) {}

func (dt *dataTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if dt.want == nil {
		dt.want = &command.Data{}
	}

	if diff := cmp.Diff(dt.want, tc.data, cmpopts.EquateEmpty(), cmpopts.IgnoreUnexported(command.Data{}), cmpopts.IgnoreFields(command.Data{}, "OS"), dt.opts); diff != "" {
		t.Errorf("%s produced incorrect Data (-want, +got):\n%s", tc.prefix, diff)
	}
}
