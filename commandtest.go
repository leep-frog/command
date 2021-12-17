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
	gotRunContents  [][]string

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

	data := &Data{}
	fo := NewFakeOutput()
	err := runNodes(rtc.Node, fo, data, rtc.Args)
	if rtc.WantErr == nil && err != nil {
		t.Errorf("RunNodes(%v) returned error (%v) when shouldn't have", rtc.Args, err)
	}
	if rtc.WantErr != nil {
		if err == nil {
			t.Errorf("RunNodes(%v) returned no error when should have returned %v", rtc.Args, rtc.WantErr)
		} else if diff := cmp.Diff(rtc.WantErr.Error(), err.Error()); diff != "" {
			t.Errorf("RunNodes(%v) returned unexpected error (-want, +got):\n%s", rtc.Args, diff)
		}
	}

	// Check Data.
	wantData := rtc.WantData
	if wantData == nil {
		wantData = &Data{}
	}
	if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty()); diff != "" && !rtc.SkipDataCheck {
		t.Errorf("RunNodes(%v) returned unexpected Data (-want, +got):\n%s", rtc.Args, diff)
	}

	// Check Stderr and Stdout.
	if diff := cmp.Diff(rtc.WantStdout, fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("RunNodes(%v) sent wrong data to stdout (-want, +got):\n%s", rtc.Args, diff)
	}
	if diff := cmp.Diff(rtc.WantStderr, fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("RunNodes(%v) sent wrong data to stderr (-want, +got):\n%s", rtc.Args, diff)
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

	args := etc.Args
	wantData := etc.WantData
	if wantData == nil {
		wantData = &Data{}
	}
	if etc.RequiresSetup {
		setupFile := setupForTest(t, etc.SetupContents)
		args = append([]string{setupFile}, args...)
		wantData.Set(SetupArgName, StringValue(setupFile))
		t.Cleanup(func() { os.Remove(setupFile) })
	}

	runResponses := &etc.RunResponses
	oldRun := run
	run = stubRunResponses(t, etc, runResponses)
	defer func() { run = oldRun }()

	input := NewInput(args, nil)

	fo := NewFakeOutput()
	data := &Data{}

	eData, err := execute(etc.Node, input, fo, data)
	if etc.WantErr == nil && err != nil {
		t.Errorf("execute(%v) returned error (%v) when shouldn't have", etc.Args, err)
	}
	if etc.WantErr != nil {
		if err == nil {
			t.Errorf("execute(%v) returned no error when should have returned %v", etc.Args, etc.WantErr)
		} else if diff := cmp.Diff(etc.WantErr.Error(), err.Error()); diff != "" {
			t.Errorf("execute(%v) returned unexpected error (-want, +got):\n%s", etc.Args, diff)
		}
	}

	// Check ExecuteData.
	wantEData := etc.WantExecuteData
	if wantEData == nil {
		wantEData = &ExecuteData{}
	}
	if eData == nil {
		eData = &ExecuteData{}
	}
	if diff := cmp.Diff(wantEData, eData, cmpopts.IgnoreFields(ExecuteData{}, "Executor")); diff != "" {
		t.Errorf("execute(%v) returned unexpected ExecuteData (-want, +got):\n%s", etc.Args, diff)
	}

	// Check Data.
	if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty()); diff != "" && !etc.SkipDataCheck {
		t.Errorf("execute(%v) returned unexpected Data (-want, +got):\n%s", etc.Args, diff)
	}

	// Check Stderr and Stdout.
	if diff := cmp.Diff(etc.WantStdout, fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("execute(%v) sent wrong data to stdout (-want, +got):\n%s", etc.Args, diff)
	}
	if diff := cmp.Diff(etc.WantStderr, fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("execute(%v) sent wrong data to stderr (-want, +got):\n%s", etc.Args, diff)
	}

	// Check input (if relevant).
	if etc.testInput {
		wantInput := etc.wantInput
		if wantInput == nil {
			wantInput = &Input{}
		}
		if diff := cmp.Diff(wantInput, input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
			t.Errorf("execute(%v) incorrectly modified input (-want, +got):\n%s", etc.Args, diff)
		}
	}

	// Check all run responses were used.
	if len(*runResponses) > 0 {
		t.Errorf("unused run responses: %v", runResponses)
	}

	// Check proper commands were run.
	if diff := cmp.Diff(etc.WantRunContents, etc.gotRunContents); diff != "" {
		t.Errorf("execute(%v) produced unexpected bash commands:\n%s", etc.Args, diff)
	}
}

