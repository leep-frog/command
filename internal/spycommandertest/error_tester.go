package spycommandertest

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/internal/testutil"
)

type errorTester struct {
	want               error
	skipErrorTypeCheck bool

	isBranchingError         func(error) bool
	wantIsBranchingError     bool
	isUsageError             func(error) bool
	wantIsUsageError         bool
	isNotEnoughArgsError     func(error) bool
	wantIsNotEnoughArgsError bool
	isExtraArgsError         func(error) bool
	wantIsExtraArgsError     bool
	isValidationError        func(error) bool
	wantIsValidationError    bool
}

func (*errorTester) setup(*testing.T, *testContext) {}
func (et *errorTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	testutil.CmpError(t, tc.prefix, et.want, tc.err)

	if !et.skipErrorTypeCheck {
		testutil.Cmp(t, fmt.Sprintf("IsBranchingError(%s) returned incorrect value", tc.prefix), et.wantIsBranchingError, et.isBranchingError(tc.err))
		testutil.Cmp(t, fmt.Sprintf("IsUsageError(%s) returned incorrect value", tc.prefix), et.wantIsUsageError, et.isUsageError(tc.err))
		testutil.Cmp(t, fmt.Sprintf("IsNotEnoughArgsError(%s) returned incorrect value", tc.prefix), et.wantIsNotEnoughArgsError, et.isNotEnoughArgsError(tc.err))
		testutil.Cmp(t, fmt.Sprintf("IsExtraArgsError(%s) returned incorrect value", tc.prefix), et.wantIsExtraArgsError, et.isExtraArgsError(tc.err))
		testutil.Cmp(t, fmt.Sprintf("IsValidationError(%s) returned incorrect value", tc.prefix), et.wantIsValidationError, et.isValidationError(tc.err))
	} else if et.wantIsBranchingError || et.wantIsUsageError || et.wantIsNotEnoughArgsError || et.wantIsExtraArgsError || et.wantIsValidationError {
		t.Fatalf("At least one errorTester.want*Error field was true, but error type checks are disabled")
	}
}
