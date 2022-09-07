package command

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type errorEdge struct {
	e error
}

func (ee *errorEdge) Next(*Input, *Data) (*Node, error) {
	return nil, ee.e
}

func (ee *errorEdge) UsageNext() *Node {
	return nil
}

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name      string
		etc       *ExecuteTestCase
		postCheck func(*testing.T)
	}{
		{
			name: "handles nil node",
		},
		{
			name: "fails if unprocessed args",
			etc: &ExecuteTestCase{
				Args:       []string{"hello"},
				WantErr:    fmt.Errorf("Unprocessed extra args: [hello]"),
				WantStderr: "Unprocessed extra args: [hello]\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
					remaining: []int{0},
				},
			},
		},
		// Single arg tests.
		{
			name: "Fails if arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(Arg[string]("s", testDesc)),
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"s\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Fails if edge fails",
			etc: &ExecuteTestCase{
				Args: []string{"hello"},
				Node: &Node{
					Processor: Arg[string]("s", testDesc),
					Edge: &errorEdge{
						e: fmt.Errorf("bad news bears"),
					},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantErr: fmt.Errorf("bad news bears"),
				WantData: &Data{Values: map[string]interface{}{
					"s": "hello",
				}},
			},
		},
		{
			name: "Fails if int arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(Arg[int]("i", testDesc)),
				WantErr:    fmt.Errorf(`Argument "i" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"i\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "Fails if float arg and no argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(Arg[float64]("f", testDesc)),
				WantErr:    fmt.Errorf(`Argument "f" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"f\" requires at least 1 argument, got 0\n",
			},
		},
		// CompleteForExecute tests for single Arg
		{
			name: "CompleteForExecute for Arg fails if no arg provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				WantErr:    fmt.Errorf(`Argument "is" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"is\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "CompleteForExecute for Arg fails completer returns error",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] failed to fetch completion for \"is\": oopsie"),
				WantStderr: "[CompleteForExecute] failed to fetch completion for \"is\": oopsie\n",
			},
		},
		{
			name: "CompleteForExecute for Arg fails if returned completion is nil",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return nil, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] nil completion returned for \"is\""),
				WantStderr: "[CompleteForExecute] nil completion returned for \"is\"\n",
			},
		},
		{
			name: "CompleteForExecute for Arg fails if 0 suggestions",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return &Completion{}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"is\", got 0: []"),
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"is\", got 0: []\n",
			},
		},
		{
			name: "CompleteForExecute for Arg fails if multiple suggestions",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"1", "4"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"is\", got 2: [1 4]"),
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"is\", got 2: [1 4]\n",
			},
		},
		{
			name: "CompleteForExecute for Arg fails if suggestions is wrong type",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"someString"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "someString"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "someString": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"someString\": invalid syntax\n",
			},
		},
		{
			name: "CompleteForExecute for Arg works if one suggestion",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"123"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"is": 123,
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg completes on best effort",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](CompleteForExecuteBestEffort()), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"123"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"is": 123,
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg doesn't complete or error on best effort if no suggestions",
			etc: &ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](CompleteForExecuteBestEffort()), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
					return &Completion{}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "h"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg doesn't complete or error on best effort if multiple suggestions",
			etc: &ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](CompleteForExecuteBestEffort()), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"hey", "hi"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "h"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg doesn't complete or error on best effort if error",
			etc: &ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](CompleteForExecuteBestEffort()), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "h"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg doesn't complete or error on best effort if nil Completion",
			etc: &ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](CompleteForExecuteBestEffort()), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
					return nil, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "h"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg works when only one prefix matches",
			etc: &ExecuteTestCase{
				Args: []string{"4"},
				Node: SerialNodes(Arg[int]("is", testDesc, CompleteForExecute[int](), CompleterFromFunc(func(i int, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"123", "234", "345", "456", "567"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "456"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"is": 456,
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg fails if multiple completions",
			etc: &ExecuteTestCase{
				Args: []string{"f"},
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"one", "two", "three", "four", "five", "six"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "f"},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]"),
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]\n",
			},
		},
		{
			name: "CompleteForExecute for Arg works for string",
			etc: &ExecuteTestCase{
				Args: []string{"fi"},
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"one", "two", "three", "four", "five", "six"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "five"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "five",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg works for multiple, independent args",
			etc: &ExecuteTestCase{
				Args: []string{"fi", "tr"},
				Node: SerialNodes(
					Arg[string]("s", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"one", "two", "three", "four", "five", "six"},
						}, nil
					})),
					Arg[string]("s2", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"un", "deux", "trois", "quatre"},
						}, nil
					})),
				),
				wantInput: &Input{
					args: []*inputArg{
						{value: "five"},
						{value: "trois"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s":  "five",
						"s2": "trois",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg fails if one of completions fails for independent args",
			etc: &ExecuteTestCase{
				Args: []string{"fi", "mouse", "tr"},
				Node: SerialNodes(
					Arg[string]("s", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"one", "two", "three", "four", "five", "six"},
						}, nil
					})),
					Arg[string]("s2", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return nil, fmt.Errorf("rats")
					})),
					Arg[string]("s3", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"un", "deux", "trois", "quatre"},
						}, nil
					})),
				),
				wantInput: &Input{
					args: []*inputArg{
						{value: "five"},
						{value: "mouse"},
						{value: "tr"},
					},
					remaining: []int{2},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "five",
					},
				},
				WantStderr: "[CompleteForExecute] failed to fetch completion for \"s2\": rats\n",
				WantErr:    fmt.Errorf("[CompleteForExecute] failed to fetch completion for \"s2\": rats"),
			},
		},
		{
			name: "CompleteForExecute for Arg works if one of completions fails on best effort for independent args",
			etc: &ExecuteTestCase{
				Args: []string{"fi", "mouse", "tr"},
				Node: SerialNodes(
					Arg[string]("s", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"one", "two", "three", "four", "five", "six"},
						}, nil
					})),
					Arg[string]("s2", testDesc, CompleteForExecute[string](CompleteForExecuteBestEffort()), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return nil, fmt.Errorf("rats")
					})),
					Arg[string]("s3", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"un", "deux", "trois", "quatre"},
						}, nil
					})),
				),
				wantInput: &Input{
					args: []*inputArg{
						{value: "five"},
						{value: "mouse"},
						{value: "trois"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s":  "five",
						"s2": "mouse",
						"s3": "trois",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg transforms last arg *after* CompleteForExecute",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](),
					CompleterFromFunc(func(s string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"abc"},
						}, nil
					}),
					NewTransformer(func(s string, d *Data) (string, error) {
						return s + "?", nil
					}),
				)),
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc?"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "abc?",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg transforms last arg *after* CompleteForExecute and sub completion",
			etc: &ExecuteTestCase{
				Args: []string{"bra"},
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](),
					CompleterFromFunc(func(s string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"alpha", "bravo", "charlie", "brown"},
						}, nil
					}),
					NewTransformer(func(s string, d *Data) (string, error) {
						return s + "?", nil
					}),
				)),
				wantInput: &Input{
					args: []*inputArg{
						{value: "bravo?"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "bravo?",
					},
				},
			},
		},
		{
			name: "CompleteForExecute for Arg with transformer fails if no match",
			etc: &ExecuteTestCase{
				Args: []string{"br"},
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](),
					CompleterFromFunc(func(s string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"alpha", "bravo", "charlie", "brown"},
						}, nil
					}),
					NewTransformer(func(s string, d *Data) (string, error) {
						return s + "?", nil
					}),
				)),
				wantInput: &Input{
					args: []*inputArg{
						{value: "br"},
					},
				},
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [bravo brown]\n",
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [bravo brown]"),
			},
		},
		{
			name: "CompleteForExecute for Arg transforms last arg if CompleteForExecute fails with best effort",
			etc: &ExecuteTestCase{
				Args: []string{"br"},
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](CompleteForExecuteBestEffort()),
					CompleterFromFunc(func(s string, d *Data) (*Completion, error) {
						return &Completion{
							Suggestions: []string{"alpha", "bravo", "charlie", "brown"},
						}, nil
					}),
					NewTransformer(func(s string, d *Data) (string, error) {
						return s + "?", nil
					}),
				)),
				wantInput: &Input{
					args: []*inputArg{
						{value: "br?"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"s": "br?",
					},
				},
			},
		},
		{
			name: "CompleteForExecute is properly set in data",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(s string, d *Data) (*Completion, error) {
					d.Set("CFE", d.completeForExecute)
					return &Completion{Suggestions: []string{"abcde"}}, nil
				}))),
				Args: []string{"ab"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abcde"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"CFE": true,
					"s":   "abcde",
				}},
			},
		},
		// CompleteForExecuteAllowExactMatch tests
		{
			name: "CompleteForExecute fails if exact match and ExactMatch option not provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](),
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"Hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "Hello"},
					},
				},
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 3: [Hello Hello! HelloThere]\n",
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 3: [Hello Hello! HelloThere]"),
			},
		},
		{
			name: "CompleteForExecuteAllowExactMatch fails if partial match",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](CompleteForExecuteAllowExactMatch()),
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"Hel"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "Hel"},
					},
				},
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 3: [Hello Hello! HelloThere]\n",
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 3: [Hello Hello! HelloThere]"),
			},
		},
		{
			name: "CompleteForExecuteAllowExactMatch works if exact match",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](CompleteForExecuteAllowExactMatch()),
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"Hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "Hello"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": "Hello",
				}},
			},
		},
		{
			name: "CompleteForExecuteAllowExactMatch works if exact match with sub match",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](CompleteForExecuteAllowExactMatch()),
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"HelloThere"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "HelloThere"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": "HelloThere",
				}},
			},
		},
		{
			name: "CompleteForExecuteAllowExactMatch works if only sub match",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					CompleteForExecute[string](CompleteForExecuteAllowExactMatch()),
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"HelloThere!"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "HelloThere!"},
					},
				},
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 0: []\n",
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 0: []"),
			},
		},
		// FileCompleter with CompleteForExecute
		{
			name: "FileCompleter with CompleteForExecute properly completes a single directory",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{"do"},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "docs")},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": FilepathAbs(t, "docs"),
				}},
			},
		},
		{
			name: "FileCompleter with CompleteForExecute properly completes a full directory",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{"docs"},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "docs")},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": FilepathAbs(t, "docs"),
				}},
			},
		},
		{
			name: "FileCompleter with CompleteForExecute properly completes a full directory with trailing slash",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{fmt.Sprintf("docs%c", filepath.Separator)},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "docs")},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": FilepathAbs(t, "docs"),
				}},
			},
		},
		{
			name: "FileCompleter with CompleteForExecute properly completes nested directory",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{filepath.Join("sourcerer", "c")},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "sourcerer", "cmd")},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": FilepathAbs(t, "sourcerer", "cmd"),
				}},
			},
		},
		{
			name: "FileCompleter with CompleteForExecute properly completes nested file",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{filepath.Join("sourcerer", "cmd", "l")},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "sourcerer", "cmd", "load_sourcerer.sh")},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": FilepathAbs(t, "sourcerer", "cmd", "load_sourcerer.sh"),
				}},
			},
		},
		{
			name: "FileCompleter with CompleteForExecute properly completes a single file",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{"v"},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "validator.go")},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": FilepathAbs(t, "validator.go"),
				}},
			},
		},
		{
			name: "FileCompleter with CompleteForExecute fails if multiple options",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{"ca"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "ca"},
					},
				},
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [cache cache_]\n",
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [cache cache_]"),
			},
		},
		{
			name: "FileCompleter with CompleteForExecute fails if no options",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileNode("s", testDesc, CompleteForExecute[string]())),
				Args: []string{"uhhh"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "uhhh"},
					},
				},
				WantStderr: "[CompleteForExecute] nil completion returned for \"s\"\n",
				WantErr:    fmt.Errorf("[CompleteForExecute] nil completion returned for \"s\""),
			},
		},
		// CompleteForExecute tests for ListArg
		{
			name: "CompleteForExecute for ListArg fails if no arg provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 2 arguments, got 0`),
				WantStderr: "Argument \"sl\" requires at least 2 arguments, got 0\n",
			},
		},
		{
			name: "CompleteForExecute for ListArg fails completer returns error",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] failed to fetch completion for \"sl\": oopsie"),
				WantStderr: "[CompleteForExecute] failed to fetch completion for \"sl\": oopsie\n",
			},
		},
		{
			name: "CompleteForExecute for ListArg fails if returned completion is nil",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return nil, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] nil completion returned for \"sl\""),
				WantStderr: "[CompleteForExecute] nil completion returned for \"sl\"\n",
			},
		},
		{
			name: "CompleteForExecute for ListArg fails if 0 suggestions",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"sl\", got 0: []"),
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"sl\", got 0: []\n",
			},
		},
		{
			name: "CompleteForExecute for ListArg fails if multiple suggestions",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"alpha", "bravo"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"sl\", got 2: [alpha bravo]"),
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"sl\", got 2: [alpha bravo]\n",
			},
		},
		{
			name: "CompleteForExecute for ListArg fails if suggestions is wrong type",
			etc: &ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 3, CompleteForExecute[[]int](), CompleterFromFunc(func(sl []int, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"alpha"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "alpha"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "alpha": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"alpha\": invalid syntax\n",
			},
		},
		{
			name: "CompleteForExecute for ListArg fails if still not enough args",
			etc: &ExecuteTestCase{
				Args: []string{"alpha", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 3, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "charlie"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "alpha"},
						{value: "charlie"},
					},
				},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"sl\" requires at least 3 arguments, got 2\n",
				WantData: &Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "charlie"},
					},
				},
			},
		},
		{
			name: "CompleteForExecute for ListArg works if one suggestion",
			etc: &ExecuteTestCase{
				Args: []string{"alpha", "bravo", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "alpha"},
						{value: "bravo"},
						{value: "charlie"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
		},
		{
			name: "CompleteForExecute for ListArg works when only one prefix matches",
			etc: &ExecuteTestCase{
				Args: []string{"alpha", "bravo", "c"},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "bravo", "charlie", "delta", "epsilon"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "alpha"},
						{value: "bravo"},
						{value: "charlie"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
		},
		{
			name: "CompleteForExecute for ListArg fails if no distinct filter",
			etc: &ExecuteTestCase{
				Args: []string{"alpha", "bravo", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "alpha"},
						{value: "bravo"},
						{value: ""},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"sl\", got 3: [alpha bravo charlie]"),
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"sl\", got 3: [alpha bravo charlie]\n",
			},
		},
		{
			name: "CompleteForExecute for ListArg works with distinct filter",
			etc: &ExecuteTestCase{
				Args: []string{"alpha", "bravo", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "alpha"},
						{value: "bravo"},
						{value: "charlie"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
		},
		{
			name: "CompleteForExecute for ListArg completes multiple args",
			etc: &ExecuteTestCase{
				Args: []string{"a", "br", "c"},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, CompleteForExecute[[]string](), CompleterFromFunc(func(sl []string, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "alpha"},
						{value: "bravo"},
						{value: "charlie"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
		},
		{
			name: "CompleteForExecute for ListArg fails if multiple completions",
			etc: &ExecuteTestCase{
				Args: []string{"f"},
				Node: SerialNodes(Arg[string]("s", testDesc, CompleteForExecute[string](), CompleterFromFunc(func(i string, d *Data) (*Completion, error) {
					return &Completion{
						Suggestions: []string{"one", "two", "three", "four", "five", "six"},
					}, nil
				}))),
				wantInput: &Input{
					args: []*inputArg{
						{value: "f"},
					},
				},
				WantErr:    fmt.Errorf("[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]"),
				WantStderr: "[CompleteForExecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]\n",
			},
		},
		// Default value tests
		{
			name: "Uses default if no arg provided",
			etc: &ExecuteTestCase{
				Node:      SerialNodes(OptionalArg("s", testDesc, Default("settled"))),
				wantInput: &Input{},
				WantData: &Data{Values: map[string]interface{}{
					"s": "settled",
				}},
			},
		},
		{
			name: "Uses DefaultFunc if no arg provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(OptionalArg("s", testDesc, DefaultFunc(func(d *Data) (string, error) {
					return "heyo", nil
				}))),
				wantInput: &Input{},
				WantData: &Data{Values: map[string]interface{}{
					"s": "heyo",
				}},
			},
		},
		{
			name: "Failure if DefaultFunc failure for arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(OptionalArg("s", testDesc, DefaultFunc(func(d *Data) (string, error) {
					return "oops", fmt.Errorf("bad news bears")
				}))),
				wantInput:  &Input{},
				WantErr:    fmt.Errorf("failed to get default: bad news bears"),
				WantStderr: "failed to get default: bad news bears\n",
			},
		},
		{
			name: "Flag defaults get set",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag("s", 's', testDesc, Default("defStr")),
						NewFlag("s2", '2', testDesc, DefaultFunc(func(d *Data) (string, error) {
							return "dos", nil
						})),
						NewFlag("it", 't', testDesc, Default(-456)),
						NewFlag("i", 'i', testDesc, DefaultFunc(func(d *Data) (int, error) {
							return 123, nil
						})),
						NewFlag("fs", 'f', testDesc, Default([]float64{1.2, 3.4, -5.6})),
					),
				),
				Args: []string{"--it", "7", "-2", "dos"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--it"},
						{value: "7"},
						{value: "-2"},
						{value: "dos"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "defStr",
					"s2": "dos",
					"it": 7,
					"i":  123,
					"fs": []float64{1.2, 3.4, -5.6},
				}},
			},
		},
		{
			name: "Flag defaults get set",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag("s", 's', testDesc, Default("defStr")),
						NewFlag("s2", '2', testDesc, DefaultFunc(func(d *Data) (string, error) {
							// This flag is set, so this error func shouldn't be run at all,
							// hence why we don't expect to see this error.
							return "dos", fmt.Errorf("nooooooo")
						})),
						NewFlag("it", 't', testDesc, Default(-456)),
						NewFlag("i", 'i', testDesc, DefaultFunc(func(d *Data) (int, error) {
							return 123, fmt.Errorf("uh oh")
						})),
						NewFlag("fs", 'f', testDesc, Default([]float64{1.2, 3.4, -5.6})),
					),
				),
				Args: []string{"--it", "7", "-2", "dos"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--it"},
						{value: "7"},
						{value: "-2"},
						{value: "dos"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s2": "dos",
					"it": 7,
					"fs": []float64{1.2, 3.4, -5.6},
				}},
				WantErr:    fmt.Errorf("failed to get default: uh oh"),
				WantStderr: "failed to get default: uh oh\n",
			},
		},
		{
			name: "Default doesn't fill in required argument",
			etc: &ExecuteTestCase{
				Node:       SerialNodes(Arg("s", testDesc, Default("settled"))),
				wantInput:  &Input{},
				WantStderr: "Argument \"s\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
			},
		},
		// Simple arg tests
		{
			name: "Processes single string arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s": "hello",
				}},
			},
		},
		{
			name: "Processes single int arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[int]("i", testDesc)),
				Args: []string{"123"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 123,
				}},
			},
		},
		{
			name: "Int arg fails if not an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[int]("i", testDesc)),
				Args: []string{"12.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "12.3"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "12.3": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"12.3\": invalid syntax\n",
			},
		},
		{
			name: "Processes single float arg",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[float64]("f", testDesc)),
				Args: []string{"-12.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-12.3"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"f": -12.3,
				}},
			},
		},
		{
			name: "Float arg fails if not a float",
			etc: &ExecuteTestCase{
				Node: SerialNodes(Arg[float64]("f", testDesc)),
				Args: []string{"twelve"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "twelve"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
				WantStderr: "strconv.ParseFloat: parsing \"twelve\": invalid syntax\n",
			},
		},
		// List args
		{
			name: "List fails if not enough args",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 1)),
				Args: []string{"hello", "there", "sir"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "sir"},
					},
					remaining: []int{2},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there"},
				}},
				WantErr:    fmt.Errorf("Unprocessed extra args: [sir]"),
				WantStderr: "Unprocessed extra args: [sir]\n",
			},
		},
		{
			name: "Processes string list if minimum provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello"},
				}},
			},
		},
		{
			name: "Processes string list if some optional provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2)),
				Args: []string{"hello", "there"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there"},
				}},
			},
		},
		{
			name: "Processes string list if max args provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2)),
				Args: []string{"hello", "there", "maam"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "maam"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there", "maam"},
				}},
			},
		},
		{
			name: "Unbounded string list fails if less than min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 4, UnboundedList)),
				Args: []string{"hello", "there", "kenobi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "kenobi"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there", "kenobi"},
				}},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 4 arguments, got 3`),
				WantStderr: "Argument \"sl\" requires at least 4 arguments, got 3\n",
			},
		},
		{
			name: "Processes unbounded string list if min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList)),
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello"},
				}},
			},
		},
		{
			name: "Processes unbounded string list if more than min provided",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, UnboundedList)),
				Args: []string{"hello", "there", "kenobi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "kenobi"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there", "kenobi"},
				}},
			},
		},
		{
			name: "Processes int list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 1, 2)),
				Args: []string{"1", "-23"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
						{value: "-23"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"il": []int{1, -23},
				}},
			},
		},
		{
			name: "Int list fails if an arg isn't an int",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 1, 2)),
				Args: []string{"1", "four", "-23"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
						{value: "four"},
						{value: "-23"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "four": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"four\": invalid syntax\n",
			},
		},
		{
			name: "Processes float list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[float64]("fl", testDesc, 1, 2)),
				Args: []string{"0.1", "-2.3"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
						{value: "-2.3"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"fl": []float64{0.1, -2.3},
				}},
			},
		},
		{
			name: "Float list fails if an arg isn't an float",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[float64]("fl", testDesc, 1, 2)),
				Args: []string{"0.1", "four", "-23"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
						{value: "four"},
						{value: "-23"},
					},
				},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "four": invalid syntax`),
				WantStderr: "strconv.ParseFloat: parsing \"four\": invalid syntax\n",
			},
		},
		// Multiple args
		{
			name: "Processes multiple args",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2)),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"il": []int{0, 1},
					"s":  "two",
					"fl": []float64{0.3, -4},
				}},
			},
		},
		{
			name: "Fails if extra args when multiple",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2)),
				Args: []string{"0", "1", "two", "0.3", "-4", "0.5", "6"},
				wantInput: &Input{
					remaining: []int{6},
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
						{value: "0.5"},
						{value: "6"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"il": []int{0, 1},
					"s":  "two",
					"fl": []float64{0.3, -4, 0.5},
				}},
				WantErr:    fmt.Errorf("Unprocessed extra args: [6]"),
				WantStderr: "Unprocessed extra args: [6]\n",
			},
		},
		// Executor tests.
		{
			name: "Sets executable with SimpleExecutableNode",
			etc: &ExecuteTestCase{
				Node: SerialNodes(SimpleExecutableNode("hello", "there")),
				WantExecuteData: &ExecuteData{
					Executable: []string{"hello", "there"},
				},
			},
		},
		{
			name: "FunctionWrap sets ExecuteData.FunctionWrap",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableNode("hello", "there"),
					FunctionWrap(),
				),
				WantExecuteData: &ExecuteData{
					Executable:   []string{"hello", "there"},
					FunctionWrap: true,
				},
			},
		},
		{
			name: "Sets executable with ExecutableNode",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", "", 0, UnboundedList),
					ExecutableNode(func(o Output, d *Data) ([]string, error) {
						o.Stdoutln("hello")
						o.Stderr("there")
						return d.StringList("SL"), nil
					}),
				),
				Args:       []string{"abc", "def"},
				WantStdout: "hello\n",
				WantStderr: "there",
				WantExecuteData: &ExecuteData{
					Executable: []string{"abc", "def"},
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"SL": []string{"abc", "def"},
					},
				},
			},
		},
		{
			name: "ExecutableNode returning error",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", "", 0, UnboundedList),
					ExecutableNode(func(o Output, d *Data) ([]string, error) {
						return d.StringList("SL"), fmt.Errorf("bad news bears")
					}),
				),
				Args:    []string{"abc", "def"},
				WantErr: fmt.Errorf("bad news bears"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"SL": []string{"abc", "def"},
					},
				},
			},
		},
		{
			name: "Sets executable with processor",
			etc: &ExecuteTestCase{
				Node: SerialNodes(SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
					ed.Executable = []string{"hello", "there"}
					return nil
				}, nil)),
				WantExecuteData: &ExecuteData{
					Executable: []string{"hello", "there"},
				},
			},
		},
		{
			name: "executes with proper data",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2), printArgsNode()),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"il": []int{0, 1},
					"s":  "two",
					"fl": []float64{0.3, -4},
				}},
				WantStdout: strings.Join([]string{
					"fl: [0.3 -4]",
					"il: [0 1]",
					`s: two`,
					"",
				}, "\n"),
			},
		},
		{
			name: "executor error is returned",
			etc: &ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2), ExecuteErrNode(func(o Output, d *Data) error {
					return o.Stderrf("bad news bears")
				})),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
						{value: "1"},
						{value: "two"},
						{value: "0.3"},
						{value: "-4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"il": []int{0, 1},
					"s":  "two",
					"fl": []float64{0.3, -4},
				}},
				WantStderr: "bad news bears",
				WantErr:    fmt.Errorf("bad news bears"),
			},
		},
		// ArgValidator tests
		// StringDoesNotEqual
		{
			name: "string dne works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, NEQ("bad")),
				},
				Args: []string{"good"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "good"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "good",
				}},
			},
		},
		{
			name: "string dne fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, NEQ("bad")),
				},
				Args: []string{"bad"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "bad"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "bad",
				}},
				WantStderr: "validation for \"strArg\" failed: [NEQ] value cannot equal bad\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [NEQ] value cannot equal bad`),
			},
		},
		// Contains
		{
			name: "contains works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, Contains("good")),
				},
				Args: []string{"goodbye"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "goodbye"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "goodbye",
				}},
			},
		},
		{
			name: "contains fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, Contains("good")),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "hello",
				}},
				WantStderr: "validation for \"strArg\" failed: [Contains] value doesn't contain substring \"good\"\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [Contains] value doesn't contain substring "good"`),
			},
		},
		{
			name: "AddOptions works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc).AddOptions(Contains("good")),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "hello",
				}},
				WantStderr: "validation for \"strArg\" failed: [Contains] value doesn't contain substring \"good\"\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [Contains] value doesn't contain substring "good"`),
			},
		},
		// MatchesRegex
		{
			name: "matches regex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, MatchesRegex("a+b=?c")),
				},
				Args: []string{"equiation: aabcdef"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "equiation: aabcdef"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "equiation: aabcdef",
				}},
			},
		},
		{
			name: "matches regex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, MatchesRegex(".*", "i+")),
				},
				Args: []string{"team"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "team"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "team",
				}},
				WantStderr: "validation for \"strArg\" failed: [MatchesRegex] value \"team\" doesn't match regex \"i+\"\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [MatchesRegex] value "team" doesn't match regex "i+"`),
			},
		},
		// ListMatchesRegex
		{
			name: "ListMatchesRegex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, UnboundedList, ValidatorList(MatchesRegex("a+b=?c", "^eq"))),
				},
				Args: []string{"equiation: aabcdef"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "equiation: aabcdef"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{"equiation: aabcdef"},
				}},
			},
		},
		{
			name: "ListMatchesRegex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, UnboundedList, ValidatorList(MatchesRegex(".*", "i+"))),
				},
				Args: []string{"equiation: aabcdef", "oops"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "equiation: aabcdef"},
						{value: "oops"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{"equiation: aabcdef", "oops"},
				}},
				WantStderr: "validation for \"slArg\" failed: [MatchesRegex] value \"oops\" doesn't match regex \"i+\"\n",
				WantErr:    fmt.Errorf(`validation for "slArg" failed: [MatchesRegex] value "oops" doesn't match regex "i+"`),
			},
		},
		// IsRegex
		{
			name: "IsRegex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, IsRegex()),
				},
				Args: []string{".*"},
				wantInput: &Input{
					args: []*inputArg{
						{value: ".*"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": ".*",
				}},
			},
		},
		{
			name: "IsRegex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, IsRegex()),
				},
				Args: []string{"*"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "*"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "*",
				}},
				WantStderr: "validation for \"strArg\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`\n",
				WantErr:    fmt.Errorf("validation for \"strArg\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"),
			},
		},
		// ListIsRegex
		{
			name: "ListIsRegex works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, UnboundedList, ValidatorList(IsRegex())),
				},
				Args: []string{".*", " +"},
				wantInput: &Input{
					args: []*inputArg{
						{value: ".*"},
						{value: " +"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{".*", " +"},
				}},
			},
		},
		{
			name: "ListIsRegex fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, UnboundedList, ValidatorList(IsRegex())),
				},
				Args: []string{".*", "+"},
				wantInput: &Input{
					args: []*inputArg{
						{value: ".*"},
						{value: "+"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{".*", "+"},
				}},
				WantStderr: "validation for \"slArg\" failed: [IsRegex] value \"+\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `+`\n",
				WantErr:    fmt.Errorf("validation for \"slArg\" failed: [IsRegex] value \"+\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `+`"),
			},
		},
		// FileExists and FilesExist
		{
			name: "FileExists works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, FileExists()),
				},
				Args: []string{"execute_test.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "execute_test.go",
				}},
			},
		},
		{
			name: "FileExists fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, FileExists()),
				},
				Args: []string{"execute_test.gone"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.gone"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "execute_test.gone",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [FileExists] file "execute_test.gone" does not exist`),
				WantStderr: "validation for \"S\" failed: [FileExists] file \"execute_test.gone\" does not exist\n",
			},
		},
		{
			name: "FilesExist works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(FileExists())),
				},
				Args: []string{"execute_test.go", "execute.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
						{value: "execute.go"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"execute_test.go", "execute.go"},
				}},
			},
		},
		{
			name: "FilesExist fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(FileExists())),
				},
				Args: []string{"execute_test.go", "execute.gone"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
						{value: "execute.gone"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"execute_test.go", "execute.gone"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [FileExists] file "execute.gone" does not exist`),
				WantStderr: "validation for \"SL\" failed: [FileExists] file \"execute.gone\" does not exist\n",
			},
		},
		// IsDir and AreDirs
		{
			name: "IsDir works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, IsDir()),
				},
				Args: []string{"testdata"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testdata"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "testdata",
				}},
			},
		},
		{
			name: "IsDir fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, IsDir()),
				},
				Args: []string{"tested"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "tested"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "tested",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsDir] file "tested" does not exist`),
				WantStderr: "validation for \"S\" failed: [IsDir] file \"tested\" does not exist\n",
			},
		},
		{
			name: "IsDir fails when not a directory",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, IsDir()),
				},
				Args: []string{"execute_test.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute_test.go"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "execute_test.go",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsDir] argument "execute_test.go" is a file`),
				WantStderr: "validation for \"S\" failed: [IsDir] argument \"execute_test.go\" is a file\n",
			},
		},
		{
			name: "AreDirs works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(IsDir())),
				},
				Args: []string{"testdata", "cache"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testdata"},
						{value: "cache"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"testdata", "cache"},
				}},
			},
		},
		{
			name: "AreDirs fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(IsDir())),
				},
				Args: []string{"testdata", "cash"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testdata"},
						{value: "cash"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"testdata", "cash"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsDir] file "cash" does not exist`),
				WantStderr: "validation for \"SL\" failed: [IsDir] file \"cash\" does not exist\n",
			},
		},
		{
			name: "AreDirs fails when not a directory",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(IsDir())),
				},
				Args: []string{"testdata", "execute.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testdata"},
						{value: "execute.go"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"testdata", "execute.go"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsDir] argument "execute.go" is a file`),
				WantStderr: "validation for \"SL\" failed: [IsDir] argument \"execute.go\" is a file\n",
			},
		},
		// IsFile and AreFiles
		{
			name: "IsFile works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, IsFile()),
				},
				Args: []string{"execute.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "execute.go",
				}},
			},
		},
		{
			name: "IsFile fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, IsFile()),
				},
				Args: []string{"tested"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "tested"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "tested",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsFile] file "tested" does not exist`),
				WantStderr: "validation for \"S\" failed: [IsFile] file \"tested\" does not exist\n",
			},
		},
		{
			name: "IsFile fails when not a file",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("S", testDesc, IsFile()),
				},
				Args: []string{"testdata"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "testdata"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"S": "testdata",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsFile] argument "testdata" is a directory`),
				WantStderr: "validation for \"S\" failed: [IsFile] argument \"testdata\" is a directory\n",
			},
		},
		{
			name: "AreFiles works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(IsFile())),
				},
				Args: []string{"execute.go", "cache.go"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
						{value: "cache.go"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"execute.go", "cache.go"},
				}},
			},
		},
		{
			name: "AreFiles fails when does not exist",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(IsFile())),
				},
				Args: []string{"execute.go", "cash"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
						{value: "cash"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"execute.go", "cash"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsFile] file "cash" does not exist`),
				WantStderr: "validation for \"SL\" failed: [IsFile] file \"cash\" does not exist\n",
			},
		},
		{
			name: "AreFiles fails when not a directory",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ValidatorList(IsFile())),
				},
				Args: []string{"execute.go", "testdata"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "execute.go"},
						{value: "testdata"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"execute.go", "testdata"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsFile] argument "testdata" is a directory`),
				WantStderr: "validation for \"SL\" failed: [IsFile] argument \"testdata\" is a directory\n",
			},
		},
		// InList & string menus
		{
			name: "InList works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, InList("abc", "def", "ghi")),
				},
				Args: []string{"def"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "def"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "def",
				}},
			},
		},
		{
			name: "InList fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, InList("abc", "def", "ghi")),
				},
				Args: []string{"jkl"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "jkl"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "jkl",
				}},
				WantStderr: "validation for \"strArg\" failed: [InList] argument must be one of [abc def ghi]\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [InList] argument must be one of [abc def ghi]`),
			},
		},
		{
			name: "MenuArg works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: MenuArg("strArg", testDesc, "abc", "def", "ghi"),
				},
				Args: []string{"def"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "def"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "def",
				}},
			},
		},
		{
			name: "MenuArg fails if provided is not in list",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: MenuArg("strArg", testDesc, "abc", "def", "ghi"),
				},
				Args: []string{"jkl"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "jkl"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "jkl",
				}},
				WantStderr: "validation for \"strArg\" failed: [InList] argument must be one of [abc def ghi]\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [InList] argument must be one of [abc def ghi]`),
			},
		},
		{
			name: "MenuFlag works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: []string{"--sf", "def"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--sf"},
						{value: "def"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sf": "def",
				}},
			},
		},
		{
			name: "MenuFlag works with AddOptions(default)",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi", "xyz").AddOptions(Default("xyz")),
					),
				),
				WantData: &Data{Values: map[string]interface{}{
					"sf": "xyz",
				}},
			},
		},
		{
			name: "MenuFlag fails if provided is not in list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: []string{"-s", "jkl"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-s"},
						{value: "jkl"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"sf": "jkl",
				}},
				WantStderr: "validation for \"sf\" failed: [InList] argument must be one of [abc def ghi]\n",
				WantErr:    fmt.Errorf(`validation for "sf" failed: [InList] argument must be one of [abc def ghi]`),
			},
		},
		// MinLength
		{
			name: "MinLength works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, MinLength(3)),
				},
				Args: []string{"hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "hello",
				}},
			},
		},
		{
			name: "MinLength works for exact count match",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, MinLength(3)),
				},
				Args: []string{"hey"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hey"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "hey",
				}},
			},
		},
		{
			name: "MinLength fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, MinLength(3)),
				},
				Args: []string{"hi"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "hi"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "hi",
				}},
				WantStderr: "validation for \"strArg\" failed: [MinLength] value must be at least 3 characters\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [MinLength] value must be at least 3 characters`),
			},
		},
		// IntEQ
		{
			name: "IntEQ works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, EQ(24)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 24,
				}},
			},
		},
		{
			name: "IntEQ fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, EQ(24)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 25,
				}},
				WantStderr: "validation for \"i\" failed: [EQ] value isn't equal to 24\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [EQ] value isn't equal to 24`),
			},
		},
		// IntNE
		{
			name: "IntNE works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, NEQ(24)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 25,
				}},
			},
		},
		{
			name: "IntNE fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, NEQ(24)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 24,
				}},
				WantStderr: "validation for \"i\" failed: [NEQ] value cannot equal 24\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [NEQ] value cannot equal 24`),
			},
		},
		// IntLT
		{
			name: "IntLT works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, LT(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 24,
				}},
			},
		},
		{
			name: "IntLT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, LT(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 25,
				}},
				WantStderr: "validation for \"i\" failed: [LT] value isn't less than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [LT] value isn't less than 25`),
			},
		},
		{
			name: "IntLT fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, LT(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 26,
				}},
				WantStderr: "validation for \"i\" failed: [LT] value isn't less than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [LT] value isn't less than 25`),
			},
		},
		// IntLTE
		{
			name: "IntLTE works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, LTE(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 24,
				}},
			},
		},
		{
			name: "IntLTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, LTE(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 25,
				}},
			},
		},
		{
			name: "IntLTE fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, LTE(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 26,
				}},
				WantStderr: "validation for \"i\" failed: [LTE] value isn't less than or equal to 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [LTE] value isn't less than or equal to 25`),
			},
		},
		// IntGT
		{
			name: "IntGT fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, GT(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 24,
				}},
				WantStderr: "validation for \"i\" failed: [GT] value isn't greater than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [GT] value isn't greater than 25`),
			},
		},
		{
			name: "IntGT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, GT(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 25,
				}},
				WantStderr: "validation for \"i\" failed: [GT] value isn't greater than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [GT] value isn't greater than 25`),
			},
		},
		{
			name: "IntGT works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, GT(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 26,
				}},
			},
		},
		// IntGTE
		{
			name: "IntGTE fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, GTE(25)),
				},
				Args: []string{"24"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "24"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 24,
				}},
				WantStderr: "validation for \"i\" failed: [GTE] value isn't greater than or equal to 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [GTE] value isn't greater than or equal to 25`),
			},
		},
		{
			name: "IntGTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, GTE(25)),
				},
				Args: []string{"25"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "25"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 25,
				}},
			},
		},
		{
			name: "IntGTE works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, GTE(25)),
				},
				Args: []string{"26"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "26"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 26,
				}},
			},
		},
		// IntPositive
		{
			name: "IntPositive fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, Positive[int]()),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": -1,
				}},
				WantStderr: "validation for \"i\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Positive] value isn't positive`),
			},
		},
		{
			name: "IntPositive fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, Positive[int]()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 0,
				}},
				WantStderr: "validation for \"i\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Positive] value isn't positive`),
			},
		},
		{
			name: "IntPositive works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, Positive[int]()),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 1,
				}},
			},
		},
		// IntNegative
		{
			name: "IntNegative works when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, Negative[int]()),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": -1,
				}},
			},
		},
		{
			name: "IntNegative fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, Negative[int]()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 0,
				}},
				WantStderr: "validation for \"i\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Negative] value isn't negative`),
			},
		},
		{
			name: "IntNegative fails when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, Negative[int]()),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 1,
				}},
				WantStderr: "validation for \"i\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Negative] value isn't negative`),
			},
		},
		// IntNonNegative
		{
			name: "IntNonNegative fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, NonNegative[int]()),
				},
				Args: []string{"-1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": -1,
				}},
				WantStderr: "validation for \"i\" failed: [NonNegative] value isn't non-negative\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [NonNegative] value isn't non-negative`),
			},
		},
		{
			name: "IntNonNegative works when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, NonNegative[int]()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 0,
				}},
			},
		},
		{
			name: "IntNonNegative works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[int]("i", testDesc, NonNegative[int]()),
				},
				Args: []string{"1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"i": 1,
				}},
			},
		},
		// FloatEQ
		{
			name: "FloatEQ works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, EQ(2.4)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
			},
		},
		{
			name: "FloatEQ fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, EQ(2.4)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
				WantStderr: "validation for \"flArg\" failed: [EQ] value isn't equal to 2.4\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [EQ] value isn't equal to 2.4`),
			},
		},
		// FloatNE
		{
			name: "FloatNE works",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, NEQ(2.4)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
			},
		},
		{
			name: "FloatNE fails",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, NEQ(2.4)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
				WantStderr: "validation for \"flArg\" failed: [NEQ] value cannot equal 2.4\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [NEQ] value cannot equal 2.4`),
			},
		},
		// FloatLT
		{
			name: "FloatLT works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, LT(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
			},
		},
		{
			name: "FloatLT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, LT(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
				WantStderr: "validation for \"flArg\" failed: [LT] value isn't less than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [LT] value isn't less than 2.5`),
			},
		},
		{
			name: "FloatLT fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, LT(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
				WantStderr: "validation for \"flArg\" failed: [LT] value isn't less than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [LT] value isn't less than 2.5`),
			},
		},
		// FloatLTE
		{
			name: "FloatLTE works when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, LTE(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
			},
		},
		{
			name: "FloatLTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, LTE(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
			},
		},
		{
			name: "FloatLTE fails when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, LTE(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
				WantStderr: "validation for \"flArg\" failed: [LTE] value isn't less than or equal to 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [LTE] value isn't less than or equal to 2.5`),
			},
		},
		// FloatGT
		{
			name: "FloatGT fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, GT(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
				WantStderr: "validation for \"flArg\" failed: [GT] value isn't greater than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [GT] value isn't greater than 2.5`),
			},
		},
		{
			name: "FloatGT fails when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, GT(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
				WantStderr: "validation for \"flArg\" failed: [GT] value isn't greater than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [GT] value isn't greater than 2.5`),
			},
		},
		{
			name: "FloatGT works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, GT(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
			},
		},
		// FloatGTE
		{
			name: "FloatGTE fails when less than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, GTE(2.5)),
				},
				Args: []string{"2.4"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.4"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
				WantStderr: "validation for \"flArg\" failed: [GTE] value isn't greater than or equal to 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [GTE] value isn't greater than or equal to 2.5`),
			},
		},
		{
			name: "FloatGTE works when equal to",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, GTE(2.5)),
				},
				Args: []string{"2.5"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.5"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
			},
		},
		{
			name: "FloatGTE works when greater than",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, GTE(2.5)),
				},
				Args: []string{"2.6"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "2.6"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
			},
		},
		// FloatPositive
		{
			name: "FloatPositive fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, Positive[float64]()),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": -0.1,
				}},
				WantStderr: "validation for \"flArg\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Positive] value isn't positive`),
			},
		},
		{
			name: "FloatPositive fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, Positive[float64]()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 0.0,
				}},
				WantStderr: "validation for \"flArg\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Positive] value isn't positive`),
			},
		},
		{
			name: "FloatPositive works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, Positive[float64]()),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 0.1,
				}},
			},
		},
		// FloatNegative
		{
			name: "FloatNegative works when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, Negative[float64]()),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": -0.1,
				}},
			},
		},
		{
			name: "FloatNegative fails when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, Negative[float64]()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 0.0,
				}},
				WantStderr: "validation for \"flArg\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Negative] value isn't negative`),
			},
		},
		{
			name: "FloatNegative fails when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, Negative[float64]()),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 0.1,
				}},
				WantStderr: "validation for \"flArg\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Negative] value isn't negative`),
			},
		},
		// FloatNonNegative
		{
			name: "FloatNonNegative fails when negative",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, NonNegative[float64]()),
				},
				Args: []string{"-0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-0.1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": -0.1,
				}},
				WantStderr: "validation for \"flArg\" failed: [NonNegative] value isn't non-negative\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [NonNegative] value isn't non-negative`),
			},
		},
		{
			name: "FloatNonNegative works when zero",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, NonNegative[float64]()),
				},
				Args: []string{"0"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 0.0,
				}},
			},
		},
		{
			name: "FloatNonNegative works when positive",
			etc: &ExecuteTestCase{
				Node: &Node{
					Processor: Arg[float64]("flArg", testDesc, NonNegative[float64]()),
				},
				Args: []string{"0.1"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "0.1"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"flArg": 0.1,
				}},
			},
		},
		// Flag nodes
		{
			name: "empty flag node works",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode()},
			},
		},
		{
			name: "flag node allows empty",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode(NewFlag[string]("strFlag", 'f', testDesc))},
			},
		},
		{
			name: "flag node fails if no argument",
			etc: &ExecuteTestCase{
				Node:       &Node{Processor: NewFlagNode(NewFlag[string]("strFlag", 'f', testDesc))},
				Args:       []string{"--strFlag"},
				WantStderr: "Argument \"strFlag\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "strFlag" requires at least 1 argument, got 0`),
				wantInput: &Input{
					args: []*inputArg{
						{value: "--strFlag"},
					},
				},
			},
		},
		{
			name: "flag node parses flag",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode(NewFlag[string]("strFlag", 'f', testDesc))},
				Args: []string{"--strFlag", "hello"},
				WantData: &Data{Values: map[string]interface{}{
					"strFlag": "hello",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--strFlag"},
						{value: "hello"},
					},
				},
			},
		},
		{
			name: "flag node parses short name flag",
			etc: &ExecuteTestCase{
				Node: &Node{Processor: NewFlagNode(NewFlag[string]("strFlag", 'f', testDesc))},
				Args: []string{"-f", "hello"},
				WantData: &Data{Values: map[string]interface{}{
					"strFlag": "hello",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-f"},
						{value: "hello"},
					},
				},
			},
		},
		{
			name: "flag node parses flag in the middle",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewFlag[string]("strFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "--strFlag", "hello", "deux"},
				WantData: &Data{Values: map[string]interface{}{
					"strFlag": "hello",
					"filler":  []string{"un", "deux"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--strFlag"},
						{value: "hello"},
						{value: "deux"},
					},
				},
			},
		},
		{
			name: "flag node parses short name flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewFlag[string]("strFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"uno", "dos", "-f", "hello"},
				WantData: &Data{Values: map[string]interface{}{
					"filler":  []string{"uno", "dos"},
					"strFlag": "hello",
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "uno"},
						{value: "dos"},
						{value: "-f"},
						{value: "hello"},
					},
				},
			},
		},
		// Int flag
		{
			name: "parses int flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewFlag[int]("intFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "deux", "-f", "3", "quatre"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "deux"},
						{value: "-f"},
						{value: "3"},
						{value: "quatre"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"filler":  []string{"un", "deux", "quatre"},
					"intFlag": 3,
				}},
			},
		},
		{
			name: "handles invalid int flag value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewFlag[int]("intFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "deux", "-f", "trois", "quatre"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "deux"},
						{value: "-f"},
						{value: "trois"},
						{value: "quatre"},
					},
					remaining: []int{0, 1, 4},
				},
				WantStderr: "strconv.Atoi: parsing \"trois\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "trois": invalid syntax`),
			},
		},
		// Float flag
		{
			name: "parses float flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewFlag[float64]("floatFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"--floatFlag", "-1.2", "three"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--floatFlag"},
						{value: "-1.2"},
						{value: "three"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"filler":    []string{"three"},
					"floatFlag": -1.2,
				}},
			},
		},
		{
			name: "handles invalid float flag value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewFlag[float64]("floatFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"--floatFlag", "twelve", "eleven"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--floatFlag"},
						{value: "twelve"},
						{value: "eleven"},
					},
					remaining: []int{2},
				},
				WantStderr: "strconv.ParseFloat: parsing \"twelve\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
			},
		},
		// Bool flag
		{
			name: "bool flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(BoolFlag("boolFlag", 'b', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"okay", "--boolFlag", "then"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "okay"},
						{value: "--boolFlag"},
						{value: "then"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"filler":   []string{"okay", "then"},
					"boolFlag": true,
				}},
			},
		},
		{
			name: "short bool flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(BoolFlag("boolFlag", 'b', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"okay", "-b", "then"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "okay"},
						{value: "-b"},
						{value: "then"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"filler":   []string{"okay", "then"},
					"boolFlag": true,
				}},
			},
		},
		// flag list tests
		{
			name: "flag list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewListFlag[string]("slFlag", 's', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "--slFlag", "hello", "there"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--slFlag"},
						{value: "hello"},
						{value: "there"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"filler": []string{"un"},
					"slFlag": []string{"hello", "there"},
				}},
			},
		},
		{
			name: "flag list fails if not enough",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewListFlag[string]("slFlag", 's', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "--slFlag", "hello"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--slFlag"},
						{value: "hello"},
					},
					remaining: []int{0},
				},
				WantStderr: "Argument \"slFlag\" requires at least 2 arguments, got 1\n",
				WantErr:    fmt.Errorf(`Argument "slFlag" requires at least 2 arguments, got 1`),
				WantData: &Data{Values: map[string]interface{}{
					"slFlag": []string{"hello"},
				}},
			},
		},
		// Int list
		{
			name: "int list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewListFlag[int]("ilFlag", 'i', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "-i", "2", "4", "8", "16", "32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "-i"},
						{value: "2"},
						{value: "4"},
						{value: "8"},
						{value: "16"},
						{value: "32"},
						{value: "64"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"filler": []string{"un", "64"},
					"ilFlag": []int{2, 4, 8, 16, 32},
				}},
			},
		},
		{
			name: "int list transform failure",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewListFlag[int]("ilFlag", 'i', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "-i", "2", "4", "8", "16.0", "32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "-i"},
						{value: "2"},
						{value: "4"},
						{value: "8"},
						{value: "16.0"},
						{value: "32"},
						{value: "64"},
					},
					remaining: []int{0, 7},
				},
				WantStderr: "strconv.Atoi: parsing \"16.0\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "16.0": invalid syntax`),
			},
		},
		// Float list
		{
			name: "float list works",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewListFlag[float64]("flFlag", 'f', testDesc, 0, 3)),
					ListArg[string]("filler", testDesc, 1, 3),
				),
				Args: []string{"un", "-f", "2", "-4.4", "0.8", "16.16", "-32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "-f"},
						{value: "2"},
						{value: "-4.4"},
						{value: "0.8"},
						{value: "16.16"},
						{value: "-32"},
						{value: "64"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"filler": []string{"un", "16.16", "-32", "64"},
					"flFlag": []float64{2, -4.4, 0.8},
				}},
			},
		},
		{
			name: "float list transform failure",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(NewListFlag[float64]("flFlag", 'f', testDesc, 0, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "--flFlag", "2", "4", "eight", "16.0", "32", "64"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "un"},
						{value: "--flFlag"},
						{value: "2"},
						{value: "4"},
						{value: "eight"},
						{value: "16.0"},
						{value: "32"},
						{value: "64"},
					},
					remaining: []int{0, 5, 6, 7},
				},
				WantStderr: "strconv.ParseFloat: parsing \"eight\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "eight": invalid syntax`),
			},
		},
		// Misc. flag tests
		{
			name: "processes multiple flags",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewListFlag[float64]("coordinates", 'c', testDesc, 2, 0),
						BoolFlag("boo", 'o', testDesc),
						NewListFlag[string]("names", 'n', testDesc, 1, 2),
						NewFlag[int]("rating", 'r', testDesc),
					),
					ListArg[string]("extra", testDesc, 0, 10),
				),
				Args: []string{"its", "--boo", "a", "-r", "9", "secret", "-n", "greggar", "groog", "beggars", "--coordinates", "2.2", "4.4", "message."},
				wantInput: &Input{
					args: []*inputArg{
						{value: "its"},
						{value: "--boo"},
						{value: "a"},
						{value: "-r"},
						{value: "9"},
						{value: "secret"},
						{value: "-n"},
						{value: "greggar"},
						{value: "groog"},
						{value: "beggars"},
						{value: "--coordinates"},
						{value: "2.2"},
						{value: "4.4"},
						{value: "message."},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"boo":         true,
					"extra":       []string{"its", "a", "secret", "message."},
					"names":       []string{"greggar", "groog", "beggars"},
					"coordinates": []float64{2.2, 4.4},
					"rating":      9,
				}},
			},
		},
		// BoolValueFlag
		{
			name: "BoolValueFlag works with true value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolValueFlag("light", 'l', testDesc, "hello there"),
					),
				),
				Args: []string{"--light"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--light"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"light": "hello there",
				}},
			},
		},
		{
			name: "BoolValueFlag works with false value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolValueFlag("light", 'l', testDesc, "hello there"),
					),
				),
			},
		},
		{
			name: "BoolValuesFlag works with true value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolValuesFlag("light", 'l', testDesc, "hello there", "general kenobi"),
					),
				),
				Args: []string{"--light"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--light"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"light": "hello there",
				}},
			},
		},
		{
			name: "BoolValuesFlag works with false value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolValuesFlag("light", 'l', testDesc, "hello there", "general kenobi"),
					),
				),
				WantData: &Data{Values: map[string]interface{}{
					"light": "general kenobi",
				}},
			},
		},
		// Multi-flag tests
		{
			name: "Multiple bool flags work as a multi-flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-qwer"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-qwer"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
					"run":      "hello there",
				}},
			},
		},
		{
			name: "Multi-flag fails if unknown flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Args: []string{"-qwy"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-qwy"},
					},
					remaining: []int{0},
				},
				WantData: &Data{Values: map[string]interface{}{
					"quick": true,
					"where": true,
				}},
				WantStderr: "Unknown flag code \"-y\" used in multi-flag\n",
				WantErr:    fmt.Errorf(`Unknown flag code "-y" used in multi-flag`),
			},
		},
		{
			name: "Multi-flag fails if uncombinable flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						NewListFlag[int]("two", 't', testDesc, 0, UnboundedList),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Args: []string{"-ert"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-ert"},
					},
					remaining: []int{0},
				},
				WantData: &Data{Values: map[string]interface{}{
					"everyone": true,
					"run":      true,
				}},
				WantStderr: "Flag \"two\" is not combinable\n",
				WantErr:    fmt.Errorf(`Flag "two" is not combinable`),
			},
		},
		// Duplicate flag tests
		{
			name: "Duplicate flags get caught in multi-flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-qwerq"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-qwerq"},
					},
					remaining: []int{0},
				},
				WantData: &Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
					"run":      "hello there",
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
		},
		{
			name: "Duplicate flags get caught in regular flags",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-q", "--quick"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-q"},
						{value: "--quick"},
					},
					remaining: []int{1},
				},
				WantData: &Data{Values: map[string]interface{}{
					"quick": true,
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
		},
		{
			name: "Duplicate flags get caught when multi, then regular flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-qwer", "--quick"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "-qwer"},
						{value: "--quick"},
					},
					remaining: []int{1},
				},
				WantData: &Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
					"run":      "hello there",
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
		},
		{
			name: "Duplicate flags get caught when regular, then multi flag",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"--quick", "-weqr"},
				wantInput: &Input{
					args: []*inputArg{
						{value: "--quick"},
						{value: "-weqr"},
					},
					remaining: []int{1},
				},
				WantData: &Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
		},
		// Transformer tests.
		{
			name: "args get transformed",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("strArg", testDesc, NewTransformer(func(v string, d *Data) (string, error) {
						return strings.ToUpper(v), nil
					})),
					Arg[int]("intArg", testDesc, NewTransformer(func(v int, d *Data) (int, error) {
						return 10 * v, nil
					})),
				),
				Args: []string{"hello", "12"},
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "HELLO",
					"intArg": 120,
				}},
				wantInput: &Input{
					args: []*inputArg{{value: "HELLO"}, {value: "120"}},
				},
			},
		},
		{
			name: "list arg get transformed with TransformerList",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 2, 3, TransformerList(NewTransformer(func(v string, d *Data) (string, error) {
						return strings.ToUpper(v), nil
					}))),
				),
				Args: []string{"hello", "there", "general", "kenobi"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"HELLO", "THERE", "GENERAL", "KENOBI"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "HELLO"},
						{value: "THERE"},
						{value: "GENERAL"},
						{value: "KENOBI"},
					},
				},
			},
		},
		{
			name: "list arg transformer fails if number of args increases",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 2, 3, NewTransformer(func(v []string, d *Data) ([]string, error) {
						return append(v, "!"), nil
					})),
				),
				Args:       []string{"hello", "there", "general", "kenobi"},
				WantErr:    fmt.Errorf("[sl] Transformers must return a value that is the same length as the original arguments"),
				WantStderr: "[sl] Transformers must return a value that is the same length as the original arguments\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "general"},
						{value: "kenobi"},
					},
				},
			},
		},
		{
			name: "list arg transformer fails if number of args decreases",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 2, 3, NewTransformer(func(v []string, d *Data) ([]string, error) {
						return v[:len(v)-1], nil
					})),
				),
				Args:       []string{"hello", "there", "general", "kenobi"},
				WantErr:    fmt.Errorf("[sl] Transformers must return a value that is the same length as the original arguments"),
				WantStderr: "[sl] Transformers must return a value that is the same length as the original arguments\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "hello"},
						{value: "there"},
						{value: "general"},
						{value: "kenobi"},
					},
				},
			},
		},
		// Stdoutln tests
		{
			name: "stdoutln works",
			etc: &ExecuteTestCase{
				Node:       printlnNode(true, "one", 2, 3.0),
				WantStdout: "one 2 3\n",
			},
		},
		{
			name: "stderrln works",
			etc: &ExecuteTestCase{
				Node:       printlnNode(false, "uh", 0),
				WantStderr: "uh 0\n",
				WantErr:    fmt.Errorf("uh 0"),
			},
		},
		// BranchNode tests
		{
			name: "branch node requires branch argument",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, nil),
				WantStderr: "Branching argument must be one of [b h]\n",
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
			},
		},
		{
			name: "branch node requires matching branch argument",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, nil),
				Args:       []string{"uh"},
				WantStderr: "Branching argument must be one of [b h]\n",
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
				wantInput: &Input{
					args: []*inputArg{
						{value: "uh"},
					},
					remaining: []int{0},
				},
			},
		},
		{
			name: "branch node forwards to proper node",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, nil),
				Args:       []string{"h"},
				WantStdout: "hello",
				wantInput: &Input{
					args: []*inputArg{
						{value: "h"},
					},
				},
			},
		},
		{
			name: "branch node forwards to default if none provided",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, printNode("default")),
				WantStdout: "default",
			},
		},
		{
			name: "branch node forwards to default if unknown provided",
			etc: &ExecuteTestCase{
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, SerialNodes(ListArg[string]("sl", testDesc, 0, UnboundedList), printArgsNode().Processor)),
				Args:       []string{"good", "morning"},
				WantStdout: "sl: [good morning]\n",
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"good", "morning"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "good"},
						{value: "morning"},
					},
				},
			},
		},
		{
			name: "branch node forwards to synonym",
			etc: &ExecuteTestCase{
				Args: []string{"B"},
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, printNode("default"), BranchSynonyms(map[string][]string{
					"b": {"bee", "B", "Be"},
				})),
				wantInput: &Input{
					args: []*inputArg{
						{value: "B"},
					},
				},
				WantStdout: "goodbye",
			},
		},
		{
			name: "branch node fails if synonym to unknown command",
			etc: &ExecuteTestCase{
				Args: []string{"uh"},
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, nil, BranchSynonyms(map[string][]string{
					"o": {"uh"},
				})),
				wantInput: &Input{
					args: []*inputArg{
						{value: "uh"},
					},
					remaining: []int{0},
				},
				WantStderr: "Branching argument must be one of [b h]\n",
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
			},
		},
		{
			name: "branch node forwards to default if synonym to unknown command",
			etc: &ExecuteTestCase{
				Args: []string{"uh"},
				Node: BranchNode(map[string]*Node{
					"h": printNode("hello"),
					"b": printNode("goodbye"),
				}, SerialNodes(ListArg[string]("sl", testDesc, 0, UnboundedList), printArgsNode().Processor), BranchSynonyms(map[string][]string{
					"o": {"uh"},
				})),
				wantInput: &Input{
					args: []*inputArg{
						{value: "uh"},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"sl": []string{"uh"},
					},
				},
				WantStdout: "sl: [uh]\n",
			},
		},
		{
			name: "branch node forwards to spaced synonym",
			etc: &ExecuteTestCase{
				Args: []string{"bee"},
				Node: BranchNode(map[string]*Node{
					"h":          printNode("hello"),
					"b bee B Be": printNode("goodbye"),
				}, printNode("default")),
				wantInput: &Input{
					args: []*inputArg{
						{value: "bee"},
					},
				},
				WantStdout: "goodbye",
			},
		},
		// NodeRepeater tests
		{
			name: "NodeRepeater fails if not enough",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(3, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "NodeRepeater fails if middle node doen't have enough",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 1)),
				Args: []string{"k1", "100", "k2"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
					},
				},
				WantErr:    fmt.Errorf(`Argument "VALUE" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"VALUE\" requires at least 1 argument, got 0\n",
			},
		},
		{
			name: "NodeRepeater fails if too many",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1"},
					"values": []int{100},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
					remaining: []int{2, 3},
				},
				WantErr:    fmt.Errorf(`Unprocessed extra args: [k2 200]`),
				WantStderr: "Unprocessed extra args: [k2 200]\n",
			},
		},
		{
			name: "NodeRepeater accepts minimum when no optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts minimum when optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 3)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts minimum when unlimited optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 3)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts maximum when no optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts maximum when optional",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 1)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater with unlimited optional accepts a bunch",
			etc: &ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, UnboundedList)),
				Args: []string{"k1", "100", "k2", "200", "k3", "300", "k4", "400", "...", "0", "kn", "999"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2", "k3", "k4", "...", "kn"},
					"values": []int{100, 200, 300, 400, 0, 999},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "k1"},
						{value: "100"},
						{value: "k2"},
						{value: "200"},
						{value: "k3"},
						{value: "300"},
						{value: "k4"},
						{value: "400"},
						{value: "..."},
						{value: "0"},
						{value: "kn"},
						{value: "999"},
					},
				},
			},
		},
		// ListBreaker tests
		{
			name: "Handles broken list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 0, UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{"ghi", "jkl"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "List breaker before min value",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 3, UnboundedList, ListUntilSymbol("ghi")),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def"},
				}},
				WantErr:    fmt.Errorf(`Argument "SL" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"SL\" requires at least 3 arguments, got 2\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
					remaining: []int{2, 3},
				},
			},
		},
		{
			name: "Handles broken list with discard",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi", DiscardBreaker())),
					ListArg[string]("SL2", testDesc, 0, UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{"jkl"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "Handles unbroken list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 0, UnboundedList),
				),
				Args: []string{"abc", "def", "ghif", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def", "ghif", "jkl"},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghif"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "Fails if arguments required after broken list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 1, UnboundedList),
				),
				Args: []string{"abc", "def", "ghif", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def", "ghif", "jkl"},
				}},
				WantErr:    fmt.Errorf(`Argument "SL2" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SL2\" requires at least 1 argument, got 0\n",
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghif"},
						{value: "jkl"},
					},
				},
			},
		},
		// StringListListNode tests
		{
			name: "StringListListNode works if no breakers",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def", "ghi", "jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "ghi"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListNode works with unbounded list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListNode works with bounded list",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl"},
				WantData: &Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListNode works if ends with operator",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl", "|"},
				WantData: &Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
						{value: "|"},
					},
				},
			},
		},
		{
			name: "StringListListNode fails if extra args",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl", "|", "other", "stuff"},
				WantData: &Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
						{value: "|"},
						{value: "ghi"},
						{value: "||"},
						{value: "|"},
						{value: "jkl"},
						{value: "|"},
						{value: "other"},
						{value: "stuff"},
					},
					remaining: []int{8, 9},
				},
				WantErr:    fmt.Errorf("Unprocessed extra args: [other stuff]"),
				WantStderr: "Unprocessed extra args: [other stuff]\n",
			},
		},
		// FileContents test
		{
			name: "file gets read properly",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileContents("FILE", testDesc)),
				Args: []string{filepath.Join("testdata", "one.txt")},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "testdata", "one.txt")},
					},
				},
				WantData: &Data{
					Values: map[string]interface{}{
						"FILE": []string{"hello", "there"},
					},
				},
			},
		},
		{
			name: "FileContents fails for unknown file",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FileContents("FILE", testDesc)),
				Args: []string{filepath.Join("uh")},
				wantInput: &Input{
					args: []*inputArg{
						{value: FilepathAbs(t, "uh")},
					},
				},
				WantStderr: fmt.Sprintf("validation for \"FILE\" failed: [FileExists] file %q does not exist\n", FilepathAbs(t, "uh")),
				WantErr:    fmt.Errorf(`validation for "FILE" failed: [FileExists] file %q does not exist`, FilepathAbs(t, "uh")),
				WantData: &Data{
					Values: map[string]interface{}{
						"FILE": FilepathAbs(t, "uh"),
					},
				},
			},
		},
		// ConditionalProcessor tests
		{
			name: "ConditionalProcessor runs if function returns true",
			etc: &ExecuteTestCase{
				Args: []string{"abc", "def"},
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					ConditionalProcessor(
						printArgsNode(),
						func(i *Input, d *Data) bool {
							return true
						},
					),
					Arg[string]("s2", testDesc),
				),
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
					},
				},
				WantStdout: strings.Join([]string{
					"s: abc",
					"s2: def",
					"",
				}, "\n"),
				WantData: &Data{Values: map[string]interface{}{
					"s":  "abc",
					"s2": "def",
				}},
			},
		},
		{
			name: "ConditionalProcessor does not run if function returns false",
			etc: &ExecuteTestCase{
				Args: []string{"abc", "def"},
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					ConditionalProcessor(
						printArgsNode(),
						func(i *Input, d *Data) bool {
							return false
						},
					),
					Arg[string]("s2", testDesc),
				),
				wantInput: &Input{
					args: []*inputArg{
						{value: "abc"},
						{value: "def"},
					},
				},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "abc",
					"s2": "def",
				}},
			},
		},
		// EchoExecuteData
		{
			name: "EchoExecuteData ignores empty ExecuteData.Executable",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					EchoExecuteData(),
				),
			},
		},
		{
			name: "EchoExecuteData outputs ExecuteData.Executable",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableNode("un", "deux", "trois"),
					EchoExecuteData(),
				),
				WantExecuteData: &ExecuteData{
					Executable: []string{"un", "deux", "trois"},
				},
				WantStdout: "un\ndeux\ntrois\n",
			},
		},
		{
			name: "EchoExecuteDataf ignores empty ExecuteData.Executable",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					EchoExecuteDataf("RUNNING CODE:\n%s\nDONE CODE\n"),
				),
			},
		},
		{
			name: "EchoExecuteData outputs ExecuteData.Executable",
			etc: &ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableNode("un", "deux", "trois"),
					EchoExecuteDataf("RUNNING CODE:\n%s\nDONE CODE\n"),
				),
				WantExecuteData: &ExecuteData{
					Executable: []string{"un", "deux", "trois"},
				},
				WantStdout: strings.Join([]string{
					"RUNNING CODE:",
					"un",
					"deux",
					"trois",
					"DONE CODE",
					"",
				}, "\n"),
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.etc == nil {
				test.etc = &ExecuteTestCase{}
			}
			test.etc.testInput = true
			ExecuteTest(t, test.etc)
		})
	}
}

func abc() *Node {
	return BranchNode(map[string]*Node{
		"t": ShortcutNode("TEST_SHORTCUT", nil,
			CacheNode("TEST_CACHE", nil, SerialNodes(
				&tt{},
				Arg[string]("PATH", testDesc, SimpleCompleter[string]("clh111", "abcd111")),
				Arg[string]("TARGET", testDesc, SimpleCompleter[string]("clh222", "abcd222")),
				Arg[string]("FUNC", testDesc, SimpleCompleter[string]("clh333", "abcd333")),
			))),
	}, nil, DontCompleteSubcommands())
}

type tt struct{}

func (t *tt) Usage(*Usage) {}
func (t *tt) Execute(input *Input, output Output, data *Data, e *ExecuteData) error {
	t.do(input)
	return nil
}

func (t *tt) do(input *Input) {
	if s, ok := input.Peek(); ok && strings.Contains(s, ":") {
		if ss := strings.Split(s, ":"); len(ss) == 2 {
			input.Pop()
			input.PushFront(ss...)
		}
	}
}

func (t *tt) Complete(input *Input, data *Data) (*Completion, error) {
	t.do(input)
	return nil, nil
}

func TestComplete(t *testing.T) {
	for _, test := range []struct {
		name           string
		ctc            *CompleteTestCase
		filepathAbs    string
		filepathAbsErr error
	}{
		{
			name: "stuff",
			ctc: &CompleteTestCase{
				Node: abc(),
				Args: "cmd t clh:abc",
				Want: []string{"abcd222"},
				WantData: &Data{Values: map[string]interface{}{
					"PATH":   "clh",
					"TARGET": "abc",
				}},
			},
		},
		// Basic tests
		{
			name: "empty graph",
			ctc: &CompleteTestCase{
				Node:    &Node{},
				WantErr: fmt.Errorf("Unprocessed extra args: []"),
			},
		},
		{
			name: "returns suggestions of first node if empty",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("un", "deux", "trois")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
				),
				Want: []string{"deux", "trois", "un"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "",
				}},
			},
		},
		{
			name: "returns suggestions of first node if up to first arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
				),
				Args: "cmd t",
				Want: []string{"three", "two"},
				WantData: &Data{Values: map[string]interface{}{
					"s": "t",
				}},
			},
		},
		{
			name: "returns suggestions of middle node if that's where we're at",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three ",
				Want: []string{"dos", "uno"},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{""},
				}},
			},
		},
		{
			name: "returns suggestions of middle node if partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three d",
				Want: []string{"dos"},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"d"},
				}},
			},
		},
		{
			name: "returns suggestions in list",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three dos ",
				Want: []string{"dos", "uno"},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"dos", ""},
				}},
			},
		},
		{
			name: "returns suggestions for last arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three uno dos ",
				Want: []string{"1", "2"},
				WantData: &Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"uno", "dos"},
				}},
			},
		},
		{
			name: "returns nothing if iterate through all nodes",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three uno dos 1 what now",
				WantData: &Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"uno", "dos"},
					"i":  1,
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [what now]"),
			},
		},
		{
			name: "works if empty and list starts",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, SimpleCompleter[[]string]("uno", "dos")),
				),
				Want: []string{"dos", "uno"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{""},
				}},
			},
		},
		{
			name: "only returns suggestions matching prefix",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, SimpleCompleter[[]string]("zzz-1", "zzz-2", "yyy-3", "zzz-4")),
				),
				Args: "cmd zz",
				Want: []string{"zzz-1", "zzz-2", "zzz-4"},
				WantData: &Data{Values: map[string]interface{}{
					"sl": []string{"zz"},
				}},
			},
		},
		// Ensure completion iteration stops if necessary.
		{
			name: "stop iterating if a completion returns nil",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("PATH", "dd", SimpleCompleter[string]()),
					ListArg[string]("SUB_PATH", "stc", 0, UnboundedList, SimpleCompleter[[]string]("un", "deux", "trois")),
				),
				Args: "cmd p",
				WantData: &Data{Values: map[string]interface{}{
					"PATH": "p",
				}},
			},
		},
		{
			name: "stop iterating if a completion returns an error",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("PATH", "dd", CompleterFromFunc(func(string, *Data) (*Completion, error) {
						return nil, fmt.Errorf("ruh-roh")
					})),
					ListArg[string]("SUB_PATH", "stc", 0, UnboundedList, SimpleCompleter[[]string]("un", "deux", "trois")),
				),
				Args:    "cmd p",
				WantErr: fmt.Errorf("ruh-roh"),
				WantData: &Data{Values: map[string]interface{}{
					"PATH": "p",
				}},
			},
		},
		// Flag completion
		{
			name: "bool flag gets set if not last one",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd -g ",
				Want: []string{"1", "2"},
				WantData: &Data{
					Values: map[string]interface{}{
						"good": true,
					},
				},
			},
		},
		{
			name: "arg flag gets set if not last one",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --greeting howdy ",
				Want: []string{"1", "2"},
				WantData: &Data{
					Values: map[string]interface{}{
						"greeting": "howdy",
					},
				},
			},
		},
		{
			name: "list arg flag gets set if not last one",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --names alice bob charlie ",
				Want: []string{"1", "2"},
				WantData: &Data{
					Values: map[string]interface{}{
						"names": []string{"alice", "bob", "charlie"},
					},
				},
			},
		},
		{
			name: "multiple flags get set if not last one",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd -n alice bob charlie --good -h howdy ",
				Want: []string{"1", "2"},
				WantData: &Data{
					Values: map[string]interface{}{
						"names":    []string{"alice", "bob", "charlie"},
						"good":     true,
						"greeting": "howdy",
					},
				},
			},
		},
		{
			name: "flag name gets completed if single hyphen at end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd -",
				Want: []string{"--good", "--greeting", "--names", "-g", "-h", "-n"},
			},
		},
		{
			name: "flag name gets completed if double hyphen at end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --",
				Want: []string{"--good", "--greeting", "--names"},
			},
		},
		{
			name: "flag name gets completed if it's the only arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -",
				Want: []string{"--good", "--greeting", "--names", "-g", "-h", "-n"},
			},
		},
		{
			name: "completes for single flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 --greeting h",
				Want: []string{"hey", "hi"},
				WantData: &Data{Values: map[string]interface{}{
					"greeting": "h",
				}},
			},
		},
		{
			name: "completes for single short flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h he",
				Want: []string{"hey"},
				WantData: &Data{Values: map[string]interface{}{
					"greeting": "he",
				}},
			},
		},
		{
			name: "completes for list flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ",
				Want: []string{"johnny", "ralph", "renee"},
				WantData: &Data{Values: map[string]interface{}{
					"greeting": "hey",
					"names":    []string{""},
				}},
			},
		},
		{
			name: "completes distinct secondary for list flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleDistinctCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ralph ",
				Want: []string{"johnny", "renee"},
				WantData: &Data{Values: map[string]interface{}{
					"greeting": "hey",
					"names":    []string{"ralph", ""},
				}},
			},
		},
		{
			name: "completes last flag",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleDistinctCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
						NewFlag[float64]("float", 'f', testDesc, SimpleCompleter[float64]("1.23", "12.3", "123.4")),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ralph renee johnny -f ",
				Want: []string{"1.23", "12.3", "123.4"},
				WantData: &Data{Values: map[string]interface{}{
					"greeting": "hey",
					"names":    []string{"ralph", "renee", "johnny"},
				}},
			},
		},
		{
			name: "completes arg if flag arg isn't at the end",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						NewFlag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						NewListFlag[string]("names", 'n', testDesc, 1, 2, SimpleDistinctCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					ListArg[string]("i", testDesc, 1, 2, SimpleCompleter[[]string]("hey", "ooo")),
				),
				Args: "cmd 1 -h hello bravo --names ralph renee johnny ",
				Want: []string{"hey", "ooo"},
				WantData: &Data{Values: map[string]interface{}{
					"i":        []string{"1", "bravo", ""},
					"greeting": "hello",
					"names":    []string{"ralph", "renee", "johnny"},
				}},
			},
		},
		// Multi-flag tests
		{
			name: "Multi-flags don't get completed",
			ctc: &CompleteTestCase{
				Args: "cmd -qwer",
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
				),
				WantData: &Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    true,
					"everyone": true,
					"run":      true,
				}},
			},
		},
		{
			name: "Args after multi-flags get completed",
			ctc: &CompleteTestCase{
				Args: "cmd -qwer ",
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def", "ghi")),
				),
				Want: []string{"abc", "def", "ghi"},
				WantData: &Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    true,
					"everyone": true,
					"run":      true,
					"s":        "",
				}},
			},
		},
		{
			name: "Args after multi-flags get completed, even if unknown flag included",
			ctc: &CompleteTestCase{
				Args: "cmd -qwertyuiop ",
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def", "ghi")),
				),
				Want: []string{"abc", "def", "ghi"},
				WantData: &Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    true,
					"everyone": true,
					"run":      true,
					"to":       true,
					"s":        "",
				}},
			},
		},
		{
			name: "Args after multi-flags get completed, even if uncombinable flag is included",
			ctc: &CompleteTestCase{
				Args: "cmd -qwz ",
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						NewFlag[string]("zf", 'z', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def", "ghi")),
				),
				Want: []string{"abc", "def", "ghi"},
				WantData: &Data{Values: map[string]interface{}{
					"quick": true,
					"where": true,
					"s":     "",
				}},
			},
		},
		// Duplicate flag tests
		{
			name: "Repeated flag still gets completed",
			ctc: &CompleteTestCase{
				Args: "cmd -z firstZ -z ",
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						NewFlag[string]("zf", 'z', testDesc, SimpleCompleter[string]("zyx", "wvu", "tsr")),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Want: []string{"tsr", "wvu", "zyx"},
				WantData: &Data{Values: map[string]interface{}{
					"zf": "",
				}},
			},
		},
		{
			name: "Repeated flag still gets completed even if other repetition in multi-flags",
			ctc: &CompleteTestCase{
				Args: "cmd --quick -qwrqw --where -z firstZ -z ",
				Node: SerialNodes(
					NewFlagNode(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						NewFlag[string]("zf", 'z', testDesc, SimpleCompleter[string]("zyx", "wvu", "tsr")),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Want: []string{"tsr", "wvu", "zyx"},
				WantData: &Data{Values: map[string]interface{}{
					"quick": true,
					"where": true,
					"run":   true,
					"zf":    "",
				}},
			},
		},
		// Transformer arg tests.
		{
			name: "handles nil option",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc),
				},
				Args: "cmd abc",
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "abc",
				}},
			},
		},
		{
			name: "list handles nil option",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, 2),
				},
				Args: "cmd abc",
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{"abc"},
				}},
			},
		},
		{
			name: "transformer doesn't transform value during completion",
			ctc: &CompleteTestCase{
				Node: SerialNodes(Arg[string]("strArg", testDesc,
					NewTransformer(func(string, *Data) (string, error) {
						return "newStuff", nil
					}))),
				Args: "cmd abc",
				WantData: &Data{Values: map[string]interface{}{
					"strArg": "abc",
				}},
			},
		},
		{
			name:        "FileTransformer doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, FileTransformer()),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{Values: map[string]interface{}{
					"strArg": filepath.Join("relative", "path.txt"),
				}},
			},
		},
		{
			name:        "FileTransformer for list doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: ListArg[string]("strArg", testDesc, 1, 2, TransformerList(FileTransformer())),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{Values: map[string]interface{}{
					"strArg": []string{filepath.Join("relative", "path.txt")},
				}},
			},
		},
		{
			name:           "handles transform error",
			filepathAbsErr: fmt.Errorf("bad news bears"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: Arg[string]("strArg", testDesc, FileTransformer()),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &Data{Values: map[string]interface{}{
					"strArg": filepath.Join("relative", "path.txt"),
				}},
			},
		},
		{
			name:        "transformer list doesn't transforms values during completion",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, 2, NewTransformer(func(sl []string, d *Data) ([]string, error) {
						var r []string
						for _, s := range sl {
							r = append(r, fmt.Sprintf("_%s_", s))
						}
						return r, nil
					})),
				},
				Args: "cmd uno dos",
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{
						"uno",
						"dos",
					},
				}},
			},
		},
		{
			name:        "transformer list transforms values that aren't at the end",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("slArg", testDesc, 1, 1, NewTransformer(func(sl []string, d *Data) ([]string, error) {
						var r []string
						for _, s := range sl {
							r = append(r, fmt.Sprintf("_%s_", s))
						}
						return r, nil
					})),
					Arg[string]("sArg", testDesc, NewTransformer(func(s string, d *Data) (string, error) {
						return s + "!", nil
					})),
				),
				Args: "cmd uno dos t",
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{
						"_uno_",
						"_dos_",
					},
					"sArg": "t",
				}},
			},
		},
		{
			name:        "transformer list transforms values on a best effort basis",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("slArg", testDesc, 1, 1, NewTransformer(func(sl []string, d *Data) ([]string, error) {
						var r []string
						for _, s := range sl {
							r = append(r, fmt.Sprintf("_%s_", s))
						}
						return r, nil
					})),
					Arg[string]("sArg", testDesc, NewTransformer(func(s string, d *Data) (string, error) {
						return "oh", fmt.Errorf("Nooooooo")
					})),
					Arg[string]("sArg2", testDesc, NewTransformer(func(s string, d *Data) (string, error) {
						return "oh yea", fmt.Errorf("nope")
					})),
				),
				Args: "cmd uno dos tres q",
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{
						"_uno_",
						"_dos_",
					},
					"sArg":  "tres",
					"sArg2": "q",
				}},
			},
		},
		{
			name:           "handles transform list error",
			filepathAbsErr: fmt.Errorf("bad news bears"),
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, 2, TransformerList(FileTransformer())),
				},
				Args: fmt.Sprintf("cmd %s %s", filepath.Join("relative", "path.txt"), filepath.Join("other.txt")),
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{
						filepath.Join("relative", "path.txt"),
						filepath.Join("other.txt"),
					},
				}},
			},
		},
		{
			name: "handles list transformer of incorrect type",
			ctc: &CompleteTestCase{
				Node: &Node{
					Processor: ListArg[string]("slArg", testDesc, 1, 2, TransformerList(FileTransformer())),
				},
				Args: "cmd 123",
				WantData: &Data{Values: map[string]interface{}{
					"slArg": []string{"123"},
				}},
			},
		},
		// FileNode
		{
			name:        "FileNode includes a vanilla FileCompleter",
			filepathAbs: filepath.Join("."),
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					FileNode("fn", testDesc),
				),
				Args: "cmd ",
				WantData: &Data{Values: map[string]interface{}{
					"fn": "",
				}},
				Want: []string{
					".git/",
					"_testdata_symlink/",
					"arg.go",
					"autocomplete.go",
					"bash_node.go",
					"bash_node_test.go",
					"bool_operator.go",
					"cache.go",
					"cache/",
					"cache_test.go",
					"cmd/",
					"color/",
					"commandtest.go",
					"completer.go",
					"completer_test.go",
					"custom_nodes.go",
					"data.go",
					"data_test.go",
					"docs/",
					"execute.go",
					"execute_test.go",
					"file_functions.go",
					"file_functions.txt",
					"flag.go",
					"float_operator.go",
					"go.mod",
					"go.sum",
					"input.go",
					"input_test.go",
					"int_operator.go",
					"option.go",
					"os.go",
					"output.go",
					"output_test.go",
					"prompt.go",
					"README.md",
					"shortcut.go",
					"shortcut_test.go",
					"sourcerer/",
					"static_cli.go",
					"static_cli_test.go",
					"stdin.go",
					"string_operator.go",
					"testdata/",
					"usage_test.go",
					"validator.go",
					" ",
				},
			},
		},
		{
			name:        "FileNode uses provided FileCompleter option",
			filepathAbs: filepath.Join("."),
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					FileNode("fn", testDesc, &FileCompleter[string]{
						FileTypes: []string{".sum", ".mod"},
					}),
				),
				Args: "cmd ",
				WantData: &Data{Values: map[string]interface{}{
					"fn": "",
				}},
				Want: []string{
					".git/",
					"_testdata_symlink/",
					"cache/",
					"cmd/",
					"color/",
					"docs/",
					"go.mod",
					"go.sum",
					"sourcerer/",
					"testdata/",
					" ",
				},
			},
		},
		{
			name:        "FileCompleter works with absolute path",
			filepathAbs: filepath.Join("."),
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("fn", testDesc, CompleterFromFunc(func(s string, d *Data) (*Completion, error) {
						_, thisFile, _, ok := runtime.Caller(0)
						if !ok {
							return nil, fmt.Errorf("failed to get runtime caller")
						}
						fc := &FileCompleter[string]{
							Directory: filepath.Join(filepath.Dir(thisFile), "testdata"),
						}
						return fc.Complete(s, d)
					})),
				),
				Args: "cmd ",
				WantData: &Data{Values: map[string]interface{}{
					"fn": "",
				}},
				Want: []string{
					".surprise",
					"cases/",
					"dir1/",
					"dir2/",
					"dir3/",
					"dir4/",
					"four.txt",
					"METADATA",
					"metadata_/",
					"moreCases/",
					"one.txt",
					"three.txt",
					"two.txt",
					" ",
				},
			},
		},
		// BranchNode completion tests.
		{
			name: "completes branch options",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts")))),
				Want: []string{"a", "alpha", "bravo", "command", "default", "opts"},
				WantData: &Data{Values: map[string]interface{}{
					"default": []string{""},
				}},
			},
		},
		{
			name: "doesn't complete branch options if complete arg is false",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts"))), DontCompleteSubcommands()),
				Want: []string{"command", "default", "opts"},
				WantData: &Data{Values: map[string]interface{}{
					"default": []string{""},
				}},
			},
		},
		{
			name: "completes for specific branch",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts")))),
				Args: "cmd alpha ",
				Want: []string{"other", "stuff"},
				WantData: &Data{Values: map[string]interface{}{
					"hello": "",
				}},
			},
		},
		{
			name: "branch node doesn't complete if no default and no branch match",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, nil),
				Args:    "cmd some thing else",
				WantErr: fmt.Errorf("Branching argument must be one of [a alpha bravo]"),
			},
		},
		{
			name: "branch node returns default node error if branch completion is false",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(SimpleProcessor(nil, func(i *Input, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("bad news bears")
				})), DontCompleteSubcommands()),
				Args:    "cmd ",
				WantErr: fmt.Errorf("bad news bears"),
			},
		},
		{
			name: "branch node returns default node error and branch completions",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(SimpleProcessor(nil, func(i *Input, d *Data) (*Completion, error) {
					return nil, fmt.Errorf("bad news bears")
				}))),
				Args:    "cmd ",
				Want:    []string{"a", "alpha", "bravo"},
				WantErr: fmt.Errorf("bad news bears"),
			},
		},
		{
			name: "completes branch options with partial completion",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts", "ahhhh")))),
				Args: "cmd a",
				Want: []string{"a", "ahhhh", "alpha"},
				WantData: &Data{Values: map[string]interface{}{
					"default": []string{"a"},
				}},
			},
		},
		{
			name: "completes default options",
			ctc: &CompleteTestCase{
				Node: BranchNode(map[string]*Node{
					"a":     {},
					"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
					"bravo": {},
				}, SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts")))),
				Args: "cmd something ",
				WantData: &Data{Values: map[string]interface{}{
					"default": []string{"something", ""},
				}},
				Want: []string{"command", "default", "opts"},
			},
		},
		// MenuArg tests.
		{
			name: "MenuArg completes choices",
			ctc: &CompleteTestCase{
				Node: SerialNodes(MenuArg("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd ",
				Want: []string{"abc", "def", "ghi"},
				WantData: &Data{Values: map[string]interface{}{
					"sm": "",
				}},
			},
		},
		{
			name: "MenuArg completes partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(MenuArg("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd g",
				Want: []string{"ghi"},
				WantData: &Data{Values: map[string]interface{}{
					"sm": "g",
				}},
			},
		},
		{
			name: "MenuArg completes none if no match",
			ctc: &CompleteTestCase{
				Node: SerialNodes(MenuArg("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd j",
				WantData: &Data{Values: map[string]interface{}{
					"sm": "j",
				}},
			},
		},
		{
			name: "MenuFlag completes choices",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: "cmd --sf ",
				Want: []string{"abc", "def", "ghi"},
				WantData: &Data{Values: map[string]interface{}{
					"sf": "",
				}},
			},
		},
		{
			name: "MenuArg completes partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: "cmd -s g",
				Want: []string{"ghi"},
				WantData: &Data{Values: map[string]interface{}{
					"sf": "g",
				}},
			},
		},
		{
			name: "MenuFlag completes none",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewFlagNode(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: "cmd -s j",
				WantData: &Data{Values: map[string]interface{}{
					"sf": "j",
				}},
			},
		},
		// Commands with different value types.
		{
			name: "int arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(Arg[int]("iArg", testDesc, SimpleCompleter[int]("12", "45", "456", "468", "7"))),
				Args: "cmd 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]interface{}{
					"iArg": 4,
				}},
			},
		},
		{
			name: "optional int arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(OptionalArg[int]("iArg", testDesc, SimpleCompleter[int]("12", "45", "456", "468", "7"))),
				Args: "cmd 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]interface{}{
					"iArg": 4,
				}},
			},
		},
		{
			name: "int list arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[int]("iArg", testDesc, 2, 3, SimpleCompleter[[]int]("12", "45", "456", "468", "7"))),
				Args: "cmd 1 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]interface{}{
					"iArg": []int{1, 4},
				}},
			},
		},
		{
			name: "int list arg gets completed if previous one was invalid",
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[int]("iArg", testDesc, 2, 3, SimpleCompleter[[]int]("12", "45", "456", "468", "7"))),
				Args: "cmd one 4",
				Want: []string{"45", "456", "468"},
			},
		},
		{
			name: "int list arg optional args get completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(ListArg[int]("iArg", testDesc, 2, 3, SimpleCompleter[[]int]("12", "45", "456", "468", "7"))),
				Args: "cmd 1 2 3 4",
				Want: []string{"45", "456", "468"},
				WantData: &Data{Values: map[string]interface{}{
					"iArg": []int{1, 2, 3, 4},
				}},
			},
		},
		{
			name: "float arg gets completed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(Arg[float64]("fArg", testDesc, SimpleCompleter[float64]("12", "4.5", "45.6", "468", "7"))),
				Args: "cmd 4",
				Want: []string{"4.5", "45.6", "468"},
				WantData: &Data{Values: map[string]interface{}{
					"fArg": 4.0,
				}},
			},
		},
		{
			name: "float list arg gets completed",
			ctc: &CompleteTestCase{
				Node:     SerialNodes(ListArg[float64]("fArg", testDesc, 1, 2, SimpleCompleter[[]float64]("12", "4.5", "45.6", "468", "7"))),
				Want:     []string{"12", "4.5", "45.6", "468", "7"},
				WantData: &Data{Values: map[string]interface{}{}},
			},
		},
		{
			name: "bool arg gets completed",
			ctc: &CompleteTestCase{
				Node:     SerialNodes(BoolNode("bArg", testDesc)),
				Want:     []string{"0", "1", "F", "FALSE", "False", "T", "TRUE", "True", "f", "false", "t", "true"},
				WantData: &Data{Values: map[string]interface{}{}},
			},
		},
		// NodeRepeater
		{
			name: "NodeRepeater completes first node",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Want: []string{"alpha", "bravo", "brown", "charlie"},
				WantData: &Data{Values: map[string]interface{}{
					"keys": []string{""},
				}},
			},
		},
		{
			name: "NodeRepeater completes first node partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Args: "cmd b",
				Want: []string{"bravo", "brown"},
				WantData: &Data{Values: map[string]interface{}{
					"keys": []string{"b"},
				}},
			},
		},
		{
			name: "NodeRepeater completes second node",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Args: "cmd brown ",
				Want: []string{"1", "121", "1213121"},
				WantData: &Data{Values: map[string]interface{}{
					"keys": []string{"brown"},
				}},
			},
		},
		{
			name: "NodeRepeater completes second node partial",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(1, 2)),
				Args: "cmd brown 12",
				Want: []string{"121", "1213121"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"brown"},
					"values": []int{12},
				}},
			},
		},
		{
			name: "NodeRepeater completes second required iteration",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 0)),
				Args: "cmd brown 12 c",
				Want: []string{"charlie"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "c"},
					"values": []int{12},
				}},
			},
		},
		{
			name: "NodeRepeater completes optional iteration",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 1)),
				Args: "cmd brown 12 charlie 21 alpha 1",
				Want: []string{"1", "121", "1213121"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha"},
					"values": []int{12, 21, 1},
				}},
			},
		},
		{
			name: "NodeRepeater completes unbounded optional iteration",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, UnboundedList)),
				Args: "cmd brown 12 charlie 21 alpha 100 delta 98 b",
				Want: []string{"bravo", "brown"},
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha", "delta", "b"},
					"values": []int{12, 21, 100, 98},
				}},
			},
		},
		{
			name: "NodeRepeater doesn't complete beyond repeated iterations",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 1)),
				Args: "cmd brown 12 charlie 21 alpha 100 b",
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha"},
					"values": []int{12, 21, 100},
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [b]"),
			},
		},
		{
			name: "NodeRepeater works if fully processed",
			ctc: &CompleteTestCase{
				Node: SerialNodes(sampleRepeaterNode(2, 1), Arg[string]("S", testDesc, SimpleCompleter[string]("un", "deux", "trois"))),
				Args: "cmd brown 12 charlie 21 alpha 100",
				WantData: &Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha"},
					"values": []int{12, 21, 100},
				}},
			},
		},
		// ListBreaker tests
		{
			name: "Suggests things after broken list",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi"), SimpleCompleter[[]string]("un", "deux", "trois")),
					ListArg[string]("SL2", testDesc, 0, UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: []string{"one", "three", "two"},
				WantData: &Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{"ghi", ""},
				}},
			},
		},
		{
			name: "Suggests things after broken list with discard",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi", DiscardBreaker()), SimpleCompleter[[]string]("un", "deux", "trois")),
					ListArg[string]("SL2", testDesc, 0, UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: []string{"one", "three", "two"},
				WantData: &Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{""},
				}},
			},
		},
		{
			name: "Suggests things before list is broken",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, UnboundedList, ListUntilSymbol("ghi"), SimpleCompleter[[]string]("un", "deux", "trois", "uno")),
					ListArg[string]("SL2", testDesc, 0, UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def un",
				Want: []string{"un", "uno"},
				WantData: &Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def", "un"},
				}},
			},
		},
		// StringListListNode
		{
			name: "StringListListNode works if no breakers",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: []string{"one", "three", "two"},
				WantData: &Data{Values: map[string]interface{}{
					"SLL": [][]string{{"abc", "def", "ghi", ""}},
				}},
			},
		},
		{
			name: "StringListListNode works with breakers",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def | ghi t",
				Want: []string{"three", "two"},
				WantData: &Data{Values: map[string]interface{}{
					"SLL": [][]string{{"abc", "def"}, {"ghi", "t"}},
				}},
			},
		},
		{
			name: "completes args after StringListListNode",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					StringListListNode("SLL", testDesc, "|", 1, 1, SimpleCompleter[[]string]("one", "two", "three")),
					Arg[string]("S", testDesc, SimpleCompleter[string]("un", "deux", "trois")),
				),
				Args: "cmd abc def | ghi | ",
				Want: []string{"deux", "trois", "un"},
				WantData: &Data{
					Values: map[string]interface{}{
						"SLL": [][]string{{"abc", "def"}, {"ghi"}},
						"S":   "",
					},
				},
			},
		},
		// BashNode
		{
			name: "BashNode runs in Completion context",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewBashCommand[string]("b", []string{"echo haha"}),
					Arg[string]("s", testDesc),
				),
				RunResponses: []*FakeRun{{
					Stdout: []string{"hehe"},
				}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo haha",
				}},
				WantData: &Data{Values: map[string]interface{}{
					"b": "hehe",
					"s": "",
				}},
			},
		},
		{
			name: "BashNode fails in Completion context",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewBashCommand[string]("b", []string{"echo haha"}),
					Arg[string]("s", testDesc),
				),
				RunResponses: []*FakeRun{{
					Err: fmt.Errorf("argh"),
				}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo haha",
				}},
				WantErr: fmt.Errorf("failed to execute bash command: argh"),
			},
		},
		{
			name: "BashNode does not run in Completion context when option provided",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					NewBashCommand("b", []string{"echo haha"}, DontRunOnComplete[string]()),
					Arg[string]("s", testDesc),
				),
				WantData: &Data{Values: map[string]interface{}{
					"s": "",
				}},
			},
		},
		// BashCompleter
		{
			name: "BashCompleter doesn't complete if bash failure",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, BashCompleter[string]("echo abc def ghi")),
				),
				RunResponses: []*FakeRun{{
					Err: fmt.Errorf("oopsie"),
				}},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo abc def ghi",
				}},
				WantErr:  fmt.Errorf("failed to fetch autocomplete suggestions with bash command: failed to execute bash command: oopsie"),
				WantData: &Data{Values: map[string]interface{}{"s": ""}},
			},
		},
		{
			name: "BashCompleter completes even if wrong type returned (since just fetches string list)",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[int]("i", testDesc, BashCompleter[int]("echo abc def ghi")),
				),
				RunResponses: []*FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: []string{
					"abc",
					"def",
					"ghi",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo abc def ghi",
				}},
				WantData: &Data{Values: map[string]interface{}{}},
			},
		},
		{
			name: "BashCompleter completes arg",
			ctc: &CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, BashCompleter[string]("echo abc def ghi")),
				),
				RunResponses: []*FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: []string{
					"abc",
					"def",
					"ghi",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo abc def ghi",
				}},
				//WantErr: fmt.Errorf(`failed to fetch autocomplete suggestions with bash command: strconv.Atoi: parsing "abc def ghi": invalid syntax`),
				WantData: &Data{Values: map[string]interface{}{"s": ""}},
			},
		},
		{
			name: "BashCompleter completes arg with partial completion",
			ctc: &CompleteTestCase{
				Args: "cmd d",
				Node: SerialNodes(
					Arg[string]("s", testDesc, BashCompleter[string]("echo abc def ghi")),
				),
				RunResponses: []*FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: []string{
					"def",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo abc def ghi",
				}},
				WantData: &Data{Values: map[string]interface{}{"s": "d"}},
			},
		},
		{
			name: "BashCompleter completes arg with opts",
			ctc: &CompleteTestCase{
				Args: "cmd abc ghi ",
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, BashCompleterWithOpts[[]string](&Completion{Distinct: true}, "echo abc def ghi")),
				),
				RunResponses: []*FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: []string{
					"def",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"echo abc def ghi",
				}},
				WantData: &Data{Values: map[string]interface{}{"sl": []string{"abc", "ghi", ""}}},
			},
		},
		// ConditionalProcessor tests
		{
			name: "ConditionProcessor runs if function returns true",
			ctc: &CompleteTestCase{
				Args: "cmd alpha ",
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					ConditionalProcessor(
						Arg[string]("s2", testDesc, SimpleCompleter[string]("bravo", "charlie")),
						func(i *Input, d *Data) bool {
							return true
						},
					),
					Arg[string]("s3", testDesc, SimpleCompleter[string]("delta", "epsilon")),
				),
				WantData: &Data{Values: map[string]interface{}{
					"s":  "alpha",
					"s2": "",
				}},
				Want: []string{"bravo", "charlie"},
			},
		},
		{
			name: "ConditionProcessor does not run if function returns true",
			ctc: &CompleteTestCase{
				Args: "cmd alpha ",
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					ConditionalProcessor(
						Arg[string]("s2", testDesc, SimpleCompleter[string]("bravo", "charlie")),
						func(i *Input, d *Data) bool {
							return false
						},
					),
					Arg[string]("s3", testDesc, SimpleCompleter[string]("delta", "epsilon")),
				),
				WantData: &Data{Values: map[string]interface{}{
					"s":  "alpha",
					"s3": "",
				}},
				Want: []string{"delta", "epsilon"},
			},
		},
		/* Useful comment for commenting out tests */
	} {
		t.Run(test.name, func(t *testing.T) {
			StubValue(t, &filepathAbs, func(s string) (string, error) {
				return filepath.Join(test.filepathAbs, s), test.filepathAbsErr
			})
			CompleteTest(t, test.ctc)
		})
	}
}

func printNode(s string) *Node {
	return &Node{
		Processor: ExecutorNode(func(output Output, _ *Data) {
			output.Stdout(s)
		}),
	}
}

func printlnNode(stdout bool, a ...interface{}) *Node {
	return &Node{
		Processor: ExecuteErrNode(func(output Output, _ *Data) error {
			if !stdout {
				return output.Stderrln(a...)
			}
			output.Stdoutln(a...)
			return nil
		}),
	}
}

func printArgsNode() *Node {
	return &Node{
		Processor: ExecutorNode(func(output Output, data *Data) {
			var keys []string
			for k := range data.Values {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				output.Stdoutf("%s: %v\n", k, data.Values[k])
			}
		}),
	}
}

func sampleRepeaterNode(minN, optionalN int) Processor {
	return NodeRepeater(SerialNodes(
		Arg[string]("KEY", testDesc, CustomSetter(func(v string, d *Data) {
			if !d.Has("keys") {
				d.Set("keys", []string{v})
			} else {
				d.Set("keys", append(d.StringList("keys"), v))
			}
		}), SimpleCompleter[string]("alpha", "bravo", "charlie", "brown")),
		Arg[int]("VALUE", testDesc, CustomSetter(func(v int, d *Data) {
			if !d.Has("values") {
				d.Set("values", []int{v})
			} else {
				d.Set("values", append(d.IntList("values"), v))
			}
		}), SimpleCompleter[int]("1", "121", "1213121")),
	), minN, optionalN)
}

func TestRunNodes(t *testing.T) {
	sum := SerialNodes(
		Description("Adds A and B"),
		Arg[int]("A", "The first value"),
		Arg[int]("B", "The second value"),
		ExecutorNode(func(o Output, d *Data) {
			o.Stdoutln(d.Int("A") + d.Int("B"))
		}),
	)
	for _, test := range []struct {
		name string
		rtc  *RunNodeTestCase
	}{
		// execute tests (without keyword)
		{
			name: "no keyword requires arguments",
			rtc: &RunNodeTestCase{
				Node: sum,
				WantStderr: strings.Join([]string{
					`Argument "A" requires at least 1 argument, got 0`,
					fmt.Sprintf("\n======= Command Usage =======\n%s", GetUsage(sum).String()),
					"",
				}, "\n"),
				WantErr: fmt.Errorf(`Argument "A" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "no keyword fails with extra",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"5", "7", "9"},
				WantStderr: strings.Join([]string{
					`Unprocessed extra args: [9]`,
					fmt.Sprintf("\n======= Command Usage =======\n%s", GetUsage(sum).String()),
					"",
				}, "\n"),
				WantErr: fmt.Errorf(`Unprocessed extra args: [9]`),
			},
		},
		{
			name: "successfully runs node if no keyword provided",
			rtc: &RunNodeTestCase{
				Node:       sum,
				Args:       []string{"5", "7"},
				WantStdout: "12\n",
			},
		},
		// execute tests with keyword
		{
			name: "execute requires arguments",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"execute", "TMP_FILE"},
				WantStderr: strings.Join([]string{
					`Argument "A" requires at least 1 argument, got 0`,
					fmt.Sprintf("\n======= Command Usage =======\n%s", GetUsage(sum).String()),
					"",
				}, "\n"),
				WantErr: fmt.Errorf(`Argument "A" requires at least 1 argument, got 0`),
			},
		},
		{
			name: "execute fails with extra",
			rtc: &RunNodeTestCase{
				Node: sum,
				Args: []string{"execute", "TMP_FILE", "5", "7", "9"},
				WantStderr: strings.Join([]string{
					`Unprocessed extra args: [9]`,
					fmt.Sprintf("\n======= Command Usage =======\n%s", GetUsage(sum).String()),
					"",
				}, "\n"),
				WantErr: fmt.Errorf(`Unprocessed extra args: [9]`),
			},
		},
		{
			name: "successfully runs node via execute keyword",
			rtc: &RunNodeTestCase{
				Node:       sum,
				Args:       []string{"execute", "TMP_FILE", "5", "7"},
				WantStdout: "12\n",
			},
		},
		{
			name: "execute data",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					SimpleExecutableNode(
						"echo hello",
						"echo there",
					),
				),
				Args: []string{"execute", "TMP_FILE"},
				WantFileContents: []string{
					"echo hello",
					"echo there",
				},
			},
		},
		// Autocomplete tests
		{
			name: "autocompletes empty",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					ListArg[string]("SL_ARG", "", 1, UnboundedList, SimpleCompleter[[]string]("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", ""},
				WantStdout: strings.Join([]string{
					"four",
					"one",
					"three",
					"two",
					"",
				}, "\n"),
			},
		},
		{
			name: "autocompletes empty with command",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					ListArg[string]("SL_ARG", "", 1, UnboundedList, SimpleCompleter[[]string]("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", "cmd "},
				WantStdout: strings.Join([]string{
					"four",
					"one",
					"three",
					"two",
					"",
				}, "\n"),
			},
		},
		{
			name: "autocompletes partial arg",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					ListArg[string]("SL_ARG", "", 1, UnboundedList, SimpleCompleter[[]string]("one", "two", "three", "four")),
				),
				Args: []string{"autocomplete", "cmd t"},
				WantStdout: strings.Join([]string{
					"three",
					"two",
					"",
				}, "\n"),
			},
		},
		{
			name: "autocompletes later args",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					ListArg[string]("SL_ARG", "", 1, UnboundedList, SimpleCompleter[[]string]("one", "two", "three", "four")),
				),
				Args:       []string{"autocomplete", "cmd three f"},
				WantStdout: "four\n",
			},
		},
		{
			name: "autocompletes nothing if past last arg",
			rtc: &RunNodeTestCase{
				Node: SerialNodes(
					ListArg[string]("SL_ARG", "", 1, 0, SimpleCompleter[[]string]("one", "two", "three", "four")),
				),
				Args:       []string{"autocomplete", "cmd three f"},
				WantStderr: "Unprocessed extra args: [f]\n",
				WantErr:    fmt.Errorf("Unprocessed extra args: [f]"),
			},
		},
		// Usage tests
		{
			name: "prints usage",
			rtc: &RunNodeTestCase{
				Node:       sum,
				Args:       []string{"usage"},
				WantStdout: GetUsage(sum).String() + "\n",
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.rtc.SkipDataCheck = true
			RunNodeTest(t, test.rtc)
		})
	}
}

