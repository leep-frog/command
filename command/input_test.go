package command

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/internal/spycommand"
	"github.com/leep-frog/command/internal/spyinput"
	"github.com/leep-frog/command/internal/testutil"
)

func runePtr(r rune) *rune {
	return &r
}

func TestPushFront(t *testing.T) {
	for _, test := range []struct {
		name string
		i    *Input
		sl   []string
		want *Input
	}{
		{
			name: "handles empty list",
			i: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero"}, {Value: "one"}, {Value: "two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{1, 3, 4},
			}},
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero"}, {Value: "one"}, {Value: "two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{1, 3, 4},
			}},
		},
		{
			name: "adds list",
			sl:   []string{"zero.one", "zero.two"},
			i: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero"}, {Value: "one"}, {Value: "two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{1, 3, 4},
			}},
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero"}, {Value: "zero.one"}, {Value: "zero.two"}, {Value: "one"}, {Value: "two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{1, 2, 3, 5, 6},
			}},
		},
		{
			name: "adds list to the front",
			sl:   []string{"zero.one", "zero.two"},
			i: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero"}, {Value: "one"}, {Value: "two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{0, 1, 3, 4},
			}},
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero.one"}, {Value: "zero.two"}, {Value: "zero"}, {Value: "one"}, {Value: "two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{0, 1, 2, 3, 5, 6},
			}},
		},
		{
			name: "adds list with offset",
			sl:   []string{"two.one", "two.two"},
			i: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero"}, {Value: "one"}, {Value: "two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{0, 1, 3, 4},
				Offset:    2,
			}},
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "zero"}, {Value: "one"}, {Value: "two"}, {Value: "two.one"}, {Value: "two.two"}, {Value: "three"}, {Value: "four"}},
				Remaining: []int{0, 1, 3, 4, 5, 6},
				Offset:    2,
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.i.PushFront(test.sl...)
			if diff := cmp.Diff(test.want, test.i, cmp.AllowUnexported(Input{}, spycommand.InputArg{})); diff != "" {
				t.Errorf("i.PushFront(%v) resulted in incorrect Input object:\n%s", test.sl, diff)
			}
		})
	}
}

func TestPop(t *testing.T) {
	input := NewInput([]string{
		"one",
		"two",
		"three",
	}, nil)

	for idx, want := range []struct {
		s  string
		ok bool
	}{
		{
			s:  "one",
			ok: true,
		},
		{
			s:  "two",
			ok: true,
		},
		{
			s:  "three",
			ok: true,
		},
		{},
		{},
	} {
		got, gotOK := input.Pop(nil)
		if want.ok != gotOK {
			t.Fatalf("Pop() (%d) returned %v for okay, want %v", idx, gotOK, want.ok)
		}
		if want.s != got {
			t.Fatalf("Pop() (%d) returned %q, want %q", idx, got, want.s)
		}
	}

	if diff := cmp.Diff(input.si.Args, []*spycommand.InputArg{{Value: "one"}, {Value: "two"}, {Value: "three"}}, cmp.AllowUnexported(spycommand.InputArg{})); diff != "" {
		t.Errorf("Input.args changed improperly (-want, +got):\n%s", diff)
	}
}

