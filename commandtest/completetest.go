package commandtest

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/commondels"
)

// CompleteTest runs a test on command autocompletion.
func CompleteTest(t *testing.T, ctc *CompleteTestCase) {
	t.Helper()

	if ctc == nil {
		ctc = &CompleteTestCase{}
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

	tc.autocompletion, tc.err = autocomplete(ctc.Node, ctc.Args, ctc.PassthroughArgs, tc.data)

	for _, tester := range testers {
		tester.check(t, tc)
	}
}
