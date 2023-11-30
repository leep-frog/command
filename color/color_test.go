package color

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/testutil"
)

func TestFormat(t *testing.T) {
	for _, test := range []struct {
		name      string
		format    *Format
		wantCalls [][]interface{}
	}{
		{
			name:   "Background color",
			format: Background(3),
			wantCalls: [][]interface{}{{
				"setab", "3",
			}},
		},
		{
			name:   "Text color",
			format: Text(6),
			wantCalls: [][]interface{}{{
				"setaf", "6",
			}},
		},
		{
			name:   "Bold",
			format: Bold(),
			wantCalls: [][]interface{}{{
				"bold",
			}},
		},
		{
			name:   "Underline",
			format: Underline(),
			wantCalls: [][]interface{}{{
				"smul",
			}},
		},
		{
			name:   "End Unerline",
			format: EndUnderline(),
			wantCalls: [][]interface{}{{
				"rmul",
			}},
		},
		{
			name:   "Reset",
			format: Reset(),
			wantCalls: [][]interface{}{{
				"reset",
			}},
		},
		{
			name:   "Init",
			format: Init(),
			wantCalls: [][]interface{}{{
				"init",
			}},
		},
		{
			name: "Multi format",
			format: MultiFormat(
				Text(5),
				Bold(),
				Underline(),
				Background(7),
				Text(11),
			),
			wantCalls: [][]interface{}{{
				"setaf", "5",
				"bold",
				"smul",
				"setab", "7",
				"setaf", "11",
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls [][]interface{}
			testutil.StubValue(t, &TputCommand, func(output commondels.Output, args ...interface{}) error {
				calls = append(calls, args)
				return nil
			})

			test.format.Apply(nil)
			if diff := cmp.Diff(test.wantCalls, calls); diff != "" {
				t.Errorf("Format %v produced incorrect tput calls (-want, +got):\n%s", test.format, diff)
			}
		})
	}
}
