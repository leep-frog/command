package spycommandertest

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/spycommandtest"
)

type testContext struct {
	prefix   string
	testCase testCase

	data  *command.Data
	fo    *commandtest.Output
	input *command.Input

	err   error
	panic interface{}

	eData          *command.ExecuteData
	autocompletion *command.Autocompletion
}

func setupForTest(t *testing.T, contents []string) string {
	t.Helper()

	f, err := os.CreateTemp("", "command_test_setup")
	if err != nil {
		t.Fatalf(`os.CreateTemp("", "command_test_setup") returned error: %v`, err)
	}
	t.Cleanup(func() { f.Close() })
	for _, s := range contents {
		fmt.Fprintln(f, s)
	}
	return f.Name()
}

type commandTester interface {
	setup(*testing.T, *testContext)
	check(*testing.T, *testContext)
}

type testCase interface {
	GetEnv() map[string]string
}

type executeFn func(command.Node, *command.Input, command.Output, *command.Data, *command.ExecuteData) error

type usageFn func(command.Node, *command.Input) (*command.Usage, error)

type nameProcessor interface {
	command.Processor
	Name() string
}

type ExecuteTestFunctionBag struct {
	ExFn         executeFn
	UFn          usageFn
	SetupArg     nameProcessor
	SerialNodes  func(...command.Processor) command.Node
	HelpBehavior func(command.Node, *command.Input, command.Output, func(error) bool) error

	IsBranchingError     func(error) bool
	IsUsageError         func(error) bool
	IsNotEnoughArgsError func(error) bool
	IsExtraArgsError     func(error) bool
	IsValidationError    func(error) bool
}

// ExecuteTest runs a command execution test.
func ExecuteTest(t *testing.T, etc *commandtest.ExecuteTestCase, ietc *spycommandtest.ExecuteTestCase, bag *ExecuteTestFunctionBag) {
	t.Helper()

	if etc == nil {
		etc = &commandtest.ExecuteTestCase{}
	}

	if etc.WantData == nil && !etc.SkipDataCheck {
		etc.WantData = &command.Data{}
	}

	if ietc == nil {
		ietc = &spycommandtest.ExecuteTestCase{}
	}

	tc := &testContext{
		prefix:   fmt.Sprintf("Execute(%v)", etc.Args),
		testCase: etc,
		data:     &command.Data{OS: etc.OS},
		fo:       commandtest.NewOutput(),
	}
	t.Cleanup(tc.fo.Close)
	args := etc.Args
	if etc.RequiresSetup {
		setupFile := setupForTest(t, etc.SetupContents)
		args = append([]string{setupFile}, args...)
		etc.WantData.Set(bag.SetupArg.Name(), setupFile)
		t.Cleanup(func() { os.Remove(setupFile) })
	}
	var helpFlag bool
	for i, a := range args {
		if a == "--help" {
			args = append(args[:i], args[i+1:]...)
			helpFlag = true
			break
		}
	}
	tc.input = command.NewInput(args, nil)

	testers := []commandTester{
		&outputTester{etc.WantStdout, etc.WantStderr},
		&errorTester{
			etc.WantErr,
			ietc.SkipErrorTypeCheck,
			bag.IsBranchingError,
			ietc.WantIsBranchingError,
			bag.IsUsageError,
			ietc.WantIsUsageError,
			bag.IsNotEnoughArgsError,
			ietc.WantIsNotEnoughArgsError,
			bag.IsExtraArgsError,
			ietc.WantIsExtraArgsError,
			bag.IsValidationError,
			ietc.WantIsValidationError,
		},
		&executeDataTester{etc.WantExecuteData},
		&RunResponseTester{etc.RunResponses, etc.WantRunContents, nil},
		&dataTester{etc.SkipDataCheck, etc.WantData, etc.DataCmpOpts},
		&inputTester{ietc.SkipInputCheck, ietc.WantInput},
		&envTester{},
		&panicTester{etc.WantPanic},
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}

	n := etc.Node
	if etc.RequiresSetup {
		n = bag.SerialNodes(bag.SetupArg, n)
	}

	func() {
		defer func() {
			tc.panic = recover()
		}()
		if helpFlag {
			tc.err = bag.HelpBehavior(n, tc.input, tc.fo, bag.IsUsageError)
		} else {
			tc.eData = &command.ExecuteData{}
			tc.err = bag.ExFn(n, tc.input, tc.fo, tc.data, tc.eData)
		}
	}()

	for _, tester := range testers {
		tester.check(t, tc)
	}
}

func ChangeTest[T commandtest.Changeable](t *testing.T, want, original T, opts ...cmp.Option) {
	wantChanged := reflect.ValueOf(want).IsValid() && !reflect.ValueOf(want).IsNil()
	if original.Changed() != wantChanged {
		if wantChanged {
			t.Errorf("object didn't change when it should have")
		} else {
			t.Errorf("object changed when it shouldn't have")
		}
	}

	if wantChanged {
		if diff := cmp.Diff(want, original, opts...); diff != "" {
			t.Errorf("object changed incorrectly (-want, +got):\n%s", diff)
		}
	}
}