func TestSnapshots(t *testing.T) {
	input := ParseExecuteArgs([]string{"one", "two", "three"})
	var snapshots []spycommand.InputSnapshot
	var wantValues [][]string
	for _, test := range []struct {
		name               string // identifier for test output
		f                  func()
		wantSnapshot       []string
		wantUsed           []string
		wantNumRemaining   int
		wantRemaining      []string
		wantFullyProcessed bool
		wantNumSnapshots   int
		wantConvertedArgs  []string
	}{
		{
			name:              "first",
			wantSnapshot:      []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
			wantNumRemaining:  3,
			wantRemaining:     []string{"one", "two", "three"},
			wantConvertedArgs: []string{"one", "two", "three"},
			wantNumSnapshots:  1,
		},
		{
			name:              "second",
			f:                 func() { input.PushFrontAt(2, "two.one", "two.two") },
			wantSnapshot:      []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
			wantNumRemaining:  5,
			wantRemaining:     []string{"one", "two", "two.one", "two.two", "three"},
			wantNumSnapshots:  2,
			wantConvertedArgs: []string{"one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "third",
			f:                 func() { input.si.Offset = 1 },
			wantSnapshot:      []string{"two", "two.one", "two.two", "three"},
			wantNumRemaining:  4,
			wantRemaining:     []string{"two", "two.one", "two.two", "three"},
			wantNumSnapshots:  3,
			wantConvertedArgs: []string{"one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "fourth",
			f:                 func() { input.PopN(3, 0, nil, nil) },
			wantSnapshot:      []string{"three"},
			wantNumRemaining:  1,
			wantRemaining:     []string{"three"},
			wantUsed:          []string{"two", "two.one", "two.two"},
			wantNumSnapshots:  4,
			wantConvertedArgs: []string{"one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "fifth",
			f:                 func() { input.si.Offset = 0 },
			wantSnapshot:      []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "three"},
			wantNumRemaining:  2,
			wantRemaining:     []string{"one", "three"},
			wantUsed:          []string{"two", "two.one", "two.two"},
			wantNumSnapshots:  5,
			wantConvertedArgs: []string{"one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "sixth",
			f:                 func() { input.PushFront("zero.one", "zero.two", "zero.three") },
			wantSnapshot:      []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "three"},
			wantNumRemaining:  5,
			wantRemaining:     []string{"zero.one", "zero.two", "zero.three", "one", "three"},
			wantUsed:          []string{"two", "two.one", "two.two"},
			wantNumSnapshots:  6,
			wantConvertedArgs: []string{"zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "seventh",
			f:                 func() { input.si.Offset = 3 },
			wantSnapshot:      []string{"one", "three"},
			wantNumRemaining:  2,
			wantRemaining:     []string{"one", "three"},
			wantUsed:          []string{"two", "two.one", "two.two"},
			wantNumSnapshots:  7,
			wantConvertedArgs: []string{"zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "eighth",
			f:                 func() { input.Pop(nil) },
			wantSnapshot:      []string{"three"},
			wantNumRemaining:  1,
			wantRemaining:     []string{"three"},
			wantUsed:          []string{"one", "two", "two.one", "two.two"},
			wantNumSnapshots:  8,
			wantConvertedArgs: []string{"zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "ninth",
			f:                 func() { input.si.Offset = 0 },
			wantSnapshot:      []string{"negative.one", "zero.one", "zero.two", "zero.three", "three"},
			wantNumRemaining:  4,
			wantRemaining:     []string{"zero.one", "zero.two", "zero.three", "three"},
			wantUsed:          []string{"one", "two", "two.one", "two.two"},
			wantNumSnapshots:  9,
			wantConvertedArgs: []string{"zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			name:              "tenth",
			f:                 func() { input.PushFront("negative.one") },
			wantSnapshot:      []string{"negative.one", "zero.one", "zero.two", "zero.three", "three"},
			wantNumRemaining:  5,
			wantRemaining:     []string{"negative.one", "zero.one", "zero.two", "zero.three", "three"},
			wantUsed:          []string{"one", "two", "two.one", "two.two"},
			wantNumSnapshots:  10,
			wantConvertedArgs: []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			name: "11th",
			f: func() {
				input.PushBreakers(&simpleListBreaker{
					breakFunc: func(s string, d *Data) bool { return s == "three" },
				})
				input.PopNAt(1, 0, UnboundedList, nil, nil)
				input.PopBreakers(1)
			},
			wantSnapshot:      []string{"negative.one", "three"},
			wantNumRemaining:  2,
			wantRemaining:     []string{"negative.one", "three"},
			wantUsed:          []string{"zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two"},
			wantNumSnapshots:  11,
			wantConvertedArgs: []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			name:               "12th",
			f:                  func() { input.PopN(0, UnboundedList, nil, nil) },
			wantRemaining:      []string{},
			wantUsed:           []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
			wantFullyProcessed: true,
			wantNumSnapshots:   12,
			wantConvertedArgs:  []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
	} {
		if test.f != nil {
			test.f()
		}
		snapshots = append(snapshots, input.Snapshot())
		wantValues = append(wantValues, test.wantSnapshot)

		testutil.Cmp(t, fmt.Sprintf("%s: input.NumRemaining() returned incorrect value", test.name), test.wantNumRemaining, input.NumRemaining())
		testutil.Cmp(t, fmt.Sprintf("%s: input.Remaining() returned incorrect value", test.name), test.wantRemaining, input.Remaining())
		testutil.Cmp(t, fmt.Sprintf("%s: input.Used() returned incorrect value", test.name), test.wantUsed, input.Used())
		testutil.Cmp(t, fmt.Sprintf("%s: input.FullyProcessed() returned incorrect value", test.name), test.wantFullyProcessed, input.FullyProcessed())
		testutil.Cmp(t, fmt.Sprintf("%s: input.NumSnapshots() returned incorrect value", test.name), test.wantNumSnapshots, input.NumSnapshots())
		testutil.Cmp(t, fmt.Sprintf("%s: input.ConvertedArgs() returned incorrect value", test.name), test.wantConvertedArgs, input.ConvertedArgs())
	}

	var snapshotValues [][]string
	for _, s := range snapshots {
		snapshotValues = append(snapshotValues, input.GetSnapshot(s))
	}
	if diff := cmp.Diff(wantValues, snapshotValues); diff != "" {
		t.Errorf("Input.Snapshots failed with snapshot diff (-want, +got):\n%s", diff)
	}

	wantInput := &Input{&spyinput.SpyInput[InputBreaker]{
		SnapshotCount: 7,
		Args: []*spycommand.InputArg{
			{Value: "zero.one", Snapshots: snapshotsMap(1, 2, 3, 4, 5)},
			{Value: "zero.two", Snapshots: snapshotsMap(1, 2, 3, 4, 5)},
			{Value: "zero.three", Snapshots: snapshotsMap(1, 2, 3, 4, 5)},
			{Value: "one", Snapshots: snapshotsMap(1, 2, 3, 4)},
			{Value: "two", Snapshots: snapshotsMap(1, 2)},
			{Value: "two.one", Snapshots: snapshotsMap(1, 2)},
			{Value: "two.two", Snapshots: snapshotsMap(1, 2)},
			{Value: "three", Snapshots: snapshotsMap(1, 2, 3, 4, 5, 6)},
		},
	}}
	if diff := cmp.Diff(wantInput, input, cmp.AllowUnexported(Input{}, spycommand.InputArg{})); diff == "" {
		t.Errorf("Input.Snapshots failed with input diff (-want, +got):\n%s", diff)
	}
}

func TestPopN(t *testing.T) {
	for _, test := range []struct {
		name      string
		input     []string
		n         int
		optN      int
		modify    func([]*string)
		want      []string
		wantOK    bool
		wantInput *Input
		breakers  []InputBreaker
	}{
		{
			name:      "pops none",
			wantOK:    true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{}},
		},
		{
			name:   "pops none from list",
			input:  []string{"hello"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}},
				Remaining: []int{0},
			}},
		},
		{
			name:   "returns all if unbounded list",
			input:  []string{"hello", "there", "person"},
			optN:   UnboundedList,
			want:   []string{"hello", "there", "person"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
			}},
		},
		{
			name:  "breaks unbounded list at breaker",
			input: []string{"hello", "there", "person", "how", "are", "you"},
			optN:  UnboundedList,
			want:  []string{"hello", "there", "person"},
			breakers: []InputBreaker{
				&simpleListBreaker{
					breakFunc: func(s string, d *Data) bool { return s == "how" },
				},
			},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}, {Value: "how"}, {Value: "are"}, {Value: "you"}},
				Remaining: []int{3, 4, 5},
			}},
		},
		{
			name:  "breaks unbounded list at breaker with discard",
			input: []string{"hello", "there", "person", "how", "are", "you"},
			optN:  UnboundedList,
			want:  []string{"hello", "there", "person"},
			breakers: []InputBreaker{
				&simpleListBreaker{
					breakFunc: func(s string, d *Data) bool { return s == "how" },
					discard:   true,
				},
			},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}, {Value: "how"}, {Value: "are"}, {Value: "you"}},
				Remaining: []int{4, 5},
			}},
		},
		{
			name:  "pops all when no ListBreaker breaks",
			input: []string{"hello", "there", "person", "how", "are", "you"},
			optN:  UnboundedList,
			want:  []string{"hello", "there", "person", "how", "are", "you"},
			breakers: []InputBreaker{
				&simpleListBreaker{
					breakFunc: func(s string, d *Data) bool { return s == "no match" },
					discard:   true,
				},
			},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}, {Value: "how"}, {Value: "are"}, {Value: "you"}},
			}},
		},
		{
			name:   "pops requested amount from list",
			input:  []string{"hello", "there", "person"},
			n:      2,
			want:   []string{"hello", "there"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
				Remaining: []int{2},
			}},
		},
		{
			name:  "still returns values when too many requested",
			input: []string{"hello", "there", "person"},
			n:     4,
			want:  []string{"hello", "there", "person"},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
			}},
		},
		{
			name:  "modifies input",
			input: []string{"hello", "there", "person"},
			n:     2,
			want:  []string{"hello", "there"},
			modify: func(s []*string) {
				*s[0] = "goodbye"
			},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "goodbye"}, {Value: "there"}, {Value: "person"}},
				Remaining: []int{2},
			}},
		},
		{
			name:  "modifies when not enough",
			input: []string{"hello", "there", "person"},
			n:     4,
			modify: func(s []*string) {
				*s[1] = "good"
			},
			want: []string{"hello", "there", "person"},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{{Value: "hello"}, {Value: "good"}, {Value: "person"}},
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := NewInput(test.input, nil)
			gotPtrs, gotOK := input.PopN(test.n, test.optN, test.breakers, nil)
			var got []string
			for _, p := range gotPtrs {
				got = append(got, *p)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("PopN(%d, %d) returned incorrect values (-want, +got):\n%s", test.n, test.optN, diff)
			}

			if test.wantOK != gotOK {
				t.Fatalf("PopN(%d, %d) returned %v for ok, want %v", test.n, test.optN, gotOK, test.wantOK)
			}

			if test.modify != nil {
				test.modify(gotPtrs)
			}

			if diff := cmp.Diff(test.wantInput, input, cmp.AllowUnexported(Input{}, spycommand.InputArg{}), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("PopN(%d, %d) resulted in incorrect input (-want, +got):\n%s", test.n, test.optN, diff)
			}
		})
	}
}

