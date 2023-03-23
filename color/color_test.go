package color

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestFormat(t *testing.T) {
	for _, test := range []struct {
		name      string
		format    *Format
		wantCalls []*call
	}{
		{
			name:   "Background color",
			format: Background(3),
			wantCalls: []*call{{
				Name: "tput",
				Args: []interface{}{"setab", "3"},
			}},
		},
		{
			name:   "Text color",
			format: Text(6),
			wantCalls: []*call{{
				Name: "tput",
				Args: []interface{}{"setaf", "6"},
			}},
		},
		{
			name:   "Bold",
			format: Bold(),
			wantCalls: []*call{{
				Name: "tput",
				Args: []interface{}{"bold"},
			}},
		},
		{
			name:   "Underline",
			format: Underline(),
			wantCalls: []*call{{
				Name: "tput",
				Args: []interface{}{"smul"},
			}},
		},
		{
			name:   "End Unerline",
			format: EndUnderline(),
			wantCalls: []*call{{
				Name: "tput",
				Args: []interface{}{"rmul"},
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
			wantCalls: []*call{{
				Name: "tput",
				Args: []interface{}{
					"setaf", "5",
					"bold",
					"smul",
					"setab", "7",
					"setaf", "11",
				},
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []*call
			command.StubValue(t, &TputCommand, func(name string, args ...interface{}) error {
				calls = append(calls, &call{name, args})
				return nil
			})

			test.format.Apply()
			if diff := cmp.Diff(test.wantCalls, calls); diff != "" {
				t.Errorf("Format %v produced incorrect tput calls (-want, +got):\n%s", test.format, diff)
			}
		})
	}
}

type call struct {
	Name string
	Args []interface{}
}
