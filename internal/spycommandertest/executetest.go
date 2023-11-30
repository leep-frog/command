package spycommandertest

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spycommandtest"
)

type testContext struct {
	prefix   string
	testCase testCase

	data  *commondels.Data
	fo    *commondels.FakeOutput
	input *commondels.Input

	err   error
	panic interface{}

	eData          *commondels.ExecuteData
	autocompletion *commondels.Autocompletion
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

type noOpTester struct{}

func (*noOpTester) setup(*testing.T, *testContext) {}

func (*noOpTester) check(t *testing.T, tc *testContext) {}

func checkIf(cond bool, ct commandTester) commandTester {
	if cond {
		return ct
	}
	return &noOpTester{}
}

type executeFn func(commondels.Node, *commondels.Input, commondels.Output, *commondels.Data, *commondels.ExecuteData) error

type usageFn func(commondels.Node, *commondels.Input) (*commondels.Usage, error)

type nameProcessor interface {
	commondels.Processor
	Name() string
}

// ExecuteTest runs a command execution test.
func ExecuteTest(t *testing.T, etc *commandtest.ExecuteTestCase, ietc *spycommandtest.ExecuteTestCase, exFn executeFn, uFn usageFn, setupArg nameProcessor, serialNodes func(...commondels.Processor) commondels.Node) {
	t.Helper()

	if etc == nil {
		etc = &commandtest.ExecuteTestCase{}
	}

	if etc.WantData == nil {
		etc.WantData = &commondels.Data{}
	}

	if ietc == nil {
		ietc = &spycommandtest.ExecuteTestCase{
			// TODO: Change TestInput to SkipInputCheck (similar to SkipDataCheck)
			// default to testing input
			TestInput: true,
		}
	}

	tc := &testContext{
		prefix:   fmt.Sprintf("Execute(%v)", etc.Args),
		testCase: etc,
		data:     &commondels.Data{OS: etc.OS},
		fo:       commondels.NewFakeOutput(),
	}
	t.Cleanup(tc.fo.Close)
	args := etc.Args
	if etc.RequiresSetup {
		setupFile := setupForTest(t, etc.SetupContents)
		args = append([]string{setupFile}, args...)
		etc.WantData.Set(setupArg.Name(), setupFile)
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
	tc.input = commondels.NewInput(args, nil)

	testers := []commandTester{
		&outputTester{etc.WantStdout, etc.WantStderr},
		&errorTester{etc.WantErr},
		&executeDataTester{etc.WantExecuteData},
		&runResponseTester{etc.RunResponses, etc.WantRunContents, nil},
		checkIf(!etc.SkipDataCheck, &dataTester{etc.WantData, etc.DataCmpOpts}),
		checkIf(ietc.TestInput, &inputTester{ietc.WantInput}),
		&envTester{},
		&panicTester{etc.WantPanic},
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}

	n := etc.Node
	if etc.RequiresSetup {
		n = serialNodes(setupArg, n)
	}

	if helpFlag {
		// TODO: This is synced with usageExecutorHelper in sourcerer (use interface to share logic? ie move this check into execute function?)
		var u *commondels.Usage
		u, tc.err = uFn(n, tc.input)
		if tc.err != nil {
			tc.fo.Err(tc.err)
		} else {
			tc.fo.Stdoutln(u.String())
		}

	} else {
		func() {
			defer func() {
				tc.panic = recover()
			}()
			tc.eData = &commondels.ExecuteData{}
			tc.err = exFn(n, tc.input, tc.fo, tc.data, tc.eData)
		}()
	}

	for _, tester := range testers {
		tester.check(t, tc)
	}
}

// ChangeTest tests if a command object has changed properly.
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
