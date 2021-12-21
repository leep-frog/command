package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const (
	AliasDesc  = "  *: Start of new aliasable section"
	CacheDesc  = "  ^: Start of new cachable section"
	BranchDesc = "  <: Start of subcommand branches"
)

type UsageTestCase struct {
	Node       *Node
	WantString []string
}

type ExecuteTestCase struct {
	Node *Node
	Args []string

	WantData        *Data
	WantExecuteData *ExecuteData
	WantStdout      []string
	WantStderr      []string
	WantErr         error

	// Whether or not to test actual input against wantInput.
	testInput bool
	wantInput *Input

	// WantRunContents are the set of commands that should have been run in bash.
	WantRunContents [][]string

	// RequiresSetup indicates whether or not the command requires setup
	RequiresSetup bool
	SetupContents []string

	// RunResponses are the stubbed responses to return from exec.Cmd.Run.
	RunResponses []*FakeRun

	// Options to skip checks
	SkipDataCheck bool
}

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
	defer f.Close()
	for _, s := range contents {
		fmt.Fprintln(f, s)
	}
	return f.Name()
}

func UsageTest(t *testing.T, utc *UsageTestCase) {
	t.Helper()

	if utc == nil {
		utc = &UsageTestCase{}
	}

	if diff := cmp.Diff(strings.Join(utc.WantString, "\n"), GetUsage(utc.Node).String()); diff != "" {
		t.Errorf("UsageString() returned incorrect response (-want, +got):\n%s", diff)
	}
}

type RunNodeTestCase struct {
	Node *Node

	Args     []string
	WantData *Data
	WantErr  error

	WantStdout []string
	WantStderr []string

	WantFileContents []string

	SkipDataCheck bool
}

func RunNodeTest(t *testing.T, rtc *RunNodeTestCase) {
	t.Helper()

	// Define prefix before TMP_FILE is switched out
	prefix := fmt.Sprintf("RunNodes(%v)", rtc.Args)

	var f *os.File
	for i, line := range rtc.Args {
		if line == "TMP_FILE" {
			var err error
			f, err = ioutil.TempFile("", "leep-run-node-test")
			if err != nil {
				t.Fatalf("failed to create tmp file: %v", err)
			}
			rtc.Args[i] = f.Name()
			break
		}
	}

	if rtc == nil {
		rtc = &RunNodeTestCase{}
	}

	tc := &testContext{
		data: &Data{},
		fo:   NewFakeOutput(),
	}
	defer tc.fo.Close()

	testers := []commandTester{
		&outputTester{rtc.WantStdout, rtc.WantStderr},
		&errorTester{rtc.WantErr},
		checkIf(!rtc.SkipDataCheck, &dataTester{rtc.WantData}),
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}
	tc.err = runNodes(rtc.Node, tc.fo, tc.data, rtc.Args)

	for _, tester := range testers {
		tester.check(t, prefix, tc)
	}

	var fileContents []string
	if f != nil {
		b, err := ioutil.ReadFile(f.Name())
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		fileContents = strings.Split(string(b), "\n")
		if len(fileContents) == 1 && fileContents[0] == "" {
			fileContents = nil
		}
	}
	if diff := cmp.Diff(rtc.WantFileContents, fileContents); diff != "" {
		t.Errorf("RunNodes(%v) sent incorrect data to file (-want, +got):\n%s", rtc.Args, diff)
	}
}

