package spycommandertest

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/stubs"
	"github.com/leep-frog/command/internal/testutil"
)

type runResponseTester struct {
	runResponses   []*commandtest.FakeRun
	want           []*commandtest.RunContents
	gotRunContents []*commandtest.RunContents
}

type fakeWriteCloser struct {
	*bytes.Buffer
}

func (f *fakeWriteCloser) Close() error {
	return nil
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
		rrt.gotRunContents = append(rrt.gotRunContents, &commandtest.RunContents{cmd.Args[0], cmd.Args[1:], cmd.Dir, stdinContents})

		r := rrt.runResponses[0]
		rrt.runResponses = rrt.runResponses[1:]
		testutil.Write(t, cmd.Stdout, r.Stdout)
		testutil.Write(t, cmd.Stderr, r.Stderr)
		if r.F != nil {
			r.F(t)
		}
		return r.Err
	}
}

func (rrt *runResponseTester) setup(t *testing.T, tc *testContext) {
	testutil.StubValue(t, &stubs.Run, rrt.stubRunResponses(t))
	testutil.StubValue(t, &stubs.StubStdinPipe, func(cmd *exec.Cmd) (io.WriteCloser, error) {
		if cmd.Stdin != nil {
			return nil, fmt.Errorf("cmd.Stdin is already set")
		}
		f := &fakeWriteCloser{bytes.NewBufferString("")}
		cmd.Stdin = f
		return f, nil
	})
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
