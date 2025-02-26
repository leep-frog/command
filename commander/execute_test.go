package commander

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/spycommand"
	"github.com/leep-frog/command/internal/spycommander"
	"github.com/leep-frog/command/internal/spycommandertest"
	"github.com/leep-frog/command/internal/spycommandtest"
	"github.com/leep-frog/command/internal/stubs"
	"github.com/leep-frog/command/internal/testutil"
)

// executeTest is a wrapper around spycommandertest.ExecuteTest
func executeTest(t *testing.T, etc *commandtest.ExecuteTestCase, ietc *spycommandtest.ExecuteTestCase) {
	t.Helper()
	if ietc == nil {
		ietc = &spycommandtest.ExecuteTestCase{}
	}
	ietc.SkipInputCheck = false
	ietc.SkipErrorTypeCheck = false
	spycommandertest.ExecuteTest(t, etc, ietc, &spycommandertest.ExecuteTestFunctionBag{
		spycommander.Execute,
		spycommander.Use,
		SetupArg,
		SerialNodes,
		spycommander.HelpBehavior,
		IsBranchingError,
		IsUsageError,
		IsNotEnoughArgsError,
		command.IsExtraArgsError,
		IsValidationError,
	})
}

// changeTest is a wrapper around spycommandertest.ChangeTest
func changeTest[T commandtest.Changeable](t *testing.T, want, original T, opts ...cmp.Option) {
	t.Helper()
	spycommandertest.ChangeTest[T](t, want, original, opts...)
}

// autocompleteTest is a wrapper around spycommandertest.AutocompleteTest
func autocompleteTest(t *testing.T, ctc *commandtest.CompleteTestCase, ictc *spycommandtest.CompleteTestCase) {
	t.Helper()
	if ictc == nil {
		ictc = &spycommandtest.CompleteTestCase{}
	}
	ictc.SkipErrorTypeCheck = false
	spycommandertest.AutocompleteTest(t, ctc, ictc, &spycommandertest.CompleteTestFunctionBag{
		spycommander.Autocomplete,
		IsBranchingError,
		IsUsageError,
		IsNotEnoughArgsError,
		command.IsExtraArgsError,
		IsValidationError,
	})
}

type errorEdge struct {
	e error
}

type simpleType struct {
	S string
	N int
}

func (ee *errorEdge) Next(*command.Input, *command.Data) (command.Node, error) {
	return nil, ee.e
}

func (ee *errorEdge) UsageNext(input *command.Input, data *command.Data) (command.Node, error) {
	return nil, nil
}

