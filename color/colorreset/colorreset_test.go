package colorreset

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
)

func applyColor(s string, fs ...color.Format) string {
	return fmt.Sprintf("%s%s%s", color.OutputCode(color.MultiFormat(fs...)), s, color.OutputCode(color.Reset))
}

func TestColorReset(t *testing.T) {
	for _, test := range []struct {
		name  string
		stdin []string
		etc   *commandtest.ExecuteTestCase
	}{
		{
			name: "handles no input",
			etc: &commandtest.ExecuteTestCase{
				WantStdout: strings.Join([]string{
					// empty output
				}, "\n"),
			},
		},
		{
			name: "handles unformatted input",
			stdin: []string{
				"some input",
			},
			etc: &commandtest.ExecuteTestCase{
				WantStdout: strings.Join([]string{
					"some input",
					"",
				}, "\n"),
			},
		},
		{
			name: "clears formatting",
			stdin: []string{
				fmt.Sprintf("%s %s", applyColor("some", color.Green), applyColor("input", color.Bold)),
			},
			etc: &commandtest.ExecuteTestCase{
				WantStdout: strings.Join([]string{
					"some input",
					"",
				}, "\n"),
			},
		},
		{
			name: "clears multiple formatting and handles m ending in and out of formatting grouping (which is end of color code stuff)",
			stdin: []string{
				fmt.Sprintf("%sm %sm", applyColor("somem", color.Green.Background(), color.Underline), applyColor("inputm", color.Red, color.Underline, color.Yellow.Background(), color.Bold)),
			},
			etc: &commandtest.ExecuteTestCase{
				WantStdout: strings.Join([]string{
					"somemm inputmm",
					"",
				}, "\n"),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cr := &resetter{
				"fake-cli-name",
				strings.NewReader(strings.Join(test.stdin, "\n")),
			}
			test.etc.Node = cr.Node()
			commandertest.ExecuteTest(t, test.etc)
		})
	}
}

func TestMetadata(t *testing.T) {
	c := CLI("fake-cli-name")
	commandtest.Cmp(t, "CLI().Name()", "fake-cli-name", c.Name())
	commandtest.Cmp(t, "CLI().Setup()", nil, c.Setup())
	commandtest.Cmp(t, "CLI().Changed()", false, c.Changed())
}