// testBoolFlag and testBoolFlagInterface allows us to use one test
// method with test cases that have different parameterized types.
type testBoolFlag[T any] struct {
	name      string
	bf        *boolFlag[T]
	wantTrue  T
	wantFalse T
}

func (tbf *testBoolFlag[T]) runTest(t *testing.T) {
	t.Run(tbf.name, func(t *testing.T) {
		if diff := cmp.Diff(tbf.wantTrue, tbf.bf.TrueValue()); diff != "" {
			t.Errorf("boolFlag.TrueValue() returned incorrect value (-want, +got):\n%s", diff)
		}

		if diff := cmp.Diff(tbf.wantFalse, tbf.bf.FalseValue()); diff != "" {
			t.Errorf("boolFlag.FalseValue() returned incorrect value (-want, +got):\n%s", diff)
		}
	})
}

type testBoolFlagInterface interface {
	runTest(t *testing.T)
}

func TestBoolFlag(t *testing.T) {
	a := "asdf"
	pa := &a
	z := "zxcv"
	pz := &z
	for _, test := range []testBoolFlagInterface{
		&testBoolFlag[string]{
			name:      "Works for BoolValueFlag[string]",
			bf:        BoolValueFlag("bf", 'b', testDesc, "asdf"),
			wantTrue:  "asdf",
			wantFalse: "",
		},
		&testBoolFlag[string]{
			name:      "Works for BoolValuesFlag[string]",
			bf:        BoolValuesFlag("bf", 'b', testDesc, "asdf", "zxcv"),
			wantTrue:  "asdf",
			wantFalse: "zxcv",
		},
		&testBoolFlag[*string]{
			name:     "Works for BoolValueFlag[*string]",
			bf:       BoolValueFlag("bf", 'b', testDesc, pa),
			wantTrue: pa,
		},
		&testBoolFlag[*string]{
			name:      "Works for BoolValuesFlag[*string]",
			bf:        BoolValuesFlag("bf", 'b', testDesc, pa, pz),
			wantTrue:  pa,
			wantFalse: pz,
		},
		&testBoolFlag[*string]{
			name:     "Works for BoolValuesFlag[*string] when false value is explicitly nil",
			bf:       BoolValuesFlag("bf", 'b', testDesc, pa, nil),
			wantTrue: pa,
		},
	} {
		test.runTest(t)
	}
}
