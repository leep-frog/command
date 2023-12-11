package commander

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/spycommand"
	"github.com/leep-frog/command/internal/spycommandtest"
)

const (
	testDesc = "test desc"
)

func TestShortcutExecute(t *testing.T) {
	sc := &simpleShortcutCLIT{}
	for _, test := range []struct {
		name   string
		etc    *commandtest.ExecuteTestCase
		ietc   *spycommandtest.ExecuteTestCase
		am     map[string]map[string][]string
		wantAC *simpleShortcutCLIT
	}{
		{
			name: "shortcut requires arg",
			etc: &commandtest.ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"sl\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		// Add shortcut tests.
		{
			name: "requires an shortcut value",
			etc: &commandtest.ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"a"},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "a"},
					},
				},
			},
		},
		{
			name: "requires a non-empty shortcut value",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", ""},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "",
				}},
				WantErr:    fmt.Errorf("validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1"),
				WantStderr: "validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{},
					},
				},
			},
		},
		{
			name: "doesn't override add command",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "a", "hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "a",
				}},
				WantErr:    fmt.Errorf("cannot create shortcut for reserved value"),
				WantStderr: "cannot create shortcut for reserved value\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "a"},
						{Value: "hello"},
					},
					Remaining: []int{2},
				},
			},
		},
		{
			name: "doesn't override delete command",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "d"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "d",
				}},
				WantErr:    fmt.Errorf("cannot create shortcut for reserved value"),
				WantStderr: "cannot create shortcut for reserved value\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "d"},
					},
				},
			},
		},
		// We don't really need to test other overrides (like we do for add and
		// delete above) since the user can still delete and add if they
		// accidentally override.
		{
			name: "ignores execute data from children nodes",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(Arg[string]("s", testDesc), SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					ed.Executable = []string{
						"ab cd",
						"",
						"e",
					}
					ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
						o.Stdout("here we are")
						return o.Stderr("unfortunately")
					})
					return nil
				}, nil), nil)),
				Args: []string{"a", "b", "c"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "b",
					"s":        "c",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "b"},
						{Value: "c", Snapshots: spycommand.SnapshotsMap(1)},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", ""},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "",
				}},
				WantErr:    fmt.Errorf("validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1"),
				WantStderr: "validation for \"SHORTCUT\" failed: [MinLength] length must be at least 1\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: ""},
					},
				},
			},
		},
		{
			name: "errors on too many values",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "overload", "five", "four", "three", "two", "one"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "overload",
					"sl":       []string{"five", "four", "three"},
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [two one]"),
				WantStderr: strings.Join([]string{
					`Unprocessed extra args: [two one]`,
					``,
					`======= Command Usage =======`,
					`* sl [ sl sl ]`,
					``,
					`Arguments:`,
					`  sl: test desc`,
					``,
					`Symbols:`,
					`  *: Start of new shortcut-able section`,
					``,
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:     true,
				WantIsExtraArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "overload"},
						{Value: "five", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "four", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "three", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "two", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "one", Snapshots: spycommand.SnapshotsMap(1)},
					},
					Remaining: []int{5, 6},
				},
			},
		},
		{
			name: "fails to add empty shortcut list",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "empty"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "empty",
				}},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "empty"},
					},
				},
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "adds shortcut list when just enough",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "grizzly", Snapshots: spycommand.SnapshotsMap(1)},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
				}},
				WantStderr: "Shortcut \"bearMinimum\" already exists\n",
				WantErr:    fmt.Errorf(`Shortcut "bearMinimum" already exists`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "grizzly"},
					},
					Remaining: []int{2},
				},
			},
		},
		{
			name: "adds shortcut list when maximum amount",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly", "teddy", "brown"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "grizzly", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "teddy", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "brown", Snapshots: spycommand.SnapshotsMap(1)},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2), Arg[int]("i", testDesc), ListArg[float64]("fl", testDesc, 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly", "teddy", "brown"},
					"i":        3,
					"fl":       []float64{2.2, -1.1},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "grizzly", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "teddy", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "brown", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "3", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "2.2", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "-1.1", Snapshots: spycommand.SnapshotsMap(1)},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2), Arg[int]("i", testDesc), ListArg[float64]("fl", testDesc, 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "grizzly", Snapshots: spycommand.SnapshotsMap(1)},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList), Arg[int]("i", testDesc), ListArg[float64]("fl", testDesc, 10, 0))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"grizzly", "teddy", "brown", "3", "2.2", "-1.1"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "grizzly", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "teddy", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "brown", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "3", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "2.2", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "-1.1", Snapshots: spycommand.SnapshotsMap(1)},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					&Transformer[[]string]{F: func([]string, *command.Data) ([]string, error) {
						return []string{"papa", "mama", "baby"}, nil
					}}))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
					"sl":       []string{"papa", "mama", "baby"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "papa", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "mama", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "baby", Snapshots: spycommand.SnapshotsMap(1)},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					&Transformer[[]string]{F: func([]string, *command.Data) ([]string, error) {
						return nil, fmt.Errorf("bad news bears")
					}}))),
				Args: []string{"a", "bearMinimum", "grizzly", "teddy", "brown"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": "bearMinimum",
				}},
				WantStderr: "Custom transformer failed: bad news bears\n",
				WantErr:    fmt.Errorf("Custom transformer failed: bad news bears"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					SnapshotCount: 1,
					Args: []*spycommand.InputArg{
						{Value: "a"},
						{Value: "bearMinimum"},
						{Value: "grizzly", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "teddy", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "brown", Snapshots: spycommand.SnapshotsMap(1)},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"t", "grizzly", "other"},
				WantErr:    fmt.Errorf("InputTransformer returned an empty list"),
				WantStderr: "InputTransformer returned an empty list\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "t"},
						{Value: "grizzly"},
						{Value: "other"},
					},
					Remaining: []int{0, 1, 2},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"t"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"teddy"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "teddy"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"tee"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"tee"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "tee"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"t", "grizzly"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"teddy", "grizzly"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "teddy"},
						{Value: "grizzly"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"t", "grizzly"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"teddy", "brown", "grizzly"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "teddy"},
						{Value: "brown"},
						{Value: "grizzly"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg("sl", testDesc, 1, 2, UpperCaseTransformer()))),
				Args: []string{"t", "grizzly"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"TEDDY", "BROWN", "GRIZZLY"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "TEDDY"},
						{Value: "BROWN"},
						{Value: "GRIZZLY"},
					},
				},
			},
		},
		// Arg with shortcut opt tests
		{
			name: "shortcut opt works with no shortcuts",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"zero"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"zero"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "zero"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"hello", "dee"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "d"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "d"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: []string{"hello", "dee", "trois"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "d", "trois"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "d"},
						{Value: "trois"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList,
					ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: []string{"f", "dee"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"four", "two", "deux"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "four"},
						{Value: "two"},
						{Value: "deux"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc)),
					Arg("s", testDesc, ShortcutOpt[string]("pioneer", sc)),
					OptionalArg("o", testDesc, ShortcutOpt[string]("pioneer", sc)),
				),
				Args: []string{"un", "dee", "z", "f"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"un", "two", "deux"},
					"s":  "zero",
					"o":  "four",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "two"},
						{Value: "deux"},
						{Value: "zero"},
						{Value: "four"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: []string{"f", "zero", "zero", "n1", "dee", "n2", "n3", "t", "u", "n4", "n5", ""},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "four"},
						{Value: "0"},
						{Value: "0"},
						{Value: "n1"},
						{Value: "two"},
						{Value: "deux"},
						{Value: "n2"},
						{Value: "n3"},
						{Value: "three"},
						{Value: "trois"},
						{Value: "tres"},
						{Value: "un"},
						{Value: "n4"},
						{Value: "n5"},
						{Value: ""},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"f", "zero", "n1", "t"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "n1", "three", "trois", "tres"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "four"},
						{Value: "0"},
						{Value: "n1"},
						{Value: "three"},
						{Value: "trois"},
						{Value: "tres"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 1, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					UpperCaseTransformer(),
				)),
				Args: []string{"f", "zero", "n1", "t"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"FOUR", "0", "N1", "THREE", "TROIS", "TRES"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "FOUR"},
						{Value: "0"},
						{Value: "N1"},
						{Value: "THREE"},
						{Value: "TROIS"},
						{Value: "TRES"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"dee"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"two", "deux"},
				}},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"sl\" requires at least 3 arguments, got 2\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "two"},
						{Value: "deux"},
					},
				},
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "works if shortcut adds enough args",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres"},
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc))),
				Args: []string{"t"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"three", "trois", "tres"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "three"},
						{Value: "trois"},
						{Value: "tres"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), OptionalArg[int]("i", testDesc)),
				Args: []string{"t"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"three", "trois", "tres"},
					"s":  "III",
					"i":  3,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "three"},
						{Value: "trois"},
						{Value: "tres"},
						{Value: "III"},
						{Value: "3"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), OptionalArg[int]("i", testDesc)),
				Args: []string{"I", "II", "III", "t"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"I", "II", "III"},
					"s":  "t",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "I"},
						{Value: "II"},
						{Value: "III"},
						{Value: "t"},
					},
				},
			},
		},
		// Get shortcut tests.
		{
			name: "Get shortcut requires argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"g"},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "g"},
					},
				},
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "Get shortcut handles missing shortcut type",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"g", "h"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"h"},
				}},
				WantErr:    fmt.Errorf(`No shortcuts exist for shortcut type "pioneer"`),
				WantStderr: "No shortcuts exist for shortcut type \"pioneer\"\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "g"},
						{Value: "h"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"g", "h", "i", "j", "k", "l", "m"},
				WantData: &command.Data{Values: map[string]interface{}{
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
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "g"},
						{Value: "h"},
						{Value: "i"},
						{Value: "j"},
						{Value: "k"},
						{Value: "l"},
						{Value: "m"},
					},
				},
			},
		},
		// List shortcuts
		{
			name: "lists shortcut handles unset map",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"l"},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "l"},
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
			etc: &commandtest.ExecuteTestCase{
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
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "l"},
					},
				},
			},
		},
		// Search shortcut tests.
		{
			name: "search shortcut requires argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"s"},
				WantStderr: "Argument \"regexp\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "regexp" requires at least 1 argument, got 0`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "s"},
					},
				},
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "search shortcut fails on invalid regexp",
			etc: &commandtest.ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"s", ":)"},
				WantStderr: "validation for \"regexp\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`\n",
				WantErr:    fmt.Errorf("validation for \"regexp\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &command.Data{Values: map[string]interface{}{
					"regexp": []string{":)"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "s"},
						{Value: ":)"},
					},
				},
				WantIsValidationError: true,
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"s", "ga$"},
				WantData: &command.Data{Values: map[string]interface{}{
					"regexp": []string{"ga$"},
				}},
				WantStdout: strings.Join([]string{
					"j: bazzinga",
					"z: omega",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "s"},
						{Value: "ga$"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"s", "a$", "^.: [aeiou]"},
				WantData: &command.Data{Values: map[string]interface{}{
					"regexp": []string{"a$", "^.: [aeiou]"},
				}},
				WantStdout: strings.Join([]string{
					"k: alpha",
					"z: omega",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "s"},
						{Value: "a$"},
						{Value: "^.: [aeiou]"},
					},
				},
			},
		},
		// Delete shortcut tests.
		{
			name: "Delete requires argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args:       []string{"d"},
				WantErr:    fmt.Errorf(`Argument "SHORTCUT" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SHORTCUT\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "d"},
					},
				},
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "Delete returns error if shortcut group does not exist",
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "e"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"e"},
				}},
				WantErr:    fmt.Errorf("Shortcut group has no shortcuts yet."),
				WantStderr: "Shortcut group has no shortcuts yet.\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "d"},
						{Value: "e"},
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
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "tee"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"tee"},
				}},
				WantStderr: "Shortcut \"tee\" does not exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "d"},
						{Value: "tee"},
					},
				},
			},
		},
		{
			name: "Deletes an shortcut",
			am: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"teddy", "grizzly"},
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "t"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"t"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "d"},
						{Value: "t"},
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
					"colors": []string{"brown", "abc  defk"},
					"t":      []string{"teddy"},
					"g":      []string{"grizzly"},
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"d", "t", "penguin", "colors", "bare"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SHORTCUT": []string{"t", "penguin", "colors", "bare"},
				}},
				WantStderr: strings.Join([]string{
					"Shortcut \"penguin\" does not exist",
					"Shortcut \"bare\" does not exist",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "d"},
						{Value: "t"},
						{Value: "penguin"},
						{Value: "colors"},
						{Value: "bare"},
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
		// Usage tests
		{
			name: "Usage doc",
			am: map[string]map[string][]string{
				"pioneer": {
					"p":      []string{"polar", "pooh"},
					"colors": []string{"brown", "abc  defk"},
					"t":      []string{"teddy"},
					"g":      []string{"grizzly"},
				},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: []string{"--help"},
				WantStdout: strings.Join([]string{
					"* sl [ sl sl ]",
					"",
					"Arguments:",
					"  sl: test desc",
					"",
					"Symbols:",
					"  *: Start of new shortcut-able section",
					"",
				}, "\n"),
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

			executeTest(t, test.etc, test.ietc)
			changeTest(t, test.wantAC, sc, cmp.AllowUnexported(simpleShortcutCLIT{}))
		})
	}
}

func TestAliasComplete(t *testing.T) {
	sc := &simpleShortcutCLIT{}
	for _, test := range []struct {
		name string
		ctc  *commandtest.CompleteTestCase
		mp   map[string]map[string][]string
	}{
		{
			name: "suggests arg suggestions, but not command names",
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"deux", "trois", "un"},
				},
			},
		},
		// Add shortcut test
		{
			name: "suggests nothing for shortcut",
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd a ",
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args:    "cmd alpha b ",
				WantErr: fmt.Errorf("InputTransformer returned an empty list"),
			},
		},
		{
			name: "suggests regular things after shortcut",
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd a b ",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): "b",
					"sl":               []string{""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"deux", "trois", "un"},
				},
			},
		},
		{
			name: "suggests regular things after shortcut",
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd a b ",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): "b",
					"sl":               []string{""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"deux", "trois", "un"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd g ",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "alright", "any", "balloon", "bear"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd g b",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"b"},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"balloon", "bear"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd g alright balloon ",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"alright", "balloon", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "any", "bear"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd d ",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "alright", "any", "balloon", "bear"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd d b",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"b"},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"balloon", "bear"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2))),
				Args: "cmd d alright balloon ",
				WantData: &command.Data{Values: map[string]interface{}{
					ShortcutArg.Name(): []string{"alright", "balloon", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "any", "bear"},
				},
			},
		},
		// Execute shortcut tests
		{
			name: "suggests regular things for regular command",
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd zero ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"zero", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"deux", "trois", "un"},
				},
			},
		},
		{
			name: "doesn't replace last argument if it's one",
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd dee",
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd dee t",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"d", "t"},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"trois"},
				},
			},
		},
		{
			name: "replaced args are considered in distinct ops",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			ctc: &commandtest.CompleteTestCase{
				Node: ShortcutNode("pioneer", sc, SerialNodes(ListArg[string]("sl", testDesc, 1, 2,
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				))),
				Args: "cmd dee ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"deux", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"trois", "un"},
				},
			},
		},
		// Arg with shortcut opt tests
		{
			name: "shortcut opt suggests regular things for regular command",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"))),
				Args: "cmd zero ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"zero", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"deux", "trois", "un"},
				},
			},
		},
		{
			name: "shortcut opt doesn't replace last argument if it's one",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"d"},
				},
			},
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd hello dee",
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd hello dee t",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "d", "t"},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"trois"},
				},
			},
		},
		{
			name: "shortcut opt replaced args are considered in distinct ops",
			mp: map[string]map[string][]string{
				"pioneer": {
					"dee": []string{"deux"},
				},
			},
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd dee ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"deux", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"trois", "un"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd dee t ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"deux", "trois", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"un"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois"),
				)),
				Args: "cmd f dee ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"four", "two", "deux", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"trois", "un"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois", "five", "six"),
				)),
				Args: "cmd f zero zero n1 dee n2 n3 t u n4 n5 ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "0", "n1", "two", "deux", "n2", "n3", "three", "trois", "tres", "un", "n4", "n5", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"five", "six"},
				},
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList, ShortcutOpt[[]string]("pioneer", sc),
					SimpleDistinctCompleter[[]string]("un", "deux", "trois", "five", "six"),
				)),
				Args: "cmd f zero n1 t",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"four", "0", "n1", "t"},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"trois"},
				},
			},
		},
		{
			name: "shortcut values bleed over into next argument for suggestion",
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), Arg[string]("i", testDesc, SimpleCompleter[string]("alpha", "beta"))),
				Args: "cmd t ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"three", "trois", "tres"},
					"s":  "III",
					"i":  "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "beta"},
				},
			},
		},
		{
			name: "don't shortcut for later args",
			mp: map[string]map[string][]string{
				"pioneer": {
					"t": []string{"three", "trois", "tres", "III"},
				},
			},
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg("sl", testDesc, 3, 0, ShortcutOpt[[]string]("pioneer", sc)), Arg[string]("s", testDesc), Arg[string]("i", testDesc, SimpleCompleter[string]("alpha", "beta"))),
				Args: "cmd I II III t ",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"I", "II", "III"},
					"s":  "t",
					"i":  "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "beta"},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sc.mp = test.mp
			autocompleteTest(t, test.ctc, nil)
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
	f := func(sl []string, d *command.Data) ([]string, error) {
		r := make([]string, 0, len(sl))
		for _, v := range sl {
			r = append(r, strings.ToUpper(v))
		}
		return r, nil
	}
	return &Transformer[[]string]{F: f}
}
