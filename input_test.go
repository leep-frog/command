package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
			i: &Input{
				args:      []*inputArg{{value: "zero"}, {value: "one"}, {value: "two"}, {value: "three"}, {value: "four"}},
				remaining: []int{1, 3, 4},
			},
			want: &Input{
				args:      []*inputArg{{value: "zero"}, {value: "one"}, {value: "two"}, {value: "three"}, {value: "four"}},
				remaining: []int{1, 3, 4},
			},
		},
		{
			name: "adds list",
			sl:   []string{"zero.one", "zero.two"},
			i: &Input{
				args:      []*inputArg{{value: "zero"}, {value: "one"}, {value: "two"}, {value: "three"}, {value: "four"}},
				remaining: []int{1, 3, 4},
			},
			want: &Input{
				args:      []*inputArg{{value: "zero"}, {value: "zero.one"}, {value: "zero.two"}, {value: "one"}, {value: "two"}, {value: "three"}, {value: "four"}},
				remaining: []int{1, 2, 3, 5, 6},
			},
		},
		{
			name: "adds list to the front",
			sl:   []string{"zero.one", "zero.two"},
			i: &Input{
				args:      []*inputArg{{value: "zero"}, {value: "one"}, {value: "two"}, {value: "three"}, {value: "four"}},
				remaining: []int{0, 1, 3, 4},
			},
			want: &Input{
				args:      []*inputArg{{value: "zero.one"}, {value: "zero.two"}, {value: "zero"}, {value: "one"}, {value: "two"}, {value: "three"}, {value: "four"}},
				remaining: []int{0, 1, 2, 3, 5, 6},
			},
		},
		{
			name: "adds list with offset",
			sl:   []string{"two.one", "two.two"},
			i: &Input{
				args:      []*inputArg{{value: "zero"}, {value: "one"}, {value: "two"}, {value: "three"}, {value: "four"}},
				remaining: []int{0, 1, 3, 4},
				offset:    2,
			},
			want: &Input{
				args:      []*inputArg{{value: "zero"}, {value: "one"}, {value: "two"}, {value: "two.one"}, {value: "two.two"}, {value: "three"}, {value: "four"}},
				remaining: []int{0, 1, 3, 4, 5, 6},
				offset:    2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.i.PushFront(test.sl...)
			if diff := cmp.Diff(test.want, test.i, cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
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
		got, gotOK := input.Pop()
		if want.ok != gotOK {
			t.Fatalf("Pop() (%d) returned %v for okay, want %v", idx, gotOK, want.ok)
		}
		if want.s != got {
			t.Fatalf("Pop() (%d) returned %q, want %q", idx, got, want.s)
		}
	}

	if diff := cmp.Diff(input.args, []*inputArg{{value: "one"}, {value: "two"}, {value: "three"}}, cmp.AllowUnexported(inputArg{})); diff != "" {
		t.Errorf("Input.args changed improperly (-want, +got):\n%s", diff)
	}
}

func TestSnapshots(t *testing.T) {
	input := ParseExecuteArgs([]string{"one", "two", "three"})
	var snapshots []inputSnapshot
	var wantValues [][]string
	for _, test := range []struct {
		f            func()
		wantSnapshot []string
	}{
		{
			wantSnapshot: []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			f:            func() { input.PushFrontAt(2, "two.one", "two.two") },
			wantSnapshot: []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "two", "two.one", "two.two", "three"},
		},
		{
			f:            func() { input.offset = 1 },
			wantSnapshot: []string{"two", "two.one", "two.two", "three"},
		},
		{
			f:            func() { input.PopN(3, 0, nil) },
			wantSnapshot: []string{"three"},
		},
		{
			f:            func() { input.offset = 0 },
			wantSnapshot: []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "three"},
		},
		{
			f:            func() { input.PushFront("zero.one", "zero.two", "zero.three") },
			wantSnapshot: []string{"negative.one", "zero.one", "zero.two", "zero.three", "one", "three"},
		},
		{
			f:            func() { input.offset = 3 },
			wantSnapshot: []string{"one", "three"},
		},
		{
			f:            func() { input.Pop() },
			wantSnapshot: []string{"three"},
		},
		{
			f:            func() { input.offset = 0 },
			wantSnapshot: []string{"negative.one", "zero.one", "zero.two", "zero.three", "three"},
		},
		{
			f:            func() { input.PushFront("negative.one") },
			wantSnapshot: []string{"negative.one", "zero.one", "zero.two", "zero.three", "three"},
		},
		{
			f:            func() { input.PopN(1, 2, nil) },
			wantSnapshot: []string{"zero.three", "three"},
		},
		{
			f: func() { input.PopN(0, 100, nil) },
		},
	} {
		if test.f != nil {
			test.f()
		}
		snapshots = append(snapshots, input.Snapshot())
		wantValues = append(wantValues, test.wantSnapshot)
	}

	var snapshotValues [][]string
	for _, s := range snapshots {
		snapshotValues = append(snapshotValues, input.GetSnapshot(s))
	}
	if diff := cmp.Diff(wantValues, snapshotValues); diff != "" {
		t.Errorf("Input.Snapshots failed with snapshot diff (-want, +got):\n%s", diff)
	}

	wantInput := &Input{
		snapshotCount: 7,
		args: []*inputArg{
			{value: "zero.one", snapshots: snapshotsMap(1, 2, 3, 4, 5)},
			{value: "zero.two", snapshots: snapshotsMap(1, 2, 3, 4, 5)},
			{value: "zero.three", snapshots: snapshotsMap(1, 2, 3, 4, 5)},
			{value: "one", snapshots: snapshotsMap(1, 2, 3, 4)},
			{value: "two", snapshots: snapshotsMap(1, 2)},
			{value: "two.one", snapshots: snapshotsMap(1, 2)},
			{value: "two.two", snapshots: snapshotsMap(1, 2)},
			{value: "three", snapshots: snapshotsMap(1, 2, 3, 4, 5, 6)},
		},
	}
	if diff := cmp.Diff(wantInput, input, cmp.AllowUnexported(Input{}, inputArg{})); diff == "" {
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
		breaker   *ListBreaker
	}{
		{
			name:      "pops none",
			wantOK:    true,
			wantInput: &Input{},
		},
		{
			name:   "pops none from list",
			input:  []string{"hello"},
			wantOK: true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}},
				remaining: []int{0},
			},
		},
		{
			name:   "returns all if unbounded list",
			input:  []string{"hello", "there", "person"},
			optN:   UnboundedList,
			want:   []string{"hello", "there", "person"},
			wantOK: true,
			wantInput: &Input{
				args: []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
			},
		},
		{
			name:    "breaks unbounded list at breaker",
			input:   []string{"hello", "there", "person", "how", "are", "you"},
			optN:    UnboundedList,
			want:    []string{"hello", "there", "person"},
			breaker: ListUntilSymbol("how"),
			wantOK:  true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}, {value: "how"}, {value: "are"}, {value: "you"}},
				remaining: []int{3, 4, 5},
			},
		},
		{
			name:    "breaks unbounded list at breaker with discard",
			input:   []string{"hello", "there", "person", "how", "are", "you"},
			optN:    UnboundedList,
			want:    []string{"hello", "there", "person"},
			breaker: ListUntilSymbol("how", DiscardBreaker()),
			wantOK:  true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}, {value: "how"}, {value: "are"}, {value: "you"}},
				remaining: []int{4, 5},
			},
		},
		{
			name:    "pops all when no ListBreaker breaks",
			input:   []string{"hello", "there", "person", "how", "are", "you"},
			optN:    UnboundedList,
			want:    []string{"hello", "there", "person", "how", "are", "you"},
			breaker: ListUntilSymbol("no match"),
			wantOK:  true,
			wantInput: &Input{
				args: []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}, {value: "how"}, {value: "are"}, {value: "you"}},
			},
		},
		{
			name:   "pops requested amount from list",
			input:  []string{"hello", "there", "person"},
			n:      2,
			want:   []string{"hello", "there"},
			wantOK: true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
				remaining: []int{2},
			},
		},
		{
			name:  "still returns values when too many requested",
			input: []string{"hello", "there", "person"},
			n:     4,
			want:  []string{"hello", "there", "person"},
			wantInput: &Input{
				args: []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
			},
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
			wantInput: &Input{
				args:      []*inputArg{{value: "goodbye"}, {value: "there"}, {value: "person"}},
				remaining: []int{2},
			},
		},
		{
			name:  "modifies when not enough",
			input: []string{"hello", "there", "person"},
			n:     4,
			modify: func(s []*string) {
				*s[1] = "good"
			},
			want: []string{"hello", "there", "person"},
			wantInput: &Input{
				args: []*inputArg{{value: "hello"}, {value: "good"}, {value: "person"}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := NewInput(test.input, nil)
			gotPtrs, gotOK := input.PopN(test.n, test.optN, test.breaker)
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

			if diff := cmp.Diff(test.wantInput, input, cmp.AllowUnexported(Input{}, inputArg{}), cmpopts.EquateEmpty()); diff != "" {
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
			wantInput: &Input{},
		},
		{
			name:   "pops none when offset",
			offset: 1,
			wantOK: true,
			wantInput: &Input{
				offset: 1,
			},
		},
		{
			name:   "returns false if big offset and n",
			offset: 1,
			n:      1,
			wantInput: &Input{
				offset: 1,
			},
		},
		{
			name:   "pops none from list",
			input:  []string{"hello"},
			wantOK: true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}},
				remaining: []int{0},
			},
		},
		{
			name:   "pops none from list with offset",
			input:  []string{"hello"},
			offset: 1,
			optN:   2,
			wantOK: true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}},
				remaining: []int{0},
				offset:    1,
			},
		},
		{
			name:   "returns all if unbounded list",
			input:  []string{"hello", "there", "person"},
			optN:   UnboundedList,
			want:   []string{"hello", "there", "person"},
			wantOK: true,
			wantInput: &Input{
				args: []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
			},
		},
		{
			name:   "returns remaining if unbounded list",
			offset: 1,
			input:  []string{"hello", "there", "person"},
			optN:   UnboundedList,
			want:   []string{"there", "person"},
			wantOK: true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
				remaining: []int{0},
				offset:    1,
			},
		},
		{
			name:   "pops requested amount from list",
			input:  []string{"hello", "there", "person"},
			n:      2,
			want:   []string{"hello", "there"},
			wantOK: true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
				remaining: []int{2},
			},
		},
		{
			name:   "pops requested amount from list with offset",
			input:  []string{"hello", "there", "general", "kenobi"},
			offset: 1,
			n:      2,
			want:   []string{"there", "general"},
			wantOK: true,
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "general"}, {value: "kenobi"}},
				remaining: []int{0, 3},
				offset:    1,
			},
		},
		{
			name:  "still returns values when too many requested",
			input: []string{"hello", "there", "person"},
			n:     4,
			want:  []string{"hello", "there", "person"},
			wantInput: &Input{
				args: []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
			},
		},
		{
			name:   "still returns values when too many requested with offset",
			input:  []string{"hello", "there", "person"},
			offset: 2,
			n:      4,
			want:   []string{"person"},
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "person"}},
				remaining: []int{0, 1},
				offset:    2,
			},
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
			wantInput: &Input{
				args:      []*inputArg{{value: "goodbye"}, {value: "there"}, {value: "person"}},
				remaining: []int{2},
			},
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
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "general"}, {value: "kenobi"}},
				remaining: []int{0, 1},
				offset:    2,
			},
		},
		{
			name:  "modifies when not enough",
			input: []string{"hello", "there", "person"},
			n:     4,
			modify: func(s []*string) {
				*s[1] = "good"
			},
			want: []string{"hello", "there", "person"},
			wantInput: &Input{
				args: []*inputArg{{value: "hello"}, {value: "good"}, {value: "person"}},
			},
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
			wantInput: &Input{
				args:      []*inputArg{{value: "hello"}, {value: "there"}, {value: "general"}, {value: "motors"}},
				remaining: []int{0, 1, 2},
				offset:    3,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := NewInput(test.input, nil)
			input.offset = test.offset
			gotPtrs, gotOK := input.PopN(test.n, test.optN, nil)
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

			if diff := cmp.Diff(test.wantInput, input, cmp.AllowUnexported(Input{}, inputArg{}), cmpopts.EquateEmpty()); diff != "" {
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
			want: &Input{
				args:      []*inputArg{{value: ""}},
				remaining: []int{0},
			},
		},
		{
			name:  "handles empty command",
			input: "cmd",
			want: &Input{
				args:      []*inputArg{{value: ""}},
				remaining: []int{0},
			},
		},
		{
			name:  "converts single argument",
			input: "cmd one",
			want: &Input{
				args:      []*inputArg{{value: "one"}},
				remaining: []int{0},
			},
		},
		{
			name:   "includes passthrough args",
			input:  "cmd one two",
			ptArgs: []string{"nOne", "zero"},
			want: &Input{
				args: []*inputArg{
					{value: "nOne"},
					{value: "zero"},
					{value: "one"},
					{value: "two"},
				},
				remaining: []int{0, 1, 2, 3},
			},
		},
		{
			name:  "converts single argument with quote",
			input: `cmd "one`,
			want: &Input{
				args:      []*inputArg{{value: "one"}},
				delimiter: runePtr('"'),
				remaining: []int{0},
			},
		},
		{
			name:  "converts quoted argument",
			input: `cmd "one"`,
			want: &Input{
				args:      []*inputArg{{value: "one"}},
				remaining: []int{0},
			},
		},
		{
			name:  "ignores last argument if quote",
			input: `cmd one "`,
			want: &Input{
				args:      []*inputArg{{value: "one"}, {value: ""}},
				delimiter: runePtr('"'),
				remaining: []int{0, 1},
			},
		},
		{
			name:  "space character",
			input: "cmd ab cd",
			want: &Input{
				args: []*inputArg{
					{value: "ab"},
					{value: "cd"},
				},
				remaining: []int{0, 1},
			},
		},
		{
			name:  "multiple space characters",
			input: "cmd ab cd  ef       gh",
			want: &Input{
				args: []*inputArg{
					{value: "ab"},
					{value: "cd"},
					{value: "ef"},
					{value: "gh"},
				},
				remaining: []int{0, 1, 2, 3},
			},
		},
		{
			name:  "quotation between words",
			input: "cmd a'b c'd",
			want: &Input{
				args:      []*inputArg{{value: "ab cd"}},
				remaining: []int{0},
			},
		},
		{
			name:  "escaped space character",
			input: `cmd ab\ cd`,
			want: &Input{
				args:      []*inputArg{{value: "ab cd"}},
				remaining: []int{0},
			},
		},
		{
			name:  "escaped space character between words",
			input: "cmd ab\\ cd",
			want: &Input{
				args:      []*inputArg{{value: "ab cd"}},
				remaining: []int{0},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ParseCompLine(test.input, test.ptArgs)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(Input{}, inputArg{})); diff != "" {
				t.Fatalf("ParseCompLine(%v) created incorrect args (-want, +got):\n%s", test.input, diff)
			}
		})
	}
}
