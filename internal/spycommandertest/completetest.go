package spycommandertest

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spycommandtest"
)

// CompleteTest runs a test on command autocompletion.
func CompleteTest(t *testing.T, ctc *commandtest.CompleteTestCase, ictc *spycommandtest.CompleteTestCase, autocompleteFn func(commondels.Node, string, []string, *commondels.Data) (*commondels.Autocompletion, error)) {
	t.Helper()

	if ctc == nil {
		ctc = &commandtest.CompleteTestCase{}
	}
	if ictc == nil {
		ictc = &spycommandtest.CompleteTestCase{}
	}

	tc := &testContext{
		prefix:   fmt.Sprintf("Autocomplete(%v)", ctc.Args),
		testCase: ctc,
		data:     &commondels.Data{OS: ctc.OS},
	}

	testers := []commandTester{
		&runResponseTester{ctc.RunResponses, ctc.WantRunContents, nil},
		&errorTester{ctc.WantErr},
		&autocompleteTester{ctc.Want},
		checkIf(!ctc.SkipDataCheck, &dataTester{ctc.WantData, ctc.DataCmpOpts}),
		&envTester{},
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}

	tc.autocompletion, tc.err = autocompleteFn(ctc.Node, ctc.Args, ctc.PassthroughArgs, tc.data)

	for _, tester := range testers {
		tester.check(t, tc)
	}
}