func TestExecute(t *testing.T) {

	StubRuntimeCaller(t, "some/file/path", true)
	rcNode := RuntimeCaller()
	StubRuntimeCaller(t, "some/file/path", false)
	rcErrNode := RuntimeCaller()

	_ = rcNode
	_ = rcErrNode

	envArgProcessor := &EnvArg{
		Name:     "ENV_VAR",
		Optional: true,
	}
	optionalString := OptionalArg[string]("opt-arg", "desc")
	stringFlag := Flag[string]("opt-flag", 'o', "flag-desc")
	simpleBoolFlag := BoolFlag("bool-flag", 'b', "bool-flag-desc")

	fos := &commandtest.FakeOS{}
	for _, test := range []struct {
		name       string
		etc        *commandtest.ExecuteTestCase
		ietc       *spycommandtest.ExecuteTestCase
		osGetwd    string
		osGetwdErr error
		postCheck  func(*testing.T)
	}{
		{
			name: "handles nil node",
		},
		{
			name: "fails if unprocessed args",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"hello"},
				WantErr:    fmt.Errorf("Unprocessed extra args: [hello]"),
				WantStderr: "Unprocessed extra args: [hello]\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
					Remaining: []int{0},
				},
			},
		},
		// Single arg tests.
		{
			name: "Fails if arg and no argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(Arg[string]("s", testDesc)),
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"s\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "Fails if edge fails",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"hello"},
				Node: &SimpleNode{
					Processor: Arg[string]("s", testDesc),
					Edge: &errorEdge{
						e: fmt.Errorf("bad news bears"),
					},
				},
				WantErr: fmt.Errorf("bad news bears"),
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "Fails if int arg and no argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(Arg[int]("i", testDesc)),
				WantErr:    fmt.Errorf(`Argument "i" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"i\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "Fails if float arg and no argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(Arg[float64]("f", testDesc)),
				WantErr:    fmt.Errorf(`Argument "f" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"f\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		// Complexecute tests for single Arg
		{
			name: "Complexecute for Arg fails if no arg provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				WantErr:    fmt.Errorf(`Argument "is" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"is\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "Complexecute for Arg fails completer returns error",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				WantErr:    fmt.Errorf("[Complexecute] failed to fetch completion for \"is\": oopsie"),
				WantStderr: "[Complexecute] failed to fetch completion for \"is\": oopsie\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg fails if returned completion is nil",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return nil, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] nil completion returned for \"is\""),
				WantStderr: "[Complexecute] nil completion returned for \"is\"\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg fails if 0 suggestions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return &command.Completion{}, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"is\", got 0: []"),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"is\", got 0: []\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg fails if multiple suggestions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"1", "4"},
					}, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"is\", got 2: [1 4]"),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"is\", got 2: [1 4]\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg fails if suggestions is wrong type",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"someString"},
					}, nil
				}))),
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "someString": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"someString\": invalid syntax\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "someString"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg works if one suggestion",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"123"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"is": 123,
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "123"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg completes on best effort",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{Lenient: true}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"123"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"is": 123,
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "123"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg doesn't complete or error on best effort if no suggestions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{Lenient: true}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "h"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg doesn't complete or error on best effort if multiple suggestions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{Lenient: true}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"hey", "hi"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "h"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg doesn't complete or error on best effort if error",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{Lenient: true}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "h"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg doesn't complete or error on best effort if nil command.Completion",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"h"},
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{Lenient: true}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
					return nil, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "h",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "h"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg works when only one prefix matches",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"4"},
				Node: SerialNodes(Arg[int]("is", testDesc, &Complexecute[int]{}, CompleterFromFunc(func(i int, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"123", "234", "345", "456", "567"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"is": 456,
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "456"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg fails if multiple completions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"f"},
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"one", "two", "three", "four", "five", "six"},
					}, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]"),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "f"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg works for string",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"fi"},
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"one", "two", "three", "four", "five", "six"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "five",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "five"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg works for multiple, independent args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"fi", "tr"},
				Node: SerialNodes(
					Arg[string]("s", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"one", "two", "three", "four", "five", "six"},
						}, nil
					})),
					Arg[string]("s2", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"un", "deux", "trois", "quatre"},
						}, nil
					})),
				),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s":  "five",
						"s2": "trois",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "five"},
						{Value: "trois"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg fails if one of completions fails for independent args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"fi", "mouse", "tr"},
				Node: SerialNodes(
					Arg[string]("s", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"one", "two", "three", "four", "five", "six"},
						}, nil
					})),
					Arg[string]("s2", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return nil, fmt.Errorf("rats")
					})),
					Arg[string]("s3", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"un", "deux", "trois", "quatre"},
						}, nil
					})),
				),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "five",
					},
				},
				WantStderr: "[Complexecute] failed to fetch completion for \"s2\": rats\n",
				WantErr:    fmt.Errorf("[Complexecute] failed to fetch completion for \"s2\": rats"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "five"},
						{Value: "mouse"},
						{Value: "tr"},
					},
					Remaining: []int{2},
				},
			},
		},
		{
			name: "Complexecute for Arg works if one of completions fails on best effort for independent args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"fi", "mouse", "tr"},
				Node: SerialNodes(
					Arg[string]("s", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"one", "two", "three", "four", "five", "six"},
						}, nil
					})),
					Arg[string]("s2", testDesc, &Complexecute[string]{Lenient: true}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return nil, fmt.Errorf("rats")
					})),
					Arg[string]("s3", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"un", "deux", "trois", "quatre"},
						}, nil
					})),
				),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s":  "five",
						"s2": "mouse",
						"s3": "trois",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "five"},
						{Value: "mouse"},
						{Value: "trois"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg transforms last arg *after* Complexecute",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{},
					CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"abc"},
						}, nil
					}),
					&Transformer[string]{F: func(s string, d *command.Data) (string, error) {
						return s + "?", nil
					}},
				)),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "abc?",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc?"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg transforms last arg *after* Complexecute and sub completion",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"bra"},
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{},
					CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"alpha", "bravo", "charlie", "brown"},
						}, nil
					}),
					&Transformer[string]{F: func(s string, d *command.Data) (string, error) {
						return s + "?", nil
					}},
				)),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "bravo?",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "bravo?"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg with transformer fails if no match",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"br"},
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{},
					CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"alpha", "bravo", "charlie", "brown"},
						}, nil
					}),
					&Transformer[string]{F: func(s string, d *command.Data) (string, error) {
						return s + "?", nil
					}},
				)),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [bravo brown]\n",
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [bravo brown]"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "br"},
					},
				},
			},
		},
		{
			name: "Complexecute for Arg transforms last arg if Complexecute fails with best effort",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"br"},
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{Lenient: true},
					CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"alpha", "bravo", "charlie", "brown"},
						}, nil
					}),
					&Transformer[string]{F: func(s string, d *command.Data) (string, error) {
						return s + "?", nil
					}},
				)),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "br?",
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "br?"},
					},
				},
			},
		},
		{
			name: "Complexecute is properly set in data",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
					d.Set("CFE", d.Complexecute)
					return &command.Completion{Suggestions: []string{"abcde"}}, nil
				}))),
				Args: []string{"ab"},
				WantData: &command.Data{Values: map[string]interface{}{
					"CFE": true,
					"s":   "abcde",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abcde"},
					},
				},
			},
		},
		{
			name: "Complexecute succeeds if exact match",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{},
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"Hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "Hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "Hello"},
					},
				},
			},
		},
		{
			name: "Complexecute fails if partial match",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{},
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args:       []string{"Hel"},
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"s\", got 3: [Hello Hello! HelloThere]\n",
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"s\", got 3: [Hello Hello! HelloThere]"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "Hel"},
					},
				},
			},
		},
		{
			name: "Complexecute works if exact match",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{Lenient: true},
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"Hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "Hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "Hello"},
					},
				},
			},
		},
		{
			name: "Complexecute works if exact match with sub match",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{Lenient: true},
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args: []string{"HelloThere"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "HelloThere",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "HelloThere"},
					},
				},
			},
		},
		{
			name: "Complexecute works if only sub match",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc,
					&Complexecute[string]{},
					SimpleCompleter[string]("Hello", "HelloThere", "Hello!", "Goodbye"),
				)),
				Args:       []string{"HelloThere!"},
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"s\", got 0: []\n",
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"s\", got 0: []"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "HelloThere!"},
					},
				},
			},
		},
		// FileCompleter with Complexecute
		{
			name: "FileCompleter with Complexecute properly completes a single directory",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args: []string{"te"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": testutil.FilepathAbs(t, "testdata"),
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "testdata")},
					},
				},
			},
		},
		{
			name: "FileListArgument with Complexecute properly completes a single directory",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileListArgument("s", testDesc, 2, 0, &Complexecute[[]string]{})),
				Args: []string{"co2test", "te"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": []string{
						testutil.FilepathAbs(t, "co2test"),
						testutil.FilepathAbs(t, "testdata"),
					},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "co2test")},
						{Value: testutil.FilepathAbs(t, "testdata")},
					},
				},
			},
		},
		{
			name: "FileCompleter with Complexecute properly completes a full directory",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args: []string{"testdata"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": testutil.FilepathAbs(t, "testdata"),
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "testdata")},
					},
				},
			},
		},
		{
			name: "FileCompleter with Complexecute properly completes a full directory with trailing slash",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args: []string{fmt.Sprintf("testdata%c", filepath.Separator)},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": testutil.FilepathAbs(t, "testdata"),
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "testdata")},
					},
				},
			},
		},
		{
			name: "FileCompleter with Complexecute properly completes nested directory",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args: []string{filepath.Join("testdata", "c")},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": testutil.FilepathAbs(t, "testdata", "cases"),
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "testdata", "cases")},
					},
				},
			},
		},
		{
			name: "FileCompleter with Complexecute properly completes nested file",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args: []string{filepath.Join("testdata", "cases", "o")},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": testutil.FilepathAbs(t, "testdata", "cases", "other.txt"),
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "testdata", "cases", "other.txt")},
					},
				},
			},
		},
		{
			name: "FileCompleter with Complexecute properly completes a single file",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args: []string{"v"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": testutil.FilepathAbs(t, "validator.go"),
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "validator.go")},
					},
				},
			},
		},
		{
			name: "FileCompleter with Complexecute fails if multiple options (autofilling letters)",
			etc: &commandtest.ExecuteTestCase{
				OS:         &commandtest.FakeOS{},
				Node:       SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args:       []string{"t"},
				WantStderr: filepath.FromSlash("[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [testdata/ transformer.go]\n"),
				WantErr:    fmt.Errorf(filepath.FromSlash("[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [testdata/ transformer.go]")),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "t"},
					},
				},
			},
		},
		{
			name: "FileCompleter with Complexecute fails if no options",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(FileArgument("s", testDesc, &Complexecute[string]{})),
				Args:       []string{"uhhh"},
				WantStderr: "[Complexecute] nil completion returned for \"s\"\n",
				WantErr:    fmt.Errorf("[Complexecute] nil completion returned for \"s\""),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "uhhh"},
					},
				},
			},
		},
		{
			name:    "FileCompleter with Complexecute and ExcludePwd",
			osGetwd: testutil.FilepathAbs(t, "cotest"),
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, &Complexecute[string]{}, &FileCompleter[string]{
						ExcludePwd:  true,
						IgnoreFiles: true,
					}),
				),
				Args: []string{"c"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": filepath.FromSlash("co2test/"),
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: filepath.FromSlash("co2test/")},
					},
				},
			},
		},
		// Complexecute tests for ListArg
		{
			name: "Complexecute for ListArg fails if no arg provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 2 arguments, got 0`),
				WantStderr: "Argument \"sl\" requires at least 2 arguments, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
			},
		},
		{
			name: "Complexecute for ListArg fails completer returns error",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return nil, fmt.Errorf("oopsie")
				}))),
				WantErr:    fmt.Errorf("[Complexecute] failed to fetch completion for \"sl\": oopsie"),
				WantStderr: "[Complexecute] failed to fetch completion for \"sl\": oopsie\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg fails if returned completion is nil",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return nil, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] nil completion returned for \"sl\""),
				WantStderr: "[Complexecute] nil completion returned for \"sl\"\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg fails if 0 suggestions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{}, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"sl\", got 0: []"),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"sl\", got 0: []\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg fails if multiple suggestions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"alpha", "bravo"},
					}, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"sl\", got 2: [alpha bravo]"),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"sl\", got 2: [alpha bravo]\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg fails if suggestions is wrong type",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{""},
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 3, &Complexecute[[]int]{}, CompleterFromFunc(func(sl []int, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"alpha"},
					}, nil
				}))),
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "alpha": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"alpha\": invalid syntax\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "alpha"},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg fails if still not enough args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"alpha", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 3, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "charlie"},
					}, nil
				}))),
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"sl\" requires at least 3 arguments, got 2\n",
				WantData: &command.Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "charlie"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "alpha"},
						{Value: "charlie"},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg works if one suggestion",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"alpha", "bravo", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "alpha"},
						{Value: "bravo"},
						{Value: "charlie"},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg works when only one prefix matches",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"alpha", "bravo", "c"},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "bravo", "charlie", "delta", "epsilon"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "alpha"},
						{Value: "bravo"},
						{Value: "charlie"},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg fails if no distinct filter",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"alpha", "bravo", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"sl\", got 3: [alpha bravo charlie]"),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"sl\", got 3: [alpha bravo charlie]\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "alpha"},
						{Value: "bravo"},
						{Value: ""},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg works with distinct filter",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"alpha", "bravo", ""},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Distinct:    true,
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "alpha"},
						{Value: "bravo"},
						{Value: "charlie"},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg completes multiple args",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"a", "br", "c"},
				Node: SerialNodes(ListArg[string]("sl", testDesc, 2, 3, &Complexecute[[]string]{}, CompleterFromFunc(func(sl []string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"alpha", "bravo", "charlie"},
					}, nil
				}))),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"sl": []string{"alpha", "bravo", "charlie"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "alpha"},
						{Value: "bravo"},
						{Value: "charlie"},
					},
				},
			},
		},
		{
			name: "Complexecute for ListArg fails if multiple completions",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"f"},
				Node: SerialNodes(Arg[string]("s", testDesc, &Complexecute[string]{}, CompleterFromFunc(func(i string, d *command.Data) (*command.Completion, error) {
					return &command.Completion{
						Suggestions: []string{"one", "two", "three", "four", "five", "six"},
					}, nil
				}))),
				WantErr:    fmt.Errorf("[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]"),
				WantStderr: "[Complexecute] requires exactly one suggestion to be returned for \"s\", got 2: [five four]\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "f"},
					},
				},
			},
		},
		// Arg convenience functions
		{
			name: "Arg.Provided, Get, GetOrDefault, GetOrDefaultFunc when argument is present",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"some-string"},
				Node: SerialNodes(
					optionalString,
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						df, _ := optionalString.GetOrDefaultFunc(d, func(d *command.Data) (string, error) { return "funcDflt", nil })
						o.Stdoutln(
							optionalString.Provided(d),
							optionalString.Get(d),
							optionalString.GetOrDefault(d, "dflt"),
							df,
						)
						return nil
					}},
				),
				WantStdout: "true some-string some-string some-string\n",
				WantData: &command.Data{Values: map[string]interface{}{
					optionalString.Name(): "some-string",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "some-string"},
					},
				},
			},
		},
		{
			name: "Arg.Provided, GetOrDefault, GetOrDefaultFunc when argument is not present",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					optionalString,
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						df, _ := optionalString.GetOrDefaultFunc(d, func(d *command.Data) (string, error) { return "funcDflt", nil })
						o.Stdoutln(
							optionalString.Provided(d),
							optionalString.GetOrDefault(d, "dflt"),
							df,
						)
						return nil
					}},
				),
				WantStdout: "false dflt funcDflt\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{},
			},
		},
		{
			name: "Arg.GetOrDefaultFunc returns error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					optionalString,
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						df, err := optionalString.GetOrDefaultFunc(d, func(d *command.Data) (string, error) { return "funcDflt", fmt.Errorf("oops") })
						o.Stdoutln(df)
						return o.Err(err)
					}},
				),
				WantStdout: "funcDflt\n",
				WantStderr: "oops\n",
				WantErr:    fmt.Errorf("oops"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{},
			},
		},
		{
			name: "Arg.Desc",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					optionalString,
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(optionalString.Desc())
						return nil
					}},
				),
				WantStdout: "desc\n",
			},
		},
		// Flag convenience functions
		{
			name: "Flag.Get",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--opt-flag", "some-string"},
				Node: SerialNodes(
					FlagProcessor(
						stringFlag,
					),
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(stringFlag.Provided(d), stringFlag.Get(d), stringFlag.GetOrDefault(d, "dflt"))
						return nil
					}},
				),
				WantStdout: "true some-string some-string\n",
				WantData: &command.Data{Values: map[string]interface{}{
					stringFlag.Name(): "some-string",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--opt-flag"},
						{Value: "some-string"},
					},
				},
			},
		},
		{
			name: "Flag.Provided and Flag.GetOrDefault",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						stringFlag,
					),
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(stringFlag.Provided(d), stringFlag.GetOrDefault(d, "dflt"))
						return nil
					}},
				),
				WantStdout: "false dflt\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{},
			},
		},
		{
			name: "Flag.Desc",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						stringFlag,
					),
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(stringFlag.Desc())
						return nil
					}},
				),
				WantStdout: "flag-desc\n",
			},
		},
		// BoolFlag convenience functions
		{
			name: "BoolFlag.Get",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--bool-flag"},
				Node: SerialNodes(
					FlagProcessor(
						simpleBoolFlag,
					),
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(simpleBoolFlag.Provided(d), simpleBoolFlag.Get(d), simpleBoolFlag.GetOrDefault(d, false))
						return nil
					}},
				),
				WantStdout: "true true true\n",
				WantData: &command.Data{Values: map[string]interface{}{
					simpleBoolFlag.Name(): true,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--bool-flag"},
					},
				},
			},
		},
		{
			name: "BoolFlag.Provided and BoolFlag.GetOrDefault",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						simpleBoolFlag,
					),
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(simpleBoolFlag.Provided(d), simpleBoolFlag.GetOrDefault(d, true))
						return nil
					}},
				),
				WantStdout: "false true\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{},
			},
		},
		{
			name: "BoolFlag.Desc",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						simpleBoolFlag,
					),
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(simpleBoolFlag.Desc())
						return nil
					}},
				),
				WantStdout: "bool-flag-desc\n",
			},
		},
		// Default value tests
		{
			name: "Uses Default value if no arg provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(OptionalArg("s", testDesc, Default("heyo"))),
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "heyo",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{},
			},
		},
		{
			name: "Flag defaults get set",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag("s", 's', testDesc, Default("defStr")),
						Flag("s2", 'S', testDesc, Default("dos")),
						Flag("it", 't', testDesc, Default(-456)),
						Flag("i", 'i', testDesc, Default(123)),
						Flag("fs", 'f', testDesc, Default([]float64{1.2, 3.4, -5.6})),
					),
				),
				Args: []string{"--it", "7", "-S", "dos"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "defStr",
					"s2": "dos",
					"it": 7,
					"i":  123,
					"fs": []float64{1.2, 3.4, -5.6},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--it"},
						{Value: "7"},
						{Value: "-S"},
						{Value: "dos"},
					},
				},
			},
		},
		{
			name: "Default doesn't fill in required argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(Arg("s", testDesc, Default("settled"))),
				WantStderr: "Argument \"s\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "s" requires at least 1 argument, got 0`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput:                &spycommandtest.SpyInput{},
			},
		},
		// Simple arg tests
		{
			name: "Processes single string arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[string]("s", testDesc)),
				Args: []string{"hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "Processes single int arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[int]("i", testDesc)),
				Args: []string{"123"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 123,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "123"},
					},
				},
			},
		},
		{
			name: "Int arg fails if not an int",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(Arg[int]("i", testDesc)),
				Args:       []string{"12.3"},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "12.3": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"12.3\": invalid syntax\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "12.3"},
					},
				},
			},
		},
		{
			name: "Processes single float arg",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(Arg[float64]("f", testDesc)),
				Args: []string{"-12.3"},
				WantData: &command.Data{Values: map[string]interface{}{
					"f": -12.3,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-12.3"},
					},
				},
			},
		},
		{
			name: "Float arg fails if not a float",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(Arg[float64]("f", testDesc)),
				Args:       []string{"twelve"},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
				WantStderr: "strconv.ParseFloat: parsing \"twelve\": invalid syntax\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "twelve"},
					},
				},
			},
		},
		// List args
		{
			name: "List fails if not enough args",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 1)),
				Args: []string{"hello", "there", "sir"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there"},
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [sir]"),
				WantStderr: strings.Join([]string{
					`Unprocessed extra args: [sir]`,
					``,
					`======= Command Usage =======`,
					`sl [ sl ]`,
					``,
					`Arguments:`,
					`  sl: test desc`,
					``,
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "there"},
						{Value: "sir"},
					},
					Remaining: []int{2},
				},
			},
		},
		{
			name: "Processes string list if minimum provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2)),
				Args: []string{"hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "Processes string list if some optional provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2)),
				Args: []string{"hello", "there"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "there"},
					},
				},
			},
		},
		{
			name: "Processes string list if max args provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, 2)),
				Args: []string{"hello", "there", "maam"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there", "maam"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "there"},
						{Value: "maam"},
					},
				},
			},
		},
		{
			name: "Unbounded string list fails if less than min provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 4, command.UnboundedList)),
				Args: []string{"hello", "there", "kenobi"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there", "kenobi"},
				}},
				WantErr:    fmt.Errorf(`Argument "sl" requires at least 4 arguments, got 3`),
				WantStderr: "Argument \"sl\" requires at least 4 arguments, got 3\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsNotEnoughArgsError: true,
				WantIsUsageError:         true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "there"},
						{Value: "kenobi"},
					},
				},
			},
		},
		{
			name: "Processes unbounded string list if min provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList)),
				Args: []string{"hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "Processes unbounded string list if more than min provided",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[string]("sl", testDesc, 1, command.UnboundedList)),
				Args: []string{"hello", "there", "kenobi"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"hello", "there", "kenobi"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "there"},
						{Value: "kenobi"},
					},
				},
			},
		},
		{
			name: "Processes int list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 1, 2)),
				Args: []string{"1", "-23"},
				WantData: &command.Data{Values: map[string]interface{}{
					"il": []int{1, -23},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1"},
						{Value: "-23"},
					},
				},
			},
		},
		{
			name: "Int list fails if an arg isn't an int",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(ListArg[int]("il", testDesc, 1, 2)),
				Args:       []string{"1", "four", "-23"},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "four": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"four\": invalid syntax\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1"},
						{Value: "four"},
						{Value: "-23"},
					},
				},
			},
		},
		{
			name: "Processes float list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[float64]("fl", testDesc, 1, 2)),
				Args: []string{"0.1", "-2.3"},
				WantData: &command.Data{Values: map[string]interface{}{
					"fl": []float64{0.1, -2.3},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0.1"},
						{Value: "-2.3"},
					},
				},
			},
		},
		{
			name: "Float list fails if an arg isn't an float",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(ListArg[float64]("fl", testDesc, 1, 2)),
				Args:       []string{"0.1", "four", "-23"},
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "four": invalid syntax`),
				WantStderr: "strconv.ParseFloat: parsing \"four\": invalid syntax\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0.1"},
						{Value: "four"},
						{Value: "-23"},
					},
				},
			},
		},
		// Multiple args
		{
			name: "Processes multiple args",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2)),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"il": []int{0, 1},
					"s":  "two",
					"fl": []float64{0.3, -4},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
						{Value: "1"},
						{Value: "two"},
						{Value: "0.3"},
						{Value: "-4"},
					},
				},
			},
		},
		{
			name: "Fails if extra args when multiple",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2)),
				Args: []string{"0", "1", "two", "0.3", "-4", "0.5", "6"},
				WantData: &command.Data{Values: map[string]interface{}{
					"il": []int{0, 1},
					"s":  "two",
					"fl": []float64{0.3, -4, 0.5},
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [6]"),
				WantStderr: strings.Join([]string{`Unprocessed extra args: [6]`,
					``,
					`======= Command Usage =======`,
					`il il s fl [ fl fl ]`,
					``,
					`Arguments:`,
					`  fl: test desc`,
					`  il: test desc`,
					`  s: test desc`,
					``,
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
				WantInput: &spycommandtest.SpyInput{
					Remaining: []int{6},
					Args: []*spycommand.InputArg{
						{Value: "0"},
						{Value: "1"},
						{Value: "two"},
						{Value: "0.3"},
						{Value: "-4"},
						{Value: "0.5"},
						{Value: "6"},
					},
				},
			},
		},
		// Executor tests.
		{
			name: "Sets executable with SimpleExecutableProcessor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(SimpleExecutableProcessor("hello", "there")),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"hello", "there"},
				},
			},
		},
		{
			name: "FunctionWrap sets command.ExecuteData.FunctionWrap",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("hello", "there"),
					FunctionWrap(),
				),
				WantExecuteData: &command.ExecuteData{
					Executable:   []string{"hello", "there"},
					FunctionWrap: true,
				},
			},
		},
		{
			name: "Sets executable with ExecutableProcessor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", "", 0, command.UnboundedList),
					ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
						o.Stdoutln("hello")
						o.Stderr("there")
						return d.StringList("SL"), nil
					}),
				),
				Args:       []string{"abc", "def"},
				WantStdout: "hello\n",
				WantStderr: "there",
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"abc", "def"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"SL": []string{"abc", "def"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
					},
				},
			},
		},
		{
			name: "ExecutableProcessor returning error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", "", 0, command.UnboundedList),
					ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
						return d.StringList("SL"), fmt.Errorf("bad news bears")
					}),
				),
				Args:    []string{"abc", "def"},
				WantErr: fmt.Errorf("bad news bears"),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"SL": []string{"abc", "def"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
					},
				},
			},
		},
		{
			name: "SimpleExecutableProcessor appends executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("do some", "stuff"),
					SimpleExecutableProcessor("and then", "even", "MORE"),
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"do some",
						"stuff",
						"and then",
						"even",
						"MORE",
					},
				},
			},
		},
		{
			name: "ExecutableProcessor appends executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
						return []string{
							"do some",
							"stuff",
						}, nil
					}),
					ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
						return []string{
							"and then",
							"even",
							"MORE",
						}, nil
					}),
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"do some",
						"stuff",
						"and then",
						"even",
						"MORE",
					},
				},
			},
		},
		{
			name: "Sets executable with processor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					ed.Executable = []string{"hello", "there"}
					return nil
				}, nil)),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"hello", "there"},
				},
			},
		},
		// SuperSimpleProcessor tests
		{
			name: "sets data with SuperSimpleProcessor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
					d.Set("key", "value")
					return nil
				})),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"key": "value",
					},
				},
			},
		},
		{
			name: "returns error with SuperSimpleProcessor",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
					d.Set("key", "value")
					return fmt.Errorf("argh")
				})),
				WantErr:    fmt.Errorf("argh"),
				WantStderr: "argh\n",
				WantData: &command.Data{
					Values: map[string]interface{}{
						"key": "value",
					},
				},
			},
		},
		// DataTransformer tests
		{
			name: "DataTransformer transforms simple types",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"123"},
				Node: SerialNodes(
					Arg[string]("S", testDesc),
					DataTransformer[string, int]("S", func(s string) (int, error) {
						return strconv.Atoi(s)
					}),
				),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"S": 123,
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "123"},
					},
				},
			},
		},
		{
			name: "DataTransformer transforms to struct types",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"hello"},
				Node: SerialNodes(
					Arg[string]("S", testDesc),
					DataTransformer[string, *simpleType]("S", func(s string) (*simpleType, error) {
						return &simpleType{fmt.Sprintf("%s there", s), 12}, nil
					}),
					DataTransformer[*simpleType, *simpleType]("S", func(st *simpleType) (*simpleType, error) {
						return &simpleType{fmt.Sprintf("%s; General Kenobi", st.S), st.N * st.N}, nil
					}),
				),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"S": &simpleType{
							S: "hello there; General Kenobi",
							N: 144,
						},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "DataTransformer fails if not set",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					DataTransformer[string, int]("S", func(s string) (int, error) {
						return strconv.Atoi(s)
					}),
				),
				WantStderr: "[DataTransformer] key is not set in command.Data\n",
				WantErr:    fmt.Errorf("[DataTransformer] key is not set in command.Data"),
			},
		},
		{
			name: "DataTransformer fails if function error",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"twelve"},
				Node: SerialNodes(
					Arg[string]("S", testDesc),
					DataTransformer[string, int]("S", func(s string) (int, error) {
						return strconv.Atoi(s)
					}),
				),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"S": "twelve",
					},
				},
				WantStderr: "[DataTransformer] failed to convert data: strconv.Atoi: parsing \"twelve\": invalid syntax\n",
				WantErr:    fmt.Errorf("[DataTransformer] failed to convert data: strconv.Atoi: parsing \"twelve\": invalid syntax"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "twelve"},
					},
				},
			},
		},
		// osenv tests
		{
			name: "EnvArg returns nil if no env",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					envArgProcessor,
				),
			},
		},
		{
			name: "EnvArg adds environment variable to data",
			etc: &commandtest.ExecuteTestCase{
				Env: map[string]string{
					envArgProcessor.Name: "heyo",
					"other":              "env-var",
				},
				Node: SerialNodes(
					envArgProcessor,
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(envArgProcessor.Provided(d), envArgProcessor.Get(d))
						return nil
					}},
				),
				WantStdout: "true heyo\n",
				WantData: &command.Data{
					Values: map[string]interface{}{
						envArgProcessor.Name: "heyo",
					},
				},
			},
		},
		{
			name: "EnvArg does nothing if variable not defined",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					envArgProcessor,
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						o.Stdoutln(envArgProcessor.Provided(d))
						return nil
					}},
				),
				WantStdout: "false\n",
			},
		},
		{
			name: "SetEnvVar sets variable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					ed.Executable = append(ed.Executable, d.OS.SetEnvVar("abc", "def"))
					return nil
				}, nil)),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fos.SetEnvVar("abc", "def"),
					},
				},
			},
		},
		{
			name: "UnsetEnvVar unsets variable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
					ed.Executable = append(ed.Executable, d.OS.UnsetEnvVar("abc"))
					return nil
				}, nil)),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fos.UnsetEnvVar("abc"),
					},
				},
			},
		},
		{
			name: "SetEnvVarProcessor sets variable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SetEnvVarProcessor("abc", "def"),
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fos.SetEnvVar("abc", "def"),
					},
				},
			},
		},
		{
			name: "UnsetEnvVarProcessor unsets variable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					UnsetEnvVarProcessor("abc"),
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						fos.UnsetEnvVar("abc"),
					},
				},
			},
		},
		{
			name: "[Un]SetEnvVar appends executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("do some", "stuff"),
					SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						ed.Executable = append(ed.Executable,
							d.OS.SetEnvVar("abc", "def"),
							d.OS.UnsetEnvVar("ghi"),
						)
						return nil
					}, nil),
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"do some",
						"stuff",
						fos.SetEnvVar("abc", "def"),
						fos.UnsetEnvVar("ghi"),
					},
				},
			},
		},
		{
			name: "[Un]SetEnvVarProcessor appends executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("do some", "stuff"),
					SetEnvVarProcessor("abc", "def"),
					UnsetEnvVarProcessor("ghi"),
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"do some",
						"stuff",
						fos.SetEnvVar("abc", "def"),
						fos.UnsetEnvVar("ghi"),
					},
				},
			},
		},
		// PrintlnProcessor tests
		{
			name: "PrintlnProcessor prints output",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					PrintlnProcessor("hello there"),
				),
				WantStdout: "hello there\n",
			},
		},
		// Getwd tests
		{
			name:    "sets data with Getwd",
			osGetwd: "some/dir",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Getwd,
					&ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
						o.Stdoutf("wd: %s", Getwd.Get(d))
						return nil
					}},
				),
				WantStdout: "wd: some/dir",
				WantData: &command.Data{
					Values: map[string]interface{}{
						GetwdKey: "some/dir",
					},
				},
			},
		},
		{
			name:       "returns error from Getwd",
			osGetwdErr: fmt.Errorf("whoops"),
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(Getwd),
				WantErr:    fmt.Errorf("failed to get current directory: whoops"),
				WantStderr: "failed to get current directory: whoops\n",
			},
		},
		// RuntimeCaller tests
		{
			name: "sets and gets data with RuntimeCaller",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					rcNode,
					&ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
						o.Stdoutf("rc: %s", RuntimeCaller().Get(d))
						return nil
					}},
				),
				WantStdout: "rc: some/file/path",
				WantData: &command.Data{
					Values: map[string]interface{}{
						RuntimeCallerKey: "some/file/path",
					},
				},
			},
		},
		{
			name: "returns error from RuntimeCaller",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					rcErrNode,
				),
				WantErr:    fmt.Errorf("runtime.Caller failed to retrieve filepath info"),
				WantStderr: "runtime.Caller failed to retrieve filepath info\n",
			},
		},
		// Other tests
		{
			name: "executes with proper data",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2), printArgsNode()),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				WantData: &command.Data{Values: map[string]interface{}{
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
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
						{Value: "1"},
						{Value: "two"},
						{Value: "0.3"},
						{Value: "-4"},
					},
				},
			},
		},
		{
			name: "executor error is returned",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(ListArg[int]("il", testDesc, 2, 0), Arg[string]("s", testDesc), ListArg[float64]("fl", testDesc, 1, 2), &ExecutorProcessor{func(o command.Output, d *command.Data) error {
					return o.Stderrf("bad news bears")
				}}),
				Args: []string{"0", "1", "two", "0.3", "-4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"il": []int{0, 1},
					"s":  "two",
					"fl": []float64{0.3, -4},
				}},
				WantStderr: "bad news bears",
				WantErr:    fmt.Errorf("bad news bears"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
						{Value: "1"},
						{Value: "two"},
						{Value: "0.3"},
						{Value: "-4"},
					},
				},
			},
		},
		// ArgValidator tests
		// StringDoesNotEqual
		{
			name: "string dne works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, NEQ("bad")),
				},
				Args: []string{"good"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "good",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "good"},
					},
				},
			},
		},
		{
			name: "string dne fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, NEQ("bad")),
				},
				Args: []string{"bad"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "bad",
				}},
				WantStderr: "validation for \"strArg\" failed: [NEQ] value cannot equal bad\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [NEQ] value cannot equal bad`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "bad"},
					},
				},
			},
		},
		// Contains
		{
			name: "contains works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, Contains("good")),
				},
				Args: []string{"goodbye"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "goodbye",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "goodbye"},
					},
				},
			},
		},
		{
			name: "contains fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, Contains("good")),
				},
				Args: []string{"hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hello",
				}},
				WantStderr: "validation for \"strArg\" failed: [Contains] value doesn't contain substring \"good\"\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [Contains] value doesn't contain substring "good"`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "AddOptions works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc).AddOptions(Contains("good")),
				},
				Args: []string{"hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hello",
				}},
				WantStderr: "validation for \"strArg\" failed: [Contains] value doesn't contain substring \"good\"\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [Contains] value doesn't contain substring "good"`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		// Not works
		{
			name: "not fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, Not(Contains("good"))),
				},
				Args: []string{"goodbye"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "goodbye",
				}},
				WantStderr: "validation for \"strArg\" failed: [Not(Contains(\"good\"))] failed\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [Not(Contains("good"))] failed`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "goodbye"},
					},
				},
			},
		},
		{
			name: "not works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, Not(Contains("good"))),
				},
				Args: []string{"hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		// MatchesRegex
		{
			name: "matches regex works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, MatchesRegex("a+b=?c")),
				},
				Args: []string{"equiation: aabcdef"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "equiation: aabcdef",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "equiation: aabcdef"},
					},
				},
			},
		},
		{
			name: "matches regex fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, MatchesRegex(".*", "i+")),
				},
				Args: []string{"team"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "team",
				}},
				WantStderr: "validation for \"strArg\" failed: [MatchesRegex] value \"team\" doesn't match regex \"i+\"\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [MatchesRegex] value "team" doesn't match regex "i+"`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "team"},
					},
				},
			},
		},
		// ListMatchesRegex
		{
			name: "ListMatchesRegex works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, command.UnboundedList, ListifyValidatorOption(MatchesRegex("a+b=?c", "^eq"))),
				},
				Args: []string{"equiation: aabcdef"},
				WantData: &command.Data{Values: map[string]interface{}{
					"slArg": []string{"equiation: aabcdef"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "equiation: aabcdef"},
					},
				},
			},
		},
		{
			name: "ListMatchesRegex fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, command.UnboundedList, ListifyValidatorOption(MatchesRegex(".*", "i+"))),
				},
				Args: []string{"equiation: aabcdef", "oops"},
				WantData: &command.Data{Values: map[string]interface{}{
					"slArg": []string{"equiation: aabcdef", "oops"},
				}},
				WantStderr: "validation for \"slArg\" failed: [MatchesRegex] value \"oops\" doesn't match regex \"i+\"\n",
				WantErr:    fmt.Errorf(`validation for "slArg" failed: [MatchesRegex] value "oops" doesn't match regex "i+"`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "equiation: aabcdef"},
						{Value: "oops"},
					},
				},
			},
		},
		// IsRegex
		{
			name: "IsRegex works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, IsRegex()),
				},
				Args: []string{".*"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": ".*",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ".*"},
					},
				},
			},
		},
		{
			name: "IsRegex fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, IsRegex()),
				},
				Args: []string{"*"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "*",
				}},
				WantStderr: "validation for \"strArg\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`\n",
				WantErr:    fmt.Errorf("validation for \"strArg\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "*"},
					},
				},
			},
		},
		// ListIsRegex
		{
			name: "ListIsRegex works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, command.UnboundedList, ListifyValidatorOption(IsRegex())),
				},
				Args: []string{".*", " +"},
				WantData: &command.Data{Values: map[string]interface{}{
					"slArg": []string{".*", " +"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ".*"},
						{Value: " +"},
					},
				},
			},
		},
		{
			name: "ListIsRegex fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, command.UnboundedList, ListifyValidatorOption(IsRegex())),
				},
				Args: []string{".*", "+"},
				WantData: &command.Data{Values: map[string]interface{}{
					"slArg": []string{".*", "+"},
				}},
				WantStderr: "validation for \"slArg\" failed: [IsRegex] value \"+\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `+`\n",
				WantErr:    fmt.Errorf("validation for \"slArg\" failed: [IsRegex] value \"+\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `+`"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: ".*"},
						{Value: "+"},
					},
				},
			},
		},
		// FileExists, FileDoesNotExist, and FilesExist
		{
			name: "FileExists works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, FileExists()),
				},
				Args: []string{"execute_test.go"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "execute_test.go",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute_test.go"},
					},
				},
			},
		},
		{
			name: "FileExists fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, FileExists()),
				},
				Args: []string{"execute_test.gone"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "execute_test.gone",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [FileExists] file "execute_test.gone" does not exist`),
				WantStderr: "validation for \"S\" failed: [FileExists] file \"execute_test.gone\" does not exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute_test.gone"},
					},
				},
			},
		},
		{
			name: "FileDoesNotExist fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, FileDoesNotExist()),
				},
				Args: []string{"execute_test.go"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "execute_test.go",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [FileDoesNotExist] file "execute_test.go" does exist`),
				WantStderr: "validation for \"S\" failed: [FileDoesNotExist] file \"execute_test.go\" does exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute_test.go"},
					},
				},
			},
		},
		{
			name: "FileDoesNotExist works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, FileDoesNotExist()),
				},
				Args: []string{"execute_test.gone"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "execute_test.gone",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute_test.gone"},
					},
				},
			},
		},
		{
			name: "FilesExist works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(FileExists())),
				},
				Args: []string{"execute_test.go", "execute.go"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"execute_test.go", "execute.go"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute_test.go"},
						{Value: "execute.go"},
					},
				},
			},
		},
		{
			name: "FilesExist fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(FileExists())),
				},
				Args: []string{"execute_test.go", "execute.gone"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"execute_test.go", "execute.gone"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [FileExists] file "execute.gone" does not exist`),
				WantStderr: "validation for \"SL\" failed: [FileExists] file \"execute.gone\" does not exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute_test.go"},
						{Value: "execute.gone"},
					},
				},
			},
		},
		// IsDir and AreDirs
		{
			name: "IsDir works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, IsDir()),
				},
				Args: []string{"testdata"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "testdata",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "testdata"},
					},
				},
			},
		},
		{
			name: "IsDir fails when does not exist",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, IsDir()),
				},
				Args: []string{"tested"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "tested",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsDir] file "tested" does not exist`),
				WantStderr: "validation for \"S\" failed: [IsDir] file \"tested\" does not exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "tested"},
					},
				},
			},
		},
		{
			name: "IsDir fails when not a directory",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, IsDir()),
				},
				Args: []string{"execute_test.go"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "execute_test.go",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsDir] argument "execute_test.go" is a file`),
				WantStderr: "validation for \"S\" failed: [IsDir] argument \"execute_test.go\" is a file\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute_test.go"},
					},
				},
			},
		},
		{
			name: "AreDirs works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(IsDir())),
				},
				Args: []string{"testdata", "co2test"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"testdata", "co2test"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "testdata"},
						{Value: "co2test"},
					},
				},
			},
		},
		{
			name: "AreDirs fails when does not exist",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(IsDir())),
				},
				Args: []string{"testdata", "co3test"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"testdata", "co3test"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsDir] file "co3test" does not exist`),
				WantStderr: "validation for \"SL\" failed: [IsDir] file \"co3test\" does not exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "testdata"},
						{Value: "co3test"},
					},
				},
			},
		},
		{
			name: "AreDirs fails when not a directory",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(IsDir())),
				},
				Args: []string{"testdata", "execute.go"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"testdata", "execute.go"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsDir] argument "execute.go" is a file`),
				WantStderr: "validation for \"SL\" failed: [IsDir] argument \"execute.go\" is a file\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "testdata"},
						{Value: "execute.go"},
					},
				},
			},
		},
		// IsFile and AreFiles
		{
			name: "IsFile works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, IsFile()),
				},
				Args: []string{"execute.go"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "execute.go",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute.go"},
					},
				},
			},
		},
		{
			name: "IsFile fails when does not exist",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, IsFile()),
				},
				Args: []string{"tested"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "tested",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsFile] file "tested" does not exist`),
				WantStderr: "validation for \"S\" failed: [IsFile] file \"tested\" does not exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "tested"},
					},
				},
			},
		},
		{
			name: "IsFile fails when not a file",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("S", testDesc, IsFile()),
				},
				Args: []string{"testdata"},
				WantData: &command.Data{Values: map[string]interface{}{
					"S": "testdata",
				}},
				WantErr:    fmt.Errorf(`validation for "S" failed: [IsFile] argument "testdata" is a directory`),
				WantStderr: "validation for \"S\" failed: [IsFile] argument \"testdata\" is a directory\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "testdata"},
					},
				},
			},
		},
		{
			name: "AreFiles works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(IsFile())),
				},
				Args: []string{"execute.go", "cache.go"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"execute.go", "cache.go"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute.go"},
						{Value: "cache.go"},
					},
				},
			},
		},
		{
			name: "AreFiles fails when does not exist",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(IsFile())),
				},
				Args: []string{"execute.go", "cash"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"execute.go", "cash"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsFile] file "cash" does not exist`),
				WantStderr: "validation for \"SL\" failed: [IsFile] file \"cash\" does not exist\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute.go"},
						{Value: "cash"},
					},
				},
			},
		},
		{
			name: "AreFiles fails when not a directory",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("SL", testDesc, 1, 3, ListifyValidatorOption(IsFile())),
				},
				Args: []string{"execute.go", "testdata"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"execute.go", "testdata"},
				}},
				WantErr:    fmt.Errorf(`validation for "SL" failed: [IsFile] argument "testdata" is a directory`),
				WantStderr: "validation for \"SL\" failed: [IsFile] argument \"testdata\" is a directory\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "execute.go"},
						{Value: "testdata"},
					},
				},
			},
		},
		// InList & string menus
		{
			name: "InList works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, InList("abc", "def", "ghi")),
				},
				Args: []string{"def"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "def",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "def"},
					},
				},
			},
		},
		{
			name: "InList fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, InList("abc", "def", "ghi")),
				},
				Args: []string{"jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "jkl",
				}},
				WantStderr: "validation for \"strArg\" failed: [InList] argument must be one of [abc def ghi]\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [InList] argument must be one of [abc def ghi]`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "MenuArg works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: MenuArg("strArg", testDesc, "abc", "def", "ghi"),
				},
				Args: []string{"def"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "def",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "def"},
					},
				},
			},
		},
		{
			name: "MenuArg fails if provided is not in list",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: MenuArg("strArg", testDesc, "abc", "def", "ghi"),
				},
				Args: []string{"jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "jkl",
				}},
				WantStderr: "validation for \"strArg\" failed: [InList] argument must be one of [abc def ghi]\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [InList] argument must be one of [abc def ghi]`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "MenuFlag works",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: []string{"--sf", "def"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sf": "def",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--sf"},
						{Value: "def"},
					},
				},
			},
		},
		{
			name: "MenuFlag works with AddOptions(default)",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi", "xyz").AddOptions(Default("xyz")),
					),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"sf": "xyz",
				}},
			},
		},
		{
			name: "MenuFlag fails if provided is not in list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: []string{"-s", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sf": "jkl",
				}},
				WantStderr: "validation for \"sf\" failed: [InList] argument must be one of [abc def ghi]\n",
				WantErr:    fmt.Errorf(`validation for "sf" failed: [InList] argument must be one of [abc def ghi]`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-s"},
						{Value: "jkl"},
					},
				},
			},
		},
		// MinLength
		{
			name: "MinLength works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, MinLength[string, string](3)),
				},
				Args: []string{"hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "MinLength works for exact count match",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, MinLength[string, string](3)),
				},
				Args: []string{"hey"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hey",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hey"},
					},
				},
			},
		},
		{
			name: "MinLength fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, MinLength[string, string](3)),
				},
				Args: []string{"hi"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hi",
				}},
				WantStderr: "validation for \"strArg\" failed: [MinLength] length must be at least 3\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [MinLength] length must be at least 3`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hi"},
					},
				},
			},
		},
		// MaxLength
		{
			name: "MaxLength works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[int]("strArg", testDesc, 0, command.UnboundedList, MaxLength[int, []int](3)),
				},
				Args: []string{"1234", "56"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": []int{
						1234,
						56,
					},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1234"},
						{Value: "56"},
					},
				},
			},
		},
		{
			name: "MaxLength works for exact count match",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[int]("strArg", testDesc, 0, command.UnboundedList, MaxLength[int, []int](3)),
				},
				Args: []string{"1234", "56", "78901"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": []int{
						1234,
						56,
						78901,
					},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1234"},
						{Value: "56"},
						{Value: "78901"},
					},
				},
			},
		},
		{
			name: "MaxLength fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[int]("strArg", testDesc, 0, command.UnboundedList, MaxLength[int, []int](3)),
				},
				Args: []string{"1234", "56", "78901", "234"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": []int{
						1234,
						56,
						78901,
						234,
					},
				}},
				WantStderr: "validation for \"strArg\" failed: [MaxLength] length must be at most 3\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [MaxLength] length must be at most 3`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1234"},
						{Value: "56"},
						{Value: "78901"},
						{Value: "234"},
					},
				},
			},
		},
		// Length
		{
			name: "Length works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, Length[string, string](3)),
				},
				Args: []string{"hey"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hey",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hey"},
					},
				},
			},
		},
		{
			name: "Length fails for too few",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, Length[string, string](3)),
				},
				Args: []string{"hi"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "hi",
				}},
				WantStderr: "validation for \"strArg\" failed: [Length] length must be exactly 3\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [Length] length must be exactly 3`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hi"},
					},
				},
			},
		},
		{
			name: "Length fails for too many",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, Length[string, string](4)),
				},
				Args: []string{"howdy"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "howdy",
				}},
				WantStderr: "validation for \"strArg\" failed: [Length] length must be exactly 4\n",
				WantErr:    fmt.Errorf(`validation for "strArg" failed: [Length] length must be exactly 4`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "howdy"},
					},
				},
			},
		},
		// IntEQ
		{
			name: "IntEQ works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, EQ(24)),
				},
				Args: []string{"24"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 24,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "24"},
					},
				},
			},
		},
		{
			name: "IntEQ fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, EQ(24)),
				},
				Args: []string{"25"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 25,
				}},
				WantStderr: "validation for \"i\" failed: [EQ] value isn't equal to 24\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [EQ] value isn't equal to 24`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "25"},
					},
				},
			},
		},
		// IntNE
		{
			name: "IntNE works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, NEQ(24)),
				},
				Args: []string{"25"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 25,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "25"},
					},
				},
			},
		},
		{
			name: "IntNE fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, NEQ(24)),
				},
				Args: []string{"24"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 24,
				}},
				WantStderr: "validation for \"i\" failed: [NEQ] value cannot equal 24\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [NEQ] value cannot equal 24`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "24"},
					},
				},
			},
		},
		// IntLT
		{
			name: "IntLT works when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, LT(25)),
				},
				Args: []string{"24"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 24,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "24"},
					},
				},
			},
		},
		{
			name: "IntLT fails when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, LT(25)),
				},
				Args: []string{"25"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 25,
				}},
				WantStderr: "validation for \"i\" failed: [LT] value isn't less than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [LT] value isn't less than 25`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "25"},
					},
				},
			},
		},
		{
			name: "IntLT fails when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, LT(25)),
				},
				Args: []string{"26"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 26,
				}},
				WantStderr: "validation for \"i\" failed: [LT] value isn't less than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [LT] value isn't less than 25`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "26"},
					},
				},
			},
		},
		// IntLTE
		{
			name: "IntLTE works when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, LTE(25)),
				},
				Args: []string{"24"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 24,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "24"},
					},
				},
			},
		},
		{
			name: "IntLTE works when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, LTE(25)),
				},
				Args: []string{"25"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 25,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "25"},
					},
				},
			},
		},
		{
			name: "IntLTE fails when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, LTE(25)),
				},
				Args: []string{"26"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 26,
				}},
				WantStderr: "validation for \"i\" failed: [LTE] value isn't less than or equal to 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [LTE] value isn't less than or equal to 25`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "26"},
					},
				},
			},
		},
		// IntGT
		{
			name: "IntGT fails when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, GT(25)),
				},
				Args: []string{"24"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 24,
				}},
				WantStderr: "validation for \"i\" failed: [GT] value isn't greater than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [GT] value isn't greater than 25`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "24"},
					},
				},
			},
		},
		{
			name: "IntGT fails when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, GT(25)),
				},
				Args: []string{"25"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 25,
				}},
				WantStderr: "validation for \"i\" failed: [GT] value isn't greater than 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [GT] value isn't greater than 25`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "25"},
					},
				},
			},
		},
		{
			name: "IntGT works when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, GT(25)),
				},
				Args: []string{"26"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 26,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "26"},
					},
				},
			},
		},
		// IntGTE
		{
			name: "IntGTE fails when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, GTE(25)),
				},
				Args: []string{"24"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 24,
				}},
				WantStderr: "validation for \"i\" failed: [GTE] value isn't greater than or equal to 25\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [GTE] value isn't greater than or equal to 25`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "24"},
					},
				},
			},
		},
		{
			name: "IntGTE works when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, GTE(25)),
				},
				Args: []string{"25"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 25,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "25"},
					},
				},
			},
		},
		{
			name: "IntGTE works when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, GTE(25)),
				},
				Args: []string{"26"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 26,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "26"},
					},
				},
			},
		},
		// IntPositive
		{
			name: "IntPositive fails when negative",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, Positive[int]()),
				},
				Args: []string{"-1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": -1,
				}},
				WantStderr: "validation for \"i\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Positive] value isn't positive`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-1"},
					},
				},
			},
		},
		{
			name: "IntPositive fails when zero",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, Positive[int]()),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 0,
				}},
				WantStderr: "validation for \"i\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Positive] value isn't positive`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "IntPositive works when positive",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, Positive[int]()),
				},
				Args: []string{"1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 1,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1"},
					},
				},
			},
		},
		// IntNegative
		{
			name: "IntNegative works when negative",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, Negative[int]()),
				},
				Args: []string{"-1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": -1,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-1"},
					},
				},
			},
		},
		{
			name: "IntNegative fails when zero",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, Negative[int]()),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 0,
				}},
				WantStderr: "validation for \"i\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Negative] value isn't negative`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "IntNegative fails when positive",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, Negative[int]()),
				},
				Args: []string{"1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 1,
				}},
				WantStderr: "validation for \"i\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [Negative] value isn't negative`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1"},
					},
				},
			},
		},
		// IntNonNegative
		{
			name: "IntNonNegative fails when negative",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, NonNegative[int]()),
				},
				Args: []string{"-1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": -1,
				}},
				WantStderr: "validation for \"i\" failed: [NonNegative] value isn't non-negative\n",
				WantErr:    fmt.Errorf(`validation for "i" failed: [NonNegative] value isn't non-negative`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-1"},
					},
				},
			},
		},
		{
			name: "IntNonNegative works when zero",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, NonNegative[int]()),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 0,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "IntNonNegative works when positive",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("i", testDesc, NonNegative[int]()),
				},
				Args: []string{"1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"i": 1,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "1"},
					},
				},
			},
		},
		// FloatEQ
		{
			name: "FloatEQ works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, EQ(2.4)),
				},
				Args: []string{"2.4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.4"},
					},
				},
			},
		},
		{
			name: "FloatEQ fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, EQ(2.4)),
				},
				Args: []string{"2.5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
				WantStderr: "validation for \"flArg\" failed: [EQ] value isn't equal to 2.4\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [EQ] value isn't equal to 2.4`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.5"},
					},
				},
			},
		},
		// FloatNE
		{
			name: "FloatNE works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, NEQ(2.4)),
				},
				Args: []string{"2.5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.5"},
					},
				},
			},
		},
		{
			name: "FloatNE fails",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, NEQ(2.4)),
				},
				Args: []string{"2.4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
				WantStderr: "validation for \"flArg\" failed: [NEQ] value cannot equal 2.4\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [NEQ] value cannot equal 2.4`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.4"},
					},
				},
			},
		},
		// FloatLT
		{
			name: "FloatLT works when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, LT(2.5)),
				},
				Args: []string{"2.4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.4"},
					},
				},
			},
		},
		{
			name: "FloatLT fails when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, LT(2.5)),
				},
				Args: []string{"2.5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
				WantStderr: "validation for \"flArg\" failed: [LT] value isn't less than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [LT] value isn't less than 2.5`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.5"},
					},
				},
			},
		},
		{
			name: "FloatLT fails when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, LT(2.5)),
				},
				Args: []string{"2.6"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
				WantStderr: "validation for \"flArg\" failed: [LT] value isn't less than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [LT] value isn't less than 2.5`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.6"},
					},
				},
			},
		},
		// FloatLTE
		{
			name: "FloatLTE works when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, LTE(2.5)),
				},
				Args: []string{"2.4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.4"},
					},
				},
			},
		},
		{
			name: "FloatLTE works when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, LTE(2.5)),
				},
				Args: []string{"2.5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.5"},
					},
				},
			},
		},
		{
			name: "FloatLTE fails when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, LTE(2.5)),
				},
				Args: []string{"2.6"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
				WantStderr: "validation for \"flArg\" failed: [LTE] value isn't less than or equal to 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [LTE] value isn't less than or equal to 2.5`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.6"},
					},
				},
			},
		},
		// FloatGT
		{
			name: "FloatGT fails when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, GT(2.5)),
				},
				Args: []string{"2.4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
				WantStderr: "validation for \"flArg\" failed: [GT] value isn't greater than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [GT] value isn't greater than 2.5`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.4"},
					},
				},
			},
		},
		{
			name: "FloatGT fails when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, GT(2.5)),
				},
				Args: []string{"2.5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
				WantStderr: "validation for \"flArg\" failed: [GT] value isn't greater than 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [GT] value isn't greater than 2.5`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.5"},
					},
				},
			},
		},
		{
			name: "FloatGT works when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, GT(2.5)),
				},
				Args: []string{"2.6"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.6"},
					},
				},
			},
		},
		// FloatGTE
		{
			name: "FloatGTE fails when less than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, GTE(2.5)),
				},
				Args: []string{"2.4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.4,
				}},
				WantStderr: "validation for \"flArg\" failed: [GTE] value isn't greater than or equal to 2.5\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [GTE] value isn't greater than or equal to 2.5`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.4"},
					},
				},
			},
		},
		{
			name: "FloatGTE works when equal to",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, GTE(2.5)),
				},
				Args: []string{"2.5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.5,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.5"},
					},
				},
			},
		},
		{
			name: "FloatGTE works when greater than",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, GTE(2.5)),
				},
				Args: []string{"2.6"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 2.6,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "2.6"},
					},
				},
			},
		},
		// FloatPositive
		{
			name: "FloatPositive fails when negative",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, Positive[float64]()),
				},
				Args: []string{"-0.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": -0.1,
				}},
				WantStderr: "validation for \"flArg\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Positive] value isn't positive`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-0.1"},
					},
				},
			},
		},
		{
			name: "FloatPositive fails when zero",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, Positive[float64]()),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 0.0,
				}},
				WantStderr: "validation for \"flArg\" failed: [Positive] value isn't positive\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Positive] value isn't positive`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "FloatPositive works when positive",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, Positive[float64]()),
				},
				Args: []string{"0.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 0.1,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0.1"},
					},
				},
			},
		},
		// FloatNegative
		{
			name: "FloatNegative works when negative",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, Negative[float64]()),
				},
				Args: []string{"-0.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": -0.1,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-0.1"},
					},
				},
			},
		},
		{
			name: "FloatNegative fails when zero",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, Negative[float64]()),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 0.0,
				}},
				WantStderr: "validation for \"flArg\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Negative] value isn't negative`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "FloatNegative fails when positive",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, Negative[float64]()),
				},
				Args: []string{"0.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 0.1,
				}},
				WantStderr: "validation for \"flArg\" failed: [Negative] value isn't negative\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [Negative] value isn't negative`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0.1"},
					},
				},
			},
		},
		// FloatNonNegative
		{
			name: "FloatNonNegative fails when negative",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, NonNegative[float64]()),
				},
				Args: []string{"-0.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": -0.1,
				}},
				WantStderr: "validation for \"flArg\" failed: [NonNegative] value isn't non-negative\n",
				WantErr:    fmt.Errorf(`validation for "flArg" failed: [NonNegative] value isn't non-negative`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-0.1"},
					},
				},
			},
		},
		{
			name: "FloatNonNegative works when zero",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, NonNegative[float64]()),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 0.0,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "FloatNonNegative works when positive",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[float64]("flArg", testDesc, NonNegative[float64]()),
				},
				Args: []string{"0.1"},
				WantData: &command.Data{Values: map[string]interface{}{
					"flArg": 0.1,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0.1"},
					},
				},
			},
		},
		// Between inclusive
		{
			name: "Between inclusive fails when less than lower bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, true)),
				},
				Args: []string{"-4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": -4,
				}},
				WantStderr: "validation for \"iArg\" failed: [Between] value is less than lower bound (-3)\n",
				WantErr:    fmt.Errorf("validation for \"iArg\" failed: [Between] value is less than lower bound (-3)"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-4"},
					},
				},
			},
		},
		{
			name: "Between inclusive succeeds when equals lower bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, true)),
				},
				Args: []string{"-3"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": -3,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-3"},
					},
				},
			},
		},
		{
			name: "Between inclusive succeeds when between bounds",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, true)),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 0,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "Between inclusive succeeds when equals upper bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, true)),
				},
				Args: []string{"4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 4,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "4"},
					},
				},
			},
		},
		{
			name: "Between inclusive fails when greater than upper bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, true)),
				},
				Args: []string{"5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 5,
				}},
				WantStderr: "validation for \"iArg\" failed: [Between] value is greater than upper bound (4)\n",
				WantErr:    fmt.Errorf("validation for \"iArg\" failed: [Between] value is greater than upper bound (4)"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "5"},
					},
				},
			},
		},
		// Between exclusive
		{
			name: "Between exclusive fails when less than lower bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, false)),
				},
				Args: []string{"-4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": -4,
				}},
				WantStderr: "validation for \"iArg\" failed: [Between] value is less than lower bound (-3)\n",
				WantErr:    fmt.Errorf("validation for \"iArg\" failed: [Between] value is less than lower bound (-3)"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-4"},
					},
				},
			},
		},
		{
			name: "Between exclusive fails when equals lower bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, false)),
				},
				Args: []string{"-3"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": -3,
				}},
				WantStderr: "validation for \"iArg\" failed: [Between] value equals exclusive lower bound (-3)\n",
				WantErr:    fmt.Errorf("validation for \"iArg\" failed: [Between] value equals exclusive lower bound (-3)"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-3"},
					},
				},
			},
		},
		{
			name: "Between exclusive succeeds when between bounds",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, false)),
				},
				Args: []string{"0"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 0,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "0"},
					},
				},
			},
		},
		{
			name: "Between exclusive fails when equals upper bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, false)),
				},
				Args: []string{"4"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 4,
				}},
				WantStderr: "validation for \"iArg\" failed: [Between] value equals exclusive upper bound (4)\n",
				WantErr:    fmt.Errorf("validation for \"iArg\" failed: [Between] value equals exclusive upper bound (4)"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "4"},
					},
				},
			},
		},
		{
			name: "Between exclusive fails when greater than upper bound",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{
					Processor: Arg[int]("iArg", testDesc, Between(-3, 4, false)),
				},
				Args: []string{"5"},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 5,
				}},
				WantStderr: "validation for \"iArg\" failed: [Between] value is greater than upper bound (4)\n",
				WantErr:    fmt.Errorf("validation for \"iArg\" failed: [Between] value is greater than upper bound (4)"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "5"},
					},
				},
			},
		},
		// Flag processors
		{
			name: "empty flag processor works",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{Processor: FlagProcessor()},
			},
		},
		{
			name: "flag processor allows empty",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{Processor: FlagProcessor(Flag[string]("strFlag", 'f', testDesc))},
			},
		},
		{
			name: "flag processor fails if no argument",
			etc: &commandtest.ExecuteTestCase{
				Node:       &SimpleNode{Processor: FlagProcessor(Flag[string]("strFlag", 'f', testDesc))},
				Args:       []string{"--strFlag"},
				WantStderr: "Argument \"strFlag\" requires at least 1 argument, got 0\n",
				WantErr:    fmt.Errorf(`Argument "strFlag" requires at least 1 argument, got 0`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--strFlag"},
					},
				},
			},
		},
		{
			name: "flag processor parses flag",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{Processor: FlagProcessor(Flag[string]("strFlag", 'f', testDesc))},
				Args: []string{"--strFlag", "hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strFlag": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--strFlag"},
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "flag processor parses short name flag",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{Processor: FlagProcessor(Flag[string]("strFlag", 'f', testDesc))},
				Args: []string{"-f", "hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strFlag": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-f"},
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "flag processor handles FlagNoShortName",
			etc: &commandtest.ExecuteTestCase{
				Node: &SimpleNode{Processor: FlagProcessor(Flag[string]("strFlag", FlagNoShortName, testDesc))},
				Args: []string{"--strFlag", "hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strFlag": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--strFlag"},
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "flag processor parses flag in the middle",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(Flag[string]("strFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "--strFlag", "hello", "deux"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strFlag": "hello",
					"filler":  []string{"un", "deux"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "--strFlag"},
						{Value: "hello"},
						{Value: "deux"},
					},
				},
			},
		},
		{
			name: "flag processor parses short name flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(Flag[string]("strFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"uno", "dos", "-f", "hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler":  []string{"uno", "dos"},
					"strFlag": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "uno"},
						{Value: "dos"},
						{Value: "-f"},
						{Value: "hello"},
					},
				},
			},
		},
		// Int flag
		{
			name: "parses int flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(Flag[int]("intFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "deux", "-f", "3", "quatre"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler":  []string{"un", "deux", "quatre"},
					"intFlag": 3,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "deux"},
						{Value: "-f"},
						{Value: "3"},
						{Value: "quatre"},
					},
				},
			},
		},
		{
			name: "handles invalid int flag value",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(Flag[int]("intFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args:       []string{"un", "deux", "-f", "trois", "quatre"},
				WantStderr: "strconv.Atoi: parsing \"trois\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "trois": invalid syntax`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "deux"},
						{Value: "-f"},
						{Value: "trois"},
						{Value: "quatre"},
					},
					Remaining: []int{0, 1, 4},
				},
			},
		},
		// Float flag
		{
			name: "parses float flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(Flag[float64]("floatFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"--floatFlag", "-1.2", "three"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler":    []string{"three"},
					"floatFlag": -1.2,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--floatFlag"},
						{Value: "-1.2"},
						{Value: "three"},
					},
				},
			},
		},
		{
			name: "handles invalid float flag value",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(Flag[float64]("floatFlag", 'f', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args:       []string{"--floatFlag", "twelve", "eleven"},
				WantStderr: "strconv.ParseFloat: parsing \"twelve\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "twelve": invalid syntax`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--floatFlag"},
						{Value: "twelve"},
						{Value: "eleven"},
					},
					Remaining: []int{2},
				},
			},
		},
		// Bool flag
		{
			name: "bool flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(BoolFlag("boolFlag", 'b', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"okay", "--boolFlag", "then"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler":   []string{"okay", "then"},
					"boolFlag": true,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "okay"},
						{Value: "--boolFlag"},
						{Value: "then"},
					},
				},
			},
		},
		{
			name: "short bool flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(BoolFlag("boolFlag", 'b', testDesc)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"okay", "-b", "then"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler":   []string{"okay", "then"},
					"boolFlag": true,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "okay"},
						{Value: "-b"},
						{Value: "then"},
					},
				},
			},
		},
		// flag list tests
		{
			name: "flag list works",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(ListFlag[string]("slFlag", 's', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "--slFlag", "hello", "there"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler": []string{"un"},
					"slFlag": []string{"hello", "there"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "--slFlag"},
						{Value: "hello"},
						{Value: "there"},
					},
				},
			},
		},
		{
			name: "flag list fails if not enough",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(ListFlag[string]("slFlag", 's', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args:       []string{"un", "--slFlag", "hello"},
				WantStderr: "Argument \"slFlag\" requires at least 2 arguments, got 1\n",
				WantErr:    fmt.Errorf(`Argument "slFlag" requires at least 2 arguments, got 1`),
				WantData: &command.Data{Values: map[string]interface{}{
					"slFlag": []string{"hello"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "--slFlag"},
						{Value: "hello"},
					},
					Remaining: []int{0},
				},
			},
		},
		// Int list
		{
			name: "int list works",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(ListFlag[int]("ilFlag", 'i', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args: []string{"un", "-i", "2", "4", "8", "16", "32", "64"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler": []string{"un", "64"},
					"ilFlag": []int{2, 4, 8, 16, 32},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "-i"},
						{Value: "2"},
						{Value: "4"},
						{Value: "8"},
						{Value: "16"},
						{Value: "32"},
						{Value: "64"},
					},
				},
			},
		},
		{
			name: "int list transform failure",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(ListFlag[int]("ilFlag", 'i', testDesc, 2, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args:       []string{"un", "-i", "2", "4", "8", "16.0", "32", "64"},
				WantStderr: "strconv.Atoi: parsing \"16.0\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "16.0": invalid syntax`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "-i"},
						{Value: "2"},
						{Value: "4"},
						{Value: "8"},
						{Value: "16.0"},
						{Value: "32"},
						{Value: "64"},
					},
					Remaining: []int{0, 7},
				},
			},
		},
		// Float list
		{
			name: "float list works",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(ListFlag[float64]("flFlag", 'f', testDesc, 0, 3)),
					ListArg[string]("filler", testDesc, 1, 3),
				),
				Args: []string{"un", "-f", "2", "-4.4", "0.8", "16.16", "-32", "64"},
				WantData: &command.Data{Values: map[string]interface{}{
					"filler": []string{"un", "16.16", "-32", "64"},
					"flFlag": []float64{2, -4.4, 0.8},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "-f"},
						{Value: "2"},
						{Value: "-4.4"},
						{Value: "0.8"},
						{Value: "16.16"},
						{Value: "-32"},
						{Value: "64"},
					},
				},
			},
		},
		{
			name: "float list transform failure",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(ListFlag[float64]("flFlag", 'f', testDesc, 0, 3)),
					ListArg[string]("filler", testDesc, 1, 2),
				),
				Args:       []string{"un", "--flFlag", "2", "4", "eight", "16.0", "32", "64"},
				WantStderr: "strconv.ParseFloat: parsing \"eight\": invalid syntax\n",
				WantErr:    fmt.Errorf(`strconv.ParseFloat: parsing "eight": invalid syntax`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "un"},
						{Value: "--flFlag"},
						{Value: "2"},
						{Value: "4"},
						{Value: "eight"},
						{Value: "16.0"},
						{Value: "32"},
						{Value: "64"},
					},
					Remaining: []int{0, 5, 6, 7},
				},
			},
		},
		// Flag overlapping tests
		{
			name: "flags don't eat other flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("alpha", 'a', testDesc, 0, command.UnboundedList),
						ListFlag[string]("bravo", 'b', testDesc, 0, command.UnboundedList),
						ListFlag[string]("charlie", 'c', testDesc, 0, command.UnboundedList),
					),
				),
				Args: []string{"--alpha", "hey", "there", "--dude", "--bravo", "yay", "--charlie"},
				WantData: &command.Data{Values: map[string]interface{}{
					"alpha": []string{"hey", "there", "--dude"},
					"bravo": []string{"yay"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--alpha"},
						{Value: "hey"},
						{Value: "there"},
						{Value: "--dude"},
						{Value: "--bravo"},
						{Value: "yay"},
						{Value: "--charlie"},
					},
				},
			},
		},
		{
			name: "flags don't eat other short flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("alpha", 'a', testDesc, 0, command.UnboundedList),
						ListFlag[string]("bravo", 'b', testDesc, 0, command.UnboundedList),
						ListFlag[string]("charlie", 'c', testDesc, 0, command.UnboundedList),
					),
				),
				Args: []string{"-a", "hey", "there", "-d", "-b", "yay", "-c"},
				WantData: &command.Data{Values: map[string]interface{}{
					"alpha": []string{"hey", "there", "-d"},
					"bravo": []string{"yay"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-a"},
						{Value: "hey"},
						{Value: "there"},
						{Value: "-d"},
						{Value: "-b"},
						{Value: "yay"},
						{Value: "-c"},
					},
				},
			},
		},
		{
			name: "flags don't eat valid multi flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("alpha", 'a', testDesc, 0, command.UnboundedList),
						BoolFlag("Q", 'q', testDesc),
						BoolFlag("W", 'w', testDesc),
						BoolFlag("E", 'e', testDesc),
						BoolFlag("R", 'r', testDesc),
					),
				),
				Args: []string{"-a", "hey", "there", "-qwer"},
				WantData: &command.Data{Values: map[string]interface{}{
					"alpha": []string{"hey", "there"},
					"Q":     true,
					"W":     true,
					"E":     true,
					"R":     true,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-a"},
						{Value: "hey"},
						{Value: "there"},
						{Value: "-qwer"},
					},
				},
			},
		},
		{
			name: "flags eat invalid multi flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("alpha", 'a', testDesc, 0, command.UnboundedList),
						BoolFlag("Q", 'q', testDesc),
						BoolFlag("W", 'w', testDesc),
						BoolFlag("E", 'e', testDesc),
						BoolFlag("R", 'r', testDesc),
					),
				),
				Args: []string{"-a", "hey", "there", "-qwert"},
				WantData: &command.Data{Values: map[string]interface{}{
					"alpha": []string{"hey", "there", "-qwert"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-a"},
						{Value: "hey"},
						{Value: "there"},
						{Value: "-qwert"},
					},
				},
			},
		},
		// Misc. flag tests
		{
			name: "processes multiple flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[float64]("coordinates", 'c', testDesc, 2, 0),
						BoolFlag("boo", 'b', testDesc),
						ListFlag[string]("names", 'n', testDesc, 1, 2),
						Flag[int]("rating", 'r', testDesc),
					),
					ListArg[string]("extra", testDesc, 0, command.UnboundedList),
				),
				Args: []string{"its", "--boo", "a", "-r", "9", "secret", "-n", "greggar", "groog", "beggars", "--coordinates", "2.2", "4.4", "message."},
				WantData: &command.Data{Values: map[string]interface{}{
					"boo":         true,
					"extra":       []string{"its", "a", "secret", "message."},
					"names":       []string{"greggar", "groog", "beggars"},
					"coordinates": []float64{2.2, 4.4},
					"rating":      9,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "its"},
						{Value: "--boo"},
						{Value: "a"},
						{Value: "-r"},
						{Value: "9"},
						{Value: "secret"},
						{Value: "-n"},
						{Value: "greggar"},
						{Value: "groog"},
						{Value: "beggars"},
						{Value: "--coordinates"},
						{Value: "2.2"},
						{Value: "4.4"},
						{Value: "message."},
					},
				},
			},
		},
		// FlagStop tests
		{
			name: "stops processing flags after FlagStop",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[float64]("coordinates", 'c', testDesc, 2, 0),
						BoolFlag("boo", 'b', testDesc),
						BoolFlag("yay", 'y', testDesc),
						ListFlag[string]("names", 'n', testDesc, 1, 2),
						Flag[int]("rating", 'r', testDesc),
					),
					ListArg[string]("extra", testDesc, 0, command.UnboundedList),
				),
				Args: []string{"its", "-b", "a", "-r", "9", "--", "secret", "--yay", "-n", "greggar", "groog", "beggars", "--coordinates", "2.2", "4.4", "message."},
				WantData: &command.Data{Values: map[string]interface{}{
					"boo":    true,
					"extra":  []string{"its", "a", "secret", "--yay", "-n", "greggar", "groog", "beggars", "--coordinates", "2.2", "4.4", "message."},
					"rating": 9,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "its"},
						{Value: "-b"},
						{Value: "a"},
						{Value: "-r"},
						{Value: "9"},
						{Value: "--"},
						{Value: "secret"},
						{Value: "--yay"},
						{Value: "-n"},
						{Value: "greggar"},
						{Value: "groog"},
						{Value: "beggars"},
						{Value: "--coordinates"},
						{Value: "2.2"},
						{Value: "4.4"},
						{Value: "message."},
					},
				},
			},
		},
		// BoolValueFlag
		{
			name: "BoolValueFlag works with true value",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolValueFlag("light", 'l', testDesc, "hello there"),
					),
				),
				Args: []string{"--light"},
				WantData: &command.Data{Values: map[string]interface{}{
					"light": "hello there",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--light"},
					},
				},
			},
		},
		{
			name: "BoolValueFlag works with false value",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolValueFlag("light", 'l', testDesc, "hello there"),
					),
				),
			},
		},
		{
			name: "BoolValuesFlag works with true value",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolValuesFlag("light", 'l', testDesc, "hello there", "general kenobi"),
					),
				),
				Args: []string{"--light"},
				WantData: &command.Data{Values: map[string]interface{}{
					"light": "hello there",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--light"},
					},
				},
			},
		},
		{
			name: "BoolValuesFlag works with false value",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolValuesFlag("light", 'l', testDesc, "hello there", "general kenobi"),
					),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"light": "general kenobi",
				}},
			},
		},
		// Multi-flag tests
		{
			name: "Multiple bool flags work as a multi-flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-qwer"},
				WantData: &command.Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
					"run":      "hello there",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-qwer"},
					},
				},
			},
		},
		{
			name: "Multi-flag fails if partial set of matches",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Args:       []string{"-qwy"},
				WantStderr: "Either all or no flags in a multi-flag object must be relevant for a FlagProcessor group\n",
				WantErr:    fmt.Errorf(`Either all or no flags in a multi-flag object must be relevant for a FlagProcessor group`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-qwy"},
					},
					Remaining: []int{0},
				},
			},
		},
		{
			name: "Multi-flags are ignored if no matches",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
					OptionalArg[string]("LEFTOVERS", testDesc),
				),
				Args: []string{"-nop"},
				WantData: &command.Data{Values: map[string]interface{}{
					"LEFTOVERS": "-nop",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-nop"},
					},
					Remaining: []int{},
				},
			},
		},
		{
			name: "Multi-flag fails if uncombinable flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						ListFlag[int]("two", 't', testDesc, 0, command.UnboundedList),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Args: []string{"-ert"},
				WantData: &command.Data{Values: map[string]interface{}{
					"everyone": true,
					"run":      true,
				}},
				WantStderr: "Flag \"two\" is not combinable\n",
				WantErr:    fmt.Errorf(`Flag "two" is not combinable`),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-ert"},
					},
					Remaining: []int{0},
				},
			},
		},
		// Duplicate flag tests
		{
			name: "Duplicate flags get caught in multi-flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-qwerq"},
				WantData: &command.Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
					"run":      "hello there",
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-qwerq"},
					},
					Remaining: []int{0},
				},
			},
		},
		{
			name: "Duplicate flags get caught in regular flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-q", "--quick"},
				WantData: &command.Data{Values: map[string]interface{}{
					"quick": true,
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-q"},
						{Value: "--quick"},
					},
					Remaining: []int{1},
				},
			},
		},
		{
			name: "Duplicate flags get caught when multi, then regular flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"-qwer", "--quick"},
				WantData: &command.Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
					"run":      "hello there",
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-qwer"},
						{Value: "--quick"},
					},
					Remaining: []int{1},
				},
			},
		},
		{
			name: "Duplicate flags get caught when regular, then multi flag",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolValuesFlag("run", 'r', testDesc, "hello there", "general kenobi"),
						BoolValueFlag("to", 't', testDesc, 123),
						BoolValueFlag("where", 'w', testDesc, 4.56),
					),
				),
				Args: []string{"--quick", "-weqr"},
				WantData: &command.Data{Values: map[string]interface{}{
					"quick":    true,
					"where":    4.56,
					"everyone": true,
				}},
				WantErr:    fmt.Errorf(`Flag "quick" has already been set`),
				WantStderr: "Flag \"quick\" has already been set\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--quick"},
						{Value: "-weqr"},
					},
					Remaining: []int{1},
				},
			},
		},
		// OptionalFlag tests
		{
			name: "OptionalFlag sets if default if last argument",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag("of", 'o', testDesc, "dfltValue"),
					),
				),
				Args: []string{"--of"},
				WantData: &command.Data{Values: map[string]interface{}{
					"of": "dfltValue",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--of"},
					},
				},
			},
		},
		{
			name: "OptionalFlag doesn't eat other flags",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag("of", 'o', testDesc, "dfltValue"),
						Flag[string]("sf", 's', testDesc),
					),
				),
				Args: []string{"--of", "--sf", "hello"},
				WantData: &command.Data{Values: map[string]interface{}{
					"of": "dfltValue",
					"sf": "hello",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--of"},
						{Value: "--sf"},
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "OptionalFlag gets set",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag("of", 'o', testDesc, "dfltValue"),
					),
				),
				Args: []string{"--of", "other"},
				WantData: &command.Data{Values: map[string]interface{}{
					"of": "other",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--of"},
						{Value: "other"},
					},
				},
			},
		},
		{
			name: "OptionalFlag handles error",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag("of", 'o', testDesc, 123),
					),
				),
				Args:       []string{"--of", "not-a-number"},
				WantErr:    fmt.Errorf(`strconv.Atoi: parsing "not-a-number": invalid syntax`),
				WantStderr: "strconv.Atoi: parsing \"not-a-number\": invalid syntax\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--of"},
						{Value: "not-a-number"},
					},
				},
			},
		},
		// ItemizedListFlag tests
		{
			name: "Itemized list flag requires argument",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ItemizedListFlag[string]("ilf", 'i', testDesc),
					),
				),
				Args:       []string{"--ilf"},
				WantErr:    fmt.Errorf("Argument \"ilf\" requires at least 1 argument, got 0"),
				WantStderr: "Argument \"ilf\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--ilf"},
					},
				},
			},
		},
		{
			name: "Itemized list flag only takes one argument",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ItemizedListFlag[string]("ilf", 'i', testDesc),
					),
					ListArg[string]("sl", testDesc, 0, command.UnboundedList),
				),
				Args: []string{"--ilf", "i1", "other"},
				WantData: &command.Data{Values: map[string]interface{}{
					"ilf": []string{"i1"},
					"sl":  []string{"other"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--ilf"},
						{Value: "i1"},
						{Value: "other"},
					},
				},
			},
		},
		{
			name: "Mixed itemized args",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ItemizedListFlag[string]("ilf", 'i', testDesc),
					),
					ListArg[string]("sl", testDesc, 0, command.UnboundedList),
				),
				Args: []string{"--ilf", "i1", "other", "thing", "-i", "robot", "--ilf", "phone", "okay", "-i", "enough", "then"},
				WantData: &command.Data{Values: map[string]interface{}{
					"ilf": []string{"i1", "robot", "phone", "enough"},
					"sl":  []string{"other", "thing", "okay", "then"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "--ilf"},
						{Value: "i1"},
						{Value: "other"},
						{Value: "thing"},
						{Value: "-i"},
						{Value: "robot"},
						{Value: "--ilf"},
						{Value: "phone"},
						{Value: "okay"},
						{Value: "-i"},
						{Value: "enough"},
						{Value: "then"},
					},
				},
			},
		},
		// Transformer tests.
		{
			name: "args get transformed",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					Arg[string]("strArg", testDesc, &Transformer[string]{F: func(v string, d *command.Data) (string, error) {
						return strings.ToUpper(v), nil
					}}),
					Arg[int]("intArg", testDesc, &Transformer[int]{F: func(v int, d *command.Data) (int, error) {
						return 10 * v, nil
					}}),
				),
				Args: []string{"hello", "12"},
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "HELLO",
					"intArg": 120,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{{Value: "HELLO"}, {Value: "120"}},
				},
			},
		},
		{
			name: "list arg get transformed with TransformerList",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 2, 3, TransformerList(&Transformer[string]{F: func(v string, d *command.Data) (string, error) {
						return strings.ToUpper(v), nil
					}})),
				),
				Args: []string{"hello", "there", "general", "kenobi"},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"HELLO", "THERE", "GENERAL", "KENOBI"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "HELLO"},
						{Value: "THERE"},
						{Value: "GENERAL"},
						{Value: "KENOBI"},
					},
				},
			},
		},
		{
			name: "list arg transformer fails if number of args increases",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 2, 3, &Transformer[[]string]{F: func(v []string, d *command.Data) ([]string, error) {
						return append(v, "!"), nil
					}}),
				),
				Args:       []string{"hello", "there", "general", "kenobi"},
				WantErr:    fmt.Errorf("[sl] Transformers must return a value that is the same length as the original arguments"),
				WantStderr: "[sl] Transformers must return a value that is the same length as the original arguments\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "there"},
						{Value: "general"},
						{Value: "kenobi"},
					},
				},
			},
		},
		{
			name: "list arg transformer fails if number of args decreases",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 2, 3, &Transformer[[]string]{F: func(v []string, d *command.Data) ([]string, error) {
						return v[:len(v)-1], nil
					}}),
				),
				Args:       []string{"hello", "there", "general", "kenobi"},
				WantErr:    fmt.Errorf("[sl] Transformers must return a value that is the same length as the original arguments"),
				WantStderr: "[sl] Transformers must return a value that is the same length as the original arguments\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
						{Value: "there"},
						{Value: "general"},
						{Value: "kenobi"},
					},
				},
			},
		},
		// InputTransformer tests.
		{
			name: "InputTransformer handles no arguments",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FileNumberInputTransformer(1),
				),
			},
		},
		{
			name: "InputTransformer handles non-matching arguments",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FileNumberInputTransformer(1),
					Arg[string]("s", testDesc),
					Arg[int]("i", testDesc),
				),
				Args: []string{"hello.go", "248"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "hello.go",
					"i": 248,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello.go"},
						{Value: "248"},
					},
				},
			},
		},
		{
			name: "InputTransformer expands matching arguments",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FileNumberInputTransformer(1),
					Arg[string]("s", testDesc),
					Arg[int]("i", testDesc),
				),
				Args: []string{"hello.go:248"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "hello.go",
					"i": 248,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello.go"},
						{Value: "248"},
					},
				},
			},
		},
		{
			name: "InputTransformer fails",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FileNumberInputTransformer(1),
				),
				Args:       []string{"hello.go:248:extra"},
				WantErr:    fmt.Errorf("Expected either 1 or 2 parts, got 3"),
				WantStderr: "Expected either 1 or 2 parts, got 3\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Remaining: []int{0},
					Args: []*spycommand.InputArg{
						{Value: "hello.go:248:extra"},
					},
				},
			},
		},
		{
			name: "InputTransformer expands multiple matching arguments",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					FileNumberInputTransformer(2),
					Arg[string]("s", testDesc),
					Arg[int]("i", testDesc),
					Arg[string]("s2", testDesc),
					Arg[int]("i2", testDesc),
				),
				Args: []string{"hello.go:248", "there.txt:139"},
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "hello.go",
					"i":  248,
					"s2": "there.txt",
					"i2": 139,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello.go"},
						{Value: "248"},
						{Value: "there.txt"},
						{Value: "139"},
					},
				},
			},
		},
		// Stdoutln tests
		{
			name: "stdoutln works",
			etc: &commandtest.ExecuteTestCase{
				Node:       printlnNode(true, "one", 2, 3.0),
				WantStdout: "one 2 3\n",
			},
		},
		{
			name: "stderrln works",
			etc: &commandtest.ExecuteTestCase{
				Node:       printlnNode(false, "uh", 0),
				WantStderr: "uh 0\n",
				WantErr:    fmt.Errorf("uh 0"),
			},
		},
		// BranchNode tests
		{
			name: "branch node requires branch argument",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
				},
				WantStderr: "Branching argument must be one of [b h]\n",
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:     true,
				WantIsBranchingError: true,
			},
		},
		{
			name: "branch node requires matching branch argument",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
				},
				Args:       []string{"uh"},
				WantStderr: "Branching argument must be one of [b h]\n",
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:     true,
				WantIsBranchingError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "uh"},
					},
					Remaining: []int{0},
				},
			},
		},
		{
			name: "branch node forwards to proper node",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
				},
				Args:       []string{"h"},
				WantStdout: "hello",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "h"},
					},
				},
			},
		},
		{
			name: "branch node forwards to default if none provided",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
					Default: printNode("default"),
				},
				WantStdout: "default",
			},
		},
		{
			name: "branch node forwards to default if unknown provided",
			etc: &commandtest.ExecuteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
					Default: SerialNodes(ListArg[string]("sl", testDesc, 0, command.UnboundedList), printArgsNode()),
				},
				Args:       []string{"good", "morning"},
				WantStdout: "sl: [good morning]\n",
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"good", "morning"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "good"},
						{Value: "morning"},
					},
				},
			},
		},
		{
			name: "branch node forwards to synonym",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"B"},
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
					Default: printNode("default"),
					Synonyms: BranchSynonyms(map[string][]string{
						"b": {"bee", "B", "Be"},
					}),
				},
				WantStdout: "goodbye",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "B"},
					},
				},
			},
		},
		{
			name: "branch node fails if synonym to unknown command",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"uh"},
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
					Synonyms: BranchSynonyms(map[string][]string{
						"o": {"uh"},
					}),
				},
				WantStderr: "Branching argument must be one of [b h]\n",
				WantErr:    fmt.Errorf("Branching argument must be one of [b h]"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:     true,
				WantIsBranchingError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "uh"},
					},
					Remaining: []int{0},
				},
			},
		},
		{
			name: "branch node forwards to default if synonym to unknown command",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"uh"},
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h": printNode("hello"),
						"b": printNode("goodbye"),
					},
					Default: SerialNodes(ListArg[string]("sl", testDesc, 0, command.UnboundedList), printArgsNode()),
					Synonyms: BranchSynonyms(map[string][]string{
						"o": {"uh"},
					}),
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"sl": []string{"uh"},
					},
				},
				WantStdout: "sl: [uh]\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "uh"},
					},
				},
			},
		},
		{
			name: "branch node forwards to spaced synonym",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"bee"},
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"h":          printNode("hello"),
						"b bee B Be": printNode("goodbye"),
					},
					Default: printNode("default"),
				},
				WantStdout: "goodbye",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "bee"},
					},
				},
			},
		},
		// BranchNode synonym tests
		{
			name: "branch node works with branch name",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"hello"},
				Node:       branchSynNode(),
				WantStdout: "yo",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "branch node works with branch name",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"hello"},
				Node:       branchSynNode(),
				WantStdout: "yo",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hello"},
					},
				},
			},
		},
		{
			name: "branch node works with second spaced alias",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"greetings"},
				Node:       branchSynNode(),
				WantStdout: "yo",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "greetings"},
					},
				},
			},
		},
		{
			name: "branch node works with first synonym",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"hey"},
				Node:       branchSynNode(),
				WantStdout: "yo",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "hey"},
					},
				},
			},
		},
		{
			name: "branch node works with second synonym",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"howdy"},
				Node:       branchSynNode(),
				WantStdout: "yo",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "howdy"},
					},
				},
			},
		},
		// NodeRepeater tests
		{
			name: "NodeRepeater fails if not enough",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(3, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
				WantErr:    fmt.Errorf(`Argument "KEY" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"KEY\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater fails if middle node doen't have enough",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, 1)),
				Args: []string{"k1", "100", "k2"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100},
				}},
				WantErr:    fmt.Errorf(`Argument "VALUE" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"VALUE\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
					},
				},
			},
		},
		{
			name: "NodeRepeater fails if too many",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1"},
					"values": []int{100},
				}},
				WantErr: fmt.Errorf(`Unprocessed extra args: [k2 200]`),
				WantStderr: strings.Join([]string{
					`Unprocessed extra args: [k2 200]`,
					``,
					`======= Command Usage =======`,
					`KEY VALUE`,
					``,
					`Arguments:`,
					`  KEY: test desc`,
					`  VALUE: test desc`,
					``,
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
					},
					Remaining: []int{2, 3},
				},
			},
		},
		{
			name: "NodeRepeater accepts minimum when no optional",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts minimum when optional",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 3)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts minimum when unlimited optional",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 3)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts maximum when no optional",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 0)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater accepts maximum when optional",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, 1)),
				Args: []string{"k1", "100", "k2", "200"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2"},
					"values": []int{100, 200},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
					},
				},
			},
		},
		{
			name: "NodeRepeater with unlimited optional accepts a bunch",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, command.UnboundedList)),
				Args: []string{"k1", "100", "k2", "200", "k3", "300", "k4", "400", "...", "0", "kn", "999"},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"k1", "k2", "k3", "k4", "...", "kn"},
					"values": []int{100, 200, 300, 400, 0, 999},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "k1"},
						{Value: "100"},
						{Value: "k2"},
						{Value: "200"},
						{Value: "k3"},
						{Value: "300"},
						{Value: "k4"},
						{Value: "400"},
						{Value: "..."},
						{Value: "0"},
						{Value: "kn"},
						{Value: "999"},
					},
				},
			},
		},
		// ListBreaker tests
		{
			name: "Handles broken list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 0, command.UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{"ghi", "jkl"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "ghi"},
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "List breaker before min value",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 3, command.UnboundedList, ListUntilSymbol("ghi")),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def"},
				}},
				WantErr:    fmt.Errorf(`Argument "SL" requires at least 3 arguments, got 2`),
				WantStderr: "Argument \"SL\" requires at least 3 arguments, got 2\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "ghi"},
						{Value: "jkl"},
					},
					Remaining: []int{2, 3},
				},
			},
		},
		{
			name: "Handles broken list with discard",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, func() *ListBreaker[[]string] {
						li := ListUntilSymbol("ghi")
						li.Discard = true
						return li
					}()),
					ListArg[string]("SL2", testDesc, 0, command.UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{"jkl"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "ghi"},
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "Handles unbroken list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 0, command.UnboundedList),
				),
				Args: []string{"abc", "def", "ghif", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def", "ghif", "jkl"},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "ghif"},
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "Fails if arguments required after broken list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, ListUntilSymbol("ghi")),
					ListArg[string]("SL2", testDesc, 1, command.UnboundedList),
				),
				Args: []string{"abc", "def", "ghif", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def", "ghif", "jkl"},
				}},
				WantErr:    fmt.Errorf(`Argument "SL2" requires at least 1 argument, got 0`),
				WantStderr: "Argument \"SL2\" requires at least 1 argument, got 0\n",
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsUsageError:         true,
				WantIsNotEnoughArgsError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "ghif"},
						{Value: "jkl"},
					},
				},
			},
		},
		// StringListListProcessor tests
		{
			name: "StringListListProcessor works if no breakers",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, command.UnboundedList),
				),
				Args: []string{"abc", "def", "ghi", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def", "ghi", "jkl"},
					},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "ghi"},
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListProcessor works with unbounded list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, command.UnboundedList),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "|"},
						{Value: "ghi"},
						{Value: "||"},
						{Value: "|"},
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListProcessor works with bounded list",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "|"},
						{Value: "ghi"},
						{Value: "||"},
						{Value: "|"},
						{Value: "jkl"},
					},
				},
			},
		},
		{
			name: "StringListListProcessor works if ends with operator",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl", "|"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "|"},
						{Value: "ghi"},
						{Value: "||"},
						{Value: "|"},
						{Value: "jkl"},
						{Value: "|"},
					},
				},
			},
		},
		{
			name: "StringListListProcessor fails if extra args",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, 2),
				),
				Args: []string{"abc", "def", "|", "ghi", "||", "|", "jkl", "|", "other", "stuff"},
				WantData: &command.Data{Values: map[string]interface{}{
					"SLL": [][]string{
						{"abc", "def"},
						{"ghi", "||"},
						{"jkl"},
					},
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [other stuff]"),
				WantStderr: strings.Join([]string{
					`Unprocessed extra args: [other stuff]`,
					``,
					`======= Command Usage =======`,
					`[ SLL ... ] | { [ SLL ... ] | [ SLL ... ] | }`,
					``,
					`Arguments:`,
					`  SLL: test desc`,
					``,
					`Symbols:`,
					`  |: List breaker`,
					``,
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
						{Value: "|"},
						{Value: "ghi"},
						{Value: "||"},
						{Value: "|"},
						{Value: "jkl"},
						{Value: "|"},
						{Value: "other"},
						{Value: "stuff"},
					},
					Remaining: []int{8, 9},
				},
			},
		},
		// FileContents test
		{
			name: "file gets read properly",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(FileContents("FILE", testDesc)),
				Args: []string{filepath.Join("testdata", "one.txt")},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"FILE": []string{"hello", "there"},
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "testdata", "one.txt")},
					},
				},
			},
		},
		{
			name: "FileContents fails for unknown file",
			etc: &commandtest.ExecuteTestCase{
				Node:       SerialNodes(FileContents("FILE", testDesc)),
				Args:       []string{filepath.Join("uh")},
				WantStderr: fmt.Sprintf("validation for \"FILE\" failed: [FileExists] file %q does not exist\n", testutil.FilepathAbs(t, "uh")),
				WantErr:    fmt.Errorf(`validation for "FILE" failed: [FileExists] file %q does not exist`, testutil.FilepathAbs(t, "uh")),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"FILE": testutil.FilepathAbs(t, "uh"),
					},
				},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: testutil.FilepathAbs(t, "uh")},
					},
				},
			},
		},
		// If tests
		{
			name: "If runs if function returns true",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"abc", "def"},
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					If(
						printArgsNode(),
						func(i *command.Input, d *command.Data) bool {
							return true
						},
					),
					Arg[string]("s2", testDesc),
				),
				WantStdout: strings.Join([]string{
					"s: abc",
					"s2: def",
					"",
				}, "\n"),
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "abc",
					"s2": "def",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
					},
				},
			},
		},
		{
			name: "If does not run if function returns false",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"abc", "def"},
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					If(
						printArgsNode(),
						func(i *command.Input, d *command.Data) bool {
							return false
						},
					),
					Arg[string]("s2", testDesc),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "abc",
					"s2": "def",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
						{Value: "def"},
					},
				},
			},
		},
		// IfData tests
		{
			name: "IfData runs if variable is present",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"abc"},
				Node: SerialNodes(
					OptionalArg[string]("s", testDesc),
					IfData("s", printlnNode(true, "hello")),
				),
				WantStdout: "hello\n",
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "abc",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
					},
				},
			},
		},
		{
			name: "IfData runs if bool variable is present and true",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"true"},
				Node: SerialNodes(
					OptionalArg[bool]("b", testDesc),
					IfData("b", printlnNode(true, "hello")),
				),
				WantStdout: "hello\n",
				WantData: &command.Data{Values: map[string]interface{}{
					"b": true,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "true"},
					},
				},
			},
		},
		{
			name: "IfData does not run if variable is not present",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					OptionalArg[string]("s", testDesc),
					IfData("s", printlnNode(true, "hello")),
				),
			},
		},
		{
			name: "IfData does not run if bool variable is present and false",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"false"},
				Node: SerialNodes(
					OptionalArg[bool]("b", testDesc),
					IfData("b", printlnNode(true, "hello")),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"b": false,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "false"},
					},
				},
			},
		},
		// IfElseData tests
		{
			name: "IfElseData runs t if variable is present",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"abc"},
				Node: SerialNodes(
					OptionalArg[string]("s", testDesc),
					IfElseData(
						"s",
						printlnNode(true, "hello"),
						printlnNode(true, "goodbye"),
					),
				),
				WantStdout: "hello\n",
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "abc",
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
					},
				},
			},
		},
		{
			name: "IfElseData runs t if bool variable is present and true",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"true"},
				Node: SerialNodes(
					OptionalArg[bool]("b", testDesc),
					IfElseData(
						"b",
						printlnNode(true, "hello"),
						printlnNode(true, "goodbye"),
					),
				),
				WantStdout: "hello\n",
				WantData: &command.Data{Values: map[string]interface{}{
					"b": true,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "true"},
					},
				},
			},
		},
		{
			name: "IfElseData runs f if variable is not present",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					OptionalArg[string]("s", testDesc),
					IfElseData(
						"s",
						printlnNode(true, "hello"),
						printlnNode(true, "goodbye"),
					),
				),
				WantStdout: "goodbye\n",
			},
		},
		{
			name: "IfData runs f if bool variable is present and false",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"false"},
				Node: SerialNodes(
					OptionalArg[bool]("b", testDesc),
					IfElseData(
						"b",
						printlnNode(true, "hello"),
						printlnNode(true, "goodbye"),
					),
				),
				WantStdout: "goodbye\n",
				WantData: &command.Data{Values: map[string]interface{}{
					"b": false,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "false"},
					},
				},
			},
		},
		// EchoExecuteData
		{
			name: "EchoExecuteData ignores empty command.ExecuteData.Executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					EchoExecuteData(),
				),
			},
		},
		{
			name: "EchoExecuteData outputs command.ExecuteData.Executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("un", "deux", "trois"),
					EchoExecuteData(),
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"un", "deux", "trois"},
				},
				WantStdout: "un\ndeux\ntrois\n",
			},
		},
		{
			name: "EchoExecuteData outputs command.ExecuteData.Executable to stderr",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("un", "deux", "trois"),
					&EchoExecuteDataProcessor{
						Stderr: true,
					},
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"un", "deux", "trois"},
				},
				WantStderr: "un\ndeux\ntrois\n",
			},
		},
		{
			name: "EchoExecuteDataf ignores empty command.ExecuteData.Executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					EchoExecuteDataf("RUNNING CODE:\n%s\nDONE CODE\n"),
				),
			},
		},
		{
			name: "EchoExecuteData outputs command.ExecuteData.Executable",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("un", "deux", "trois"),
					EchoExecuteDataf("RUNNING CODE:\n%s\nDONE CODE\n"),
				),
				WantExecuteData: &command.ExecuteData{
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
		{
			name: "EchoExecuteData outputs command.ExecuteData.Executable to stderr",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					SimpleExecutableProcessor("un", "deux", "trois"),
					&EchoExecuteDataProcessor{
						Stderr: true,
						Format: "RUNNING CODE:\n%s\nDONE CODE\n",
					},
				),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{"un", "deux", "trois"},
				},
				WantStderr: strings.Join([]string{
					"RUNNING CODE:",
					"un",
					"deux",
					"trois",
					"DONE CODE",
					"",
				}, "\n"),
			},
		},
		// MapArg tests
		{
			name: "MapArg converts a value with allowMissing=true",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, true),
				),
				Args: []string{"two"},
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 2,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "two"},
					},
				},
			},
		},
		{
			name: "MapArg converts a value with allowMissing=false",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, false),
				),
				Args: []string{"two"},
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 2,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "two"},
					},
				},
			},
		},
		{
			name: "MapArg converts to default value if allow missing",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, true),
				),
				Args: []string{"four"},
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 0,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "four"},
					},
				},
			},
		},
		{
			name: "MapArg fails if allow missing is false",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, false),
				),
				Args:       []string{"four"},
				WantStderr: "validation for \"m\" failed: [MapArg] key (four) is not in map; expected one of [one three two]\n",
				WantErr:    fmt.Errorf("validation for \"m\" failed: [MapArg] key (four) is not in map; expected one of [one three two]"),
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 0,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "four"},
					},
				},
			},
		},
		{
			name: "MapArg fails if allow missing is false",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, false),
				),
				Args:       []string{"four"},
				WantStderr: "validation for \"m\" failed: [MapArg] key (four) is not in map; expected one of [one three two]\n",
				WantErr:    fmt.Errorf("validation for \"m\" failed: [MapArg] key (four) is not in map; expected one of [one three two]"),
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 0,
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantIsValidationError: true,
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "four"},
					},
				},
			},
		},
		{
			name: "MapArg works with custom types",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]struct {
						A int
						B float64
					}{
						"one":   {1, 1.0},
						"two":   {2, 2.0},
						"three": {3, 3.0},
					}, true),
				),
				Args: []string{"three"},
				WantData: &command.Data{Values: map[string]interface{}{
					"m": struct {
						A int
						B float64
					}{3, 3.0},
				}},
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "three"},
					},
				},
			},
		},
		{
			name: "MapArg.Get, GetOrDefault, and GetKey works",
			etc: func() *commandtest.ExecuteTestCase {
				ma := MapArg("m", testDesc, map[string]int{
					"one":   1,
					"two":   2,
					"three": 3,
				}, true)
				otherMa := MapArg("m2", testDesc, map[string]int{
					"one":   1,
					"two":   2,
					"three": 3,
				}, true)

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						ma,
						SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							o.Stdoutln(ma.GetKey())
							o.Stdoutln(otherMa.GetKey())
							o.Stdoutln(ma.Get(d))
							o.Stdoutln(ma.GetOrDefault(d, 7))
							o.Stdoutln(otherMa.Get(d))
							o.Stdoutln(otherMa.GetOrDefault(d, 7))
							return nil
						}, nil),
					),
					Args: []string{"three"},
					WantData: &command.Data{Values: map[string]interface{}{
						"m": 3,
					}},
					WantStdout: strings.Join([]string{
						"three", // ma.GetKey
						"",      // otherMa.GetKey
						"3",     // ma.Get
						"3",     // ma.GetOrDefault
						"0",     // otherMa.Get
						"7",     // otherMa.GetOrDefault
						"",
					}, "\n"),
				}
			}(),
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "three"},
					},
				},
			},
		},
		{
			name: "MapArg.Get, GetOrDefault, and GetKey works with custom type",
			etc: func() *commandtest.ExecuteTestCase {
				type vType struct {
					A int
					B float64
				}
				ma := MapArg("m", testDesc, map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)
				otherMa := MapArg("m2", testDesc, map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)
				missingMa := MapArg("m3", testDesc, map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						ma,
						missingMa,
						SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							o.Stdoutln("GetKey", ma.GetKey())
							o.Stdoutln("Provided", ma.Provided(d))
							o.Stdoutln("Get", ma.Get(d))
							o.Stdoutln("GetOrDefault", ma.GetOrDefault(d, &vType{7, 7.7}))
							o.Stdoutln("Hit", ma.Hit())
							o.Stdoutln("===")
							o.Stdoutln("GetKey", otherMa.GetKey())
							o.Stdoutln("Provided", otherMa.Provided(d))
							o.Stdoutln("Get", otherMa.Get(d))
							o.Stdoutln("GetOrDefault", otherMa.GetOrDefault(d, &vType{7, 7.7}))
							o.Stdoutln("Hit", otherMa.Hit())
							o.Stdoutln("===")
							o.Stdoutln("GetKey", missingMa.GetKey())
							o.Stdoutln("Provided", missingMa.Provided(d))
							o.Stdoutln("Get", missingMa.Get(d))
							o.Stdoutln("GetOrDefault", missingMa.GetOrDefault(d, &vType{7, 7.7}))
							o.Stdoutln("Hit", missingMa.Hit())
							return nil
						}, nil),
					),
					Args: []string{"three", "eleven"},
					WantData: &command.Data{Values: map[string]interface{}{
						"m":  &vType{3, 3.3},
						"m3": (*vType)(nil),
					}},
					WantStdout: strings.Join([]string{
						"GetKey three",
						"Provided true",
						"Get &{3 3.3}",
						"GetOrDefault &{3 3.3}",
						"Hit true",
						"===",
						"GetKey ",
						"Provided false",
						"Get <nil>",
						"GetOrDefault &{7 7.7}",
						"Hit false",
						"===",
						"GetKey eleven",
						"Provided true",
						"Get <nil>",
						"GetOrDefault <nil>",
						"Hit false",
						"",
					}, "\n"),
				}
			}(),
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "three"},
						{Value: "eleven"},
					},
				},
			},
		},
		// MapFlag
		{
			name: "MapFlag.Get, Provided, GetOrDefault, and GetKey works with custom type for flag",
			etc: func() *commandtest.ExecuteTestCase {
				type vType struct {
					A int
					B float64
				}
				ma := MapFlag("m1", 'm', testDesc, map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)
				otherMa := MapFlag("m2", 'd', testDesc, map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)
				missingMa := MapFlag("m3", FlagNoShortName, testDesc, map[string]*vType{
					"one":   {1, 1.1},
					"two":   {2, 2.2},
					"three": {3, 3.3},
				}, true)

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						FlagProcessor(
							ma,
							otherMa,
							missingMa,
						),
						SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							o.Stdoutln("GetKey", ma.GetKey())
							o.Stdoutln("Provided", ma.Provided(d))
							o.Stdoutln("Get", ma.Get(d))
							o.Stdoutln("GetOrDefault", ma.GetOrDefault(d, &vType{7, 7.7}))
							o.Stdoutln("Hit", ma.Hit())
							o.Stdoutln("===")
							o.Stdoutln("GetKey", otherMa.GetKey())
							o.Stdoutln("Get", otherMa.Get(d))
							o.Stdoutln("Provided", otherMa.Provided(d))
							o.Stdoutln("GetOrDefault", otherMa.GetOrDefault(d, &vType{7, 7.7}))
							o.Stdoutln("Hit", otherMa.Hit())
							o.Stdoutln("===")
							o.Stdoutln("GetKey", missingMa.GetKey())
							o.Stdoutln("Get", missingMa.Get(d))
							o.Stdoutln("Provided", missingMa.Provided(d))
							o.Stdoutln("GetOrDefault", missingMa.GetOrDefault(d, &vType{7, 7.7}))
							o.Stdoutln("Hit", missingMa.Hit())
							return nil
						}, nil),
					),
					Args: []string{"-m", "three", "--m3", "eleven"},
					WantData: &command.Data{Values: map[string]interface{}{
						"m1": &vType{3, 3.3},
						"m3": (*vType)(nil),
					}},
					WantStdout: strings.Join([]string{
						"GetKey three",
						"Provided true",
						"Get &{3 3.3}",
						"GetOrDefault &{3 3.3}",
						"Hit true",
						"===",
						"GetKey ",
						"Get <nil>",
						"Provided false",
						"GetOrDefault &{7 7.7}",
						"Hit false",
						"===",
						"GetKey eleven",
						"Get <nil>",
						"Provided true",
						"GetOrDefault <nil>",
						"Hit false",
						"",
					}, "\n"),
				}
			}(),
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "-m"},
						{Value: "three"},
						{Value: "--m3"},
						{Value: "eleven"},
					},
				},
			},
		},
		// command.Usage tests
		{
			name: "works with single arg",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: SerialNodes(Arg[string]("SARG", "desc")),
				WantStdout: strings.Join([]string{
					"SARG",
					"",
					"Arguments:",
					"  SARG: desc",
					"",
				}, "\n"),
			},
		},
		// Panic tests
		{
			name: "forwards panic",
			etc: &commandtest.ExecuteTestCase{
				Node: SerialNodes(
					&ExecutorProcessor{func(o command.Output, d *command.Data) error {
						panic("oh no!")
					}},
				),
				WantPanic: "oh no!",
			},
		},
		// MutableProcessor tests
		{
			name: "Reference does not update underlying processor",
			etc: func() *commandtest.ExecuteTestCase {
				hi := PrintlnProcessor("hi")
				hello := PrintlnProcessor("hello")

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							hi = hello
							return nil
						}),
						hi,
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							fo := commandtest.NewOutput()
							if err := hi.Execute(nil, fo, nil, nil); err != nil {
								return err
							}
							fo.Close()
							d.Set("FINAL", fo.GetStdout())
							return nil
						}),
					),
					WantStdout: "hi\n",
					WantData: &command.Data{Values: map[string]interface{}{
						"FINAL": "hello\n",
					}},
				}
			}(),
		},
		{
			name: "MutableProcessor DOES update underlying processor",
			etc: func() *commandtest.ExecuteTestCase {
				hi := NewMutableProcessor[command.Processor](PrintlnProcessor("hi"))
				hello := PrintlnProcessor("hello")

				return &commandtest.ExecuteTestCase{
					Node: SerialNodes(
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							hi.Processor = &hello
							return nil
						}),
						hi,
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							fo := commandtest.NewOutput()
							if err := hi.Execute(nil, fo, nil, nil); err != nil {
								return err
							}
							fo.Close()
							d.Set("FINAL", fo.GetStdout())
							return nil
						}),
					),
					WantStdout: "hello\n",
					WantData: &command.Data{Values: map[string]interface{}{
						"FINAL": "hello\n",
					}},
				}
			}(),
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			stubs.StubGetwd(t, test.osGetwd, test.osGetwdErr)

			if test.etc == nil {
				test.etc = &commandtest.ExecuteTestCase{}
			}
			if test.etc.OS == nil {
				test.etc.OS = fos
			}
			if test.ietc == nil {
				test.ietc = &spycommandtest.ExecuteTestCase{}
			}
			executeTest(t, test.etc, test.ietc)
		})
	}
}

