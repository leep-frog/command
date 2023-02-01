package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	testDesc = "test desc"
)

func TestShortcutExecute(t *testing.T) {
	sc := &simpleShortcutCLIT{}
	for _, test := range []struct {
		name   string
		etc    *ExecuteTestCase
		am     map[string]map[string][]string
		wantAC *simpleShortcutCLIT
	}{
		{
			name: "shortcut requires arg",
			etc: &ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"sl\" requires at least 1 argument, got 0\n",
			},
		},
		// Add shortcut tests.
		{
			name: "requires an shortcut value",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
					},
				},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "requires a non-empty shortcut value",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", ""},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{},
					},
				},
				WantErr:    fmt.Errorf("validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1"),
				WantStderr: "validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1\n",
			},
		},
		{
			name: "doesn't override add command",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "a", "hello"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "a",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: "a"},
						{value: "hello"},
					},
					remaining: []int{2},
				},
				WantErr:    fmt.Errorf("cannot create shortcut for reserved value"),
				WantStderr: "cannot create shortcut for reserved value\n",
			},
		},
		{
			name: "doesn't override delete command",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "d"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "d",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: "d"},
					},
				},
				WantErr:    fmt.Errorf("cannot create shortcut for reserved value"),
				WantStderr: "cannot create shortcut for reserved value\n",
			},
		},
		// We don't really need to test other overrides (like we do for add and
		// delete above) since the user can still delete and add if they
		// accidentally override.
		{
			name: "ignores execute data from children nodes",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(Arg[string]("s", testDesc), SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
					ed.Executable = []string{
						"ab cd",
						"",
						"e",
					}
					ed.Executor = append(ed.Executor, func(o Output, d *Data) error {
						o.Stdout("here we are")
						return o.Stderr("unfortunately")
					})
					return nil
				}, nil), nil)),
				Args: []string{"a", "b", "c"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "b",
					"s":        "c",
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "b"},
						{value: "c", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"b": []string{"c"},
					},
				},
			},
		},
		{
			name: "errors on empty shortcut",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", ""},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1"),
				WantStderr: "validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1\n",
			},
		},
		{
			name: "errors on too many values",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "overload", "five", "four", "three", "two", "one"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "overload",
					"sl":       []string{"five", "four", "three"},
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "overload"},
						{value: "five", snapshots: snapshotsMap(1)},
						{value: "four", snapshots: snapshotsMap(1)},
						{value: "three", snapshots: snapshotsMap(1)},
						{value: "two", snapshots: snapshotsMap(1)},
						{value: "one", snapshots: snapshotsMap(1)},
					},
					remaining: []int{5, 6},
				},
				WantErr:    fmt.Errorf("Unprocessed extra args: [two one]"),
				WantStderr: "Unprocessed extra args: [two one]\n",
			},
		},
		{
			name: "fails to add empty shortcut list",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "empty"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "empty",
				}},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "empty"},
					},
				},
			},
		},
		{
			name: "adds shortcut list when just enough",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly"},
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly"},
					},
				},
			},
		},
		{
			name: "fails if shortcut already exists",
			am: map[string]map[string][]string{
				"pioneer": {
					"bearMinimum": nil,
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly"},
					},
					remaining: []int{2},
				},
				WantStderr: "Shortcut \"bearMinimum\" already exists\n",
				WantErr:    fmt.Errorf(`Shortcut "bearMinimum" already exists`),
			},
		},
		{
			name: "adds shortcut list when maximum amount",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly", "teddy", "brown"},
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown"},
					},
				},
			},
		},
		{
			name: "adds shortcut for multiple nodes",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2), Arg[int]("i", testDesc), ListArg[float64]("fl", testDesc, 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly", "teddy", "brown"},
					"i":        3,
					"fl":       []float64{2.2, -1.1},
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
						{value: "3", snapshots: snapshotsMap(1)},
						{value: "2.2", snapshots: snapshotsMap(1)},
						{value: "-1.1", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
					},
				},
			},
		},
		{
			name: "adds shortcut when doesn't reach nodes",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2), Arg[int]("i", testDesc), ListArg[float64]("fl", testDesc, 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly"},
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly"},
					},
				},
			},
		},
		{
			name: "adds shortcut for unbounded list",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList), Arg[int]("i", testDesc), ListArg[float64]("fl", testDesc, 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
						{value: "3", snapshots: snapshotsMap(1)},
						{value: "2.2", snapshots: snapshotsMap(1)},
						{value: "-1.1", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
					},
				},
			},
		},
		// Adding transformed arguments
		{
			name: "adds transformed arguments",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					&Transformer[[]string]{F: func([]string, *Data) ([]string, error) {
						return []string{"papa", "mama", "baby"}, nil
					}}))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"papa", "mama", "baby"},
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "papa", snapshots: snapshotsMap(1)},
						{value: "mama", snapshots: snapshotsMap(1)},
						{value: "baby", snapshots: snapshotsMap(1)},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"bearMinimum": []string{"papa", "mama", "baby"},
					},
				},
			},
		},
		{
			name: "fails if transform error",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					&Transformer[[]string]{F: func([]string, *Data) ([]string, error) {
						return nil, fmt.Errorf("bad news bears")
					}}))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
				}},
				wantInput: &Input{
					snapshotCount: 1,
					args: []*inputArg{
						{value: "a"},
						{value: "bearMinimum"},
						{value: "grizzly", snapshots: snapshotsMap(1)},
						{value: "teddy", snapshots: snapshotsMap(1)},
						{value: "brown", snapshots: snapshotsMap(1)},
					},
				},
				WantStderr: "Custom transformer failed: bad news bears\n",
				WantErr:    fmt.Errorf("Custom transformer failed: bad news bears"),
			},
		},
		// Executing node tests.
		{
			name: "Fails to replace shortcut with empty value",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{},
				},
			},
			etc: &ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"t", "grizzly", "other"},
				WantErr:    fmt.Errorf("shortcut has empty value"),
				WantStderr: "shortcut has empty value\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "t"},
						{value: "grizzly"},
						{value: "other"},
					},
					remaining: []int{0, 1, 2},
				},
			},
		},
		{
			name: "Replaces shortcut with value",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"t"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"teddy"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "teddy"},
					},
				},
			},
		},
		{
			name: "Ignores non-shortcut values",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"tee"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"tee"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "tee"},
					},
				},
			},
		},
		{
			name: "Replaces only shortcut value",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"t", "grizzly"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"teddy", "grizzly"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "teddy"},
						{value: "grizzly"},
					},
				},
			},
		},
		{
			name: "Replaces with multiple values",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "brown"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"t", "grizzly"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"teddy", "brown", "grizzly"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "teddy"},
						{value: "brown"},
						{value: "grizzly"},
					},
				},
			},
		},
		{
			name: "Replaces with multiple values and transformers",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "brown"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg("sl", testDesc, 1, 2, UpperCaseTransformer()))),
				Args: []string{"t", "grizzly"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"TEDDY", "BROWN", "GRIZZLY"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "TEDDY"},
						{value: "BROWN"},
						{value: "GRIZZLY"},
					},
				},
			},
		},
		// Arg with shortcut opt tests
		{
			name: "shortcut opt works with no shortcuts",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"zero"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"zero"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "zero"},
					},
				},
			},
		},
		{
			name: "shortcut opt replaces last argument",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"hello", "dee"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "d"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "d"},
					},
				},
			},
		},
		{
			name: "shortcut opt suggests args after replacement",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
					"t":   []string{"trois"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: []string{"hello", "dee", "trois"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "d", "trois"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "d"},
						{value: "trois"},
					},
				},
			},
		},
		{
			name: "shortcut opt replaces multiple shortcuts with more than one value",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList,
					ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: []string{"f", "dee"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"four", "two", "deux"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "four"},
						{value: "two"},
						{value: "deux"},
					},
				},
			},
		},
		{
			name: "shortcut opt replaces values across multiple args",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
					"z":   []string{"zero"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc)),
					Arg("s", testDesc, ShortcutOpt[string]("pioneer", sc)),
					OptionalArg("o", testDesc, ShortcutOpt[string]("pioneer", sc)),
				),
				Args: []string{"un", "dee", "z", "f"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"un", "two", "deux"},
					"s":  "zero",
					"o":  "four",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "two"},
						{value: "deux"},
						{value: "zero"},
						{value: "four"},
					},
				},
			},
		},
		{
			name: "shortcut opt replaces multiple shortcuts intertwined with regular args more than one value",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: []string{"f", "zero", "zero", "n1", "dee", "n2", "n3", "t", "u", "n4", "n5", ""},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "four"},
						{value: "0"},
						{value: "0"},
						{value: "n1"},
						{value: "two"},
						{value: "deux"},
						{value: "n2"},
						{value: "n3"},
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
						{value: "un"},
						{value: "n4"},
						{value: "n5"},
						{value: ""},
					},
				},
			},
		},
		{
			name: "shortcut opt replaces last value",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, UnboundedList, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"f", "zero", "n1", "t"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "n1", "three", "trois", "tres"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "four"},
						{value: "0"},
						{value: "n1"},
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
					},
				},
			},
		},
		{
			name: "shortcut happens before transform",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					UpperCaseTransformer(),
				)),
				Args: []string{"f", "zero", "n1", "t"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"FOUR", "0", "N1", "THREE", "TROIS", "TRES"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "FOUR"},
						{value: "0"},
						{value: "N1"},
						{value: "THREE"},
						{value: "TROIS"},
						{value: "TRES"},
					},
				},
			},
		},
		{
			name: "fails if shortcut doesn't add enough args",
			am: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, UnboundedList, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"dee"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"two", "deux"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "two"},
						{value: "deux"},
					},
				},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"sl\" requires at least 3 arguments, got 2\n",
			},
		},
		{
			name: "works if shortcut adds enough args",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, UnboundedList, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"t"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"three", "trois", "tres"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
					},
				},
			},
		},
		{
			name: "shortcut values bleed over into next argument",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III", "3"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), OptionalArg[int]("i", testDesc)),
				Args: []string{"t"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"three", "trois", "tres"},
					"s":  "III",
					"i":  3,
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "three"},
						{value: "trois"},
						{value: "tres"},
						{value: "III"},
						{value: "3"},
					},
				},
			},
		},
		{
			name: "don't shortcut for later args",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), OptionalArg[int]("i", testDesc)),
				Args: []string{"I", "II", "III", "t"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"I", "II", "III"},
					"s":  "t",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "I"},
						{value: "II"},
						{value: "III"},
						{value: "t"},
					},
				},
			},
		},
		// Get shortcut tests.
		{
			name: "Get shortcut requires argument",
			etc: &ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"g"},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "g"},
					},
				},
			},
		},
		{
			name: "Get shortcut handles missing shortcut type",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"g", "h"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"h"},
				}},
				WantErr:    fmt.Errorf(`No shortcuts exist for shortcut type "pioneer"`),
				WantStderr: "No shortcuts exist for shortcut type \"pioneer\"\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "g"},
						{value: "h"},
					},
				},
			},
		},
		{
			name: "Gets shortcut",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"k": []string{"alpha"},
					"m": []string{"one", "two", "three"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"g", "h", "i", "j", "k", "l", "m"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"h", "i", "j", "k", "l", "m"},
				}},
				WantStderr: strings.Join([]string{
					"Shortcut \"j\" does not exist",
					"Shortcut \"l\" does not exist",
					"",
				}, "\n"),
				WantStdout: strings.Join([]string{
					"h: ",
					"i: ",
					"k: alpha",
					"m: one two three",
					"",
				}, "\n"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "g"},
						{value: "h"},
						{value: "i"},
						{value: "j"},
						{value: "k"},
						{value: "l"},
						{value: "m"},
					},
				},
			},
		},
		// List shortcuts
		{
			name: "lists shortcut handles unset map",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"l"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "l"},
					},
				},
			},
		},
		{
			name: "lists shortcuts",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"k": []string{"alpha"},
					"m": []string{"one", "two", "three"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"l"},
				WantStdout: strings.Join([]string{
					"h: ",
					"i: ",
					"k: alpha",
					"m: one two three",
					"z: omega",
					"",
				}, "\n"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "l"},
					},
				},
			},
		},
		// Search shortcut tests.
		{
			name: "search shortcut requires argument",
			etc: &ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"s"},
				WantStderr: "Argument \"regexp\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "regexp" requires at least 1 argument, got 0`),
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
					},
				},
			},
		},
		{
			name: "search shortcut fails on invalid regexp",
			etc: &ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"s", ":)"},
				WantStderr: "validation for \"regexp\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`\n",
				WantErr:    fmt.Errorf("validation for \"regexp\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &Data{Values: map[string]interface{}{
					"regexp": []string{":)"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
						{value: ":)"},
					},
				},
			},
		},
		{
			name: "searches through shortcuts",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"j": []string{"bazzinga"},
					"k": []string{"alpha"},
					"l": []string{"garage"},
					"m": []string{"one", "two", "three"},
					"n": []string{"four"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"s", "ga$"},
				WantData: &Data{Values: map[string]interface{}{
					"regexp": []string{"ga$"},
				}},
				WantStdout: strings.Join([]string{
					"j: bazzinga",
					"z: omega",
					"",
				}, "\n"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
						{value: "ga$"},
					},
				},
			},
		},
		{
			name: "searches through shortcuts with multiple",
			am: map[string]map[string][]string{
				"pioneer": {
					"h": nil,
					"i": []string{},
					"j": []string{"bazzinga"},
					"k": []string{"alpha"},
					"l": []string{"garage"},
					"m": []string{"one", "two", "three"},
					"n": []string{"four"},
					"z": []string{"omega"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"s", "a$", "^.: [aeiou]"},
				WantData: &Data{Values: map[string]interface{}{
					"regexp": []string{"a$", "^.: [aeiou]"},
				}},
				WantStdout: strings.Join([]string{
					"k: alpha",
					"z: omega",
					"",
				}, "\n"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "s"},
						{value: "a$"},
						{value: "^.: [aeiou]"},
					},
				},
			},
		},
		// Delete shortcut tests.
		{
			name: "Delete requires argument",
			etc: &ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"d"},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
					},
				},
			},
		},
		{
			name: "Delete returns error if shortcut group does not exist",
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "e"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"e"},
				}},
				WantErr:    fmt.Errorf("Shortcut group has no shortcuts yet."),
				WantStderr: "Shortcut group has no shortcuts yet.\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "e"},
					},
				},
			},
		},
		{
			name: "Delete prints error if shortcut does not exist",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "grizzly"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "tee"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"tee"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "tee"},
					},
				},
				WantStderr: "Shortcut \"tee\" does not exist\n",
			},
		},
		{
			name: "Deletes an shortcut",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "grizzly"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "t"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"t"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "t"},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {},
				},
			},
		},
		{
			name: "Delete deletes multiple shortcuts",
			am: map[string]map[string][]string{
				"pioneer": {
					"p":      []string{"polar", "pooh"},
					"colors": []string{"brown", "black"},
					"t":      []string{"teddy"},
					"g":      []string{"grizzly"},
				},
			},
			etc: &ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "t", "penguin", "colors", "bare"},
				WantData: &Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"t", "penguin", "colors", "bare"},
				}},
				WantStderr: strings.Join([]string{
					"Shortcut \"penguin\" does not exist",
					"Shortcut \"bare\" does not exist",
					"",
				}, "\n"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "d"},
						{value: "t"},
						{value: "penguin"},
						{value: "colors"},
						{value: "bare"},
					},
				},
			},
			wantAC: &simpleShortcutCLIT{
				changed: true,
				mp: map[string]map[string][]string{
					"pioneer": {
						"p": []string{"polar", "pooh"},
						"g": []string{"grizzly"},
					},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			originalMP := map[string]map[string][]string{}
			for k1, m := range test.am {
				originalMP[k1] = map[string][]string{}
				for k2, v := range m {
					originalMP[k1][k2] = v
				}
			}
			sc.changed = false
			sc.mp = test.am

			test.etc.testInput = true
			ExecuteTest(t, test.etc)
			ChangeTest(t, test.wantAC, sc, cmp.AllowUnexported(simpleShortcutCLIT{}))
		})
	}
}

