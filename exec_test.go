package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name     string
		contents []string
		want     []string
		wantErr  error
	}{
		{
			name: "works with empty command",
		},
		{
			name: "works with simple echo commands",
			contents: []string{
				"echo one two",
				"echo three",
				"echo four",
			},
			want: []string{
				"one two",
				"three",
				"four",
				"",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Run(test.contents)
			if diff := cmp.Diff(test.wantErr, err); diff != "" {
				t.Errorf("Run(%v) returned incorrect error (-want, +got):\n%s", test.contents, diff)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Run(%v) returned incorrect output (-want, +got):\n%s", test.contents, diff)
			}
		})
	}
}