func ExecuteTest(t *testing.T, etc *ExecuteTestCase) {
	t.Helper()

	if etc == nil {
		etc = &ExecuteTestCase{}
	}

	tc := &testContext{
		data:         &Data{},
		fo:           NewFakeOutput(),
		runResponses: etc.RunResponses,
	}
	defer tc.fo.Close()
	args := etc.Args
	if etc.RequiresSetup {
		setupFile := setupForTest(t, etc.SetupContents)
		args = append([]string{setupFile}, args...)
		etc.WantData.Set(SetupArgName, StringValue(setupFile))
		t.Cleanup(func() { os.Remove(setupFile) })
	}
	tc.input = NewInput(args, nil)

	testers := []commandTester{
		&outputTester{etc.WantStdout, etc.WantStderr},
		&errorTester{etc.WantErr},
		&executeDataTester{etc.WantExecuteData},
		&runResponseTester{etc.WantRunContents},
		checkIf(!etc.SkipDataCheck, &dataTester{etc.WantData}),
		checkIf(etc.testInput, &inputTester{etc.wantInput}),
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}

	tc.eData, tc.err = execute(etc.Node, tc.input, tc.fo, tc.data)

	prefix := fmt.Sprintf("Execute(%v)", etc.Args)
	for _, tester := range testers {
		tester.check(t, prefix, tc)
	}
}

func write(t *testing.T, iow io.Writer, contents []string) {
	if _, err := bytes.NewBufferString(strings.Join(contents, "\n")).WriteTo(iow); err != nil {
		t.Fatalf("failed to write buffer to io.Writer: %v", err)
	}
}

type Changeable interface {
	Changed() bool
}

