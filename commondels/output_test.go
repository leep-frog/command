package commondels

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/internal/testutil"
)

func TestOutput(t *testing.T) {
	for _, test := range []struct {
		name       string
		fo         func(o Output) Output
		f          func(o Output) error
		wantStdout string
		wantStderr string
		wantErr    error
		wantPanic  interface{}
	}{
		{
			name: "output formats when interfaces provided",
			f: func(o Output) error {
				t := "there"
				o.Stdout("hello %s")
				o.Stdoutf("hello %s", t)
				o.Stdoutln("final", t)

				k := "kenobi"
				o.Stderr("general %s")
				o.Stderrf("general %s", k)
				o.Stderrln("finale", k)

				// These are ignored
				var nilErr error
				o.Annotate(nilErr, "nope")
				o.Annotatef(nilErr, "more %q", "nope")

				// Make sure we don't format the %s from the error
				o.Annotate(fmt.Errorf("bad news bears %%s"), "attention animals")
				o.Annotatef(fmt.Errorf("rough news rabbits %%s"), "attention %d dalmations", 101)
				return nil
			},
			wantStdout: strings.Join([]string{
				"hello %s",
				"hello there",
				"final there\n",
			}, ""),
			wantStderr: strings.Join([]string{
				"general %s",
				"general kenobi",
				"finale kenobi\n",
				"attention animals: bad news bears %s\n",
				"attention 101 dalmations: rough news rabbits %s\n",
				"",
			}, ""),
		},
		{
			name: "output.Err prints and returns err",
			f: func(o Output) error {
				err := fmt.Errorf("some %q", "error")
				return o.Err(err)
			},
			wantStderr: "some \"error\"\n",
			wantErr:    fmt.Errorf("some \"error\""),
		},
		{
			name: "output.Err ignores nil err",
			f: func(o Output) error {
				var err error
				return o.Err(err)
			},
		},
		{
			name: "output terminates on error, but not on nil",
			f: func(o Output) error {
				o.Stdout("hello")
				o.Stderr("there")

				o.Terminate(nil)

				o.Stdout("general")
				o.Stderr("kenobi")

				o.Terminate(fmt.Errorf("donzo"))

				o.Stdout("ignore")
				o.Stderr("this")
				return nil
			},
			wantPanic: TerminationErr(fmt.Errorf("donzo")),
			wantStdout: strings.Join([]string{
				"hello",
				"general",
			}, ""),
			wantStderr: strings.Join([]string{
				"there",
				"kenobi",
				"donzo\n",
			}, ""),
		},
		{
			name: "Terminatef terminates",
			f: func(o Output) error {
				o.Stdout("hello")
				o.Stderr("there")

				o.Terminatef("ahoy %s", "matey")

				o.Stdout("general")
				o.Stderr("kenobi")
				return nil
			},
			wantStdout: "hello",
			wantStderr: "thereahoy matey",
			wantPanic:  TerminationErr(fmt.Errorf("ahoy matey")),
		},
		{
			name: "Tannotate terminates",
			f: func(o Output) error {
				o.Stdout("hello")
				o.Stderr("there")

				o.Tannotate(nil, "don't mind me")

				o.Stdout("general")
				o.Stderr("kenobi")

				o.Tannotate(fmt.Errorf("do mind me"), "but")

				o.Stdout("ignore")
				o.Stderr("us")
				return nil
			},
			wantStdout: "hellogeneral",
			wantStderr: "therekenobibut: do mind me\n",
			wantPanic:  TerminationErr(fmt.Errorf("but: do mind me")),
		},
		{
			name: "Tannotate terminates",
			f: func(o Output) error {
				o.Stdout("hello")
				o.Stderr("there")

				o.Tannotatef(nil, "don't %s me", "mind")

				o.Stdout("general")
				o.Stderr("kenobi")

				o.Tannotatef(fmt.Errorf("do mind me"), "%s%s", "how", "ever")

				o.Stdout("ignore")
				o.Stderr("us")
				return nil
			},
			wantStdout: "hellogeneral",
			wantStderr: "therekenobihowever: do mind me\n",
			wantPanic:  TerminationErr(fmt.Errorf("however: do mind me")),
		},
		// IgnoreErr output
		{
			name: "Return ignored error",
			fo: func(o Output) Output {
				return NewIgnoreErrOutput(o,
					func(err error) bool {
						return err.Error() == "yup"
					},
					func(err error) bool {
						return err.Error() == "YES"
					},
				)
			},
			f: func(o Output) error {
				o.Err(fmt.Errorf("yup"))
				o.Err(fmt.Errorf("Yup"))
				o.Err(fmt.Errorf("yes"))
				o.Err(fmt.Errorf("Yes"))
				return o.Err(fmt.Errorf("YES"))
			},
			wantErr:    fmt.Errorf("YES"),
			wantStderr: "Yup\nyes\nYes\n",
		},
		{
			name: "Return non-ignored error",
			fo: func(o Output) Output {
				return NewIgnoreErrOutput(o,
					func(err error) bool {
						return err.Error() == "yup"
					},
					func(err error) bool {
						return err.Error() == "YES"
					},
				)
			},
			f: func(o Output) error {
				return o.Err(fmt.Errorf("Yes"))
			},
			wantErr:    fmt.Errorf("Yes"),
			wantStderr: "Yes\n",
		},
		// NewIgnoreAllOutput tests
		{
			name: "NewIgnoreAllOutput doesn't output, but returns errors",
			fo: func(o Output) Output {
				return NewIgnoreAllOutput()
			},
			f: func(o Output) error {
				o.Stdoutln("a")
				o.Stderrln("b")
				o.Annotate(fmt.Errorf("oops"), "c")
				return o.Err(fmt.Errorf("e"))
			},
			wantErr: fmt.Errorf("e"),
		},
		{
			name: "NewIgnoreAllOutput doesn't output, but panics",
			fo: func(o Output) Output {
				return NewIgnoreAllOutput()
			},
			f: func(o Output) error {
				o.Stdoutln("a")
				o.Stderrln("b")
				o.Terminatef("whoops")
				o.Annotate(fmt.Errorf("oops"), "c")
				return o.Err(fmt.Errorf("e"))
			},
			wantPanic: TerminationErr(fmt.Errorf("whoops")),
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			fakeO := NewFakeOutput()
			var o Output = fakeO
			if test.fo != nil {
				o = test.fo(o)
			}

			err := testutil.CmpPanic(t, "Output func()", func() error { return test.f(o) }, test.wantPanic, TerminationCmpopts())
			testutil.CmpError(t, "Output func()", test.wantErr, err)
			testutil.Cmp(t, "Output func() produced incorrect stdout", test.wantStdout, fakeO.GetStdout())
			testutil.Cmp(t, "Output func() produced incorrect stderr", test.wantStderr, fakeO.GetStderr())
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

func TestMetadata(t *testing.T) {
	t.Run("NewOutput() succeeds and prints", func(t *testing.T) {
		o := NewOutput()
		// TODO: Stub things so this doesn't actually
		o.Stdoutln("Output.Stdoutln() test")
		o.Stderrln("Output.Stderrln() test")
	})
}
