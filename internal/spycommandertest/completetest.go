package spycommandertest

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/spycommandtest"
)

type CompleteTestFunctionBag struct {
	AutocompleteFn func(command.Node, string, []string, *command.Data) (*command.Autocompletion, error)

	IsBranchingError     func(error) bool
	IsUsageError         func(error) bool
	IsNotEnoughArgsError func(error) bool
	IsExtraArgsError     func(error) bool
	IsValidationError    func(error) bool
}

// AutocompleteTest runs a test on command autocompletion.
func AutocompleteTest(t *testing.T, ctc *commandtest.CompleteTestCase, ictc *spycommandtest.CompleteTestCase, bag *CompleteTestFunctionBag) {
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
		data:     &command.Data{OS: ctc.OS},
	}

	testers := []commandTester{
		&RunResponseTester{ctc.RunResponses, ctc.WantRunContents, nil},
		&errorTester{
			ctc.WantErr,
			ictc.SkipErrorTypeCheck,
			bag.IsBranchingError,
			ictc.WantIsBranchingError,
			bag.IsUsageError,
			ictc.WantIsUsageError,
			bag.IsNotEnoughArgsError,
			ictc.WantIsNotEnoughArgsError,
			bag.IsExtraArgsError,
			ictc.WantIsExtraArgsError,
			bag.IsValidationError,
			ictc.WantIsValidationError,
		},
		&autocompleteTester{ctc.Want},
		&dataTester{ctc.SkipDataCheck, ctc.WantData, ctc.DataCmpOpts},
		&envTester{},
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}

	tc.autocompletion, tc.err = bag.AutocompleteFn(ctc.Node, ctc.Args, ctc.PassthroughArgs, tc.data)

	for _, tester := range testers {
		tester.check(t, tc)
	}
}