type runStubber interface {
	addRunResponse([]string)
}

func (etc *ExecuteTestCase) addRunResponse(lines []string) {
	etc.gotRunContents = append(etc.gotRunContents, lines)
}

func (ctc *CompleteTestCase) addRunResponse(lines []string) {
	ctc.gotRunContents = append(ctc.gotRunContents, lines)
}

func stubRunResponses(t *testing.T, rs runStubber, runResponses *[]*FakeRun) func(cmd *exec.Cmd) error {
	return func(cmd *exec.Cmd) error {
		if cmd.Path != "bash" && cmd.Path != "C:\\msys64\\usr\\bin\\bash.exe" && cmd.Path != "C:\\Windows\\system32\\bash.exe" {
			t.Fatalf(`expected cmd path to be "bash"; got %q`, cmd.Path)
		}
		if len(cmd.Args) != 2 {
			t.Fatalf("expected two args ('bash filename'), but got %v", cmd.Args)
		}
		if len(*runResponses) == 0 {
			t.Fatalf("ran out of stubbed run responses")
		}

		content, err := ioutil.ReadFile(cmd.Args[1])
		if err != nil {
			t.Fatalf("unable to read file: %v", err)
		}
		lines := strings.Split(string(content), "\n")
		rs.addRunResponse(lines)

		r := (*runResponses)[0]
		*runResponses = (*runResponses)[1:]
		write(t, cmd.Stdout, r.Stdout)
		write(t, cmd.Stderr, r.Stderr)
		if r.F != nil {
			r.F(t)
		}
		return r.Err
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

	// TODO: make type that ctc and etc both point to
	// RunResponses are the stubbed responses to return from exec.Cmd.Run.
	RunResponses []*FakeRun

	// WantRunContents are the set of commands that should have been run in bash.
	WantRunContents [][]string
	gotRunContents  [][]string
}

func CompleteTest(t *testing.T, ctc *CompleteTestCase) {
	t.Helper()
	data := &Data{}

	runResponses := &ctc.RunResponses
	oldRun := run
	run = stubRunResponses(t, ctc, runResponses)
	defer func() { run = oldRun }()

	got, err := autocomplete(ctc.Node, ctc.Args, data)
	if ctc.WantErr == nil && err != nil {
		t.Errorf("autocomplete(%v) returned error (%v) when shouldn't have", ctc.Args, err)
	}
	if ctc.WantErr != nil {
		if err == nil {
			t.Errorf("autocomplete(%v) returned no error when should have returned %v", ctc.Args, ctc.WantErr)
		} else if diff := cmp.Diff(ctc.WantErr.Error(), err.Error()); diff != "" {
			t.Errorf("autocomplete(%v) returned unexpected error (-want, +got):\n%s", ctc.Args, diff)
		}
	}

	if diff := cmp.Diff(ctc.Want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Autocomplete(%v) produced incorrect completions (-want, +got):\n%s", ctc.Args, diff)
	}

	wantData := ctc.WantData
	if wantData == nil {
		wantData = &Data{}
	}
	if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty()); diff != "" && !ctc.SkipDataCheck {
		t.Errorf("Autocomplete(%s) improperly parsed args (-want, +got)\n:%s", ctc.Args, diff)
	}

	// Check all run responses were used.
	if len(*runResponses) > 0 {
		t.Errorf("unused run responses: %v", runResponses)
	}

	// Check proper commands were run.
	if diff := cmp.Diff(ctc.WantRunContents, ctc.gotRunContents); diff != "" {
		t.Errorf("Autocomplete(%v) produced unexpected bash commands:\n%s", ctc.Args, diff)
	}
}