func TestPopNOffset(t *testing.T) {
	for _, test := range []struct {
		name      string
		input     []string
		offset    int
		n         int
		optN      int
		modify    func([]*string)
		want      []string
		wantOK    bool
		wantInput *Input
	}{
		{
			name:      "pops none",
			wantOK:    true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{}},
		},
		{
			name:   "pops none when offset",
			offset: 1,
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Offset: 1,
			}},
		},
		{
			name:   "returns false if big offset and n",
			offset: 1,
			n:      1,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Offset: 1,
			}},
		},
		{
			name:   "pops none from list",
			input:  []string{"hello"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}},
				Remaining: []int{0},
			}},
		},
		{
			name:   "pops none from list with offset",
			input:  []string{"hello"},
			offset: 1,
			optN:   2,
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}},
				Remaining: []int{0},
				Offset:    1,
			}},
		},
		{
			name:   "returns all if unbounded list",
			input:  []string{"hello", "there", "person"},
			optN:   UnboundedList,
			want:   []string{"hello", "there", "person"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
			}},
		},
		{
			name:   "returns remaining if unbounded list",
			offset: 1,
			input:  []string{"hello", "there", "person"},
			optN:   UnboundedList,
			want:   []string{"there", "person"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
				Remaining: []int{0},
				Offset:    1,
			}},
		},
		{
			name:   "pops requested amount from list",
			input:  []string{"hello", "there", "person"},
			n:      2,
			want:   []string{"hello", "there"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
				Remaining: []int{2},
			}},
		},
		{
			name:   "pops requested amount from list with offset",
			input:  []string{"hello", "there", "general", "kenobi"},
			offset: 1,
			n:      2,
			want:   []string{"there", "general"},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "general"}, {Value: "kenobi"}},
				Remaining: []int{0, 3},
				Offset:    1,
			}},
		},
		{
			name:  "still returns values when too many requested",
			input: []string{"hello", "there", "person"},
			n:     4,
			want:  []string{"hello", "there", "person"},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
			}},
		},
		{
			name:   "still returns values when too many requested with offset",
			input:  []string{"hello", "there", "person"},
			offset: 2,
			n:      4,
			want:   []string{"person"},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "person"}},
				Remaining: []int{0, 1},
				Offset:    2,
			}},
		},
		{
			name:  "modifies input",
			input: []string{"hello", "there", "person"},
			n:     2,
			want:  []string{"hello", "there"},
			modify: func(s []*string) {
				*s[0] = "goodbye"
			},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "goodbye"}, {Value: "there"}, {Value: "person"}},
				Remaining: []int{2},
			}},
		},
		{
			name:   "modifies input with offset",
			input:  []string{"hello", "there", "good", "sir"},
			n:      2,
			offset: 2,
			want:   []string{"good", "sir"},
			modify: func(s []*string) {
				*s[0] = "general"
				*s[1] = "kenobi"
			},
			wantOK: true,
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "general"}, {Value: "kenobi"}},
				Remaining: []int{0, 1},
				Offset:    2,
			}},
		},
		{
			name:  "modifies when not enough",
			input: []string{"hello", "there", "person"},
			n:     4,
			modify: func(s []*string) {
				*s[1] = "good"
			},
			want: []string{"hello", "there", "person"},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{{Value: "hello"}, {Value: "good"}, {Value: "person"}},
			}},
		},
		{
			name:   "modifies when not enough with offset",
			input:  []string{"hello", "there", "general", "kenobi"},
			n:      3,
			offset: 3,
			modify: func(s []*string) {
				*s[0] = "motors"
			},
			want: []string{"kenobi"},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "hello"}, {Value: "there"}, {Value: "general"}, {Value: "motors"}},
				Remaining: []int{0, 1, 2},
				Offset:    3,
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := NewInput(test.input, nil)
			input.si.Offset = test.offset
			gotPtrs, gotOK := input.PopN(test.n, test.optN, nil, nil)
			var got []string
			for _, p := range gotPtrs {
				got = append(got, *p)
			}
			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("PopN(%d, %d) returned incorrect values (-want, +got):\n%s", test.n, test.optN, diff)
			}

			if test.wantOK != gotOK {
				t.Fatalf("PopN(%d, %d) returned %v for ok, want %v", test.n, test.optN, gotOK, test.wantOK)
			}

			if test.modify != nil {
				test.modify(gotPtrs)
			}

			if diff := cmp.Diff(test.wantInput, input, cmp.AllowUnexported(Input{}, spycommand.InputArg{}), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("PopN(%d, %d) resulted in incorrect input (-want, +got):\n%s", test.n, test.optN, diff)
			}
		})
	}
}