func abc() command.Node {
	return &BranchNode{
		Branches: map[string]command.Node{
			"t": ShortcutNode("TEST_SHORTCUT", nil,
				CacheNode("TEST_CACHE", nil, SerialNodes(
					&tt{},
					Arg[string]("PATH", testDesc, SimpleCompleter[string]("clh111", "abcd111")),
					Arg[string]("TARGET", testDesc, SimpleCompleter[string]("clh222", "abcd222")),
					Arg[string]("FUNC", testDesc, SimpleCompleter[string]("clh333", "abcd333")),
				))),
		},
		DefaultCompletion: true,
	}
}

type tt struct{}

func (t *tt) Usage(*command.Input, *command.Data, *command.Usage) error { return nil }
func (t *tt) Execute(input *command.Input, output command.Output, data *command.Data, e *command.ExecuteData) error {
	t.do(input, data)
	return nil
}

func (t *tt) do(input *command.Input, data *command.Data) {
	if s, ok := input.Peek(); ok && strings.Contains(s, ":") {
		if ss := strings.Split(s, ":"); len(ss) == 2 {
			input.Pop(data)
			input.PushFront(ss...)
		}
	}
}

func (t *tt) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	t.do(input, data)
	return nil, nil
}

func TestComplete(t *testing.T) {
	breakerFlagProcessor := FlagProcessor(
		Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
		ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
		BoolFlag("good", 'g', testDesc),
	)
	for _, test := range []struct {
		name           string
		ctc            *commandtest.CompleteTestCase
		ictc           *spycommandtest.CompleteTestCase
		filepathAbs    string
		filepathAbsErr error
		osGetwd        string
		osGetwdErr     error
	}{
		{
			name: "stuff",
			ctc: &commandtest.CompleteTestCase{
				Node: abc(),
				Args: "cmd t clh:abc",
				Want: &command.Autocompletion{
					Suggestions: []string{"abcd222"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"PATH":   "clh",
					"TARGET": "abc",
				}},
			},
		},
		// Basic tests
		{
			name: "empty graph",
			ctc: &commandtest.CompleteTestCase{
				Node:    &SimpleNode{},
				WantErr: fmt.Errorf("Unprocessed extra args: []"),
			},
			ictc: &spycommandtest.CompleteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
			},
		},
		{
			name: "returns suggestions of first node if empty",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("un", "deux", "trois")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"deux", "trois", "un"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "",
				}},
			},
		},
		{
			name: "returns suggestions of first node if up to first arg",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
				),
				Args: "cmd t",
				Want: &command.Autocompletion{
					Suggestions: []string{"three", "two"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "t",
				}},
			},
		},
		{
			name: "returns suggestions of middle node if that's where we're at",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three ",
				Want: &command.Autocompletion{
					Suggestions: []string{"dos", "uno"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{""},
				}},
			},
		},
		{
			name: "returns suggestions of middle node if partial",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three d",
				Want: &command.Autocompletion{
					Suggestions: []string{"dos"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"d"},
				}},
			},
		},
		{
			name: "returns suggestions in list",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three dos ",
				Want: &command.Autocompletion{
					Suggestions: []string{"dos", "uno"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"dos", ""},
				}},
			},
		},
		{
			name: "returns suggestions for last arg",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three uno dos ",
				Want: &command.Autocompletion{
					Suggestions: []string{"1", "2"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"uno", "dos"},
				}},
			},
		},
		{
			name: "returns nothing if iterate through all nodes",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, SimpleCompleter[string]("one", "two", "three")),
					ListArg[string]("sl", testDesc, 0, 2, SimpleCompleter[[]string]("uno", "dos")),
					OptionalArg[int]("i", testDesc, SimpleCompleter[int]("2", "1")),
				),
				Args: "cmd three uno dos 1 what now",
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "three",
					"sl": []string{"uno", "dos"},
					"i":  1,
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [what now]"),
			},
			ictc: &spycommandtest.CompleteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
			},
		},
		{
			name: "works if empty and list starts",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, SimpleCompleter[[]string]("uno", "dos")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"dos", "uno"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{""},
				}},
			},
		},
		{
			name: "only returns suggestions matching prefix",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, SimpleCompleter[[]string]("zzz-1", "zzz-2", "yyy-3", "zzz-4")),
				),
				Args: "cmd zz",
				Want: &command.Autocompletion{
					Suggestions: []string{"zzz-1", "zzz-2", "zzz-4"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"sl": []string{"zz"},
				}},
			},
		},
		{
			name: "if fail to convert arg, then don't complete",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s1", testDesc, SimpleCompleter[string]("one", "two", "three")),
					Arg[int]("i", testDesc),
					Arg[string]("s2", testDesc, SimpleCompleter[string]("abc", "alpha")),
				),
				Args:    "cmd three two a",
				WantErr: fmt.Errorf(`strconv.Atoi: parsing "two": invalid syntax`),
				WantData: &command.Data{Values: map[string]interface{}{
					"s1": "three",
				}},
			},
		},
		// Ensure completion iteration stops if necessary.
		{
			name: "stop iterating if a completion returns nil",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("PATH", "dd", SimpleCompleter[string]()),
					ListArg[string]("SUB_PATH", "stc", 0, command.UnboundedList, SimpleCompleter[[]string]("un", "deux", "trois")),
				),
				Args: "cmd p",
				WantData: &command.Data{Values: map[string]interface{}{
					"PATH": "p",
				}},
			},
		},
		{
			name: "stop iterating if a completion returns an error",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("PATH", "dd", CompleterFromFunc(func(string, *command.Data) (*command.Completion, error) {
						return nil, fmt.Errorf("ruh-roh")
					})),
					ListArg[string]("SUB_PATH", "stc", 0, command.UnboundedList, SimpleCompleter[[]string]("un", "deux", "trois")),
				),
				Args:    "cmd p",
				WantErr: fmt.Errorf("ruh-roh"),
				WantData: &command.Data{Values: map[string]interface{}{
					"PATH": "p",
				}},
			},
		},
		{
			name: "fails if edge returns an error",
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Edge: &errorEdge{fmt.Errorf("whoops")},
				},
				Args:    "cmd p",
				WantErr: fmt.Errorf("whoops"),
			},
		},
		// Flag completion
		{
			name: "bool flag gets set if not last one",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd -g ",
				Want: &command.Autocompletion{
					Suggestions: []string{"1", "2"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"good": true,
					},
				},
			},
		},
		{
			name: "arg flag gets set if not last one",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --greeting howdy ",
				Want: &command.Autocompletion{
					Suggestions: []string{"1", "2"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"greeting": "howdy",
					},
				},
			},
		},
		{
			name: "list arg flag gets set if not last one",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --names alice bob charlie ",
				Want: &command.Autocompletion{
					Suggestions: []string{"1", "2"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"names": []string{"alice", "bob", "charlie"},
					},
				},
			},
		},
		{
			name: "multiple flags get set if not last one",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd -n alice bob charlie --good -h howdy ",
				Want: &command.Autocompletion{
					Suggestions: []string{"1", "2"},
				},
				WantData: &command.Data{
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd -",
				Want: &command.Autocompletion{
					Suggestions: []string{"--good", "--greeting", "--names"},
				},
			},
		},
		{
			name: "flag name gets completed if double hyphen at end",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --",
				Want: &command.Autocompletion{
					Suggestions: []string{"--good", "--greeting", "--names"},
				},
			},
		},
		{
			name: "flag name gets completed if it's the only arg",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -",
				Want: &command.Autocompletion{
					Suggestions: []string{"--good", "--greeting", "--names"},
				},
			},
		},
		{
			name: "partial flag name gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --gr",
				Want: &command.Autocompletion{
					Suggestions: []string{"--greeting"},
				},
			},
		},
		{
			name: "full flag name gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd --names",
				Want: &command.Autocompletion{
					Suggestions: []string{"--names"},
				},
			},
		},
		// Flag value completions
		{
			name: "completes for single flag",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 --greeting h",
				Want: &command.Autocompletion{
					Suggestions: []string{"hey", "hi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"greeting": "h",
				}},
			},
		},
		{
			name: "completes for single short flag",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h he",
				Want: &command.Autocompletion{
					Suggestions: []string{"hey"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"greeting": "he",
				}},
			},
		},
		{
			name: "completes for list flag",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ",
				Want: &command.Autocompletion{
					Suggestions: []string{"johnny", "ralph", "renee"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"greeting": "hey",
					"names":    []string{""},
				}},
			},
		},
		{
			name: "completes distinct secondary for list flag",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleDistinctCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ralph ",
				Want: &command.Autocompletion{
					Suggestions: []string{"johnny", "renee"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"greeting": "hey",
					"names":    []string{"ralph", ""},
				}},
			},
		},
		{
			name: "completes last flag",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleDistinctCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
						Flag[float64]("float", 'f', testDesc, SimpleCompleter[float64]("1.23", "12.3", "123.4")),
					),
					Arg[int]("i", testDesc, SimpleCompleter[int]("1", "2")),
				),
				Args: "cmd 1 -h hey other --names ralph renee johnny -f ",
				Want: &command.Autocompletion{
					Suggestions: []string{"1.23", "12.3", "123.4"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"greeting": "hey",
					"names":    []string{"ralph", "renee", "johnny"},
				}},
			},
		},
		{
			name: "completes arg if flag arg isn't at the end",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						Flag[string]("greeting", 'h', testDesc, SimpleCompleter[string]("hey", "hi")),
						ListFlag[string]("names", 'n', testDesc, 1, 2, SimpleDistinctCompleter[[]string]("ralph", "johnny", "renee")),
						BoolFlag("good", 'g', testDesc),
					),
					ListArg[string]("i", testDesc, 1, 2, SimpleCompleter[[]string]("hey", "ooo")),
				),
				Args: "cmd 1 -h hello bravo --names ralph renee johnny ",
				Want: &command.Autocompletion{
					Suggestions: []string{"hey", "ooo"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"i":        []string{"1", "bravo", ""},
					"greeting": "hello",
					"names":    []string{"ralph", "renee", "johnny"},
				}},
			},
		},
		// Multi-flag tests
		{
			name: "Multi-flags don't get completed",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd -qwer",
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
				),
			},
		},
		{
			name: "Args after multi-flags get completed",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd -qwer ",
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def", "ghi")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd -qwertyuiop ",
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def", "ghi")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd -qwz ",
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						Flag[string]("zf", 'z', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def", "ghi")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"quick": true,
					"where": true,
					"s":     "",
				}},
			},
		},
		// Duplicate flag tests
		{
			name: "Repeated flag still gets completed",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd -z firstZ -z ",
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						Flag[string]("zf", 'z', testDesc, SimpleCompleter[string]("zyx", "wvu", "tsr")),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"tsr", "wvu", "zyx"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"zf": "",
				}},
			},
		},
		{
			name: "Repeated flag still gets completed even if other repetition in multi-flags",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd --quick -qwrqw --where -z firstZ -z ",
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						Flag[string]("zf", 'z', testDesc, SimpleCompleter[string]("zyx", "wvu", "tsr")),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"tsr", "wvu", "zyx"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"quick": true,
					"where": true,
					"run":   true,
					"zf":    "",
				}},
			},
		},
		{
			name: "Don't suggest already seen flag names",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd -z firstZ --everyone --ilf heyo --run -",
				Node: SerialNodes(
					FlagProcessor(
						BoolFlag("everyone", 'e', testDesc),
						BoolFlag("quick", 'q', testDesc),
						BoolFlag("run", 'r', testDesc),
						BoolFlag("to", 't', testDesc),
						Flag[string]("zf", 'z', testDesc, SimpleCompleter[string]("zyx", "wvu", "tsr")),
						ItemizedListFlag[string]("ilf", 'i', testDesc),
						BoolFlag("where", 'w', testDesc),
					),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{
						// ilf still gets completed because it allows multiple.
						"--ilf",
						"--quick",
						"--to",
						"--where",
					},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"everyone": true,
					"run":      true,
					"zf":       "firstZ",
				}},
			},
		},
		// OptionalFlag tests
		{
			name: "OptionalFlag gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag[string]("of", 'o', testDesc, "dfltValue", SimpleDistinctCompleter[string]("abc", "def", "ghi")),
					),
				),
				Args: "cmd --of",
				Want: &command.Autocompletion{
					Suggestions: []string{"--of"},
				},
			},
		},
		{
			name: "OptionalFlag arg gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag[string]("of", 'o', testDesc, "dfltValue", SimpleDistinctCompleter[string]("abc", "def", "ghi")),
					),
				),
				Args: "cmd --of ",
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"of": "",
				}},
			},
		},
		{
			name: "Eats partial flag completion",
			// Eats partial flag completion because there's no great way
			// to know if the value is for this flag or not.
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag[string]("of", 'o', testDesc, "dfltValue", SimpleDistinctCompleter[string]("abc", "def", "ghi")),
						BoolFlag("bf", 'b', testDesc),
					),
				),
				Args: "cmd --of -",
				Want: &command.Autocompletion{
					Suggestions: []string{
						"--bf",
					},
				},
			},
		},
		{
			name: "Eats optional flag completion",
			// Eats partial flag completion because there's no great way
			// to know if the value is for this flag or not.
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						OptionalFlag[string]("of", 'o', testDesc, "dfltValue", SimpleDistinctCompleter[string]("abc", "def", "ghi")),
						BoolFlag("bf", 'b', testDesc),
					),
				),
				Args: "cmd --of provided -",
				WantData: &command.Data{Values: map[string]interface{}{
					"of": "provided",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"--bf",
					},
				},
			},
		},
		// ItemizedListFlag tests
		{
			name: "Itemized list flag gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ItemizedListFlag[string]("ilf", 'i', testDesc, SimpleDistinctCompleter[[]string]("abc", "def", "ghi")),
					),
				),
				Args: "cmd --ilf",
				Want: &command.Autocompletion{
					Suggestions: []string{"--ilf"},
				},
			},
		},
		{
			name: "Completes itemized list flag value",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ItemizedListFlag[string]("ilf", 'i', testDesc, SimpleDistinctCompleter[[]string]("abc", "def", "ghi")),
					),
				),
				Args: "cmd --ilf ",
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"ilf": []string{""},
				}},
			},
		},
		{
			name: "Completes later itemized list flag value",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ItemizedListFlag[string]("ilf", 'i', testDesc, SimpleDistinctCompleter[[]string]("abc", "def", "ghi")),
					),
				),
				Args: "cmd --ilf un -i d",
				WantData: &command.Data{Values: map[string]interface{}{
					"ilf": []string{"un", "d"},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"def"},
				},
			},
		},
		{
			name: "Completes distinct itemized list flag value",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ItemizedListFlag[string]("ilf", 'i', testDesc, SimpleDistinctCompleter[[]string]("abc", "def", "ghi")),
					),
				),
				Args: "cmd --ilf def -i ",
				WantData: &command.Data{Values: map[string]interface{}{
					"ilf": []string{"def", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "ghi"},
				},
			},
		},
		// Flag completion with list breaker
		{
			name: "completes flag argument when flag processor's list breaker is provided as arg option",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("lst", testDesc, 0, command.UnboundedList, breakerFlagProcessor.ListBreaker()),
					breakerFlagProcessor,
				),
				Args: "cmd v1 v2 other --names ",
				Want: &command.Autocompletion{
					Suggestions: []string{"johnny", "ralph", "renee"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"lst":   []string{"v1", "v2", "other"},
					"names": []string{""},
				}},
			},
		},
		{
			name: "second flag is recognized and completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("lst", testDesc, 0, command.UnboundedList, breakerFlagProcessor.ListBreaker()),
					breakerFlagProcessor,
				),
				Args: "cmd v1 v2 other --names un --greeting ",
				Want: &command.Autocompletion{
					Suggestions: []string{"hey", "hi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"lst":      []string{"v1", "v2", "other"},
					"names":    []string{"un"},
					"greeting": "",
				}},
			},
		},
		// DeferredCompletion tests
		{
			name: "DeferredCompletion handles nil graph",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("lf", 'f', testDesc, 0, command.UnboundedList, DeferredCompleter(nil, SimpleCompleter[[]string]("abc", "def"))),
					),
				),
				Args: "cmd --lf ab ",
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"lf": []string{"ab", ""},
				}},
			},
		},
		{
			name: "DeferredCompletion handles error",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("lf", 'f', testDesc, 0, command.UnboundedList, DeferredCompleter(nil, CompleterFromFunc(func([]string, *command.Data) (*command.Completion, error) {
							return &command.Completion{Suggestions: []string{"abc", "def"}}, fmt.Errorf("oh well")
						}))),
					),
				),
				Args:    "cmd --lf ab ",
				WantErr: fmt.Errorf("oh well"),
				WantData: &command.Data{Values: map[string]interface{}{
					"lf": []string{"ab", ""},
				}},
			},
		},
		{
			name: "DeferredCompletion executes sub graph",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("lf", 'f', testDesc, 0, command.UnboundedList, DeferredCompleter(
							SerialNodes(
								ListArg[string]("la", testDesc, 3, command.UnboundedList, SimpleCompleter[[]string]("un", "deux")),
							),
							CompleterFromFunc(func([]string, *command.Data) (*command.Completion, error) {
								return &command.Completion{Suggestions: []string{"abc", "def"}}, fmt.Errorf("oh well")
							}))),
					),
				),
				Args:    "cmd v1 v2 other --lf ab ",
				WantErr: fmt.Errorf("oh well"),
				WantData: &command.Data{Values: map[string]interface{}{
					"lf": []string{"ab", ""},
					"la": []string{"v1", "v2", "other"},
				}},
			},
		},
		{
			name: "DeferredCompletion returns error from sub graph",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[string]("lf", 'f', testDesc, 0, command.UnboundedList, DeferredCompleter(
							SerialNodes(
								ListArg[string]("la", testDesc, 4, command.UnboundedList, SimpleCompleter[[]string]("un", "deux")),
							),
							CompleterFromFunc(func([]string, *command.Data) (*command.Completion, error) {
								return &command.Completion{Suggestions: []string{"abc", "def"}}, fmt.Errorf("oh well")
							}))),
					),
				),
				Args:    "cmd v1 v2 other --lf ab ",
				WantErr: fmt.Errorf(`failed to execute DeferredCompletion graph: Argument "la" requires at least 4 arguments, got 3`),
				WantData: &command.Data{Values: map[string]interface{}{
					"lf": []string{"ab", ""},
					"la": []string{"v1", "v2", "other"},
				}},
			},
		},
		// Transformer arg tests.
		{
			name: "handles nil option",
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc),
				},
				Args: "cmd abc",
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "abc",
				}},
			},
		},
		{
			name: "list handles nil option",
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, 2),
				},
				Args: "cmd abc",
				WantData: &command.Data{Values: map[string]interface{}{
					"slArg": []string{"abc"},
				}},
			},
		},
		{
			name: "transformer doesn't transform value during completion",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(Arg[string]("strArg", testDesc,
					&Transformer[string]{F: func(string, *command.Data) (string, error) {
						return "newStuff", nil
					}})),
				Args: "cmd abc",
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": "abc",
				}},
			},
		},
		{
			name:        "FileTransformer doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, FileTransformer()),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": filepath.Join("relative", "path.txt"),
				}},
			},
		},
		{
			name:        "FileTransformer for list doesn't transform",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("strArg", testDesc, 1, 2, TransformerList(FileTransformer())),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": []string{filepath.Join("relative", "path.txt")},
				}},
			},
		},
		{
			name:           "handles transform error",
			filepathAbsErr: fmt.Errorf("bad news bears"),
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: Arg[string]("strArg", testDesc, FileTransformer()),
				},
				Args: fmt.Sprintf("cmd %s", filepath.Join("relative", "path.txt")),
				WantData: &command.Data{Values: map[string]interface{}{
					"strArg": filepath.Join("relative", "path.txt"),
				}},
			},
		},
		{
			name:        "transformer list doesn't transforms values during completion",
			filepathAbs: filepath.Join("abso", "lutely"),
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, 2, &Transformer[[]string]{F: func(sl []string, d *command.Data) ([]string, error) {
						var r []string
						for _, s := range sl {
							r = append(r, fmt.Sprintf("_%s_", s))
						}
						return r, nil
					}}),
				},
				Args: "cmd uno dos",
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("slArg", testDesc, 1, 1, &Transformer[[]string]{F: func(sl []string, d *command.Data) ([]string, error) {
						var r []string
						for _, s := range sl {
							r = append(r, fmt.Sprintf("_%s_", s))
						}
						return r, nil
					}}),
					Arg[string]("sArg", testDesc, &Transformer[string]{F: func(s string, d *command.Data) (string, error) {
						return s + "!", nil
					}}),
				),
				Args: "cmd uno dos t",
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("slArg", testDesc, 1, 1, &Transformer[[]string]{F: func(sl []string, d *command.Data) ([]string, error) {
						var r []string
						for _, s := range sl {
							r = append(r, fmt.Sprintf("_%s_", s))
						}
						return r, nil
					}}),
					Arg[string]("sArg", testDesc, &Transformer[string]{F: func(s string, d *command.Data) (string, error) {
						return "oh", fmt.Errorf("Nooooooo")
					}}),
					Arg[string]("sArg2", testDesc, &Transformer[string]{F: func(s string, d *command.Data) (string, error) {
						return "oh yea", fmt.Errorf("nope")
					}}),
				),
				Args: "cmd uno dos tres q",
				WantData: &command.Data{Values: map[string]interface{}{
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
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, 2, TransformerList(FileTransformer())),
				},
				Args: fmt.Sprintf("cmd %s %s", filepath.Join("relative", "path.txt"), filepath.Join("other.txt")),
				WantData: &command.Data{Values: map[string]interface{}{
					"slArg": []string{
						filepath.Join("relative", "path.txt"),
						filepath.Join("other.txt"),
					},
				}},
			},
		},
		{
			name: "handles list transformer of incorrect type",
			ctc: &commandtest.CompleteTestCase{
				Node: &SimpleNode{
					Processor: ListArg[string]("slArg", testDesc, 1, 2, TransformerList(FileTransformer())),
				},
				Args: "cmd 123",
				WantData: &command.Data{Values: map[string]interface{}{
					"slArg": []string{"123"},
				}},
			},
		},
		// FileArgument
		{
			name:        "FileArgument includes a vanilla FileCompleter",
			filepathAbs: filepath.Join("."),
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FileArgument("fn", testDesc),
				),
				Args: "cmd ",
				WantData: &command.Data{Values: map[string]interface{}{
					"fn": "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						filepath.FromSlash(".dot-dir/"),
						filepath.FromSlash("_testdata_symlink/"),
						"arg.go",
						"autocomplete.go",
						"branch_node.go",
						"branch_node_test.go",
						"cache.go",
						"cache_test.go",
						filepath.FromSlash("co2test/"),
						"completer.go",
						"completer_test.go",
						"conditional.go",
						filepath.FromSlash("cotest/"),
						"data_transformer.go",
						"debug.go",
						"description.go",
						"echo.go",
						"error.go",
						"execute.go",
						"execute_test.go",
						"executor.go",
						"fake.mod",
						"fake.sum",
						"file_functions.go",
						"file_functions.txt",
						"flag.go",
						"get_processor.go",
						"list_breaker.go",
						"map_arg.go",
						"menu.go",
						"mutable_processor.go",
						"node_repeater.go",
						"option.go",
						"osenv.go",
						"prompt.go",
						"runtime_caller.go",
						"runtime_caller_test.go",
						"serial_nodes.go",
						"setup.go",
						"shell_command_node.go",
						"shell_command_node_test.go",
						"shortcut.go",
						"shortcut_test.go",
						"simple_node.go",
						"simple_processor.go",
						"static_cli.go",
						"static_cli_test.go",
						filepath.FromSlash("testdata/"),
						"transformer.go",
						"usage_test.go",
						"validator.go",
						"working_directory.go",
						" ",
					},
				},
			},
		},
		{
			name:        "FileArgument uses provided FileCompleter option",
			filepathAbs: filepath.Join("."),
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FileArgument("fn", testDesc, &FileCompleter[string]{
						FileTypes: []string{".sum", ".mod"},
					}),
				),
				Args: "cmd ",
				WantData: &command.Data{Values: map[string]interface{}{
					"fn": "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						filepath.FromSlash(".dot-dir/"),
						filepath.FromSlash("_testdata_symlink/"),
						filepath.FromSlash("co2test/"),
						filepath.FromSlash("cotest/"),
						"fake.mod",
						"fake.sum",
						filepath.FromSlash("testdata/"),
						" ",
					},
				},
			},
		},
		{
			name:        "FileCompleter works with absolute path",
			filepathAbs: filepath.Join(),
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("fn", testDesc, CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
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
				WantData: &command.Data{Values: map[string]interface{}{
					"fn": "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						".surprise",
						filepath.FromSlash("cases/"),
						filepath.FromSlash("dir1/"),
						filepath.FromSlash("dir2/"),
						filepath.FromSlash("dir3/"),
						filepath.FromSlash("dir4/"),
						"four.txt",
						"METADATA",
						filepath.FromSlash("metadata_/"),
						filepath.FromSlash("moreCases/"),
						"one.txt",
						"three.txt",
						"two.txt",
						" ",
					},
				},
			},
		},
		// BranchNode completion tests.
		{
			name: "completes branch name options",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default: SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts"))),
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"a", "alpha", "bravo"},
				},
			},
		},
		{
			name: "completes default node options",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default:           SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts"))),
					DefaultCompletion: true,
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"command", "default", "opts"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"default": []string{""},
				}},
			},
		},
		{
			name: "no completions if default node is nil",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					DefaultCompletion: true,
				},
				WantErr: fmt.Errorf("Unprocessed extra args: []"),
			},
			ictc: &spycommandtest.CompleteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
			},
		},
		{
			name: "doesn't complete branch options if complete arg is false",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default:           SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts"))),
					DefaultCompletion: true,
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"command", "default", "opts"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"default": []string{""},
				}},
			},
		},
		{
			name: "completes for specific branch",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default:           SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts"))),
					DefaultCompletion: true,
				},
				Args: "cmd alpha ",
				Want: &command.Autocompletion{
					Suggestions: []string{"other", "stuff"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"hello": "",
				}},
			},
		},
		{
			name: "branch node doesn't complete if no default and no branch match",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
				},
				Args:    "cmd some thing else",
				WantErr: fmt.Errorf("Branching argument must be one of [a alpha bravo]"),
			},
			ictc: &spycommandtest.CompleteTestCase{
				WantIsBranchingError: true,
				WantIsUsageError:     true,
			},
		},
		{
			name: "branch node returns default node error if branch completion is false",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default: SerialNodes(SimpleProcessor(nil, func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return nil, fmt.Errorf("bad news bears")
					})),
					DefaultCompletion: true,
				},
				Args:    "cmd ",
				WantErr: fmt.Errorf("bad news bears"),
			},
		},
		{
			name: "branch node returns only branch completions",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default: SerialNodes(SimpleProcessor(nil, func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return nil, fmt.Errorf("bad news bears")
					})),
				},
				Args: "cmd ",
				Want: &command.Autocompletion{
					Suggestions: []string{"a", "alpha", "bravo"},
				},
			},
		},
		{
			name: "completes branch options with partial completion",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default:           SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts", "ahhhh", "alright"))),
					DefaultCompletion: true,
				},
				Args: "cmd a",
				Want: &command.Autocompletion{
					Suggestions: []string{"ahhhh", "alright"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"default": []string{"a"},
				}},
			},
		},
		{
			name: "completes default options",
			ctc: &commandtest.CompleteTestCase{
				Node: &BranchNode{
					Branches: map[string]command.Node{
						"a":     &SimpleNode{},
						"alpha": SerialNodes(OptionalArg[string]("hello", testDesc, SimpleCompleter[string]("other", "stuff"))),
						"bravo": &SimpleNode{},
					},
					Default: SerialNodes(ListArg[string]("default", testDesc, 1, 3, SimpleCompleter[[]string]("default", "command", "opts"))),
				},
				Args: "cmd something ",
				WantData: &command.Data{Values: map[string]interface{}{
					"default": []string{"something", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"command", "default", "opts"},
				},
			},
		},
		{
			name: "BranchNode only completes first name of branch",
			ctc: &commandtest.CompleteTestCase{
				Node: branchSynNode(),
				Args: "cmd ",
				Want: &command.Autocompletion{
					Suggestions: []string{"hello"},
				},
			},
		},
		// SuperSimpleProcessor tests
		{
			name: "sets data with SuperSimpleProcessor",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd",
				Node: SerialNodes(SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
					d.Set("key", "value")
					return nil
				}),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"key": "value",
						"s":   "",
					},
				},
			},
		},
		{
			name: "returns error from SuperSimpleProcessor",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd",
				Node: SerialNodes(SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
					d.Set("key", "value")
					return fmt.Errorf("ugh")
				}),
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def")),
				),
				WantErr: fmt.Errorf("ugh"),
				WantData: &command.Data{
					Values: map[string]interface{}{
						"key": "value",
					},
				},
			},
		},
		// PrintlnProcessor tests
		{
			name: "PrintlnProcessor does not print output in completion context",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					PrintlnProcessor("hello there"),
					Arg[string]("s", testDesc, SimpleCompleter[string]("okay", "then")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{
						"okay",
						"then",
					},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"s": "",
					},
				},
			},
		},
		// Getwd tests
		{
			name:    "sets data with Getwd",
			osGetwd: "some/dir",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd",
				Node: SerialNodes(
					Getwd,
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def")),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						GetwdKey: "some/dir",
						"s":      "",
					},
				},
			},
		},
		{
			name:       "returns error from Getwd",
			osGetwdErr: fmt.Errorf("whoops"),
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd",
				Node: SerialNodes(
					Getwd,
					Arg[string]("s", testDesc, SimpleCompleter[string]("abc", "def")),
				),
				WantErr: fmt.Errorf("failed to get current directory: whoops"),
			},
		},
		// MenuArg tests.
		{
			name: "MenuArg completes choices",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(MenuArg("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd ",
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"sm": "",
				}},
			},
		},
		{
			name: "MenuArg completes partial",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(MenuArg("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd g",
				Want: &command.Autocompletion{
					Suggestions: []string{"ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"sm": "g",
				}},
			},
		},
		{
			name: "MenuArg completes none if no match",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(MenuArg("sm", "desc", "abc", "def", "ghi")),
				Args: "cmd j",
				WantData: &command.Data{Values: map[string]interface{}{
					"sm": "j",
				}},
			},
		},
		{
			name: "MenuFlag completes choices",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: "cmd --sf ",
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"sf": "",
				}},
			},
		},
		{
			name: "MenuArg completes partial",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: "cmd -s g",
				Want: &command.Autocompletion{
					Suggestions: []string{"ghi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"sf": "g",
				}},
			},
		},
		{
			name: "MenuFlag completes none",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					FlagProcessor(
						MenuFlag("sf", 's', testDesc, "abc", "def", "ghi"),
					),
				),
				Args: "cmd -s j",
				WantData: &command.Data{Values: map[string]interface{}{
					"sf": "j",
				}},
			},
		},
		// Commands with different value types.
		{
			name: "int arg gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(Arg[int]("iArg", testDesc, SimpleCompleter[int]("12", "45", "456", "468", "7"))),
				Args: "cmd 4",
				Want: &command.Autocompletion{
					Suggestions: []string{"45", "456", "468"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 4,
				}},
			},
		},
		{
			name: "optional int arg gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(OptionalArg[int]("iArg", testDesc, SimpleCompleter[int]("12", "45", "456", "468", "7"))),
				Args: "cmd 4",
				Want: &command.Autocompletion{
					Suggestions: []string{"45", "456", "468"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": 4,
				}},
			},
		},
		{
			name: "int list arg gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[int]("iArg", testDesc, 2, 3, SimpleCompleter[[]int]("12", "45", "456", "468", "7"))),
				Args: "cmd 1 4",
				Want: &command.Autocompletion{
					Suggestions: []string{"45", "456", "468"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": []int{1, 4},
				}},
			},
		},
		{
			name: "int list arg gets completed if previous one was invalid",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[int]("iArg", testDesc, 2, 3, SimpleCompleter[[]int]("12", "45", "456", "468", "7"))),
				Args: "cmd one 4",
				Want: &command.Autocompletion{
					Suggestions: []string{"45", "456", "468"},
				},
			},
		},
		{
			name: "int list arg optional args get completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[int]("iArg", testDesc, 2, 3, SimpleCompleter[[]int]("12", "45", "456", "468", "7"))),
				Args: "cmd 1 2 3 4",
				Want: &command.Autocompletion{
					Suggestions: []string{"45", "456", "468"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"iArg": []int{1, 2, 3, 4},
				}},
			},
		},
		{
			name: "float arg gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(Arg[float64]("fArg", testDesc, SimpleCompleter[float64]("12", "4.5", "45.6", "468", "7"))),
				Args: "cmd 4",
				Want: &command.Autocompletion{
					Suggestions: []string{"4.5", "45.6", "468"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"fArg": 4.0,
				}},
			},
		},
		{
			name: "float list arg gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(ListArg[float64]("fArg", testDesc, 1, 2, SimpleCompleter[[]float64]("12", "4.5", "45.6", "468", "7"))),
				Want: &command.Autocompletion{
					Suggestions: []string{"12", "4.5", "45.6", "468", "7"},
				},
				WantData: &command.Data{Values: map[string]interface{}{}},
			},
		},
		{
			name: "bool arg gets completed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(BoolArg("bArg", testDesc)),
				Want: &command.Autocompletion{
					Suggestions: []string{"0", "1", "F", "FALSE", "False", "T", "TRUE", "True", "f", "false", "t", "true"},
				},
				WantData: &command.Data{Values: map[string]interface{}{}},
			},
		},
		// NodeRepeater
		{
			name: "NodeRepeater completes first node",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, 2)),
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "bravo", "brown", "charlie"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys": []string{""},
				}},
			},
		},
		{
			name: "NodeRepeater completes first node partial",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, 2)),
				Args: "cmd b",
				Want: &command.Autocompletion{
					Suggestions: []string{"bravo", "brown"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys": []string{"b"},
				}},
			},
		},
		{
			name: "NodeRepeater completes second node",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, 2)),
				Args: "cmd brown ",
				Want: &command.Autocompletion{
					Suggestions: []string{"1", "121", "1213121"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys": []string{"brown"},
				}},
			},
		},
		{
			name: "NodeRepeater completes second node partial",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(1, 2)),
				Args: "cmd brown 12",
				Want: &command.Autocompletion{
					Suggestions: []string{"121", "1213121"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"brown"},
					"values": []int{12},
				}},
			},
		},
		{
			name: "NodeRepeater completes second required iteration",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 0)),
				Args: "cmd brown 12 c",
				Want: &command.Autocompletion{
					Suggestions: []string{"charlie"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "c"},
					"values": []int{12},
				}},
			},
		},
		{
			name: "NodeRepeater completes optional iteration",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 1)),
				Args: "cmd brown 12 charlie 21 alpha 1",
				Want: &command.Autocompletion{
					Suggestions: []string{"1", "121", "1213121"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha"},
					"values": []int{12, 21, 1},
				}},
			},
		},
		{
			name: "NodeRepeater completes unbounded optional iteration",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, command.UnboundedList)),
				Args: "cmd brown 12 charlie 21 alpha 100 delta 98 b",
				Want: &command.Autocompletion{
					Suggestions: []string{"bravo", "brown"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha", "delta", "b"},
					"values": []int{12, 21, 100, 98},
				}},
			},
		},
		{
			name: "NodeRepeater doesn't complete beyond repeated iterations",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 1)),
				Args: "cmd brown 12 charlie 21 alpha 100 b",
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha"},
					"values": []int{12, 21, 100},
				}},
				WantErr: fmt.Errorf("Unprocessed extra args: [b]"),
			},
			ictc: &spycommandtest.CompleteTestCase{
				WantIsExtraArgsError: true,
				WantIsUsageError:     true,
			},
		},
		{
			name: "NodeRepeater works if fully processed",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(sampleRepeaterProcessor(2, 1), Arg[string]("S", testDesc, SimpleCompleter[string]("un", "deux", "trois"))),
				Args: "cmd brown 12 charlie 21 alpha 100",
				WantData: &command.Data{Values: map[string]interface{}{
					"keys":   []string{"brown", "charlie", "alpha"},
					"values": []int{12, 21, 100},
				}},
			},
		},
		// ListBreaker tests
		{
			name: "Suggests things after broken list",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, ListUntilSymbol("ghi"), SimpleCompleter[[]string]("un", "deux", "trois")),
					ListArg[string]("SL2", testDesc, 0, command.UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: &command.Autocompletion{
					Suggestions: []string{"one", "three", "two"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{"ghi", ""},
				}},
			},
		},
		{
			name: "Suggests things after broken list with discard",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, func() *ListBreaker[[]string] {
						li := ListUntilSymbol("ghi")
						li.Discard = true
						return li
					}(), SimpleCompleter[[]string]("un", "deux", "trois")),
					ListArg[string]("SL2", testDesc, 0, command.UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: &command.Autocompletion{
					Suggestions: []string{"one", "three", "two"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL":  []string{"abc", "def"},
					"SL2": []string{""},
				}},
			},
		},
		{
			name: "Suggests things before list is broken",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					ListArg[string]("SL", testDesc, 1, command.UnboundedList, ListUntilSymbol("ghi"), SimpleCompleter[[]string]("un", "deux", "trois", "uno")),
					ListArg[string]("SL2", testDesc, 0, command.UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def un",
				Want: &command.Autocompletion{
					Suggestions: []string{"un", "uno"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"SL": []string{"abc", "def", "un"},
				}},
			},
		},
		// StringListListProcessor
		{
			name: "StringListListProcessor works if no breakers",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, command.UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def ghi ",
				Want: &command.Autocompletion{
					Suggestions: []string{"one", "three", "two"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"SLL": [][]string{{"abc", "def", "ghi", ""}},
				}},
			},
		},
		{
			name: "StringListListProcessor works with breakers",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, command.UnboundedList, SimpleCompleter[[]string]("one", "two", "three")),
				),
				Args: "cmd abc def | ghi t",
				Want: &command.Autocompletion{
					Suggestions: []string{"three", "two"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"SLL": [][]string{{"abc", "def"}, {"ghi", "t"}},
				}},
			},
		},
		{
			name: "completes args after StringListListProcessor",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					StringListListProcessor("SLL", testDesc, "|", 1, 1, SimpleCompleter[[]string]("one", "two", "three")),
					Arg[string]("S", testDesc, SimpleCompleter[string]("un", "deux", "trois")),
				),
				Args: "cmd abc def | ghi | ",
				Want: &command.Autocompletion{
					Suggestions: []string{"deux", "trois", "un"},
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						"SLL": [][]string{{"abc", "def"}, {"ghi"}},
						"S":   "",
					},
				},
			},
		},
		// ShellCommandNode
		{
			name: "ShellCommandNode runs in command.Completion context",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					&ShellCommand[string]{ArgName: "b", CommandName: "echo", Args: []string{"haha"}},
					Arg[string]("s", testDesc),
				),
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{"hehe"},
				}},
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"haha"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					"b": "hehe",
					"s": "",
				}},
			},
		},
		{
			name: "ShellCommandNode fails in command.Completion context",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					&ShellCommand[string]{ArgName: "b", CommandName: "echo", Args: []string{"haha"}},
					Arg[string]("s", testDesc),
				),
				RunResponses: []*commandtest.FakeRun{{
					Err: fmt.Errorf("argh"),
				}},
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"haha"},
				}},
				WantErr: fmt.Errorf("failed to execute shell command: argh"),
			},
		},
		{
			name: "ShellCommandNode does not run in command.Completion context when option provided",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					&ShellCommand[string]{ArgName: "b", CommandName: "echo", Args: []string{"haha"}, DontRunOnComplete: true},
					Arg[string]("s", testDesc),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"s": "",
				}},
			},
		},
		// ShellCommandCompleter
		{
			name: "ShellCommandCompleter doesn't complete if shell failure",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, ShellCommandCompleter[string]("echo", "abc", "def", "ghi")),
				),
				RunResponses: []*commandtest.FakeRun{{
					Err: fmt.Errorf("oopsie"),
				}},
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"abc", "def", "ghi"},
				}},
				WantErr:  fmt.Errorf("failed to fetch autocomplete suggestions with shell command: failed to execute shell command: oopsie"),
				WantData: &command.Data{Values: map[string]interface{}{"s": ""}},
			},
		},
		{
			name: "ShellCommandCompleter completes even if wrong type returned (since just fetches string list)",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[int]("i", testDesc, ShellCommandCompleter[int]("echo", "abc", "def", "ghi")),
				),
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"abc",
						"def",
						"ghi",
					},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"abc", "def", "ghi"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{}},
			},
		},
		{
			name: "ShellCommandCompleter completes arg",
			ctc: &commandtest.CompleteTestCase{
				Node: SerialNodes(
					Arg[string]("s", testDesc, ShellCommandCompleter[string]("echo", "abc", "def", "ghi")),
				),
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"abc",
						"def",
						"ghi",
					},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"abc", "def", "ghi"},
				}},
				//WantErr: fmt.Errorf(`failed to fetch autocomplete suggestions with shell command: strconv.Atoi: parsing "abc def ghi": invalid syntax`),
				WantData: &command.Data{Values: map[string]interface{}{"s": ""}},
			},
		},
		{
			name: "ShellCommandCompleter completes arg with partial completion",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd d",
				Node: SerialNodes(
					Arg[string]("s", testDesc, ShellCommandCompleter[string]("echo", "abc", "def", "ghi")),
				),
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"def",
					},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"abc", "def", "ghi"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{"s": "d"}},
			},
		},
		{
			name: "ShellCommandCompleter completes arg with opts",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd abc ghi ",
				Node: SerialNodes(
					ListArg[string]("sl", testDesc, 1, 2, ShellCommandCompleterWithOpts[[]string](&command.Completion{Distinct: true}, "echo", "abc", "def", "ghi")),
				),
				RunResponses: []*commandtest.FakeRun{{
					Stdout: []string{
						"abc",
						"def",
						"ghi",
					},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{
						"def",
					},
				},
				WantRunContents: []*commandtest.RunContents{{
					Name: "echo",
					Args: []string{"abc", "def", "ghi"},
				}},
				WantData: &command.Data{Values: map[string]interface{}{"sl": []string{"abc", "ghi", ""}}},
			},
		},
		// If tests
		{
			name: "If runs if function returns true",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd alpha ",
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					If(
						Arg[string]("s2", testDesc, SimpleCompleter[string]("bravo", "charlie")),
						func(i *command.Input, d *command.Data) bool {
							return true
						},
					),
					Arg[string]("s3", testDesc, SimpleCompleter[string]("delta", "epsilon")),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "alpha",
					"s2": "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"bravo", "charlie"},
				},
			},
		},
		{
			name: "If does not run if function returns true",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd alpha ",
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					If(
						Arg[string]("s2", testDesc, SimpleCompleter[string]("bravo", "charlie")),
						func(i *command.Input, d *command.Data) bool {
							return false
						},
					),
					Arg[string]("s3", testDesc, SimpleCompleter[string]("delta", "epsilon")),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "alpha",
					"s3": "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"delta", "epsilon"},
				},
			},
		},
		// IfElse
		{
			name: "If runs t if function returns true",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd alpha ",
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					IfElse(
						Arg[string]("s2", testDesc, SimpleCompleter[string]("bravo", "charlie")),
						Arg[string]("s2", testDesc, SimpleCompleter[string]("alpha", "omega")),
						func(i *command.Input, d *command.Data) bool {
							return true
						},
					),
					Arg[string]("s3", testDesc, SimpleCompleter[string]("delta", "epsilon")),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "alpha",
					"s2": "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"bravo", "charlie"},
				},
			},
		},
		{
			name: "IfElse runs f if function returns false",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd alpha ",
				Node: SerialNodes(
					Arg[string]("s", testDesc),
					IfElse(
						Arg[string]("s2", testDesc, SimpleCompleter[string]("bravo", "charlie")),
						Arg[string]("s2", testDesc, SimpleCompleter[string]("alpha", "omega")),
						func(i *command.Input, d *command.Data) bool {
							return false
						},
					),
					Arg[string]("s3", testDesc, SimpleCompleter[string]("delta", "epsilon")),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"s":  "alpha",
					"s2": "",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"alpha", "omega"},
				},
			},
		},
		// MapArg test
		{
			name: "MapArg completes keys",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd ",
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, true),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 0,
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"one", "three", "two"},
				},
			},
		},
		{
			name: "MapArg completes some keys",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd t",
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, true),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 0,
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"three", "two"},
				},
			},
		},
		{
			name: "MapArg completes single",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd th",
				Node: SerialNodes(
					MapArg("m", testDesc, map[string]int{
						"one":   1,
						"two":   2,
						"three": 3,
					}, true),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 0,
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"three"},
				},
			},
		},
		{
			name: "MapArg completes int",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd ",
				Node: SerialNodes(
					MapArg("m", testDesc, map[int]string{
						123: "one",
						456: "two",
						789: "three",
					}, true),
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"123", "456", "789"},
				},
			},
		},
		{
			name: "MapArg completes single int",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd 4",
				Node: SerialNodes(
					MapArg("m", testDesc, map[int]string{
						123: "one",
						456: "two",
						789: "three",
						4:   "other",
					}, true),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"m": "other",
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"4", "456"},
				},
			},
		},
		// MapFlag test
		{
			name: "MapFlag completes some keys",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd --m t",
				Node: SerialNodes(
					FlagProcessor(
						MapFlag("m", 'm', testDesc, map[string]int{
							"one":   1,
							"two":   2,
							"three": 3,
						}, true),
					),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"m": 0,
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"three", "two"},
				},
			},
		},
		// FlagStop test
		{
			name: "Stops processing flags after flag stop",
			ctc: &commandtest.CompleteTestCase{
				Args: "its --boo a -r abc -- secret def --coordinates -y 1.1 2.2 ghi -n ",
				Node: SerialNodes(
					FlagProcessor(
						ListFlag[float64]("coordinates", 'c', testDesc, 2, 0),
						BoolFlag("boo", 'o', testDesc),
						BoolFlag("yay", 'y', testDesc),
						ListFlag[string]("names", 'n', testDesc, 2, 0),
						Flag[string]("rating", 'r', testDesc),
					),
					ListArg[string]("extra", testDesc, 0, command.UnboundedList, SimpleDistinctCompleter[[]string]("abc", "def", "ghi", "jkl")),
				),
				WantData: &command.Data{Values: map[string]interface{}{
					"boo":    true,
					"rating": "abc",
					"extra":  []string{"a", "secret", "def", "--coordinates", "-y", "1.1", "2.2", "ghi", "-n", ""},
				}},
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "jkl"},
				},
			},
		},
		// MutableProcessor tests
		{
			name: "Reference does not update underlying processor",
			ctc: func() *commandtest.CompleteTestCase {
				hi := SimpleProcessor(nil, func(i *command.Input, d *command.Data) (*command.Completion, error) {
					return &command.Completion{Suggestions: []string{"hi"}}, nil
				})
				hello := SimpleProcessor(nil, func(i *command.Input, d *command.Data) (*command.Completion, error) {
					return &command.Completion{Suggestions: []string{"hello"}}, nil
				})

				return &commandtest.CompleteTestCase{
					Node: SerialNodes(
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							hi = hello
							return nil
						}),
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							c, err := hi.Complete(nil, nil)
							if err != nil {
								return err
							}
							d.Set("FINAL", c.Suggestions)
							return nil
						}),
						hi,
					),
					Want: &command.Autocompletion{
						Suggestions: []string{"hi"},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						"FINAL": []string{"hello"},
					}},
				}
			}(),
		},
		{
			name: "MutableProcessor DOES update underlying processor",
			ctc: func() *commandtest.CompleteTestCase {
				hi := NewMutableProcessor(SimpleProcessor(nil, func(i *command.Input, d *command.Data) (*command.Completion, error) {
					return &command.Completion{Suggestions: []string{"hi"}}, nil
				}))
				hello := SimpleProcessor(nil, func(i *command.Input, d *command.Data) (*command.Completion, error) {
					return &command.Completion{Suggestions: []string{"hello"}}, nil
				})

				return &commandtest.CompleteTestCase{
					Node: SerialNodes(
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							hi.Processor = &hello
							return nil
						}),
						SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
							c, err := hi.Complete(nil, nil)
							if err != nil {
								return err
							}
							d.Set("FINAL", c.Suggestions)
							return nil
						}),
						hi,
					),
					Want: &command.Autocompletion{
						Suggestions: []string{"hello"},
					},
					WantData: &command.Data{Values: map[string]interface{}{
						"FINAL": []string{"hello"},
					}},
				}
			}(),
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			stubs.StubGetwd(t, test.osGetwd, test.osGetwdErr)
			testutil.StubValue(t, &filepathAbs, func(s string) (string, error) {
				return filepath.Join(test.filepathAbs, s), test.filepathAbsErr
			})
			autocompleteTest(t, test.ctc, test.ictc)
		})
	}
}

