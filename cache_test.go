package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type simpleCacheCLI struct {
	changed bool
	cache   map[string][][]string
}

func (sc *simpleCacheCLI) Changed() bool {
	return sc.changed
}

func (sc *simpleCacheCLI) MarkChanged() {
	sc.changed = true
}

func (sc *simpleCacheCLI) Cache() map[string][][]string {
	if sc.cache == nil {
		sc.cache = map[string][][]string{}
	}
	return sc.cache
}

func TestCacheExecution(t *testing.T) {
	StubValue(t, &defaultHistory, 2)

	cc := &simpleCacheCLI{}
	for _, test := range []struct {
		name      string
		etc       *ExecuteTestCase
		opts      []CacheOption
		cache     map[string][][]string
		wantCache *simpleCacheCLI
	}{
		// Tests around adding things to the cache.
		{
			name: "Fails if later nodes fail",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"s\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Fails if extra arguments and doesn't cache",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args:       []string{"dollar", "bills"},
				WantErr:    fmt.Errorf("Unprocessed extra args: [bills]"),
				WantStderr: "Unprocessed extra args: [bills]\n",
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
						{value: "bills", snapshots: snapshotsMap(1)},
					},
					remaining:     []int{1},
					snapshotCount: 1,
				},
			},
		},
		{
			name: "Fails if not enough arguments error",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					ListArg[string]("sl", testDesc, 3, 0),
				)),
				Args:       []string{"dollar", "bills"},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"sl\" requires at least 3 arguments, got 2\n",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"dollar", "bills"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
						{value: "bills", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
		},
		{
			name: "caches data on validator error",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc, MinLength(100)),
				)),
				Args:       []string{"dollar"},
				WantErr:    fmt.Errorf("validation for \"s\" failed: [MinLength] value must be at least 100 characters"),
				WantStderr: "validation for \"s\" failed: [MinLength] value must be at least 100 characters\n",
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {{"dollar"}},
				},
			},
		},
		{
			name: "Doesn't mark as changed if same values",
			cache: map[string][][]string{
				"money": {{"dollar"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
		},
		{
			name: "Doesn't mark as changed if transformed is same values",
			cache: map[string][][]string{
				"money": {{"usd"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc,
						NewTransformer(func(string, *Data) (string, error) {
							return "usd", nil
						}, false),
					))),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "usd",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "usd", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
		},
		{
			name: "Caches data on success",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {{"dollar"}},
				},
			},
		},
		{
			name: "Replaces cache value",
			cache: map[string][][]string{
				"money": {{"euro", "peso"}},
				"other": {{"one", "two"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				), CacheHistory(1)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {{"dollar"}},
					"other": {{"one", "two"}},
				},
			},
		},
		{
			name: "Adds cache value for default",
			cache: map[string][][]string{
				"money": {{"euro", "peso"}},
				"other": {{"one", "two"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {
						{"euro", "peso"},
						{"dollar"},
					},
					"other": {{"one", "two"}},
				},
			},
		},
		{
			name: "Overwrites cache value when at default",
			cache: map[string][][]string{
				"money": {{"euro", "peso"}, {"other"}},
				"other": {{"one", "two"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {
						{"other"},
						{"dollar"},
					},
					"other": {{"one", "two"}},
				},
			},
		},
		{
			name: "Adds cache value",
			cache: map[string][][]string{
				"money": {{"euro", "peso"}},
				"other": {{"one", "two"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				), CacheHistory(3)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {
						{"euro", "peso"},
						{"dollar"},
					},
					"other": {{"one", "two"}},
				},
			},
		},
		{
			name: "Overrides cache value",
			cache: map[string][][]string{
				"money": {
					{"ca-ching"},
					{"euro", "peso"},
					{"cash", "money"},
				},
				"other": {{"one", "two"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				), CacheHistory(3)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {
						{"euro", "peso"},
						{"cash", "money"},
						{"dollar"},
					},
					"other": {{"one", "two"}},
				},
			},
		},
		{
			name: "No changes if last matches",
			cache: map[string][][]string{
				"money": {
					{"ca-ching"},
					{"euro", "peso"},
					{"cash", "money"},
				},
				"other": {{"one", "two"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					ListArg[string]("sl", testDesc, 0, 4),
				), CacheHistory(3)),
				Args: []string{"cash", "money"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"cash", "money"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "cash", snapshots: snapshotsMap(1)},
						{value: "money", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
		},
		{
			name: "Caches transformed args",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc,
						NewTransformer(func(string, *Data) (string, error) {
							return "usd", nil
						}, false),
					),
				)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "usd",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "usd", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {{"usd"}},
				},
			},
		},
		{
			name: "Caches lots of args",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[int]("i", testDesc),
					Arg[string]("s", testDesc,
						NewTransformer(func(string, *Data) (string, error) {
							return "usd", nil
						}, false),
					),
					ListArg[float64]("fl", testDesc, 2, 0),
					ListArg[string]("sl", testDesc, 1, 2,
						NewTransformer(func(v []string, d *Data) ([]string, error) {
							var newSL []string
							for _, s := range v {
								newSL = append(newSL, fmt.Sprintf("$%s", s))
							}
							return newSL, nil
						}, false),
					),
				)),
				Args: []string{"123", "dollar", "3.4", "4.5", "six", "7"},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "usd",
					"i":  123,
					"fl": []float64{3.4, 4.5},
					"sl": []string{"$six", "$7"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "123", snapshots: snapshotsMap(1)},
						{value: "usd", snapshots: snapshotsMap(1)},
						{value: "3.4", snapshots: snapshotsMap(1)},
						{value: "4.5", snapshots: snapshotsMap(1)},
						{value: "$six", snapshots: snapshotsMap(1)},
						{value: "$7", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {{"123", "usd", "3.4", "4.5", "$six", "$7"}},
				},
			},
		},
		{
			name: "Executes the executor",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ExecutorNode(func(output Output, _ *Data) {
						output.Stdout("We made it!")
					}),
				)),
				Args: []string{"dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				WantStdout: "We made it!",
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][][]string{
					"money": {{"dollar"}},
				},
			},
		},
		// Tests around using the cache.
		{
			name: "Works when no cache exists",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(OptionalArg[string]("s", testDesc)))},
		},
		{
			name: "Works when cache exists",
			cache: map[string][][]string{
				"money": {{"usd"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(OptionalArg[string]("s", testDesc))),
				wantInput: &Input{
					args: []*inputArg{{value: "usd"}},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": "usd",
				}},
			},
		},
		{
			name: "Works for long cache",
			cache: map[string][][]string{
				"money": {{"usd", "1", "2", "4"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				wantInput: &Input{
					args: []*inputArg{
						{value: "usd"},
						{value: "1"},
						{value: "2"},
						{value: "4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "usd",
					"il": []int{1, 2},
					"fl": []float64{4},
				}},
			},
		},
		{
			name: "Works for long cache with multiple ones",
			cache: map[string][][]string{
				"money": {{"first", "1"}, {"second", "2"}, {"usd", "1", "2", "4"}},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				wantInput: &Input{
					args: []*inputArg{
						{value: "usd"},
						{value: "1"},
						{value: "2"},
						{value: "4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "usd",
					"il": []int{1, 2},
					"fl": []float64{4},
				}},
			},
		},
		// History
		{
			name: "Displays no history if none",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "history"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 1,
					cachePrefixData:         "",
				}},
			},
		},
		{
			name: "Displays first history",
			cache: map[string][][]string{
				"money": {
					{"first", "1"},
					{"second", "2"},
					{"usd", "1", "2", "4"},
				},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "history"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 1,
					cachePrefixData:         "",
				}},
				WantStdout: "usd 1 2 4\n",
			},
		},
		{
			name: "Displays multiple history elements",
			cache: map[string][][]string{
				"money": {
					{"first", "1"},
					{"second", "2"},
					{"usd", "1", "2", "4"},
				},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "2"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "history"},
						{value: "-n"},
						{value: "2"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 2,
					cachePrefixData:         "",
				}},
				WantStdout: strings.Join([]string{
					"second 2",
					"usd 1 2 4\n",
				}, "\n"),
			},
		},
		{
			name: "Displays all history elements with prefix flag",
			cache: map[string][][]string{
				"money": {
					{"first", "1"},
					{"second", "2"},
					{"usd", "1", "2", "4"},
				},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "history"},
						{value: "-n"},
						{value: "3"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 3,
					cachePrefixData:         "",
				}},
				WantStdout: strings.Join([]string{
					"first 1",
					"second 2",
					"usd 1 2 4\n",
				}, "\n"),
			},
		},
		{
			name: "Displays all history elements if n is too big",
			cache: map[string][][]string{
				"money": {
					{"first", "1"},
					{"second", "2"},
					{"usd", "1", "2", "4"},
				},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "33"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "history"},
						{value: "-n"},
						{value: "33"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 33,
					cachePrefixData:         "",
				}},
				WantStdout: strings.Join([]string{
					"first 1",
					"second 2",
					"usd 1 2 4\n",
				}, "\n"),
			},
		},
		// History prefix
		{
			name: "Displays history with prefix flag if no prefix",
			cache: map[string][][]string{
				"money": {
					{"first", "1"},
					{"second", "2"},
					{"usd", "1", "2", "4"},
				},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "33", "-p"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "history"},
						{value: "-n"},
						{value: "33"},
						{value: "-p"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name():     33,
					cachePrefixData:             "",
					cachePrintPrefixFlag.Name(): true,
				}},
				WantStdout: strings.Join([]string{
					"first 1",
					"second 2",
					"usd 1 2 4",
					"",
				}, "\n"),
			},
		},
		{
			name: "Displays history with prefix flag if prefix",
			cache: map[string][][]string{
				"money": {
					{"first", "1"},
					{"second", "2"},
					{"usd", "1", "2", "4"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("beforeStr", testDesc),
					Arg[int]("beforeInt", testDesc),
					CacheNode("money", cc, SerialNodes(
						Arg[string]("s", testDesc),
						ListArg[int]("il", testDesc, 2, 0),
						ListArg[float64]("fl", testDesc, 1, 3),
					)),
				),
				Args: []string{"hello", "123", "history", "-n", "33", "-p"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "123"},
						{value: "history"},
						{value: "-n"},
						{value: "33"},
						{value: "-p"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name():     33,
					cachePrefixData:             "hello 123 ",
					cachePrintPrefixFlag.Name(): true,
					"beforeStr":                 "hello",
					"beforeInt":                 123,
				}},
				WantStdout: strings.Join([]string{
					"hello 123 first 1",
					"hello 123 second 2",
					"hello 123 usd 1 2 4\n",
				}, "\n"),
			},
		},
		{
			name: "Displays history with transformed prefix flag if prefix",
			cache: map[string][][]string{
				"money": {
					{"first", "1"},
					{"second", "2"},
					{"usd", "1", "2", "4"},
				},
			},
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("beforeStr", testDesc, NewTransformer(func(s string, d *Data) (string, error) {
						return fmt.Sprintf("TRANSFORM(%s)", s), nil
					}, false)),
					Arg[int]("beforeInt", testDesc),
					CacheNode("money", cc, SerialNodes(
						Arg[string]("s", testDesc),
						ListArg[int]("il", testDesc, 2, 0),
						ListArg[float64]("fl", testDesc, 1, 3),
					)),
				),
				Args: []string{"hello", "123", "history", "-n", "33", "-p"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "TRANSFORM(hello)"},
						{value: "123"},
						{value: "history"},
						{value: "-n"},
						{value: "33"},
						{value: "-p"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name():     33,
					cachePrefixData:             "TRANSFORM(hello) 123 ",
					cachePrintPrefixFlag.Name(): true,
					"beforeStr":                 "TRANSFORM(hello)",
					"beforeInt":                 123,
				}},
				WantStdout: strings.Join([]string{
					"TRANSFORM(hello) 123 first 1",
					"TRANSFORM(hello) 123 second 2",
					"TRANSFORM(hello) 123 usd 1 2 4\n",
				}, "\n"),
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			originalCache := map[string][][]string{}
			for k, v := range test.cache {
				originalCache[k] = v
			}
			cc.changed = false
			cc.cache = test.cache

			// Generic testing.
			test.etc.testInput = true
			ExecuteTest(t, test.etc)
			ChangeTest(t, test.wantCache, cc, cmp.AllowUnexported(simpleCacheCLI{}))
		})
	}
}

func TestCacheComplete(t *testing.T) {
	cc := &simpleCacheCLI{}
	for _, test := range []struct {
		name  string
		cache map[string][][]string
		ctc   *CompleteTestCase
	}{
		{
			name: "handles empty",
			ctc: &CompleteTestCase{
				WantData: &Data{},
				WantErr:  fmt.Errorf("Unprocessed extra args: []"),
			},
		},
		{
			name: "defers completion to provided node",
			ctc: &CompleteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, SimpleCompletor[[]string]("buck", "dollar", "dollHairs", "dinero", "usd")),
				)),
				Args: "cmd $ d",
				Want: []string{"dinero", "dollHairs", "dollar"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"$", "d"},
				}},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			cc.cache = test.cache
			CompleteTest(t, test.ctc)
		})
	}
}
