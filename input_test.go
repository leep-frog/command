package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func runePtr(r rune) *rune {
	return &r
}

func TestParseArgs(t *testing.T) {
	for _, test := range []struct {
		name  string
		input []string
		want  *InputArgs
	}{
		{
			name: "handles empty input",
			want: &InputArgs{},
		},
		{
			name:  "converts single argument",
			input: []string{"one"},
			want: &InputArgs{
				args: []string{"one"},
			},
		},
		{
			name:  "converts single argument with quote",
			input: []string{`"one`},
			want: &InputArgs{
				args:      []string{"one"},
				delimiter: runePtr('"'),
			},
		},
		{
			name:  "converts quoted argument",
			input: []string{`"one"`},
			want: &InputArgs{
				args: []string{"one"},
			},
		},
		{
			name:  "ignores last argument if quote",
			input: []string{`one`, `"`},
			want: &InputArgs{
				args:      []string{"one", ""},
				delimiter: runePtr('"'),
			},
		},
		{
			name:  "escaped space character",
			input: []string{`ab\ cd`},
			want: &InputArgs{
				args: []string{"ab cd"},
			},
		},
		{
			name:  "escaped space character between words",
			input: []string{"ab\\", "cd"},
			want: &InputArgs{
				args: []string{"ab cd"},
			},
		},
		{
			name:  "quotation between words",
			input: []string{"a'b", "c'd"},
			want: &InputArgs{
				args: []string{"ab cd"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ParseArgs(test.input)
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(InputArgs{})); diff != "" {
				t.Fatalf("ParseArgs(%v) created incorrect args (-want, +got):\n%s", test.input, diff)
			}
		})
	}
}
