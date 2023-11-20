package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// VerifyPanic verifies that function `f` panics with argument `want`.
func VerifyPanic(t *testing.T, prefix string, want interface{}, f func()) {
	t.Helper()

	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("%s did not panic", prefix)
		} else if diff := cmp.Diff(want, r); diff != "" {
			t.Errorf("%s panicked with incorrect value (-want, +got):\n%s", prefix, diff)
		}
	}()

	f()
}

func TempFile(t *testing.T, pattern string) *os.File {
	tmp, err := ioutil.TempFile("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { tmp.Close() })
	return tmp
}

func StubValue[T any](t *testing.T, originalValue *T, newValue T) {
	oldValue := *originalValue
	*originalValue = newValue
	t.Cleanup(func() {
		*originalValue = oldValue
	})
}

const (
	ShortcutDesc             = "  *: Start of new shortcut-able section"
	CacheDesc                = "  ^: Start of new cachable section"
	BranchDescWithDefault    = "  ┳: Start of subcommand branches (with default node)"
	BranchDescWithoutDefault = "  ┓: Start of subcommand branches (without default node)"
)

// UsageTestCase is a test case object for testing command usage.
type UsageTestCase struct {
	// Node is the root `Node` of the command to test.
	Node Node
	// Args is the list of arguments provided to the command.
	Args []string
	// WantString is the expected usage output.
	WantString []string
	// WantErr is the error that should be returned.
	WantErr error
}

// ExecuteTestCase is a test case object for testing command execution.
type ExecuteTestCase struct {
	// Node is the root `Node` of the command to test.
	Node Node
	// Args is the list of arguments provided to the command.
	Args []string
	// Env is the map of os environment variables to stub. If nil, this is not stubbed.
	Env map[string]string
	// OS is the OS to use for the test.
	OS OS

	// WantData is the `Data` object that should be constructed.
	WantData *Data
	// SkipDataCheck skips the check on `WantData`.
	SkipDataCheck bool
	// DataCmpOpts is the set of cmp.Options that should be used
	// when comparing data.
	DataCmpOpts cmp.Options
	// WantExecuteData is the `ExecuteData` object that should be constructed.
	WantExecuteData *ExecuteData
	// WantStdout is the data that should be sent to stdout.
	WantStdout string
	// WantStderr is the data that should be sent to stderr.
	WantStderr string
	// WantErr is the error that should be returned.
	WantErr error

	// Whether or not to test actual input against wantInput.
	testInput bool
	wantInput *Input

	// WantRunContents are the set of shell commands that should have been run.
	WantRunContents []*RunContents

	// RequiresSetup indicates whether or not the command requires setup
	RequiresSetup bool
	// SetupContents is the contents of the setup file provided to the command.
	SetupContents []string

	// RunResponses are the stubbed responses to return from exec.Cmd.Run.
	RunResponses []*FakeRun
}

func (etc *ExecuteTestCase) getEnv() map[string]string {
	if etc.Env == nil {
		etc.Env = map[string]string{}
	}
	return etc.Env
}

// FakeRun is a fake shell command run.
type FakeRun struct {
	Stdout []string
	Stderr []string
	Err    error
	F      func(t *testing.T)
}

func setupForTest(t *testing.T, contents []string) string {
	t.Helper()

	f, err := ioutil.TempFile("", "command_test_setup")
	if err != nil {
		t.Fatalf(`ioutil.TempFile("", "command_test_setup") returned error: %v`, err)
	}
	t.Cleanup(func() { f.Close() })
	for _, s := range contents {
		fmt.Fprintln(f, s)
	}
	return f.Name()
}

// UsageTest runs a test on command usage.
func UsageTest(t *testing.T, utc *UsageTestCase) {
	// TODO: Remove UsageTest in favor of `ExecuteTest` with `--help` flag set.
	t.Helper()

	if utc == nil {
		utc = &UsageTestCase{}
	}

	got, err := Use(utc.Node, ParseExecuteArgs(utc.Args))
	CmpError(t, fmt.Sprintf("Use(%v)", utc.Args), utc.WantErr, err)

	if err == nil {
		if diff := cmp.Diff(strings.Join(utc.WantString, "\n"), got.String()); diff != "" {
			t.Errorf("Use(%v) returned incorrect response (-want, +got):\n%s", utc.Args, diff)
		}
	}
}

// PrependSetupArg prepends the SetupArg node to the given node.
func PreprendSetupArg(n Node) Node {
	return SerialNodes(SetupArg, n)
}

// ExecuteTest runs a command execution test.
func ExecuteTest(t *testing.T, etc *ExecuteTestCase) {
	t.Helper()

	if etc == nil {
		etc = &ExecuteTestCase{}
	}

	if etc.WantData == nil {
		etc.WantData = &Data{}
	}

	tc := &testContext{
		prefix:   fmt.Sprintf("Execute(%v)", etc.Args),
		testCase: etc,
		data:     &Data{OS: etc.OS},
		fo:       NewFakeOutput(),
	}
	t.Cleanup(tc.fo.Close)
	args := etc.Args
	if etc.RequiresSetup {
		setupFile := setupForTest(t, etc.SetupContents)
		args = append([]string{setupFile}, args...)
		etc.WantData.Set(SetupArg.Name(), setupFile)
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
	tc.input = NewInput(args, nil)

	testers := []commandTester{
		&outputTester{etc.WantStdout, etc.WantStderr},
		&errorTester{etc.WantErr},
		&executeDataTester{etc.WantExecuteData},
		&runResponseTester{etc.RunResponses, etc.WantRunContents, nil},
		checkIf(!etc.SkipDataCheck, &dataTester{etc.WantData, etc.DataCmpOpts}),
		checkIf(etc.testInput, &inputTester{etc.wantInput}),
		&envTester{},
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}

	n := etc.Node
	if etc.RequiresSetup {
		n = PreprendSetupArg(n)
	}

	if helpFlag {
		// This is synced with usageExecutorHelper in sourcerer (use interface to share logic?)
		var u *Usage
		u, tc.err = Use(n, tc.input)
		if tc.err != nil {
			tc.fo.Err(tc.err)
		} else {
			tc.fo.Stdoutln(u.String())
		}

	} else {
		tc.eData, tc.err = execute(n, tc.input, tc.fo, tc.data)
	}

	for _, tester := range testers {
		tester.check(t, tc)
	}
}

func write(t *testing.T, iow io.Writer, contents []string) {
	for _, c := range contents {
		if _, err := bytes.NewBufferString(fmt.Sprintf("%s\n", c)).WriteTo(iow); err != nil {
			t.Fatalf("failed to write buffer to io.Writer: %v", err)
		}
	}
}

// Changeable is an interface for commands that can be changed.
// Note: this is really just using a function from the `sourcerer.CLI` interface.
// TODO: Move this and ChangeTest to sourcerer package?
type Changeable interface {
	// Changed returns whether or not the undelrying command object has changed.
	Changed() bool
}

// ChangeTest tests if a command object has changed properly.
func ChangeTest[T Changeable](t *testing.T, want, original T, opts ...cmp.Option) {
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

// CompleteTestCase is a test case object for testing command autocompletion.
type CompleteTestCase struct {
	// Node is the root `Node` of the command to test.
	Node Node
	// Args is the list of arguments provided to the command.
	// Remember that args requires a dummy command argument (e.g. "cmd ")
	// since `COMP_LINE` includes that.
	Args string
	// PassthroughArgs are the passthrough args provided to the command autocompletion.
	PassthroughArgs []string
	// Env is the map of os environment variables to stub. If nil, this is not stubbed.
	Env map[string]string
	// OS is the OS to use for the test.
	OS OS

	// Want is the expected `Autocompletion` object produced by the test.
	Want *Autocompletion
	// WantErr is the error that should be returned.
	WantErr error
	// WantData is the `Data` object that should be constructed.
	WantData *Data
	// SkipDataCheck skips the check on `WantData`.
	SkipDataCheck bool
	// DataCmpOpts is the set of cmp.Options that should be used
	// when comparing data.
	DataCmpOpts cmp.Options

	// RunResponses are the stubbed responses to return from exec.Cmd.Run.
	RunResponses []*FakeRun

	// WantRunContents are the set of shell commands that should have been run.
	WantRunContents []*RunContents
}

func (ctc *CompleteTestCase) getEnv() map[string]string {
	if ctc.Env == nil {
		ctc.Env = map[string]string{}
	}
	return ctc.Env
}

// CompleteTest runs a test on command autocompletion.
func CompleteTest(t *testing.T, ctc *CompleteTestCase) {
	t.Helper()

	if ctc == nil {
		ctc = &CompleteTestCase{}
	}

	tc := &testContext{
		prefix:   fmt.Sprintf("Autocomplete(%v)", ctc.Args),
		testCase: ctc,
		data:     &Data{OS: ctc.OS},
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

type testContext struct {
	prefix   string
	testCase testCase

	data  *Data
	fo    *FakeOutput
	input *Input

	err error

	eData          *ExecuteData
	autocompletion *Autocompletion
}

type commandTester interface {
	setup(*testing.T, *testContext)
	check(*testing.T, *testContext)
}

type testCase interface {
	getEnv() map[string]string
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

type dataTester struct {
	want *Data
	opts cmp.Options
}

func (*dataTester) setup(*testing.T, *testContext) {}

func (dt *dataTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if dt.want == nil {
		dt.want = &Data{}
	}

	if diff := cmp.Diff(dt.want, tc.data, cmpopts.EquateEmpty(), cmpopts.IgnoreUnexported(Data{}), cmpopts.IgnoreFields(Data{}, "OS"), dt.opts); diff != "" {
		t.Errorf("%s produced incorrect Data (-want, +got):\n%s", tc.prefix, diff)
	}
}

type envTester struct{}

func (et *envTester) setup(t *testing.T, tc *testContext) {
	StubEnv(t, tc.testCase.getEnv())
}

func (et *envTester) check(t *testing.T, tc *testContext) {}

type outputTester struct {
	wantStdout string
	wantStderr string
}

func (*outputTester) setup(*testing.T, *testContext) {}
func (ot *outputTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if diff := cmp.Diff(ot.wantStdout, tc.fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s sent wrong data to stdout (-want, +got):\n%s", tc.prefix, diff)
	}
	if diff := cmp.Diff(ot.wantStderr, tc.fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s sent wrong data to stderr (-want, +got):\n%s", tc.prefix, diff)
	}
}

type inputTester struct {
	want *Input
}

func (*inputTester) setup(*testing.T, *testContext) {}
func (it *inputTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if it.want == nil {
		it.want = &Input{}
	}
	if diff := cmp.Diff(it.want, tc.input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
		t.Errorf("%s incorrectly modified input (-want, +got):\n%s", tc.prefix, diff)
	}
}

type runResponseTester struct {
	runResponses   []*FakeRun
	want           []*RunContents
	gotRunContents []*RunContents
}

type RunContents struct {
	Name          string
	Args          []string
	Dir           string
	StdinContents string
}

func (rrt *runResponseTester) stubRunResponses(t *testing.T) func(cmd *exec.Cmd) error {
	return func(cmd *exec.Cmd) error {
		if len(rrt.runResponses) == 0 {
			t.Fatalf("ran out of stubbed run responses")
		}

		var stdinContents string
		if cmd.Stdin != nil {
			b, err := io.ReadAll(cmd.Stdin)
			if err != nil {
				t.Fatalf("Failed to read data from cmd.Stdin: %v", err)
			}
			stdinContents = string(b)
		}

		// `cmd.Args[0]` is used instead of `cmd.Path` because `cmd.Path` can be modified,
		// like by msys for example.
		rrt.gotRunContents = append(rrt.gotRunContents, &RunContents{cmd.Args[0], cmd.Args[1:], cmd.Dir, stdinContents})

		r := rrt.runResponses[0]
		rrt.runResponses = rrt.runResponses[1:]
		write(t, cmd.Stdout, r.Stdout)
		write(t, cmd.Stderr, r.Stderr)
		if r.F != nil {
			r.F(t)
		}
		return r.Err
	}
}

func (rrt *runResponseTester) setup(t *testing.T, tc *testContext) {
	oldRun := run
	run = rrt.stubRunResponses(t)
	t.Cleanup(func() { run = oldRun })
}

func (rrt *runResponseTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if len(rrt.runResponses) > 0 {
		t.Errorf("unused run responses: %v", rrt.runResponses)
	}

	// Check proper commands were run.
	if diff := cmp.Diff(rrt.want, rrt.gotRunContents); diff != "" {
		t.Errorf("%s produced unexpected shell commands:\n%s", tc.prefix, diff)
	}
}

type errorTester struct {
	want error
}

func (*errorTester) setup(*testing.T, *testContext) {}
func (et *errorTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	CmpError(t, tc.prefix, et.want, tc.err)
}

func CmpError(t *testing.T, prefix string, wantErr, err error) {
	t.Helper()

	if wantErr == nil && err != nil {
		t.Errorf("%s returned error (%v) when shouldn't have", prefix, err)
	}
	if wantErr != nil {
		if err == nil {
			t.Errorf("%s returned no error when should have returned %v", prefix, wantErr)
		} else if diff := cmp.Diff(wantErr.Error(), err.Error()); diff != "" {
			t.Errorf("%s returned unexpected error (-want, +got):\n%s", prefix, diff)
		}
	}
}

type executeDataTester struct {
	want *ExecuteData
}

func (*executeDataTester) setup(*testing.T, *testContext) {}
func (et *executeDataTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	if et.want == nil {
		et.want = &ExecuteData{}
	}
	if tc.eData == nil {
		tc.eData = &ExecuteData{}
	}
	if diff := cmp.Diff(et.want, tc.eData, cmpopts.IgnoreFields(ExecuteData{}, "Executor")); diff != "" {
		t.Errorf("%s returned unexpected ExecuteData (-want, +got):\n%s", tc.prefix, diff)
	}
}

type autocompleteTester struct {
	want *Autocompletion
}

func (*autocompleteTester) setup(*testing.T, *testContext) {}
func (at *autocompleteTester) check(t *testing.T, tc *testContext) {
	t.Helper()

	if at.want == nil {
		at.want = &Autocompletion{}
	}
	if tc.autocompletion == nil {
		tc.autocompletion = &Autocompletion{}
	}

	if diff := cmp.Diff(at.want, tc.autocompletion); diff != "" {
		t.Errorf("%s produced incorrect completions (-want, +got):\n%s", tc.prefix, diff)
	}
}

func FilepathAbs(t *testing.T, s ...string) string {
	t.Helper()
	r, err := filepath.Abs(filepath.Join(s...))
	if err != nil {
		t.Fatalf("Failed to get absolute path for file: %v", err)
	}
	return r
}

// FakeOS is a fake OS that can be used for testing purposes.
type FakeOS struct{}

func (*FakeOS) SetEnvVar(variable, value string) string {
	return fmt.Sprintf("FAKE_SET[(variable=%s), (value=%s)]", variable, value)
}

func (*FakeOS) UnsetEnvVar(variable string) string {
	return fmt.Sprintf("FAKE_UNSET[(variable=%s)]", variable)
}