func TestParseCompLine(t *testing.T) {
	for _, test := range []struct {
		name   string
		input  string
		ptArgs []string
		want   *Input
	}{
		{
			name: "handles empty input",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: ""}},
				Remaining: []int{0},
			}},
		},
		{
			name:  "handles empty command",
			input: "cmd",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: ""}},
				Remaining: []int{0},
			}},
		},
		{
			name:  "converts single argument",
			input: "cmd one",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "one"}},
				Remaining: []int{0},
			}},
		},
		{
			name:   "includes passthrough args",
			input:  "cmd one two",
			ptArgs: []string{"nOne", "zero"},
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "nOne"},
					{Value: "zero"},
					{Value: "one"},
					{Value: "two"},
				},
				Remaining: []int{0, 1, 2, 3},
			}},
		},
		{
			name:  "converts single argument with quote",
			input: `cmd "one`,
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "one"}},
				Delimiter: runePtr('"'),
				Remaining: []int{0},
			}},
		},
		{
			name:  "converts quoted argument",
			input: `cmd "one"`,
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "one"}},
				Remaining: []int{0},
			}},
		},
		{
			name:  "ignores last argument if quote",
			input: `cmd one "`,
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "one"}, {Value: ""}},
				Delimiter: runePtr('"'),
				Remaining: []int{0, 1},
			}},
		},
		{
			name:  "space character",
			input: "cmd ab cd",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "ab"},
					{Value: "cd"},
				},
				Remaining: []int{0, 1},
			}},
		},
		{
			name:  "multiple space characters",
			input: "cmd ab cd  ef       gh",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "ab"},
					{Value: "cd"},
					{Value: "ef"},
					{Value: "gh"},
				},
				Remaining: []int{0, 1, 2, 3},
			}},
		},
		{
			name:  "quotation between words",
			input: "cmd a'b c'd",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "ab cd"}},
				Remaining: []int{0},
			}},
		},
		{
			name:  "escaped space character",
			input: `cmd ab\ cd`,
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "ab cd"}},
				Remaining: []int{0},
			}},
		},
		{
			name:  "escaped space character between words",
			input: "cmd ab\\ cd",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args:      []*spycommand.InputArg{{Value: "ab cd"}},
				Remaining: []int{0},
			}},
		},
		{
			name:  "ending backslash in word",
			input: "cmd ab cd\\",
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: `ab`},
					{Value: `cd\`},
				},
				Remaining: []int{0, 1},
			}},
		},
		{
			name:  "escaped character to start word",
			input: `cmd ab \cd`,
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "ab"},
					{Value: `\cd`},
				},
				Remaining: []int{0, 1},
			}},
		},
		{
			name:  "end with backslash while in word",
			input: `cmd ab cd ef\`,
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "ab"},
					{Value: `cd`},
					{Value: `ef\`},
				},
				Remaining: []int{0, 1, 2},
			}},
		},
		{
			name:  "end with backslash while not in word",
			input: `cmd ab cd \`,
			want: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "ab"},
					{Value: `cd`},
					{Value: `\`},
				},
				Remaining: []int{0, 1, 2},
			}},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ParseCompLine(test.input, test.ptArgs...)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(Input{}, spycommand.InputArg{})); diff != "" {
				t.Fatalf("ParseCompLine(%v) created incorrect args (-want, +got):\n%s", test.input, diff)
			}
		})
	}
}

func TestPopAtAndPeekAt(t *testing.T) {
	for _, test := range []struct {
		name       string
		input      *Input
		idx        int
		wantPeek   string
		wantPeekOK bool
		want       string
		wantOK     bool
		wantInput  *Input
	}{
		{
			name:      "empty input",
			input:     &Input{&spyinput.SpyInput[InputBreaker]{}},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{}},
		},
		{
			name: "non-empty input, but out of range",
			idx:  2,
			input: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "abc"},
					{Value: "def"},
				},
				Remaining: []int{0, 1},
			}},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "abc"},
					{Value: "def"},
				},
				Remaining: []int{0, 1},
			}},
			wantPeek:   "abc",
			wantPeekOK: true,
		},
		{
			name: "pops first element",
			idx:  0,
			input: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "abc"},
					{Value: "def"},
				},
				Remaining: []int{0, 1},
			}},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "abc"},
					{Value: "def"},
				},
				Remaining: []int{1},
			}},
			wantPeek:   "abc",
			wantPeekOK: true,
			want:       "abc",
			wantOK:     true,
		},
		{
			name: "pops second element",
			idx:  1,
			input: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "abc"},
					{Value: "def"},
				},
				Remaining: []int{0, 1},
			}},
			wantInput: &Input{&spyinput.SpyInput[InputBreaker]{
				Args: []*spycommand.InputArg{
					{Value: "abc"},
					{Value: "def"},
				},
				Remaining: []int{0},
			}},
			wantPeek:   "abc",
			wantPeekOK: true,
			want:       "def",
			wantOK:     true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			peekGot, peekGotOK := test.input.Peek()
			testutil.Cmp(t, "Peek() returned invalid string value", test.wantPeek, peekGot)
			testutil.Cmp(t, "Peek() returned invalid OK value", test.wantPeekOK, peekGotOK)

			peekAtGot, peekAtGotOK := test.input.PeekAt(test.idx)
			testutil.Cmp(t, fmt.Sprintf("PeekAt(%d) returned invalid string value", test.idx), test.want, peekAtGot)
			testutil.Cmp(t, fmt.Sprintf("PeekAt(%d) returned invalid OK value", test.idx), test.wantOK, peekAtGotOK)

			popAtGot, popAtGotOK := test.input.PopAt(test.idx, nil)
			testutil.Cmp(t, fmt.Sprintf("PopAt(%d) returned invalid string value", test.idx), test.want, popAtGot)
			testutil.Cmp(t, fmt.Sprintf("PopAt(%d) returned invalid OK value", test.idx), test.wantOK, popAtGotOK)

			testutil.Cmp(t, fmt.Sprintf("PopAt(%d) resulted in incorrect input", test.idx), test.wantInput, test.input, cmp.AllowUnexported(Input{}, spycommand.InputArg{}))
		})
	}
}

func snapshotsMap(iss ...spycommand.InputSnapshot) map[spycommand.InputSnapshot]bool {
	if len(iss) == 0 {
		return nil
	}
	m := map[spycommand.InputSnapshot]bool{}
	for _, is := range iss {
		m[is] = true
	}
	return m
}

type simpleListBreaker struct {
	breakFunc func(string, *Data) bool
	discard   bool
}

func (slb *simpleListBreaker) Break(s string, d *Data) bool {
	return slb.breakFunc != nil && slb.breakFunc(s, d)
}

func (slb *simpleListBreaker) DiscardBreak(s string, d *Data) bool {
	return slb.discard
}