func printNode(s string) command.Node {
	return &SimpleNode{
		Processor: &ExecutorProcessor{func(output command.Output, _ *command.Data) error {
			output.Stdout(s)
			return nil
		}},
	}
}

func printlnNode(stdout bool, a ...interface{}) command.Node {
	return &SimpleNode{
		Processor: &ExecutorProcessor{func(output command.Output, _ *command.Data) error {
			if !stdout {
				return output.Stderrln(a...)
			}
			output.Stdoutln(a...)
			return nil
		}},
	}
}

func printArgsNode() command.Node {
	return &SimpleNode{
		Processor: &ExecutorProcessor{func(output command.Output, data *command.Data) error {
			var keys []string
			for k := range data.Values {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				output.Stdoutf("%s: %v\n", k, data.Values[k])
			}
			return nil
		}},
	}
}

func sampleRepeaterProcessor(minN, optionalN int) command.Processor {
	return NodeRepeater(SerialNodes(
		Arg[string]("KEY", testDesc, &CustomSetter[string]{func(v string, d *command.Data) {
			if !d.Has("keys") {
				d.Set("keys", []string{v})
			} else {
				d.Set("keys", append(d.StringList("keys"), v))
			}
		}}, SimpleCompleter[string]("alpha", "bravo", "charlie", "brown")),
		Arg[int]("VALUE", testDesc, &CustomSetter[int]{func(v int, d *command.Data) {
			if !d.Has("values") {
				d.Set("values", []int{v})
			} else {
				d.Set("values", append(d.IntList("values"), v))
			}
		}}, SimpleCompleter[int]("1", "121", "1213121")),
	), minN, optionalN)
}

