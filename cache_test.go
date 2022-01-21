package command

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type simpleCacheCLI struct {
	changed bool
	cache   map[string][]string
}

func (sc *simpleCacheCLI) Changed() bool {
	return sc.changed
}

func (sc *simpleCacheCLI) MarkChanged() {
	sc.changed = true
}

func (sc *simpleCacheCLI) Cache() map[string][]string {
	if sc.cache == nil {
		sc.cache = map[string][]string{}
	}
	return sc.cache
}

func TestCacheExecution(t *testing.T) {
	cc := &simpleCacheCLI{}
	for _, test := range []struct {
		name      string
		etc       *ExecuteTestCase
		cache     map[string][]string
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
				WantStderr: []string{`Argument "s" requires at least 1 argument, got 0`},
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
				WantStderr: []string{"Unprocessed extra args: [bills]"},
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
				WantStderr: []string{`Argument "sl" requires at least 3 arguments, got 2`},
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
				WantErr:    fmt.Errorf("validation failed: [MinLength] value must be at least 100 characters"),
				WantStderr: []string{"validation failed: [MinLength] value must be at least 100 characters"},
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
				cache: map[string][]string{
					"money": {"dollar"},
				},
			},
		},
		{
			name: "Doesn't mark as changed if same values",
			cache: map[string][]string{
				"money": {"dollar"},
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
			cache: map[string][]string{
				"money": {"usd"},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc,
						NewTransformer[string](func(string) (string, error) {
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
				cache: map[string][]string{
					"money": {"dollar"},
				},
			},
		},
		{
			name: "Replaces cache value",
			cache: map[string][]string{
				"money": {"euro", "peso"},
				"other": {"one", "two"},
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
				cache: map[string][]string{
					"money": {"dollar"},
					"other": {"one", "two"},
				},
			},
		},
		{
			name: "Caches transformed args",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[string]("s", testDesc,
						NewTransformer[string](func(string) (string, error) {
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
				cache: map[string][]string{
					"money": {"usd"},
				},
			},
		},
		{
			name: "Caches lots of args",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					Arg[int]("i", testDesc),
					Arg[string]("s", testDesc,
						NewTransformer[string](func(string) (string, error) {
							return "usd", nil
						}, false),
					),
					ListArg[float64]("fl", testDesc, 2, 0),
					ListArg[string]("sl", testDesc, 1, 2,
						NewTransformer[[]string](func(v []string) ([]string, error) {
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
				cache: map[string][]string{
					"money": {"123", "usd", "3.4", "4.5", "$six", "$7"},
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
				WantStdout: []string{"We made it!"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "dollar", snapshots: snapshotsMap(1)},
					},
					snapshotCount: 1,
				},
			},
			wantCache: &simpleCacheCLI{
				changed: true,
				cache: map[string][]string{
					"money": {"dollar"},
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
			cache: map[string][]string{
				"money": {"usd"},
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
			cache: map[string][]string{
				"money": {"usd", "1", "2", "4"},
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
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			originalCache := map[string][]string{}
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
		cache map[string][]string
		ctc   *CompleteTestCase
	}{
		{
			name: "handles empty",
			ctc: &CompleteTestCase{
				WantData: &Data{}},
		},
		{
			name: "defers completion to provided node",
			ctc: &CompleteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2,
						&Completor[[]string]{
							SuggestionFetcher: &ListFetcher[[]string]{
								Options: []string{"buck", "dollar", "dollHairs", "dinero", "usd"},
							},
						},
					),
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
