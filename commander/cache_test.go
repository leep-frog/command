package commander

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/spycommand"
	"github.com/leep-frog/command/internal/spycommander"
	"github.com/leep-frog/command/internal/spycommandtest"
	"github.com/leep-frog/command/internal/testutil"
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
	testutil.StubValue(t, &CacheDefaultHistory, 2)

	cc := &simpleCacheCLI{}
	for _, test := range []struct {
		name      string
		etc       *commandtest.ExecuteTestCase
		ietc      *spycommandtest.ExecuteTestCase
		opts      []CacheOption
		cache     map[string][][]string
		wantCache *simpleCacheCLI
	}{
		// Tests around adding things to the cache.
		{
			name: "Fails if later nodes fail",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"s\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Fails if extra arguments and doesn't cache",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args:    []string{"dollar", "bills"},
				WantErr: fmt.Errorf("Unprocessed extra args: [bills]"),
				WantStderr: strings.Join([]string{
					"Unprocessed extra args: [bills]",
					``,
					spycommander.UsageErrorSectionStart,
					`^ s`,
					``,
					`Arguments:`,
					`  s: test desc`,
					``,
					`Symbols:`,
					`  ^: Start of new cachable section`,
					``,
				}, "\n"),
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "bills", Snapshots: spycommand.SnapshotsMap(1)},
					},
					Remaining:     []int{1},
					SnapshotCount: 1,
				},
			},
		},
		{
			name: "Fails if not enough arguments error",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					ListArg[string]("sl", testDesc, 3, 0),
				)),
				Args:       []string{"dollar", "bills"},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"sl\" requires at least 3 arguments, got 2\n",
				WantData: &commondels.Data{Values: map[string]interface{}{
					"sl": []string{"dollar", "bills"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "bills", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
				},
			},
		},
		{
			name: "caches data on validator error",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc, MinLength[string, string](100)),
				)),
				Args:       []string{"dollar"},
				WantErr:    fmt.Errorf("validation for \"s\" failed: [MinLength] length must be at least 100"),
				WantStderr: "validation for \"s\" failed: [MinLength] length must be at least 100\n",
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
				},
			},
		},
		{
			name: "Doesn't mark as changed if transformed is same values",
			cache: map[string][][]string{
				"money": {{"usd"}},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc,
						&Transformer[string]{F: func(string, *commondels.Data) (string, error) {
							return "usd", nil
						}},
					))),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "usd",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "usd", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
				},
			},
		},
		{
			name: "Caches data on success",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				), CacheHistory(1)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				), CacheHistory(3)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
				), CacheHistory(3)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					ListArg[string]("sl", testDesc, 0, 4),
				), CacheHistory(3)),
				Args: []string{"cash", "money"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"sl": []string{"cash", "money"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "cash", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "money", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
				},
			},
		},
		{
			name: "Caches transformed args",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc,
						&Transformer[string]{F: func(string, *commondels.Data) (string, error) {
							return "usd", nil
						}},
					),
				)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "usd",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "usd", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[int]("i", testDesc),
					Arg[string]("s", testDesc,
						&Transformer[string]{F: func(string, *commondels.Data) (string, error) {
							return "usd", nil
						}},
					),
					ListArg[float64]("fl", testDesc, 2, 0),
					ListArg[string]("sl", testDesc, 1, 2,
						&Transformer[[]string]{F: func(v []string, d *commondels.Data) ([]string, error) {
							var newSL []string
							for _, s := range v {
								newSL = append(newSL, fmt.Sprintf("$%s", s))
							}
							return newSL, nil
						}},
					),
				)),
				Args: []string{"123", "dollar", "3.4", "4.5", "six", "7"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s":  "usd",
					"i":  123,
					"fl": []float64{3.4, 4.5},
					"sl": []string{"$six", "$7"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "123", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "usd", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "3.4", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "4.5", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "$six", Snapshots: spycommand.SnapshotsMap(1)},
						{Value: "$7", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					&ExecutorProcessor{func(output commondels.Output, _ *commondels.Data) error {
						output.Stdout("We made it!")
						return nil
					}},
				)),
				Args: []string{"dollar"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "dollar",
				}},
				WantStdout: "We made it!",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "dollar", Snapshots: spycommand.SnapshotsMap(1)},
					},
					SnapshotCount: 1,
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(OptionalArg[string]("s", testDesc)))},
		},
		{
			name: "Works when cache exists",
			cache: map[string][][]string{
				"money": {{"usd"}},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(OptionalArg[string]("s", testDesc))),
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s": "usd",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{{Value: "usd"}},
				},
			},
		},
		{
			name: "Works for long cache",
			cache: map[string][][]string{
				"money": {{"usd", "1", "2", "4"}},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s":  "usd",
					"il": []int{1, 2},
					"fl": []float64{4},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "usd"},
						{Value: "1"},
						{Value: "2"},
						{Value: "4"},
					},
				},
			},
		},
		{
			name: "Works for long cache with multiple ones",
			cache: map[string][][]string{
				"money": {{"first", "1"}, {"second", "2"}, {"usd", "1", "2", "4"}},
			},
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				WantData: &commondels.Data{Values: map[string]interface{}{
					"s":  "usd",
					"il": []int{1, 2},
					"fl": []float64{4},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "usd"},
						{Value: "1"},
						{Value: "2"},
						{Value: "4"},
					},
				},
			},
		},
		// History
		{
			name: "Displays no history if none",
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 1,
					cachePrefixData:         "",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "history"},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 1,
					cachePrefixData:         "",
				}},
				WantStdout: "usd 1 2 4\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "history"},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "2"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 2,
					cachePrefixData:         "",
				}},
				WantStdout: strings.Join([]string{
					"second 2",
					"usd 1 2 4\n",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "history"},
						{Value: "-n"},
						{Value: "2"},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "3"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 3,
					cachePrefixData:         "",
				}},
				WantStdout: strings.Join([]string{
					"first 1",
					"second 2",
					"usd 1 2 4\n",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "history"},
						{Value: "-n"},
						{Value: "3"},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "33"},
				WantData: &commondels.Data{Values: map[string]interface{}{
					cacheHistoryFlag.Name(): 33,
					cachePrefixData:         "",
				}},
				WantStdout: strings.Join([]string{
					"first 1",
					"second 2",
					"usd 1 2 4\n",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "history"},
						{Value: "-n"},
						{Value: "33"},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc),
					ListArg[int]("il", testDesc, 2, 0),
					ListArg[float64]("fl", testDesc, 1, 3),
				)),
				Args: []string{"history", "-n", "33", "-p"},
				WantData: &commondels.Data{Values: map[string]interface{}{
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
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "history"},
						{Value: "-n"},
						{Value: "33"},
						{Value: "-p"},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
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
				WantData: &commondels.Data{Values: map[string]interface{}{
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
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "123"},
						{Value: "history"},
						{Value: "-n"},
						{Value: "33"},
						{Value: "-p"},
					},
				},
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
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("beforeStr", testDesc, &Transformer[string]{F: func(s string, d *commondels.Data) (string, error) {
						return fmt.Sprintf("TRANSFORM(%s)", s), nil
					}}),
					Arg[int]("beforeInt", testDesc),
					CacheNode("money", cc, SerialNodes(
						Arg[string]("s", testDesc),
						ListArg[int]("il", testDesc, 2, 0),
						ListArg[float64]("fl", testDesc, 1, 3),
					)),
				),
				Args: []string{"hello", "123", "history", "-n", "33", "-p"},
				WantData: &commondels.Data{Values: map[string]interface{}{
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
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "TRANSFORM(hello)"},
						{Value: "123"},
						{Value: "history"},
						{Value: "-n"},
						{Value: "33"},
						{Value: "-p"},
					},
				},
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
			if test.ietc == nil {
				test.ietc = &spycommandtest.ExecuteTestCase{}
			}
			test.ietc.TestInput = true
			executeTest(t, test.etc, test.ietc)
			changeTest(t, test.wantCache, cc, cmp.AllowUnexported(simpleCacheCLI{}))
		})
	}
}

func TestCacheComplete(t *testing.T) {
	cc := &simpleCacheCLI{}
	for _, test := range []struct {
		name  string
		cache map[string][][]string
		ctc   *commandtest.CompleteTestCase
	}{
		{
			name: "handles empty",
			ctc: &commandtest.CompleteTestCase{
				WantData: &commondels.Data{},
				WantErr:  fmt.Errorf("Unprocessed extra args: []"),
			},
		},
		{
			name: "defers completion to provided node",
			ctc: &commandtest.CompleteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, SimpleCompleter[[]string]("buck", "dollar", "dollHairs", "dinero", "usd")),
				)),
				Args: "cmd $ d",
				Want: &commondels.Autocompletion{
					Suggestions: []string{"dinero", "dollHairs", "dollar"},
				},
				WantData: &commondels.Data{Values: map[string]interface{}{
					"sl": []string{"$", "d"},
				}},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			cc.cache = test.cache
			autocompleteTest(t, test.ctc, nil)
		})
	}
}
