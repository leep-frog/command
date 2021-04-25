package command

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type simpleCacheCLI struct {
	changed bool
	cache   map[string][]string
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
		name       string
		n          *Node
		args       []string
		cache      map[string][]string
		wantStderr []string
		wantStdout []string
		wantErr    error
		wantEData  *ExecuteData
		wantData   *Data
		wantInput  *Input
		wantCache  *simpleCacheCLI
	}{
		// Tests around adding things to the cache.
		{
			name: "Fails if later nodes fail",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", nil),
			)),
			wantErr:    fmt.Errorf("not enough arguments"),
			wantStderr: []string{"not enough arguments"},
		},
		{
			name: "Fails if extra arguments, but still caches",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", nil),
			)),
			args:       []string{"dollar", "bills"},
			wantErr:    fmt.Errorf("Unprocessed extra args: [bills]"),
			wantStderr: []string{"Unprocessed extra args: [bills]"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("dollar"),
				},
			},
			wantInput: &Input{
				args: []*inputArg{
					{value: "dollar", snapshots: snapshotsMap(1)},
					{value: "bills", snapshots: snapshotsMap(1)},
				},
				remaining:     []int{1},
				snapshotCount: 1,
			},
			wantCache: &simpleCacheCLI{
				cache: map[string][]string{
					"money": []string{"dollar", "bills"},
				},
			},
		},
		{
			name: "caches data on validator error",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", &ArgOpt{Validators: []ArgValidator{MinLength(100)}}),
			)),
			args:       []string{"dollar"},
			wantErr:    fmt.Errorf("validation failed: [MinLength] value must be at least 100 characters"),
			wantStderr: []string{"validation failed: [MinLength] value must be at least 100 characters"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("dollar"),
				},
			},
			wantInput: &Input{
				args: []*inputArg{
					{value: "dollar", snapshots: snapshotsMap(1)},
				},
				snapshotCount: 1,
			},
			wantCache: &simpleCacheCLI{
				cache: map[string][]string{
					"money": []string{"dollar"},
				},
			},
		},
		{
			name: "Caches data on success",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", nil),
			)),
			args: []string{"dollar"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("dollar"),
				},
			},
			wantInput: &Input{
				args: []*inputArg{
					{value: "dollar", snapshots: snapshotsMap(1)},
				},
				snapshotCount: 1,
			},
			wantCache: &simpleCacheCLI{
				cache: map[string][]string{
					"money": []string{"dollar"},
				},
			},
		},
		{
			name: "Replaces cache value",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", nil),
			)),
			cache: map[string][]string{
				"money": []string{"euro", "peso"},
				"other": []string{"one", "two"},
			},
			args: []string{"dollar"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("dollar"),
				},
			},
			wantInput: &Input{
				args: []*inputArg{
					{value: "dollar", snapshots: snapshotsMap(1)},
				},
				snapshotCount: 1,
			},
			wantCache: &simpleCacheCLI{
				cache: map[string][]string{
					"money": []string{"dollar"},
					"other": []string{"one", "two"},
				},
			},
		},
		{
			name: "Caches transformed args",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", &ArgOpt{
					Transformer: SimpleTransformer(StringType, func(v *Value) (*Value, error) {
						return StringValue("usd"), nil
					}),
				}),
			)),
			args: []string{"dollar"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("usd"),
				},
			},
			wantInput: &Input{
				args: []*inputArg{
					{value: "usd", snapshots: snapshotsMap(1)},
				},
				snapshotCount: 1,
			},
			wantCache: &simpleCacheCLI{
				cache: map[string][]string{
					"money": []string{"usd"},
				},
			},
		},
		{
			name: "Caches lots of args",
			n: CacheNode("money", cc, SerialNodes(
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
			args: []string{"123", "dollar", "3.4", "4.5", "six", "7"},
			wantData: &Data{
				Values: map[string]*Value{
					"s":  StringValue("usd"),
					"i":  IntValue(123),
					"fl": FloatListValue(3.4, 4.5),
					"sl": StringListValue("$six", "$7"),
				},
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
			wantCache: &simpleCacheCLI{
				cache: map[string][]string{
					"money": []string{"123", "usd", "3.4", "4.5", "$six", "$7"},
				},
			},
		},
		{
			name: "Executes the executor",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", nil),
				ExecutorNode(func(output Output, _ *Data) error {
					output.Stdout("We made it!")
					return nil
				}),
			)),
			args: []string{"dollar"},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("dollar"),
				},
			},
			wantStdout: []string{"We made it!"},
			wantInput: &Input{
				args: []*inputArg{
					{value: "dollar", snapshots: snapshotsMap(1)},
				},
				snapshotCount: 1,
			},
			wantCache: &simpleCacheCLI{
				cache: map[string][]string{
					"money": []string{"dollar"},
				},
			},
		},
		// Tests around using the cache.
		{
			name: "Works when no cache exists",
			n:    CacheNode("money", cc, SerialNodes(OptionalStringNode("s", nil))),
		},
		{
			name: "Works when cache exists",
			n:    CacheNode("money", cc, SerialNodes(OptionalStringNode("s", nil))),
			cache: map[string][]string{
				"money": []string{"usd"},
			},
			wantInput: &Input{
				args: []*inputArg{{value: "usd"}},
			},
			wantData: &Data{
				Values: map[string]*Value{
					"s": StringValue("usd"),
				},
			},
		},
		{
			name: "Works for long cache",
			n: CacheNode("money", cc, SerialNodes(
				StringNode("s", nil),
				IntListNode("il", 2, 0, nil),
				FloatListNode("fl", 1, 3, nil),
			)),
			cache: map[string][]string{
				"money": []string{"usd", "1", "2", "4"},
			},
			wantInput: &Input{
				args: []*inputArg{
					{value: "usd"},
					{value: "1"},
					{value: "2"},
					{value: "4"},
				},
			},
			wantData: &Data{
				Values: map[string]*Value{
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
			executeTest(t, test.n, test.args, test.wantErr, test.wantEData, test.wantData, test.wantInput, test.wantStdout, test.wantStderr)

			wc := test.wantCache
			if wc == nil {
				wc = &simpleCacheCLI{
					cache: originalCache,
				}
			}
			if diff := cmp.Diff(wc, cc, cmp.AllowUnexported(simpleCacheCLI{}), cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("Cache.Execute(%v) incorrectly modified alias values:\n%s", test.args, diff)
			}
		})
	}
}

