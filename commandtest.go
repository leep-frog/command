package command

import (
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
}

type ExecuteTestOptions struct {
	testInput bool
}

func ExecuteTest(t *testing.T, etc *ExecuteTestCase, opts *ExecuteTestOptions) {
	t.Helper()

	if etc == nil {
		etc = &ExecuteTestCase{}
	}

	input := ParseArgs(etc.Args)

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
	wantData := etc.WantData
	if wantData == nil {
		wantData = &Data{}
	}
	if diff := cmp.Diff(wantData, data); diff != "" {
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
	if opts.testInput {
		wantInput := etc.wantInput
		if wantInput == nil {
			wantInput = &Input{}
		}
		if diff := cmp.Diff(wantInput, input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
			t.Errorf("execute(%v) incorrectly modified input (-want, +got):\n%s", etc.Args, diff)
		}
	}
}

type CompleteTestCase struct {
	Node *Node
	Args []string

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
