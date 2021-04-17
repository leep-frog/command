package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func runePtr(r rune) *rune {
	return &r
}

func TestPop(t *testing.T) {
	input := &Input{
		args: []string{
			"one",
			"two",
			"three",
		},
	}

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
		input     *Input
		n         int
		optN      int
		modify    func([]string)
		want      []string
		wantOK    bool
		wantInput *Input
	}{
		{
			name:      "pops none",
			input:     &Input{},
			wantOK:    true,
			wantInput: &Input{},
		},
		{
			name: "pops none from list",
			input: &Input{
				args: []string{"hello"},
			},
			wantOK: true,
			wantInput: &Input{
				args: []string{"hello"},
			},
		},
		{
			name: "returns all if unbounded list",
			input: &Input{
				args: []string{"hello", "there", "person"},
			},
			optN:   UnboundedList,
			want:   []string{"hello", "there", "person"},
			wantOK: true,
			wantInput: &Input{
				args: []string{"hello", "there", "person"},
				pos:  3,
			},
		},
		{
			name: "pops requested amount from list",
			input: &Input{
				args: []string{"hello", "there", "person"},
			},
			n:      2,
			want:   []string{"hello", "there"},
			wantOK: true,
			wantInput: &Input{
				args: []string{"hello", "there", "person"},
				pos:  2,
			},
		},
		{
			name: "still returns values when too many requested",
			input: &Input{
				args: []string{"hello", "there", "person"},
			},
			n:    4,
			want: []string{"hello", "there", "person"},
			wantInput: &Input{
				args: []string{"hello", "there", "person"},
				pos:  3,
			},
		},
		{
			name: "modifies input",
			input: &Input{
				args: []string{"hello", "there", "person"},
			},
			n:    2,
			want: []string{"hello", "there"},
			modify: func(s []string) {
				s[0] = "goodbye"
			},
			wantOK: true,
			wantInput: &Input{
				args: []string{"goodbye", "there", "person"},
				pos:  2,
			},
		},
		{
			name: "modifies when not enough",
			input: &Input{
				args: []string{"hello", "there", "person"},
			},
			n: 4,
			modify: func(s []string) {
				s[1] = "good"
			},
			want: []string{"hello", "there", "person"},
			wantInput: &Input{
				args: []string{"hello", "good", "person"},
				pos:  3,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, gotOK := test.input.PopN(test.n, test.optN)
			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("PopN(%d, %d) returned incorrect values (-want, +got):\n%s", test.n, test.optN, diff)
			}

			if test.wantOK != gotOK {
				t.Fatalf("PopN(%d, %d) returned %v for ok, want %v", test.n, test.optN, gotOK, test.wantOK)
			}

			if test.modify != nil {
				test.modify(got)
			}

			if diff := cmp.Diff(test.wantInput, test.input, cmp.AllowUnexported(Input{})); diff != "" {
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
				args: []string{"one"},
			},
		},
		{
			name:  "converts single argument with quote",
			input: []string{`"one`},
			want: &Input{
				args:      []string{"one"},
				delimiter: runePtr('"'),
			},
		},
		{
			name:  "converts quoted argument",
			input: []string{`"one"`},
			want: &Input{
				args: []string{"one"},
			},
		},
		{
			name:  "ignores last argument if quote",
			input: []string{`one`, `"`},
			want: &Input{
				args:      []string{"one", ""},
				delimiter: runePtr('"'),
			},
		},
		{
			name:  "escaped space character",
			input: []string{`ab\ cd`},
			want: &Input{
				args: []string{"ab cd"},
			},
		},
		{
			name:  "escaped space character between words",
			input: []string{"ab\\", "cd"},
			want: &Input{
				args: []string{"ab cd"},
			},
		},
		{
			name:  "quotation between words",
			input: []string{"a'b", "c'd"},
			want: &Input{
				args: []string{"ab cd"},
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