func TestAliasComplete(t *testing.T) {
	sc := &simpleShortcutCLIT{}
	for _, test := range []struct {
		name string
		ctc  *CompleteTestCase
		mp   map[string]map[string][]string
	}{
		{
			name: "suggests arg suggestions, but not command names",
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{""},
				}},
				Want: []string{"deux", "trois", "un"},
			},
		},
		// Add shortcut test
		{
			name: "suggests nothing for shortcut",
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd a ",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): "",
				}},
			},
		},
		{
			name: "fails if empty shortcut",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha": nil,
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args:    "cmd alpha b ",
				WantErr: fmt.Errorf("shortcut has empty value"),
			},
		},
		{
			name: "suggests regular things after shortcut",
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd a b ",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): "b",
					"sl":               []string{""},
				}},
				Want: []string{"deux", "trois", "un"},
			},
		},
		{
			name: "suggests regular things after shortcut",
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd a b ",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): "b",
					"sl":               []string{""},
				}},
				Want: []string{"deux", "trois", "un"},
			},
		},
		// Get shortcut test
		{
			name: "get shortcut makes suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd g ",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{""},
				}},
				Want: []string{"alpha", "alright", "any", "balloon", "bear"},
			},
		},
		{
			name: "get shortcut makes partial suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd g b",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"b"},
				}},
				Want: []string{"balloon", "bear"},
			},
		},
		{
			name: "get shortcut makes unique suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd g alright balloon ",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"alright", "balloon", ""},
				}},
				Want: []string{"alpha", "any", "bear"},
			},
		},
		// Delete shortcut test
		{
			name: "get shortcut makes suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd d ",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{""},
				}},
				Want: []string{"alpha", "alright", "any", "balloon", "bear"},
			},
		},
		{
			name: "get shortcut makes partial suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd d b",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"b"},
				}},
				Want: []string{"balloon", "bear"},
			},
		},
		{
			name: "get shortcut makes unique suggestions",
			mp: map[string]map[string][]string{
				"pioneer": {
					"alpha":   nil,
					"any":     []string{},
					"alright": nil,
					"balloon": []string{"red"},
					"bear":    []string{"lee"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd d alright balloon ",
				WantData: &Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"alright", "balloon", ""},
				}},
				Want: []string{"alpha", "any", "bear"},
			},
		},
		// Execute shortcut tests
		{
			name: "suggests regular things for regular command",
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd zero ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"zero", ""},
				}},
				Want: []string{"deux", "trois", "un"},
			},
		},
		{
			name: "doesn't replace last argument if it's one",
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd dee",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"dee"},
				}},
			},
		},
		{
			name: "suggests args after replacement",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd dee t",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"d", "t"},
				}},
				Want: []string{"trois"},
			},
		},
		{
			name: "replaced args are considered in distinct ops",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			ctc: &CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd dee ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"deux", ""},
				}},
				Want: []string{"trois", "un"},
			},
		},
		// Arg with shortcut opt tests
		{
			name: "shortcut opt suggests regular things for regular command",
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"))),
				Args: "cmd zero ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"zero", ""},
				}},
				Want: []string{"deux", "trois", "un"},
			},
		},
		{
			name: "shortcut opt doesn't replace last argument if it's one",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd hello dee",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "dee"},
				}},
			},
		},
		{
			name: "shortcut opt suggests args after replacement",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd hello dee t",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "d", "t"},
				}},
				Want: []string{"trois"},
			},
		},
		{
			name: "shortcut opt replaced args are considered in distinct ops",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd dee ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"deux", ""},
				}},
				Want: []string{"trois", "un"},
			},
		},
		{
			name: "shortcut opt replaces multiple args",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
					"t":   []string{"trois"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd dee t ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"deux", "trois", ""},
				}},
				Want: []string{"un"},
			},
		},
		{
			name: "shortcut opt replaces multiple shortcuts with more than one value",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"two", "deux"},
					"t":   []string{"three", "trois", "tres"},
					"f":   []string{"four"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd f dee ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"four", "two", "deux", ""},
				}},
				Want: []string{"trois", "un"},
			},
		},
		{
			name: "shortcut opt replaces multiple shortcuts intertwined with regular args more than one value",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois", "five", "six"),
				)),
				Args: "cmd f zero zero n1 dee n2 n3 t u n4 n5 ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""},
				}},
				Want: []string{"five", "six"},
			},
		},
		{
			name: "shortcut opt doesn't replace last value",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee":  []string{"two", "deux"},
					"t":    []string{"three", "trois", "tres"},
					"f":    []string{"four"},
					"u":    []string{"un"},
					"zero": []string{"0"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois", "five", "six"),
				)),
				Args: "cmd f zero n1 t",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "n1", "t"},
				}},
				Want: []string{"trois"},
			},
		},
		{
			name: "shortcut values bleed over into next argument for suggestion",
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), Arg[string]("i", testDesc, SimpleCompleter[string]("alpha", "beta"))),
				Args: "cmd t ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"three", "trois", "tres"},
					"s":  "III",
					"i":  "",
				}},
				Want: []string{"alpha", "beta"},
			},
		},
		{
			name: "don't shortcut for later args",
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), Arg[string]("i", testDesc, SimpleCompleter[string]("alpha", "beta"))),
				Args: "cmd I II III t ",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"I", "II", "III"},
					"s":  "t",
					"i":  "",
				}},
				Want: []string{"alpha", "beta"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sc.mp = test.mp
			CompleteTest(t, test.ctc)
		})
	}
}

type simpleShortcutCLIT struct {
	mp      map[string]map[string][]string
	changed bool
}

func (ssc *simpleShortcutCLIT) ShortcutMap() map[string]map[string][]string {
	if ssc.mp == nil {
		ssc.mp = map[string]map[string][]string{}
	}
	return ssc.mp
}

func (ssc *simpleShortcutCLIT) Changed() bool {
	return ssc.changed
}

func (ssc *simpleShortcutCLIT) MarkChanged() {
	ssc.changed = true
}

func newSimpleShortcut(existing map[string]map[string][]string) ShortcutCLI {
	return &simpleShortcutCLIT{
		mp: existing,
	}
}

func UpperCaseTransformer() ArgumentOption[[]string] {
	f := func(sl []string, d *Data) ([]string, error) {
		r := make([]string, 0, len(sl))
		for _, v := range sl {
			r = append(r, strings.ToUpper(v))
		}
		return r, nil
	}
	return &Transformer[[]string]{F: f}
}
