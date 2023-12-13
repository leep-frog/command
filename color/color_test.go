package color

import (
	"testing"

	"github.com/leep-frog/command/internal/testutil"
)

func TestFormat(t *testing.T) {
	for _, test := range []struct {
		name           string
		format         Format
		wantOutputCode string
		wantTputCalls  [][]string
	}{
		{
			name:   "Text color",
			format: Cyan,
			wantTputCalls: [][]string{{
				"setaf", "6",
			}},
			wantOutputCode: "\033[36m",
		},
		{
			name:   "Background color",
			format: Yellow.Background(),
			wantTputCalls: [][]string{{
				"setab", "3",
			}},
			wantOutputCode: "\033[43m",
		},
		{
			name:   "Bold",
			format: Bold,
			wantTputCalls: [][]string{{
				"bold",
			}},
			wantOutputCode: "\033[1m",
		},
		{
			name:   "Underline",
			format: Underline,
			wantTputCalls: [][]string{{
				"smul",
			}},
			wantOutputCode: "\033[4m",
		},
		{
			name:   "End Unerline",
			format: EndUnderline,
			wantTputCalls: [][]string{{
				"rmul",
			}},
			wantOutputCode: "\033[24m",
		},
		{
			name:   "Reset",
			format: Reset,
			wantTputCalls: [][]string{{
				"init",
			}},
			wantOutputCode: "\033[0m",
		},
		{
			name:   "empty MultiFormat",
			format: MultiFormat(),
		},
		{
			name: "MultiFormat",
			format: MultiFormat(
				Magenta,
				Bold,
				Underline,
				White.Background(),
				Blue,
			),
			wantTputCalls: [][]string{
				{"setaf", "5"},
				{"bold"},
				{"smul"},
				{"setab", "7"},
				{"setaf", "4"},
			},
			wantOutputCode: "\033[35;1;4;47;34m",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testutil.Cmp(t, "Format.tputArgs() returned incorrect value", test.wantTputCalls, test.format.tputArgs())
			testutil.Cmp(t, "Format.OutputCode() returned incorrect value", test.wantOutputCode, OutputCode(test.format))
		})
	}
}
