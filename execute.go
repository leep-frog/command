package command

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type Node struct {
	Processor Processor
	Edge      Edge
}

type Processor interface {
	Execute(*Input, Output, *Data, *ExecuteData) error
	// Complete should return complete data if there was an error or a completion can be made.
	Complete(*Input, *Data) *CompleteData
}

type Edge interface {
	Next(*Input, *Data) (*Node, error)
}

func Execute(n *Node, input *Input, output Output) (*ExecuteData, error) {
	return execute(n, input, output, &Data{})
}

func iterativeExecute(n *Node, input *Input, output Output, data *Data, eData *ExecuteData, runExecutor bool) error {
	for n != nil {
		if n.Processor != nil {
			if err := n.Processor.Execute(input, output, data, eData); err != nil {
				return err
			}
		}

		if n.Edge == nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, data); err != nil {
			return err
		}
	}

	if !input.FullyProcessed() {
		return output.Err(ExtraArgsErr(input))
	}

	if runExecutor && eData.Executor != nil {
		return eData.Executor(output, data)
	}
	return nil
}

// Separate method for testing purposes.
func execute(n *Node, input *Input, output Output, data *Data) (*ExecuteData, error) {
	eData := &ExecuteData{}
	return eData, iterativeExecute(n, input, output, data, eData, true)
}

func ExtraArgsErr(input *Input) error {
	return &extraArgsErr{input}
}

type extraArgsErr struct {
	input *Input
}

func (eae *extraArgsErr) Error() string {
	return fmt.Sprintf("Unprocessed extra args: %v", eae.input.Remaining())
}

// TODO: if this function isn't in test package, then it isn't exposed publicly.
// Find out best place to put this.
func executeTest(t *testing.T, node *Node, args []string, wantErr error, want *ExecuteData, wantData *Data, wantInput *Input, wantStdout, wantStderr []string) {
	input := ParseArgs(args)
	testExecute(t, node, args, input, wantErr, want, wantData, wantStdout, wantStderr)

	if wantInput == nil {
		wantInput = &Input{}
	}
	if diff := cmp.Diff(wantInput, input, cmpopts.EquateEmpty(), cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
		t.Errorf("execute(%v) incorrectly modified input (-want, +got):\n%s", args, diff)
	}
}

func ExecuteTest(t *testing.T, node *Node, args []string, wantErr error, want *ExecuteData, wantData *Data, wantStdout, wantStderr []string) {
	input := ParseArgs(args)
	testExecute(t, node, args, input, wantErr, want, wantData, wantStdout, wantStderr)
}

func testExecute(t *testing.T, node *Node, args []string, input *Input, wantErr error, want *ExecuteData, wantData *Data, wantStdout, wantStderr []string) {
	t.Helper()

	fo := NewFakeOutput()
	data := &Data{}

	eData, err := execute(node, input, fo, data)
	if wantErr == nil && err != nil {
		t.Fatalf("execute(%v) returned error (%v) when shouldn't have", args, err)
	}
	if wantErr != nil {
		if err == nil {
			t.Fatalf("execute(%v) returned no error when should have returned %v", args, wantErr)
		} else if diff := cmp.Diff(wantErr.Error(), err.Error()); diff != "" {
			t.Errorf("execute(%v) returned unexpected error (-want, +got):\n%s", args, diff)
		}
	}

	if want == nil {
		want = &ExecuteData{}
	}
	if eData == nil {
		eData = &ExecuteData{}
	}
	if diff := cmp.Diff(want, eData, cmpopts.IgnoreFields(ExecuteData{}, "Executor")); diff != "" {
		t.Errorf("execute(%v) returned unexpected ExecuteData (-want, +got):\n%s", args, diff)
	}

	if wantData == nil {
		wantData = &Data{}
	}
	if diff := cmp.Diff(wantData, data); diff != "" {
		t.Errorf("execute(%v) returned unexpected Data (-want, +got):\n%s", args, diff)
	}

	if diff := cmp.Diff(wantStdout, fo.GetStdout(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("execute(%v) sent wrong data to stdout (-want, +got):\n%s", args, diff)
	}
	if diff := cmp.Diff(wantStderr, fo.GetStderr(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("execute(%v) sent wrong data to stderr (-want, +got):\n%s", args, diff)
	}
}
