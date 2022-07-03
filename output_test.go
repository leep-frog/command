package command

import (
	"fmt"
	"strings"
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

					// Make sure we don't format the %s from the error
					o.Annotate(fmt.Errorf("bad news bears %%s"), "attention animals")
					o.Annotatef(fmt.Errorf("rough news rabbits %%s"), "attention %d dalmations", 101)
				})),
				WantStdout: strings.Join([]string{
					"hello %s",
					"hello there",
				}, ""),
				WantStderr: strings.Join([]string{
					"general %s",
					"general kenobi",
					"attention animals: bad news bears %s\n",
					"attention 101 dalmations: rough news rabbits %s\n",
					"",
				}, ""),
			},
		},
		{
			name: "output terminates on error, but not on nil",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ExecutorNode(func(o Output, d *Data) {
					o.Stdout("hello")
					o.Stderr("there")

					o.Terminate(nil)

					o.Stdout("general")
					o.Stderr("kenobi")

					o.Terminate(fmt.Errorf("donzo"))

					o.Stdout("ignore")
					o.Stderr("this")
				})),
				WantStdout: strings.Join([]string{
					"hello",
					"general",
				}, ""),
				WantStderr: strings.Join([]string{
					"there",
					"kenobi",
					"donzo\n",
				}, ""),
				WantErr: fmt.Errorf("donzo"),
			},
		},
		{
			name: "Terminatef terminates",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ExecutorNode(func(o Output, d *Data) {
					o.Stdout("hello")
					o.Stderr("there")

					o.Terminatef("ahoy %s", "matey")

					o.Stdout("general")
					o.Stderr("kenobi")
				})),
				WantStdout: "hello",
				WantStderr: "thereahoy matey",
				WantErr:    fmt.Errorf("ahoy matey"),
			},
		},
		{
			name: "Tannotate terminates",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ExecutorNode(func(o Output, d *Data) {
					o.Stdout("hello")
					o.Stderr("there")

					o.Tannotate(nil, "don't mind me")

					o.Stdout("general")
					o.Stderr("kenobi")

					o.Tannotate(fmt.Errorf("do mind me"), "but")

					o.Stdout("ignore")
					o.Stderr("us")
				})),
				WantStdout: "hellogeneral",
				WantStderr: "therekenobibut: do mind me\n",
				WantErr:    fmt.Errorf("but: do mind me"),
			},
		},
		{
			name: "Tannotate terminates",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ExecutorNode(func(o Output, d *Data) {
					o.Stdout("hello")
					o.Stderr("there")

					o.Tannotatef(nil, "don't %s me", "mind")

					o.Stdout("general")
					o.Stderr("kenobi")

					o.Tannotatef(fmt.Errorf("do mind me"), "%s%s", "how", "ever")

					o.Stdout("ignore")
					o.Stderr("us")
				})),
				WantStdout: "hellogeneral",
				WantStderr: "therekenobihowever: do mind me\n",
				WantErr:    fmt.Errorf("however: do mind me"),
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

	wantStdout := "output"
	wantStderr := "errput"
	if diff := cmp.Diff(wantStdout, fo.GetStdout()); diff != "" {
		t.Errorf("Incorrect output sent to stdout writer:\n%s", diff)
	}
	if diff := cmp.Diff(wantStderr, fo.GetStderr()); diff != "" {
		t.Errorf("Incorrect output sent to stderr writer:\n%s", diff)
	}
}
