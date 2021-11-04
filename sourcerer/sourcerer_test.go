package sourcerer

import (
	"io/ioutil"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
)

func TestGenerateBinaryNode(t *testing.T) {
	oldGSL := getSourceLoc
	getSourceLoc = func() (string, error) {
		return "/fake/source/location", nil
	}
	defer func() { getSourceLoc = oldGSL }()

	for _, test := range []struct {
		name     string
		clis     []CLI
		args     []string
		wantFile []string
	}{
		{
			name: "generates source file when no CLIs",
			wantFile: []string{
				"",
				`	pushd . > /dev/null`,
				`	cd "$(dirname /fake/source/location)"`,
				`	# TODO: this won't work if two separate source files are used.`,
				`	go build -o $GOPATH/bin/leep-frog-source`,
				`	popd > /dev/null`,
				`	`,
				`	function _custom_autocomplete_leep-frog-source {`,
				`		tFile=$(mktemp)`,
				`		$GOPATH/bin/leep-frog-source autocomplete ${COMP_WORDS[0]} $COMP_POINT "$COMP_LINE" > $tFile`,
				`		local IFS=$'\n'`,
				`		COMPREPLY=( $(cat $tFile) )`,
				`		rm $tFile`,
				`	}`,
				`	`,
				`	function _custom_execute_leep-frog-source {`,
				`		# tmpFile is the file to which we write ExecuteData.Executable`,
				`		tmpFile=$(mktemp)`,
				`		$GOPATH/bin/leep-frog-source execute $tmpFile "$@"`,
				`		source $tmpFile`,
				`		if [ -z "$LEEP_FROG_DEBUG" ]`,
				`		then`,
				`		  rm $tmpFile`,
				`		else`,
				`		  echo $tmpFile`,
				`		fi`,
				`	}`,
				`	`,
				`	function mancli {`,
				`		$GOPATH/bin/leep-frog-source usage "$@"`,
				`	}`,
				`	`,
			},
		},
		{
			name: "generates source file with custom filename",
			args: []string{"custom-output_file"},
			wantFile: []string{
				"",
				`	pushd . > /dev/null`,
				`	cd "$(dirname /fake/source/location)"`,
				`	# TODO: this won't work if two separate source files are used.`,
				`	go build -o $GOPATH/bin/custom-output_file`,
				`	popd > /dev/null`,
				`	`,
				`	function _custom_autocomplete_custom-output_file {`,
				`		tFile=$(mktemp)`,
				`		$GOPATH/bin/custom-output_file autocomplete ${COMP_WORDS[0]} $COMP_POINT "$COMP_LINE" > $tFile`,
				`		local IFS=$'\n'`,
				`		COMPREPLY=( $(cat $tFile) )`,
				`		rm $tFile`,
				`	}`,
				`	`,
				`	function _custom_execute_custom-output_file {`,
				`		# tmpFile is the file to which we write ExecuteData.Executable`,
				`		tmpFile=$(mktemp)`,
				`		$GOPATH/bin/custom-output_file execute $tmpFile "$@"`,
				`		source $tmpFile`,
				`		if [ -z "$LEEP_FROG_DEBUG" ]`,
				`		then`,
				`		  rm $tmpFile`,
				`		else`,
				`		  echo $tmpFile`,
				`		fi`,
				`	}`,
				`	`,
				`	function mancli {`,
				`		$GOPATH/bin/custom-output_file usage "$@"`,
				`	}`,
				`	`,
			},
		},
		{
			name: "generates source file with CLIs",
			clis: append(SimpleCommands(map[string]string{
				"x": "exit",
				"l": "ls -la",
			}), &testCLI{name: "basic", setup: []string{"his", "story"}}),
			wantFile: []string{
				"",
				`	pushd . > /dev/null`,
				`	cd "$(dirname /fake/source/location)"`,
				`	# TODO: this won't work if two separate source files are used.`,
				`	go build -o $GOPATH/bin/leep-frog-source`,
				`	popd > /dev/null`,
				`	`,
				`	function _custom_autocomplete_leep-frog-source {`,
				`		tFile=$(mktemp)`,
				`		$GOPATH/bin/leep-frog-source autocomplete ${COMP_WORDS[0]} $COMP_POINT "$COMP_LINE" > $tFile`,
				`		local IFS=$'\n'`,
				`		COMPREPLY=( $(cat $tFile) )`,
				`		rm $tFile`,
				`	}`,
				`	`,
				`	function _custom_execute_leep-frog-source {`,
				`		# tmpFile is the file to which we write ExecuteData.Executable`,
				`		tmpFile=$(mktemp)`,
				`		$GOPATH/bin/leep-frog-source execute $tmpFile "$@"`,
				`		source $tmpFile`,
				`		if [ -z "$LEEP_FROG_DEBUG" ]`,
				`		then`,
				`		  rm $tmpFile`,
				`		else`,
				`		  echo $tmpFile`,
				`		fi`,
				`	}`,
				`	`,
				`	function mancli {`,
				`		$GOPATH/bin/leep-frog-source usage "$@"`,
				`	}`,
				`	`,
				`	function _setup_for_basic_cli {`,
				`		his  `,
				`  story`,
				`	}`,
				`	alias basic='o=$(mktemp) && _setup_for_basic_cli > $o && _custom_execute_leep-frog-source basic $o'`,
				"complete -F _custom_autocomplete_leep-frog-source -o nosort basic",
				`alias l='_custom_execute_leep-frog-source l'`,
				"complete -F _custom_autocomplete_leep-frog-source -o nosort l",
				"alias x='_custom_execute_leep-frog-source x'",
				"complete -F _custom_autocomplete_leep-frog-source -o nosort x",
				"",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			o := command.NewFakeOutput()
			source(test.clis, test.args, o)
			o.Close()

			if o.GetStderr() != nil {
				t.Errorf("source(%v) produced stderr when none was expected:\n%v", test.args, o.GetStderr())
			}

			if len(o.GetStdout()) != 1 {
				t.Fatalf("source(%v) should have outputted one line (a file name), but didn't:\n%v", test.args, o.GetStdout())
			}

			cmpFile(t, "Incorrect source file generated", o.GetStdout()[0], test.wantFile)
		})
	}
}

