package spycommandertest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
)

type autocompleteTester struct {
	want *command.Autocompletion
}

func (*autocompleteTester) setup(*testing.T, *testContext) {}
func (at *autocompleteTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	if at.want == nil {
		at.want = &command.Autocompletion{}
	}
	if tc.autocompletion == nil {
		tc.autocompletion = &command.Autocompletion{}
	}

	if diff := cmp.Diff(at.want, tc.autocompletion); diff != "" {
		t.Errorf("%s produced incorrect completions (-want, +got):\n%s", tc.prefix, diff)
	}
}
