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
					StringNode("s", nil),
				)),
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
			},
		},
		{
			name: "Fails if extra arguments and doesn't cache",
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					StringNode("s", nil),
				)),
				Args:       []string{"dollar", "bills"},
				WantErr:    fmt.Errorf("Unprocessed extra args: [bills]"),
				WantStderr: []string{"Unprocessed extra args: [bills]"},
				WantData: &Data{
					"s": StringValue("dollar"),
				},
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
					StringListNode("sl", 3, 0, nil),
				)),
				Args:       []string{"dollar", "bills"},
				WantErr:    fmt.Errorf("not enough arguments"),
				WantStderr: []string{"not enough arguments"},
				WantData: &Data{
					"sl": StringListValue("dollar", "bills"),
				},
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
					StringNode("s", &ArgOpt{Validators: []ArgValidator{MinLength(100)}}),
				)),
				Args:       []string{"dollar"},
				WantErr:    fmt.Errorf("validation failed: [MinLength] value must be at least 100 characters"),
				WantStderr: []string{"validation failed: [MinLength] value must be at least 100 characters"},
				WantData: &Data{
					"s": StringValue("dollar"),
				},
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
					StringNode("s", nil),
				)),
				Args: []string{"dollar"},
				WantData: &Data{
					"s": StringValue("dollar"),
				},
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
					StringNode("s", &ArgOpt{
						Transformer: SimpleTransformer(StringType, func(v *Value) (*Value, error) {
							return StringValue("usd"), nil
						}),
					}),
				)),
				Args: []string{"dollar"},
				WantData: &Data{
					"s": StringValue("usd"),
				},
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
					StringNode("s", nil),
				)),
				Args: []string{"dollar"},
				WantData: &Data{
					"s": StringValue("dollar"),
				},
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
					StringNode("s", nil),
				)),
				Args: []string{"dollar"},
				WantData: &Data{
					"s": StringValue("dollar"),
				},
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
					StringNode("s", &ArgOpt{
						Transformer: SimpleTransformer(StringType, func(v *Value) (*Value, error) {
							return StringValue("usd"), nil
						}),
					}),
				)),
				Args: []string{"dollar"},
				WantData: &Data{
					"s": StringValue("usd"),
				},
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
					IntNode("i", nil),
					StringNode("s", &ArgOpt{
						Transformer: SimpleTransformer(StringType, func(v *Value) (*Value, error) {
							return StringValue("usd"), nil
						}),
					}),
					FloatListNode("fl", 2, 0, nil),
					StringListNode("sl", 1, 2, &ArgOpt{
						Transformer: SimpleTransformer(StringListType, func(v *Value) (*Value, error) {
							var newSL []string
							for _, s := range v.StringList() {
								newSL = append(newSL, fmt.Sprintf("$%s", s))
							}
							return StringListValue(newSL...), nil
						}),
					}),
				)),
				Args: []string{"123", "dollar", "3.4", "4.5", "six", "7"},
				WantData: &Data{
					"s":  StringValue("usd"),
					"i":  IntValue(123),
					"fl": FloatListValue(3.4, 4.5),
					"sl": StringListValue("$six", "$7"),
				},
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
					StringNode("s", nil),
					ExecutorNode(func(output Output, _ *Data) error {
						output.Stdout("We made it!")
						return nil
					}),
				)),
				Args: []string{"dollar"},
				WantData: &Data{
					"s": StringValue("dollar"),
				},
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
				Node: CacheNode("money", cc, SerialNodes(OptionalStringNode("s", nil)))},
		},
		{
			name: "Works when cache exists",
			cache: map[string][]string{
				"money": {"usd"},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(OptionalStringNode("s", nil))),
				wantInput: &Input{
					args: []*inputArg{{value: "usd"}},
				},
				WantData: &Data{
					"s": StringValue("usd"),
				},
			},
		},
		{
			name: "Works for long cache",
			cache: map[string][]string{
				"money": {"usd", "1", "2", "4"},
			},
			etc: &ExecuteTestCase{
				Node: CacheNode("money", cc, SerialNodes(
					StringNode("s", nil),
					IntListNode("il", 2, 0, nil),
					FloatListNode("fl", 1, 3, nil),
				)),
				wantInput: &Input{
					args: []*inputArg{
						{value: "usd"},
						{value: "1"},
						{value: "2"},
						{value: "4"},
					},
				},
				WantData: &Data{
					"s":  StringValue("usd"),
					"il": IntListValue(1, 2),
					"fl": FloatListValue(4),
				},
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
			ExecuteTest(t, test.etc, &ExecuteTestOptions{testInput: true})
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
					StringListNode("sl", 1, 2, &ArgOpt{
						Completor: &Completor{
							SuggestionFetcher: &ListFetcher{
								Options: []string{"buck", "dollar", "dollHairs", "dinero", "usd"},
							},
						},
					}),
				)),
				Args: "cmd $ d",
				Want: []string{"dinero", "dollHairs", "dollar"},
				WantData: &Data{
					"sl": StringListValue("$", "d"),
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			cc.cache = test.cache
			CompleteTest(t, test.ctc, nil)
		})
	}
}
