package commandtest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/commondels"
)

type dataTester struct {
	want *commondels.Data
	opts cmp.Options
}

func (*dataTester) setup(*testing.T, *testContext) {}

func (dt *dataTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if dt.want == nil {
		dt.want = &commondels.Data{}
	}

	if diff := cmp.Diff(dt.want, tc.data, cmpopts.EquateEmpty(), cmpopts.IgnoreUnexported(commondels.Data{}), cmpopts.IgnoreFields(commondels.Data{}, "OS"), dt.opts); diff != "" {
		t.Errorf("%s produced incorrect Data (-want, +got):\n%s", tc.prefix, diff)
	}
}
