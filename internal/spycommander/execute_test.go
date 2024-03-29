package spycommander

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/constants"
	"github.com/leep-frog/command/internal/spycommand"
	"github.com/leep-frog/command/internal/spycommandertest"
	"github.com/leep-frog/command/internal/spycommandtest"
)

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name                string
		etc                 *commandtest.ExecuteTestCase
		ietc                *spycommandtest.ExecuteTestCase
		isUsageError        bool
		wantUsageErrorCheck error
		ignoreErrFuncs      []func(error) bool
	}{
		{
			name: "handles empty node",
		},
		{
			name: "fails on unused args",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"abc"},
				WantStderr: "Unprocessed extra args: [abc]\n",
				WantErr:    fmt.Errorf("Unprocessed extra args: [abc]"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "abc"},
					},
					Remaining: []int{0},
				},
			},
		},
		{
			name: "returns execution error",
			etc: &commandtest.ExecuteTestCase{
				Node: &simpleNode{
					ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						return fmt.Errorf("whoops")
					},
				},
				WantErr: fmt.Errorf("whoops"),
			},
		},
		{
			name: "returns executor error",
			etc: &commandtest.ExecuteTestCase{
				Node: &simpleNode{
					ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
							return fmt.Errorf("executor whoops")
						})
						return nil
					},
				},
				WantErr: fmt.Errorf("executor whoops"),
			},
		},
		{
			name: "returns next error",
			etc: &commandtest.ExecuteTestCase{
				Node: &simpleNode{
					nx: func(i *command.Input, d *command.Data) (command.Node, error) {
						return nil, fmt.Errorf("next oops")
					},
				},
				WantErr: fmt.Errorf("next oops"),
			},
		},
		{
			name: "runs executor at the end",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("First executor")
								return nil
							})
							o.Stdoutln("First processor")
							return nil
						},
					},
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("Second executor")
								return nil
							})
							o.Stdoutln("Second processor")
							return nil
						},
					},
				),
				WantStdout: strings.Join([]string{
					"First processor",
					"Second processor",
					"First executor",
					"Second executor",
					"",
				}, "\n"),
			},
		},
		{
			name: "stops node execution at first error",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("First executor")
								return nil
							})
							o.Stdoutln("First processor")
							return fmt.Errorf("whoops 1")
						},
					},
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("Second executor")
								return nil
							})
							o.Stdoutln("Second processor")
							return nil
						},
					},
				),
				WantErr: fmt.Errorf("whoops 1"),
				WantStdout: strings.Join([]string{
					"First processor",
					"",
				}, "\n"),
			},
		},
		{
			name: "stops executor logic at first error",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("First executor")
								return fmt.Errorf("ex whoops 1")
							})
							o.Stdoutln("First processor")
							return nil
						},
					},
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("Second executor")
								return nil
							})
							o.Stdoutln("Second processor")
							return nil
						},
					},
				),
				WantErr: fmt.Errorf("ex whoops 1"),
				WantStdout: strings.Join([]string{
					"First processor",
					"Second processor",
					"First executor",
					"",
				}, "\n"),
			},
		},
		// Panics
		{
			name: "forwards panic value in node execution",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("First executor")
								return fmt.Errorf("ex whoops 1")
							})
							o.Stdoutln("First processor")
							panic("ahhh")
						},
					},
				),
				WantPanic: "ahhh",
				WantStdout: strings.Join([]string{
					"First processor",
					"",
				}, "\n"),
			},
		},
		{
			name: "forwards panic value in executor execution",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								o.Stdoutln("First executor")
								panic("argh")
							})
							o.Stdoutln("First processor")
							return nil
						},
					},
				),
				WantPanic: "argh",
				WantStdout: strings.Join([]string{
					"First processor",
					"First executor",
					"",
				}, "\n"),
			},
		},
		{
			name: "handles panic from termination",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							o.Stdoutln("First processor")
							o.Terminatef("Termination\n")
							o.Stdoutln("First processor part II")
							return nil
						},
					},
				),
				WantStderr: "Termination\n",
				WantErr:    fmt.Errorf("Termination"),
				WantStdout: strings.Join([]string{
					"First processor",
					"",
				}, "\n"),
			},
		},
		// ProcessOrExecute tests
		{
			name: "ProcessOrExecute processes a node",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							var subN command.Node
							subN = &simpleNode{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
								o.Stdoutln("hurray node!")
								return nil
							}}
							return ProcessOrExecute(subN, i, o, d, ed)
						},
					},
				),
				WantStdout: "hurray node!\n",
			},
		},
		{
			name: "ProcessOrExecute processes a processor",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							var subP command.Processor
							subP = &simpleProcessor{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
								o.Stdoutln("hurray processor!")
								return nil
							}}

							if _, ok := subP.(command.Node); ok {
								t.Fatalf("Object should only implement Processor, not Node: %v", subP)
							}
							return ProcessOrExecute(subP, i, o, d, ed)
						},
					},
				),
				WantStdout: "hurray processor!\n",
			},
		},
		// IgnoreErr tests
		{
			name: "ProcessGraphExecution doesn't return ignorable error",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							return ProcessGraphExecution(serialNodes(
								&simpleNode{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
									o.Stdoutln("First")
									return fmt.Errorf("please skip this error")
								}},
							), i, o, d, ed, func(err error) bool { return strings.Contains(err.Error(), "skip") })
						},
					},
				),
				WantStdout: "First\n",
			},
		},
		{
			name: "ProcessGraphExecution continues processing graph after ignorable error",
			etc: &commandtest.ExecuteTestCase{
				Node: serialNodes(
					&simpleNode{
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							return ProcessGraphExecution(serialNodes(
								&simpleNode{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
									o.Stdoutln("First")
									return fmt.Errorf("please skip this error")
								}},
								&simpleNode{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
									o.Stdoutln("Second")
									return nil
								}},
							), i, o, d, ed, func(err error) bool { return strings.Contains(err.Error(), "skip") })
						},
					},
				),
				WantStdout: "First\nSecond\n",
			},
		},
		/* Useful for commenting out tests. */

		// Usage tests (execute tests with help keyword)
		// Note: the Usage -> string logic is tested elsewhere so these tests should
		// simply focus on the graph traversal side of things.
		{
			name: "handles empty nodes",
			etc: &commandtest.ExecuteTestCase{
				Args:       []string{"--help"},
				WantStdout: "\n",
			},
		},
		{
			name: "displays usage",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: &simpleNode{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
					u.SetDescription("Desc")
					return nil
				}},
				WantStdout: "Desc\n",
			},
		},
		{
			name: "returns error and handles IsUsageError returns false",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: &simpleNode{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
					u.SetDescription("Desc")
					return fmt.Errorf("whoops")
				}},
				WantStderr: "whoops\n",
				WantErr:    fmt.Errorf("whoops"),
			},
			wantUsageErrorCheck: fmt.Errorf("whoops"),
			isUsageError:        false,
		},
		{
			name: "returns error and handles IsUsageError returns true",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: func() command.Node {
					var cnt int
					return &simpleNode{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
						cnt++
						switch cnt {
						case 1:
							u.SetDescription("Desc")
							u.AddArg("BOO", "boo", 1, 2)
							return fmt.Errorf("first whoops")
						case 2:
							u.AddArg("HURRAY", "h desc", 2, 1)
							return nil
						default:
							panic("Run too many times")
						}
					}}
				}(),
				WantStderr: strings.Join([]string{
					"first whoops",
					"",
					"======= Command Usage =======",
					"HURRAY HURRAY [ HURRAY ]",
					"",
					"Arguments:",
					"  HURRAY: h desc",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("first whoops"),
			},
			wantUsageErrorCheck: fmt.Errorf("first whoops"),
			isUsageError:        true,
		},
		{
			name: "traverses multiple nodes",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: serialNodes(
					&simpleNode{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
						u.SetDescription("Desc")
						u.AddFlag("un", constants.FlagNoShortName, "", "", 0, 0)
						return nil
					}},
					&simpleNode{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
						u.AddArg("ARG", "", 1, 0)
						u.AddFlag("deux", constants.FlagNoShortName, "", "", 0, 0)
						return nil
					}},
				),
				WantStdout: "Desc\nARG --un --deux\n",
			},
		},
		{
			name: "fails if UsageNext error and IsUsageError returns false",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: &simpleNode{
					u: func(i *command.Input, d *command.Data, u *command.Usage) error {
						u.SetDescription("Desc")
						u.AddFlag("un", constants.FlagNoShortName, "", "", 0, 0)
						return nil
					},
					unx: func(i *command.Input, d *command.Data) (command.Node, error) {
						return nil, fmt.Errorf("rats")
					},
				},
				WantErr:    fmt.Errorf("rats"),
				WantStderr: "rats\n",
			},
			wantUsageErrorCheck: fmt.Errorf("rats"),
			isUsageError:        false,
		},
		{
			name: "fails if UsageNext error and IsUsageError returns true",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: func() command.Node {
					var nxCnt, cnt int
					return &simpleNode{
						u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							cnt++
							switch cnt {
							case 1:
								u.AddArg("BOO", "boo", 1, 2)
								return nil
							case 2:
								u.AddArg("HURRAY", "h desc", 2, 1)
								return nil
							default:
								panic("Run too many times")
							}
						},
						unx: func(i *command.Input, d *command.Data) (command.Node, error) {
							nxCnt++
							switch nxCnt {
							case 1:
								return nil, fmt.Errorf("rats")
							case 2:
								return &simpleNode{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
									u.SetDescription("Made it!")
									return nil
								}}, nil
							default:
								panic("Run to many times")
							}
						},
					}
				}(),
				WantErr: fmt.Errorf("rats"),
				WantStderr: strings.Join([]string{
					"rats",
					"",
					"======= Command Usage =======",
					"Made it!",
					"HURRAY HURRAY [ HURRAY ]",
					"",
					"Arguments:",
					"  HURRAY: h desc",
					"",
				}, "\n"),
			},
			wantUsageErrorCheck: fmt.Errorf("rats"),
			isUsageError:        true,
		},
		// ProcessOrUsage tests
		{
			name: "ProcessOrUsage processes a node",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: serialNodes(
					&simpleNode{
						u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							var subN command.Node
							subN = &simpleNode{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
								u.SetDescription("HERE")
								return nil
							}}
							return ProcessOrUsage(subN, i, d, u)
						},
					},
				),
				WantStdout: "HERE\n",
			},
		},
		{
			name: "ProcessOrUsage processes a processor",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"--help"},
				Node: serialNodes(
					&simpleNode{
						u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							var subP command.Processor
							subP = &simpleProcessor{u: func(i *command.Input, d *command.Data, u *command.Usage) error {
								u.AddFlag("sub", constants.FlagNoShortName, "", "", 0, 0)
								return nil
							}}

							if _, ok := subP.(command.Node); ok {
								t.Fatalf("Object should only implement Processor, not Node: %v", subP)
							}
							return ProcessOrUsage(subP, i, d, u)
						},
					},
				),
				WantStdout: "--sub\n",
			},
		},
		// Execute with usage errors
		{
			name: "Extra args results in usage error with usage doc",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"extra-arg"},
				Node: serialNodes(
					&simpleNode{
						u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							u.AddFlag("sub", constants.FlagNoShortName, "", "", 0, 0)
							return nil
						},
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							o.Stdoutln("Executing logic")

							// Executors should not be executed
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								t.Fatalf("this code should not be reached")
								return fmt.Errorf("wrongo")
							})
							return nil
						},
					},
				),
				WantStdout: strings.Join([]string{
					"Executing logic",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("Unprocessed extra args: [extra-arg]"),
				WantStderr: strings.Join([]string{
					"Unprocessed extra args: [extra-arg]",
					"",
					UsageErrorSectionStart,
					"--sub",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "extra-arg"},
					},
					Remaining: []int{0},
				},
			},
		},
		{
			name: "Extra args results in usage error with usage doc -- with usage error",
			etc: &commandtest.ExecuteTestCase{
				Args: []string{"extra-arg"},
				Node: serialNodes(
					&simpleNode{
						u: func(i *command.Input, d *command.Data, u *command.Usage) error {
							u.AddFlag("sub", constants.FlagNoShortName, "", "", 0, 0)
							return fmt.Errorf("usage oops")
						},
						ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
							o.Stdoutln("Executing logic")

							// Executors should not be executed
							ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
								t.Fatalf("this code should not be reached")
								return fmt.Errorf("wrongo")
							})
							return nil
						},
					},
				),
				WantStdout: strings.Join([]string{
					"Executing logic",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("Unprocessed extra args: [extra-arg]"),
				WantStderr: strings.Join([]string{
					"Unprocessed extra args: [extra-arg]",
					"",
					UsageErrorSectionStart,
					"failed to get command usage: usage oops",
					"",
				}, "\n"),
			},
			ietc: &spycommandtest.ExecuteTestCase{
				WantInput: &spycommandtest.SpyInput{
					Args: []*spycommand.InputArg{
						{Value: "extra-arg"},
					},
					Remaining: []int{0},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.ietc == nil {
				test.ietc = &spycommandtest.ExecuteTestCase{}
			}
			test.ietc.SkipErrorTypeCheck = true
			var gotUsageErrorCheck error
			spycommandertest.ExecuteTest(t, test.etc, test.ietc, &spycommandertest.ExecuteTestFunctionBag{
				Execute,
				Use,
				nil,
				serialNodes,
				HelpBehavior,
				func(err error) bool { panic("Unsupported branching error") },
				func(err error) bool {
					if gotUsageErrorCheck != nil {
						panic("Already checked usage error")
					}
					gotUsageErrorCheck = err
					return test.isUsageError
				},
				func(err error) bool { panic("Unsupported not enough args error") },
				func(err error) bool { panic("Unsupported extra args error") },
				func(err error) bool { panic("Unsupported validation error") },
			})

			commandtest.CmpError(t, "IsUsageError()", test.wantUsageErrorCheck, gotUsageErrorCheck)
		})
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name string
		ctc  *commandtest.CompleteTestCase
		ictc *spycommandtest.CompleteTestCase
	}{
		{
			name: "fails if extra args (input for completion includes empty string as last word)",
			ctc: &commandtest.CompleteTestCase{
				WantErr: fmt.Errorf("Unprocessed extra args: []"),
			},
		},
		{
			name: "returns completion error",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return nil, fmt.Errorf("complete fail")
					},
				},
				WantErr: fmt.Errorf("complete fail"),
			},
		},
		{
			name: "returns next error",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					nx: func(i *command.Input, d *command.Data) (command.Node, error) {
						return nil, fmt.Errorf("next oops")
					},
				},
				WantErr: fmt.Errorf("next oops"),
			},
		},
		{
			name: "returns completion success",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"abc", "def", "ghi"},
						}, nil
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"abc", "def", "ghi"},
				},
			},
		},
		{
			name: "returns partial completion",
			ctc: &commandtest.CompleteTestCase{
				Args: "cmd d",
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"abc", "def", "ghi"},
						}, nil
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"def"},
				},
			},
		},
		{
			name: "returns first error",
			ctc: &commandtest.CompleteTestCase{
				Node: serialNodes(
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							return &command.Completion{
								Suggestions: []string{"abc"},
							}, fmt.Errorf("argh")
						},
					},
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							return &command.Completion{
								Suggestions: []string{"def"},
							}, nil
						},
					},
				),
				WantErr: fmt.Errorf("argh"),
				Want: &command.Autocompletion{
					Suggestions: []string{"abc"},
				},
			},
		},
		{
			name: "returns first completion",
			ctc: &commandtest.CompleteTestCase{
				Node: serialNodes(
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							return &command.Completion{
								Suggestions: []string{"abc"},
							}, nil
						},
					},
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							return &command.Completion{
								Suggestions: []string{"def"},
							}, nil
						},
					},
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"abc"},
				},
			},
		},
		{
			name: "iterates to next node if nil completion and nil error",
			ctc: &commandtest.CompleteTestCase{
				Node: serialNodes(
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							return nil, nil
						},
					},
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							return &command.Completion{
								Suggestions: []string{"def"},
							}, nil
						},
					},
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"def"},
				},
			},
		},
		// ProcessOrComplete
		{
			name: "ProcessOrExecute processes a node",
			ctc: &commandtest.CompleteTestCase{
				Node: serialNodes(
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							var subN command.Node
							subN = &simpleNode{cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
								return &command.Completion{Suggestions: []string{"un"}}, nil
							}}
							return ProcessOrComplete(subN, i, d)
						},
					},
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"un"},
				},
			},
		},
		{
			name: "ProcessOrExecute processes a processor",
			ctc: &commandtest.CompleteTestCase{
				Node: serialNodes(
					&simpleNode{
						cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
							var subP command.Processor
							subP = &simpleProcessor{cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
								return &command.Completion{Suggestions: []string{"deux"}}, nil
							}}

							if _, ok := subP.(command.Node); ok {
								t.Fatalf("Object should only implement Processor, not Node: %v", subP)
							}
							return ProcessOrComplete(subP, i, d)
						},
					},
				),
				Want: &command.Autocompletion{
					Suggestions: []string{"deux"},
				},
			},
		},
		// DeferredCompletion
		{
			name: "empty DeferredCompletion works",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions:        []string{"hi"},
							DeferredCompletion: &command.DeferredCompletion{},
						}, nil
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"hi"},
				},
			},
		},
		{
			name: "DeferredCompletion.F runs",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"hi"},
							DeferredCompletion: &command.DeferredCompletion{
								F: func(c *command.Completion, d *command.Data) (*command.Completion, error) {
									c.Suggestions = append(c.Suggestions, "there")
									return c, nil
								},
							},
						}, nil
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"hi", "there"},
				},
			},
		},
		{
			name: "DeferredCompletion.F returns error",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"hi"},
							DeferredCompletion: &command.DeferredCompletion{
								F: func(c *command.Completion, d *command.Data) (*command.Completion, error) {
									c.Suggestions = append(c.Suggestions, "there")
									return c, fmt.Errorf("oof")
								},
							},
						}, nil
					},
				},
				WantErr: fmt.Errorf("oof"),
				Want: &command.Autocompletion{
					Suggestions: []string{"hi", "there"},
				},
			},
		},
		{
			name: "DeferredCompletion.Graph runs",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"hi"},
							DeferredCompletion: &command.DeferredCompletion{
								Graph: &simpleNode{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
									d.Set("graph", "value")
									return nil
								}},
							},
						}, nil
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"hi"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"graph": "value",
				}},
			},
		},
		{
			name: "DeferredCompletion.Graph returns error",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						return &command.Completion{
							Suggestions: []string{"hi"},
							DeferredCompletion: &command.DeferredCompletion{
								Graph: &simpleNode{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
									d.Set("graph", "value")
									return fmt.Errorf("rats")
								}},
							},
						}, nil
					},
				},
				WantErr: fmt.Errorf("failed to execute DeferredCompletion graph: rats"),
				WantData: &command.Data{Values: map[string]interface{}{
					"graph": "value",
				}},
			},
		},
		{
			name: "DeferredCompletion runs Graph and then F",
			ctc: &commandtest.CompleteTestCase{
				Node: &simpleNode{
					cmpl: func(i *command.Input, d *command.Data) (*command.Completion, error) {
						values := []string{"un", "deux"}
						return &command.Completion{
							Suggestions: []string{"hi"},
							DeferredCompletion: &command.DeferredCompletion{
								F: func(c *command.Completion, d *command.Data) (*command.Completion, error) {
									c.Suggestions = append(c.Suggestions, "there")
									d.Set("F-value", values[0])
									values = values[1:]
									return c, nil
								},
								Graph: &simpleNode{ex: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
									d.Set("Graph-value", values[0])
									values = values[1:]
									return nil
								}},
							},
						}, nil
					},
				},
				Want: &command.Autocompletion{
					Suggestions: []string{"hi", "there"},
				},
				WantData: &command.Data{Values: map[string]interface{}{
					"Graph-value": "un",
					"F-value":     "deux",
				}},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.ictc == nil {
				test.ictc = &spycommandtest.CompleteTestCase{}
			}
			test.ictc.SkipErrorTypeCheck = true
			spycommandertest.AutocompleteTest(t, test.ctc, test.ictc, &spycommandertest.CompleteTestFunctionBag{
				Autocomplete,
				func(err error) bool { panic("Unsupported IsBranchingError") },
				func(err error) bool { panic("Unsupported IsUsageError") },
				func(err error) bool { panic("Unsupported IsNotEnoughArgsError") },
				func(err error) bool { panic("Unsupported IsExtraArgsError") },
				func(err error) bool { panic("Unsupported IsValidationError") },
			})
		})
	}
}