// ChangeTest tests if an object has changed.
func ChangeTest(t *testing.T, want interface{}, original Changeable, opts ...cmp.Option) {
	wantChanged := want != nil && !reflect.ValueOf(want).IsNil()
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

type CompleteTestCase struct {
	Node *Node
	// Remember that args requires a dummy command argument (e.g. "cmd ")
	Args string

	Want     []string
	WantErr  error
	WantData *Data

	SkipDataCheck bool

	// RunResponses are the stubbed responses to return from exec.Cmd.Run.
	RunResponses []*FakeRun

	// WantRunContents are the set of commands that should have been run in bash.
	WantRunContents [][]string
}

func CompleteTest(t *testing.T, ctc *CompleteTestCase) {
	t.Helper()

	if ctc == nil {
		ctc = &CompleteTestCase{}
	}

	tc := &testContext{
		data:         &Data{},
		runResponses: ctc.RunResponses,
	}

	testers := []commandTester{
		&runResponseTester{ctc.WantRunContents},
		&errorTester{ctc.WantErr},
		&autocompleteTester{ctc.Want},
		checkIf(!ctc.SkipDataCheck, &dataTester{ctc.WantData}),
	}

	for _, tester := range testers {
		tester.setup(t, tc)
	}

	tc.autocompleteSuggestions, tc.err = autocomplete(ctc.Node, ctc.Args, tc.data)

	prefix := fmt.Sprintf("Autocomplete(%v)", ctc.Args)
	for _, tester := range testers {
		tester.check(t, prefix, tc)
	}
}

type testContext struct {
	data  *Data
	fo    *FakeOutput
	input *Input

	runResponses   []*FakeRun
	gotRunContents [][]string

	err error

	eData                   *ExecuteData
	autocompleteSuggestions []string
}

type commandTester interface {
	setup(*testing.T, *testContext)
	check(*testing.T, string, *testContext)
}

type noOpTester struct{}

func (*noOpTester) setup(t *testing.T, tc *testContext) {}

func (*noOpTester) check(t *testing.T, prefix string, tc *testContext) {}

func checkIf(cond bool, ct commandTester) commandTester {
	if cond {
		return ct
	}
	return &noOpTester{}
}

type dataTester struct {
	want *Data
}

func (*dataTester) setup(t *testing.T, tc *testContext) {}

func (dt *dataTester) check(t *testing.T, prefix string, tc *testContext) {
	t.Helper()
	if dt.want == nil {
		dt.want = &Data{}
	}

	if diff := cmp.Diff(dt.want, tc.data, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s produced incorrect Data (-want, +got):\n%s", prefix, diff)
	}
}

type outputTester struct {
	wantStdout []string
	wantStderr []string
}

func (*outputTester) setup(t *testing.T, tc *testContext) {}
func (ot *outputTester) check(t *testing.T, prefix string, tc *testContext) {
	t.Helper()
	if diff := cmp.Diff(ot.wantStdout, tc.fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s sent wrong data to stdout (-want, +got):\n%s", prefix, diff)
	}
	if diff := cmp.Diff(ot.wantStderr, tc.fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("%s sent wrong data to stderr (-want, +got):\n%s", prefix, diff)
	}
}

type inputTester struct {
	want *Input
}

func (*inputTester) setup(t *testing.T, tc *testContext) {}
func (it *inputTester) check(t *testing.T, prefix string, tc *testContext) {
	t.Helper()
	if it.want == nil {
		it.want = &Input{}
	}
	if diff := cmp.Diff(it.want, tc.input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
		t.Errorf("%s incorrectly modified input (-want, +got):\n%s", prefix, diff)
	}
}

type runResponseTester struct {
	want [][]string
}

func stubRunResponses(t *testing.T, tc *testContext) func(cmd *exec.Cmd) error {
	return func(cmd *exec.Cmd) error {
		if cmd.Path != "bash" && cmd.Path != "C:\\msys64\\usr\\bin\\bash.exe" && cmd.Path != "C:\\Windows\\system32\\bash.exe" {
			t.Fatalf(`expected cmd path to be "bash"; got %q`, cmd.Path)
		}
		if len(cmd.Args) != 2 {
			t.Fatalf("expected two args ('bash filename'), but got %v", cmd.Args)
		}
		if len(tc.runResponses) == 0 {
			t.Fatalf("ran out of stubbed run responses")
		}

		content, err := ioutil.ReadFile(cmd.Args[1])
		if err != nil {
			t.Fatalf("unable to read file: %v", err)
		}
		lines := strings.Split(string(content), "\n")
		tc.gotRunContents = append(tc.gotRunContents, lines)

		r := tc.runResponses[0]
		tc.runResponses = tc.runResponses[1:]
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
	run = stubRunResponses(t, tc)
	t.Cleanup(func() { run = oldRun })
}

func (rrt *runResponseTester) check(t *testing.T, prefix string, tc *testContext) {
	t.Helper()
	if len(tc.runResponses) > 0 {
		t.Errorf("unused run responses: %v", tc.runResponses)
	}

	// Check proper commands were run.
	if diff := cmp.Diff(rrt.want, tc.gotRunContents); diff != "" {
		t.Errorf("%s produced unexpected bash commands:\n%s", prefix, diff)
	}
}

type errorTester struct {
	want error
}

func (*errorTester) setup(t *testing.T, tc *testContext) {}
func (et *errorTester) check(t *testing.T, prefix string, tc *testContext) {
	t.Helper()

	cmpError(t, prefix, et.want, tc.err)
}

func cmpError(t *testing.T, prefix string, wantErr, err error) {
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

func (*executeDataTester) setup(t *testing.T, tc *testContext) {}
func (et *executeDataTester) check(t *testing.T, prefix string, tc *testContext) {
	t.Helper()

	if et.want == nil {
		et.want = &ExecuteData{}
	}
	if tc.eData == nil {
		tc.eData = &ExecuteData{}
	}
	if diff := cmp.Diff(et.want, tc.eData, cmpopts.IgnoreFields(ExecuteData{}, "Executor")); diff != "" {
		t.Errorf("%s returned unexpected ExecuteData (-want, +got):\n%s", prefix, diff)
	}
}

type autocompleteTester struct {
	want []string
}

func (*autocompleteTester) setup(t *testing.T, tc *testContext) {}
func (at *autocompleteTester) check(t *testing.T, prefix string, tc *testContext) {
	t.Helper()

	if diff := cmp.Diff(at.want, tc.autocompleteSuggestions); diff != "" {
		t.Errorf("%s produced incorrect completions (-want, +got):\n%s", prefix, diff)
	}
}
