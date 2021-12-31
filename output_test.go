package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOutput(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *ExecuteTestCase
	}{
		{
			name: "output formats when interfaces provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ExecutorNode(func(o Output, d *Data) {
					t := "there"
					o.Stdout("hello %s")
					o.Stdoutf("hello %s", t)

					k := "kenobi"
					o.Stderr("general %s")
					o.Stderrf("general %s", k)
				})),
				WantStdout: []string{
					"hello %s",
					"hello there",
				},
				WantStderr: []string{
					"general %s",
					"general kenobi",
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			ExecuteTest(t, test.etc)
		})
	}
}

func TestOutputWriters(t *testing.T) {
	fo := NewFakeOutput()
	outW := StdoutWriter(fo)
	errW := StderrWriter(fo)

	if _, err := outW.Write([]byte("output")); err != nil {
		t.Errorf("failed to write to stdout: %v", err)
	}
	if _, err := errW.Write([]byte("errput")); err != nil {
		t.Errorf("failed to write to stderr: %v", err)
	}

	wantStdout := []string{"output"}
	wantStderr := []string{"errput"}
	if diff := cmp.Diff(wantStdout, fo.GetStdout()); diff != "" {
		t.Errorf("Incorrect output sent to stdout writer:\n%s", diff)
	}
	if diff := cmp.Diff(wantStderr, fo.GetStderr()); diff != "" {
		t.Errorf("Incorrect output sent to stderr writer:\n%s", diff)
	}
}