func serialNodes(ps ...command.Processor) command.Node {
	if len(ps) == 0 {
		return nil
	}
	root := &serialNode{ps[0], nil}
	prev := root
	for _, p := range ps[1:] {
		new := &serialNode{p, nil}
		prev.next = new
		prev = new
	}
	return root
}

type serialNode struct {
	command.Processor
	next command.Node
}

func (sn *serialNode) Next(i *command.Input, d *command.Data) (command.Node, error) {
	return sn.next, nil
}

func (sn *serialNode) UsageNext(i *command.Input, d *command.Data) (command.Node, error) {
	return sn.next, nil
}

type simpleNode struct {
	ex   func(*command.Input, command.Output, *command.Data, *command.ExecuteData) error
	cmpl func(*command.Input, *command.Data) (*command.Completion, error)
	u    func(*command.Input, *command.Data, *command.Usage) error

	nx  func(*command.Input, *command.Data) (command.Node, error)
	unx func(*command.Input, *command.Data) (command.Node, error)
}

func (sn *simpleNode) Execute(i *command.Input, o command.Output, d *command.Data, e *command.ExecuteData) error {
	if sn.ex == nil {
		return nil
	}
	return sn.ex(i, o, d, e)
}

func (sn *simpleNode) Complete(i *command.Input, d *command.Data) (*command.Completion, error) {
	if sn.cmpl == nil {
		return nil, nil
	}
	return sn.cmpl(i, d)
}

