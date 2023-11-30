package spycommand

import (
	"fmt"
	"testing"

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