func TestCacheComplete(t *testing.T) {
	cc := &simpleCacheCLI{}
	for _, test := range []struct {
		name     string
		n        *Node
		args     []string
		cache    map[string][]string
		wantData *Data
		want     []string
	}{
		{
			name:     "handles empty",
			wantData: &Data{},
		},
		{
			name: "defers completion to provided node",
			n: CacheNode("money", cc, SerialNodes(
				StringListNode("sl", 1, 2, &ArgOpt{
					Completor: &Completor{
						SuggestionFetcher: &ListFetcher{
							Options: []string{"buck", "dollar", "dollHairs", "dinero", "usd"},
						},
					},
				}),
			)),
			args: []string{"$", "d"},
			want: []string{"dinero", "dollHairs", "dollar"},
			wantData: &Data{
				Values: map[string]*Value{
					"sl": StringListValue("$", "d"),
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			cc.cache = test.cache
			data := &Data{}
			input := ParseArgs(test.args)
			got := getCompleteData(test.n, input, data)
			var results []string
			if got != nil && got.Completion != nil {
				results = got.Completion.Process(input)
			}

			if diff := cmp.Diff(test.wantData, data, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("getCompleteData(%s) improperly parsed args (-want, +got)\n:%s", test.args, diff)
			}

			if diff := cmp.Diff(test.want, results, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("getCompleteData(%s) returned incorrect suggestions (-want, +got):\n%s", test.args, diff)
			}
		})
	}
}