func branchSynNode() command.Node {
	return &BranchNode{
		Branches: map[string]command.Node{
			"hello hi greetings": printNode("yo"),
		},
		Default: printNode("default"),
		Synonyms: BranchSynonyms(map[string][]string{
			"hello": {"hey", "howdy"},
		}),
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

type fakeOpt[T any] struct{}

func (fo *fakeOpt[T]) modifyArgumentOption(*argumentOption[T]) {}

func TestPanics(t *testing.T) {
	for _, test := range []struct {
		name string
		f    func()
		want interface{}
	}{
		{
			name: "Flag with improper short name panics",
			f: func() {
				FlagProcessor(
					Flag[string]("ampersand", '&', testDesc),
				)
			},
			want: "Short flag name '&' must match regex ^[a-zA-Z0-9]$",
		},
		{
			name: "Can't add options to a boolean flag",
			f: func() {
				BoolFlag("b", 'b', testDesc).AddOptions(&fakeOpt[bool]{})
			},
			want: "Provided option is incompatible with BoolFlag",
		},
		{
			name: "Can't create arg for unsupported type",
			f: func() {
				Arg[*SimpleNode]("n", testDesc).Execute(command.NewInput([]string{"abc"}, nil), commandtest.NewOutput(), &command.Data{}, &command.ExecuteData{})
			},
			want: "no operator defined for type *commander.SimpleNode",
		},
		{
			name: "BoolFlag.Usage()",
			f: func() {
				BoolFlag("", '_', "").Processor().Usage(nil, nil, nil)
			},
			want: "Unexpected BoolFlag.Usage() call",
		},
		{
			name: "OptionalFlag.Usage()",
			f: func() {
				OptionalFlag[string]("", '_', "", "").Processor().Usage(nil, nil, nil)
			},
			want: "Unexpected OptionalFlag.Usage() call",
		},
		{
			name: "ItemizedListFlag.Usage()",
			f: func() {
				ItemizedListFlag[string]("", '_', "").Processor().Usage(nil, nil, nil)
			},
			want: "Unexpected ItemizedListFlag.Usage() call",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testutil.CmpPanic(t, test.name, func() bool { test.f(); return false }, test.want)
		})
	}
}
