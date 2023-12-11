package spycommand

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/internal/testutil"
)

func TestSnapshotsMap(t *testing.T) {
	for _, test := range []struct {
		name  string
		input []InputSnapshot
		want  map[InputSnapshot]bool
	}{
		{
			name: "nil input returns nil map",
		},
		{
			name:  "empty input returns nil map",
			input: []InputSnapshot{},
		},
		{
			name:  "single input",
			input: []InputSnapshot{3},
			want: map[InputSnapshot]bool{
				3: true,
			},
		},
		{
			name:  "multiple inputs",
			input: []InputSnapshot{3, 9, 27},
			want: map[InputSnapshot]bool{
				3:  true,
				9:  true,
				27: true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testutil.Cmp(t, fmt.Sprintf("SnapshotsMap(%v)", test.input), test.want, SnapshotsMap(test.input...))
		})
	}
}

func TestTerminate(t *testing.T) {
	for _, test := range []struct {
		name                      string
		f                         func()
		wantPanic                 any
		wantIsTerminationPanicOK  bool
		wantIsTerminationPanicErr error
		wantNoPanic               bool
	}{
		{
			name: "panic via Terminate(error)",
			f: func() {
				Terminate(fmt.Errorf("oops"))
			},
			wantPanic:                 &Terminator{fmt.Errorf("oops")},
			wantIsTerminationPanicOK:  true,
			wantIsTerminationPanicErr: fmt.Errorf("oops"),
		},
		{
			name: "Terminate(nil) does not panic",
			f: func() {
				Terminate(nil)
			},
			wantPanic:                 nil,
			wantIsTerminationPanicOK:  false,
			wantIsTerminationPanicErr: nil,
			wantNoPanic:               true,
		},
		{
			name: "panic with non-error value",
			f: func() {
				panic("ahhh")
			},
			wantPanic:                 "ahhh",
			wantIsTerminationPanicOK:  false,
			wantIsTerminationPanicErr: nil,
			wantNoPanic:               false,
		},
		{
			name: "panic with non-Terminator error value",
			f: func() {
				panic(fmt.Errorf("other"))
			},
			wantPanic:                 fmt.Errorf("other"),
			wantIsTerminationPanicOK:  false,
			wantIsTerminationPanicErr: nil,
			wantNoPanic:               false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gotPanic any
			var gotIsTerminationPanicErr error
			var gotIsTerminationPanicOK bool
			gotNoPanic := func() bool {
				defer func() {
					gotPanic = recover()
					gotIsTerminationPanicOK, gotIsTerminationPanicErr = IsTerminationPanic(gotPanic)
				}()
				test.f()
				return true
			}()

			testutil.Cmp(t, "Terminate function returned incorrect gotNoPanic value", test.wantNoPanic, gotNoPanic)
			if gotPanicErr, ok := gotPanic.(error); ok {
				testutil.CmpError(t, "Terminate function resulted in incorrect panic value", test.wantPanic.(error), gotPanicErr)
			} else {
				testutil.Cmp(t, "Terminate function resulted in incorrect panic value", test.wantPanic, gotPanic, TerminationCmpopts())
			}

			testutil.Cmp(t, "Terminate function resulted in incorrect IsTerminationPanic.ok", test.wantIsTerminationPanicOK, gotIsTerminationPanicOK)
			testutil.CmpError(t, "Terminate function resulted in incorrect IsTerminationPanic.error", test.wantIsTerminationPanicErr, gotIsTerminationPanicErr)

			// var a, b *Terminator
			// diff := cmp.Diff(a, b, TerminationCmpopts())
			// t.Fatalf("ugh: %s", diff)
		})
	}
}

// There isn't a great place to put cmpopts because if it's here, then it's not in a *test package,
// but if we put it in spycommandtest, then it can't be used here!
// So we just add tests for it for coverage reasons.
func TestTerminationCmpopts(t *testing.T) {
	for _, test := range []struct {
		name     string
		a, b     *Terminator
		wantDiff string
	}{
		{
			name: "both nil are equal",
		},
		{
			name: "one nil and one non-nil are not equal",
			a:    &Terminator{},
			wantDiff: strings.Join([]string{
				"  (*spycommand.Terminator)(",
				"- \t&{},",
				"+ \tnil,",
				"  )",
				"",
			}, "\n"),
		},
		{
			name: "both empty are equal",
			a:    &Terminator{},
			b:    &Terminator{},
		},
		{
			name: "one empty and one with error are not equal",
			a:    &Terminator{fmt.Errorf("darn")},
			b:    &Terminator{},
			wantDiff: strings.Join([]string{"  (*spycommand.Terminator)(",
				"- \t&{TerminationError: e\"darn\"},",
				"+ \t&{},",
				"  )",
				"",
			}, "\n"),
		},
		{
			name: "both populated with same error value are equal",
			a:    &Terminator{fmt.Errorf("darn")},
			b:    &Terminator{fmt.Errorf("darn")},
		},
		{
			name: "both populated with different errors are not equal",
			a:    &Terminator{fmt.Errorf("darn")},
			b:    &Terminator{fmt.Errorf("drat")},
			wantDiff: strings.Join([]string{
				"  (*spycommand.Terminator)(",
				"- \t&{TerminationError: e\"darn\"},",
				"+ \t&{TerminationError: e\"drat\"},",
				"  )",
				"",
			}, "\n"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gotDiff := strings.ReplaceAll(cmp.Diff(test.a, test.b, TerminationCmpopts()), "\u00a0", " ")

			testutil.Cmp(t, "TerminationCmpopts()", test.wantDiff, gotDiff)
		})
	}
}