func cmpFile(t *testing.T, prefix, filename string, want []string) {
	t.Helper()
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if want == nil {
		want = []string{""}
	}
	if diff := cmp.Diff(want, strings.Split(string(contents), "\n")); diff != "" {
		t.Errorf(prefix+": incorrect file contents (-want, +got):\n%s", diff)
	}
}

func TestSourcerer(t *testing.T) {
	f, err := ioutil.TempFile("", "test-leep-frog")
	if err != nil {
		t.Fatalf("failed to create tmp file: %v", err)
	}
	u := command.GetUsage((&sourcerer{}).Node()).String()
	for _, test := range []struct {
		name       string
		clis       []CLI
		args       []string
		wantStdout []string
		wantStderr []string
		wantCLIs   map[string]CLI
		wantFile   []string
	}{
		{
			name: "fails if invalid command branch",
			args: []string{"wizardry", "stuff"},
			wantStderr: []string{
				"Unprocessed extra args: [stuff]",
				u,
			},
		},
		// Execute tests
		{
			name: "fails if no file arg",
			args: []string{"execute"},
			wantStderr: []string{
				`Argument "FILE" requires at least 1 argument, got 0`,
				u,
			},
		},
		{
			name: "fails if no cli arg",
			args: []string{"execute", "file"},
			wantStderr: []string{
				`Argument "CLI" requires at least 1 argument, got 0`,
				u,
			},
		},
		{
			name: "fails if unknown CLI",
			args: []string{"execute", "file", "idk"},
			wantStderr: []string{
				`unknown CLI "idk"`,
			},
		},
		{
			name: "properly executes CLI",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						var keys []string
						for k := range *d {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						o.Stdout("Output:")
						for _, k := range keys {
							o.Stdoutf("%s: %s", k, d.Str(k))
						}
						return nil
					},
				},
			},
			args:       []string{"execute", "file", "basic"},
			wantStdout: []string{"Output:"},
		},
		{
			name: "handles processing error",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						return o.Stderr("oops")
					},
				},
			},
			args:       []string{"execute", "file", "basic"},
			wantStderr: []string{"oops"},
		},
		{
			name: "properly passes arguments to CLI",
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.StringListNode("sl", "test desc", 1, 4),
					},
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						var keys []string
						for k := range *d {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						o.Stdout("Output:")
						for _, k := range keys {
							o.Stdoutf("%s: %s", k, d.Str(k))
						}
						return nil
					},
				},
			},
			args: []string{"execute", "file", "basic", "un", "deux", "trois"},
			wantStdout: []string{
				"Output:",
				"sl: un, deux, trois",
			},
		},
		{
			name: "properly passes extra arguments to CLI",
			clis: []CLI{
				&testCLI{
					name: "basic",
				},
			},
			args: []string{"execute", "file", "basic", "un", "deux", "trois"},
			wantStderr: []string{
				"Unprocessed extra args: [un deux trois]",
				"",
				u,
			},
		},
		{
			name: "properly marks CLI as changed",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						tc.Stuff = "things"
						tc.changed = true
						return nil
					},
				},
			},
			args: []string{"execute", "file", "basic"},
			wantCLIs: map[string]CLI{
				"basic": &testCLI{
					Stuff: "things",
				},
			},
		},
		{
			name: "writes execute data to file",
			clis: []CLI{
				&testCLI{
					name: "basic",
					f: func(tc *testCLI, i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						ed.Executable = []string{"echo", "hello", "there"}
						return nil
					},
				},
			},
			args: []string{"execute", f.Name(), "basic"},
			wantFile: []string{
				"echo",
				"hello",
				"there",
			},
		},
		// Autocomplete tests
		{
			name: "autocomplete requires cli name",
			args: []string{"autocomplete"},
			wantStderr: []string{
				`Argument "CLI" requires at least 1 argument, got 0`,
				u,
			},
		},
		{
			name: "autocomplete requires comp_point",
			args: []string{"autocomplete", "idk"},
			wantStderr: []string{
				`Argument "COMP_POINT" requires at least 1 argument, got 0`,
				u,
			},
		},
		{
			name: "autocomplete requires comp_line",
			args: []string{"autocomplete", "idk", "2"},
			wantStderr: []string{
				`Argument "COMP_LINE" requires at least 1 argument, got 0`,
				u,
			},
		},
		{
			name: "autocomplete requires valid cli",
			args: []string{"autocomplete", "idk", "2", "a"},
			wantStderr: []string{
				`unknown CLI "idk"`,
			},
		},
		{
			name: "autocomplete passes empty string along for completion",
			args: []string{"autocomplete", "basic", "0", ""},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.StringNode("s", "desc", command.SimpleCompletor("alpha", "bravo", "charlie")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"alpha",
				"bravo",
				"charlie",
			),
		},
		{
			name: "autocomplete does partial completion",
			args: []string{"autocomplete", "basic", "5", "cmd b"},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.StringNode("s", "desc", command.SimpleCompletor("alpha", "bravo", "charlie", "brown", "baker")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"baker",
				"bravo",
				"brown",
			),
		},
		{
			name: "autocomplete goes along processors",
			args: []string{"autocomplete", "basic", "6", "cmd a "},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.StringNode("s", "desc", command.SimpleCompletor("alpha", "bravo", "charlie", "brown", "baker")),
						command.StringNode("z", "desz", command.SimpleCompletor("un", "deux", "trois")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"un",
				"deux",
				"trois",
			),
		},
		{
			name: "autocomplete does earlier completion if cpoint is smaller",
			args: []string{"autocomplete", "basic", "5", "cmd c "},
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.StringNode("s", "desc", command.SimpleCompletor("alpha", "bravo", "charlie", "brown", "baker")),
						command.StringNode("z", "desz", command.SimpleCompletor("un", "deux", "trois")),
					},
				},
			},
			wantStdout: autocompleteSuggestions(
				"charlie",
			),
		},
		// Usage tests
		{
			name: "usage requires cli name",
			args: []string{"usage"},
			wantStderr: []string{
				`Argument "CLI" requires at least 1 argument, got 0`,
				u,
			},
		},
		{
			name: "usage fails if too many args",
			args: []string{"usage", "idk", "and"},
			wantStderr: []string{
				"Unprocessed extra args: [and]",
				u,
			},
		},
		{
			name: "usage prints command's usage",
			clis: []CLI{
				&testCLI{
					name: "basic",
					processors: []command.Processor{
						command.StringNode("S", "desc"),
						command.IntListNode("IS", "ints", 2, 0),
						command.FloatListNode("FS", "floats", 0, command.UnboundedList),
					},
				},
			},
			args: []string{"usage", "basic"},
			wantStdout: []string{strings.Join([]string{
				"S IS IS [ FS ... ]",
				"",
				"Arguments:",
				"  FS: floats",
				"  IS: ints",
				"  S: desc",
			}, "\n")},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ioutil.WriteFile(f.Name(), nil, 0644); err != nil {
				t.Fatalf("failed to clear file: %v", err)
			}
			// Stub out real cache
			cash := cache.NewTestCache(t)
			ogc := getCache
			getCache = func() *cache.Cache {
				return cash
			}
			defer func() { getCache = ogc }()

			// Run source command
			o := command.NewFakeOutput()
			source(test.clis, test.args, o)
			o.Close()

			// Check outputs
			if diff := cmp.Diff(test.wantStdout, o.GetStdout()); diff != "" {
				t.Errorf("source(%v) sent incorrect stdout (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, o.GetStderr()); diff != "" {
				t.Errorf("source(%v) sent incorrect stderr (-want, +got):\n%s", test.args, diff)
			}

			// Check file contents
			cmpFile(t, "Sourcing produced incorrect file contents", f.Name(), test.wantFile)

			// Check cli changes
			for _, c := range test.clis {
				wantNew, wantChanged := test.wantCLIs[c.Name()]
				if wantChanged != c.Changed() {
					t.Errorf("CLI %q was incorrectly changed: want %v; got %v", c.Name(), wantChanged, c.Changed())
				}
				if wantChanged {
					if diff := cmp.Diff(wantNew, c, cmpopts.IgnoreUnexported(testCLI{})); diff != "" {
						t.Errorf("CLI %q was incorrectly updated: %v", c.Name(), diff)
					}
				}
				delete(test.wantCLIs, c.Name())
			}

			if len(test.wantCLIs) != 0 {
				for name := range test.wantCLIs {
					t.Errorf("Unknown CLI was supposed to change %q", name)
				}
			}
		})
	}
}

type testCLI struct {
	name       string
	processors []command.Processor
	f          func(*testCLI, *command.Input, command.Output, *command.Data, *command.ExecuteData) error
	changed    bool
	setup      []string
	// Used for json checking
	Stuff string
}

func (tc *testCLI) Name() string {
	return tc.name
}

func (tc *testCLI) Load(string) error { return nil }
func (tc *testCLI) Node() *command.Node {
	ns := append(tc.processors, command.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		if tc.f != nil {
			return tc.f(tc, i, o, d, ed)
		}
		return nil
	}, nil))
	return command.SerialNodes(ns[0], ns[1:]...)
}
func (tc *testCLI) Changed() bool   { return tc.changed }
func (tc *testCLI) Setup() []string { return tc.setup }

func autocompleteSuggestions(s ...string) []string {
	sort.Strings(s)
	return []string{strings.Join(s, "\n") + "\n"}
}
