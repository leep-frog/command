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

type ExecuteTestCase struct {
	Node *Node
	Args []string

	WantData        *Data
	WantExecuteData *ExecuteData
	WantStdout      []string
	WantStderr      []string
	WantErr         error

	// Arguments only used for internal testing.
	wantInput *Input

	// WantRunContents are the set of commands that should have been run in bash.
	WantRunContents [][]string
	gotRunContents  [][]string
}

type ExecuteTestOptions struct {
	// TODO: just move these into ExecuteTestCase
	testInput bool

	RequiresSetup bool
	SetupContents []string

	RunResponses    []*FakeRun
	WantRunCommands [][]string
}

type FakeRun struct {
	Stdout []string
	Stderr []string
	Err    error
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

func ExecuteTest(t *testing.T, etc *ExecuteTestCase, opts *ExecuteTestOptions) {
	t.Helper()

	if etc == nil {
		etc = &ExecuteTestCase{}
	}

	args := etc.Args
	wantData := etc.WantData
	if wantData == nil {
		wantData = &Data{}
	}
	if opts != nil && opts.RequiresSetup {
		setupFile := setupForTest(t, opts.SetupContents)
		args = append([]string{setupFile}, args...)
		wantData.Set(SetupArgName, StringValue(setupFile))
		t.Cleanup(func() { os.Remove(setupFile) })
	}

	runResponses := []*FakeRun{}
	if opts != nil {
		runResponses = opts.RunResponses
	}

	oldRun := run
	run = func(cmd *exec.Cmd) error {
		if cmd.Path != "bash" {
			t.Fatalf(`expected cmd path to be "bash"; got %q`, cmd.Path)
		}
		if len(cmd.Args) != 2 {
			t.Fatalf("expected two args ('bash filename'), but got %v", cmd.Args)
		}
		if len(runResponses) == 0 {
			t.Fatalf("ran out of stubbed run responses")
		}

		content, err := ioutil.ReadFile(cmd.Args[1])
		if err != nil {
			t.Fatalf("unable to read file: %v", err)
		}
		lines := strings.Split(string(content), "\n")
		etc.gotRunContents = append(etc.gotRunContents, lines)

		r := runResponses[0]
		runResponses = runResponses[1:]
		write(t, cmd.Stdout, r.Stdout)
		write(t, cmd.Stderr, r.Stderr)
		return r.Err
	}
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
	if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty()); diff != "" {
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
	if opts != nil && opts.testInput {
		wantInput := etc.wantInput
		if wantInput == nil {
			wantInput = &Input{}
		}
		if diff := cmp.Diff(wantInput, input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
			t.Errorf("execute(%v) incorrectly modified input (-want, +got):\n%s", etc.Args, diff)
		}
	}

	// Check all run responses were used.
	if opts != nil && len(runResponses) > 0 {
		t.Errorf("unused run responses: %v", opts.RunResponses)
	}

	// Check proper commands were run.
	if diff := cmp.Diff(etc.WantRunContents, etc.gotRunContents); diff != "" {
		t.Errorf("execute(%v) produced unexpected bash commands:\n%s", etc.Args, diff)
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
	Args string

	Want     []string
	WantData *Data
}

type CompleteTestOptions struct{}

func CompleteTest(t *testing.T, ctc *CompleteTestCase, opts *CompleteTestOptions) {
	t.Helper()
	data := &Data{}

	got := autocomplete(ctc.Node, ctc.Args, data)
	if diff := cmp.Diff(ctc.Want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Autocomplete(%v) produced incorrect completions (-want, +got):\n%s", ctc.Args, diff)
	}

	wantData := ctc.WantData
	if wantData == nil {
		wantData = &Data{}
	}
	if diff := cmp.Diff(wantData, data, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("getCompleteData(%s) improperly parsed args (-want, +got)\n:%s", ctc.Args, diff)
	}
}
