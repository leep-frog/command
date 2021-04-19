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
				args:      []string{"zero", "one", "two", "three", "four"},
				remaining: []int{1, 3, 4},
			},
			want: &Input{
				args:      []string{"zero", "one", "two", "three", "four"},
				remaining: []int{1, 3, 4},
			},
		},
		{
			name: "adds list",
			sl:   []string{"zero.one", "zero.two"},
			i: &Input{
				args:      []string{"zero", "one", "two", "three", "four"},
				remaining: []int{1, 3, 4},
			},
			want: &Input{
				args:      []string{"zero", "zero.one", "zero.two", "one", "two", "three", "four"},
				remaining: []int{1, 2, 3, 5, 6},
			},
		},
		{
			name: "adds list to the front",
			sl:   []string{"zero.one", "zero.two"},
			i: &Input{
				args:      []string{"zero", "one", "two", "three", "four"},
				remaining: []int{0, 1, 3, 4},
			},
			want: &Input{
				args:      []string{"zero.one", "zero.two", "zero", "one", "two", "three", "four"},
				remaining: []int{0, 1, 2, 3, 5, 6},
			},
		},
		{
			name: "adds list with offset",
			sl:   []string{"two.one", "two.two"},
			i: &Input{
				args:      []string{"zero", "one", "two", "three", "four"},
				remaining: []int{0, 1, 3, 4},
				offset:    2,
			},
			want: &Input{
				args:      []string{"zero", "one", "two", "two.one", "two.two", "three", "four"},
				remaining: []int{0, 1, 3, 4, 5, 6},
				offset:    2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.i.PushFront(test.sl...)
			if diff := cmp.Diff(test.want, test.i, cmp.AllowUnexported(Input{})); diff != "" {
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

	if diff := cmp.Diff(input.args, []string{"one", "two", "three"}); diff != "" {
		t.Errorf("Input.args changed improperly (-want, +got):\n%s", diff)
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
				args:      []string{"hello"},
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
				args: []string{"hello", "there", "person"},
			},
		},
		{
			name:   "pops requested amount from list",
			input:  []string{"hello", "there", "person"},
			n:      2,
			want:   []string{"hello", "there"},
			wantOK: true,
			wantInput: &Input{
				args:      []string{"hello", "there", "person"},
				remaining: []int{2},
			},
		},
		{
			name:  "still returns values when too many requested",
			input: []string{"hello", "there", "person"},
			n:     4,
			want:  []string{"hello", "there", "person"},
			wantInput: &Input{
				args: []string{"hello", "there", "person"},
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
				args:      []string{"goodbye", "there", "person"},
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
				args: []string{"hello", "good", "person"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := NewInput(test.input, nil)
			gotPtrs, gotOK := input.PopN(test.n, test.optN)
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

			if diff := cmp.Diff(test.wantInput, input, cmp.AllowUnexported(Input{}), cmpopts.EquateEmpty()); diff != "" {
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
				args:      []string{"hello"},
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
				args:      []string{"hello"},
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
				args: []string{"hello", "there", "person"},
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
				args:      []string{"hello", "there", "person"},
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
				args:      []string{"hello", "there", "person"},
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
				args:      []string{"hello", "there", "general", "kenobi"},
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
				args: []string{"hello", "there", "person"},
			},
		},
		{
			name:   "still returns values when too many requested with offset",
			input:  []string{"hello", "there", "person"},
			offset: 2,
			n:      4,
			want:   []string{"person"},
			wantInput: &Input{
				args:      []string{"hello", "there", "person"},
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
				args:      []string{"goodbye", "there", "person"},
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
				args:      []string{"hello", "there", "general", "kenobi"},
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
				args: []string{"hello", "good", "person"},
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
				args:      []string{"hello", "there", "general", "motors"},
				remaining: []int{0, 1, 2},
				offset:    3,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := NewInput(test.input, nil)
			input.offset = test.offset
			gotPtrs, gotOK := input.PopN(test.n, test.optN)
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

			if diff := cmp.Diff(test.wantInput, input, cmp.AllowUnexported(Input{}), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("PopN(%d, %d) resulted in incorrect input (-want, +got):\n%s", test.n, test.optN, diff)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	for _, test := range []struct {
		name  string
		input []string
		want  *Input
	}{
		{
			name: "handles empty input",
			want: &Input{},
		},
		{
			name:  "converts single argument",
			input: []string{"one"},
			want: &Input{
				args:      []string{"one"},
				remaining: []int{0},
			},
		},
		{
			name:  "converts single argument with quote",
			input: []string{`"one`},
			want: &Input{
				args:      []string{"one"},
				delimiter: runePtr('"'),
				remaining: []int{0},
			},
		},
		{
			name:  "converts quoted argument",
			input: []string{`"one"`},
			want: &Input{
				args:      []string{"one"},
				remaining: []int{0},
			},
		},
		{
			name:  "ignores last argument if quote",
			input: []string{`one`, `"`},
			want: &Input{
				args:      []string{"one", ""},
				delimiter: runePtr('"'),
				remaining: []int{0, 1},
			},
		},
		{
			name:  "escaped space character",
			input: []string{`ab\ cd`},
			want: &Input{
				args:      []string{"ab cd"},
				remaining: []int{0},
			},
		},
		{
			name:  "escaped space character between words",
			input: []string{"ab\\", "cd"},
			want: &Input{
				args:      []string{"ab cd"},
				remaining: []int{0},
			},
		},
		{
			name:  "quotation between words",
			input: []string{"a'b", "c'd"},
			want: &Input{
				args:      []string{"ab cd"},
				remaining: []int{0},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ParseArgs(test.input)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(Input{})); diff != "" {
				t.Fatalf("ParseArgs(%v) created incorrect args (-want, +got):\n%s", test.input, diff)
			}
		})
	}
}