func (sn *simpleNode) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	if sn.u == nil {
		return nil
	}
	return sn.u(i, d, u)
}

func (sn *simpleNode) Next(i *command.Input, d *command.Data) (command.Node, error) {
	if sn.nx == nil {
		return nil, nil
	}
	return sn.nx(i, d)
}

func (sn *simpleNode) UsageNext(i *command.Input, d *command.Data) (command.Node, error) {
	if sn.unx == nil {
		return nil, nil
	}
	return sn.unx(i, d)
}

type simpleProcessor struct {
	ex   func(*command.Input, command.Output, *command.Data, *command.ExecuteData) error
	cmpl func(*command.Input, *command.Data) (*command.Completion, error)
	u    func(*command.Input, *command.Data, *command.Usage) error
}

func (sp *simpleProcessor) Execute(i *command.Input, o command.Output, d *command.Data, e *command.ExecuteData) error {
	if sp.ex == nil {
		return nil
	}
	return sp.ex(i, o, d, e)
}

func (sp *simpleProcessor) Complete(i *command.Input, d *command.Data) (*command.Completion, error) {
	if sp.cmpl == nil {
		return nil, nil
	}
	return sp.cmpl(i, d)
}

func (sp *simpleProcessor) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	if sp.u == nil {
		return nil
	}
	return sp.u(i, d, u)
}
