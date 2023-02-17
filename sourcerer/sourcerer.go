// package sourcerer sources CLI commands in a shell environment.
// See the `main` function in github.com/leep-frog/command/examples/source.go
// for an example of how to define a source file that uses this.
package sourcerer

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
	"golang.org/x/exp/maps"
)

var (
	// The file that was used to create the source file will also
	// be used for executing and autocompleting cli commands.
	generateBinary = strings.Join([]string{
		"pushd . > /dev/null",
		`cd "$(dirname %s)"`,
		"go build -o $GOPATH/bin/_%s_runner",
		"popd > /dev/null",
		"",
	}, "\n")

	// autocompleteFunction defines a bash function for CLI autocompletion.
	autocompleteFunction = strings.Join([]string{
		"function _custom_autocomplete_%s {",
		`  local tFile=$(mktemp)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		`  $GOPATH/bin/_%s_runner autocomplete ${COMP_WORDS[0]} "$COMP_TYPE" $COMP_POINT "$COMP_LINE" > $tFile`,
		`  local IFS=$'\n'`,
		`  COMPREPLY=( $(cat $tFile) )`,
		`  rm $tFile`,
		"}",
		"",
	}, "\n")

	globalAutocompleteForAliasFunction = strings.Join([]string{
		`function _leep_frog_autocompleter {`,
		fmt.Sprintf("  %s", FileStringFromCLI(`"$1"`)),
		`  local tFile=$(mktemp)`,
		// The last argument is for extra passthrough arguments to be passed for aliaser autocompletes.
		`  $GOPATH/bin/_${file}_runner autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`,
		`  local IFS='`,
		`';`,
		`  COMPREPLY=( $(cat $tFile) )`,
		`  rm $tFile`,
		`}`,
		``,
	}, "\n")

	// autocompleteForAliasFunction defines a bash function for CLI autocompletion for aliased commands.
	// See AliaserCommand.
	autocompleteForAliasFunction = strings.Join([]string{
		"function _custom_autocomplete_for_alias_%s {",
		`  _leep_frog_autocompleter %q %s`,
		"}",
		"",
	}, "\n")

	// executeFileContents defines a bash function for CLI execution.
	executeFileContents = strings.Join([]string{
		`function _custom_execute_%s {`,
		`  # tmpFile is the file to which we write ExecuteData.Executable`,
		`  local tmpFile=$(mktemp)`,
		``,
		`  # Run the go-only code`,
		`  $GOPATH/bin/_%s_runner execute "$1" $tmpFile "${@:2}"`,
		`  # Return the error code if go code terminated with an error`,
		`  local errorCode=$?`,
		`  if [ $errorCode -ne 0 ]; then return $errorCode; fi`,
		``,
		`  # Otherwise, run the ExecuteData.Executable data`,
		`  source $tmpFile`,
		`  local errorCode=$?`,
		fmt.Sprintf(`  if [ -z "$%s" ]; then`, command.DebugEnvVar),
		`    rm $tmpFile`,
		`  else`,
		`    echo $tmpFile`,
		`  fi`,
		`  return $errorCode`,
		`}`,
		`_custom_execute_%s "$@"`,
		``,
	}, "\n")

	// setupFunctionFormat is used to run setup functions prior to a CLI command execution.
	setupFunctionFormat = strings.Join([]string{
		`function %s {`,
		`  %s`,
		"}",
		"",
	}, "\n")

	// aliasWithSetupFormat is an alias definition template for commands that require a setup function.
	aliasWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && source $GOPATH/bin/_custom_execute_%s %s $o'"
	// aliasFormat is an alias definition template for commands that don't require a setup function.
	aliasFormat = "alias %s='source $GOPATH/bin/_custom_execute_%s %s'"
)

var (
	fileArg         = command.FileArgument("FILE", "Temporary file for execution")
	targetNameArg   = command.OptionalArg[string]("TARGET_NAME", "The name of the created target in $GOPATH/bin")
	passthroughArgs = command.ListArg[string]("ARG", "Arguments that get passed through to relevant CLI command", 0, command.UnboundedList)
	// See the below link for more details on COMP_* details:
	// https://www.gnu.org/software/bash/manual/html_node/Bash-Variables.html#Bash-Variables
	compTypeArg  = command.Arg[int]("COMP_TYPE", "COMP_TYPE variable from bash complete function")
	compPointArg = command.Arg[int]("COMP_POINT", "COMP_POINT variable from bash complete function")
	compLineArg  = command.Arg[string]("COMP_LINE", "COMP_LINE variable from bash complete function", &command.Transformer[string]{F: func(s string, d *command.Data) (string, error) {
		if cPoint := compPointArg.Get(d); cPoint < len(s) {
			return s[:cPoint], nil
		}
		return s, nil
	}})
	autocompletePassthroughArgs = command.ListArg[string]("PASSTHROUGH_ARG", "Arguments that get passed through to autocomplete command", 0, command.UnboundedList)
)

// CLI provides a way to construct CLIs in go, with tab-completion.
// Note, this has to be an interface (as opposed to a struct) because of the Load function.
type CLI interface {
	// We use json unmarshaling so if a CLI type wants custom marshaling/unmarshaling,
	// they just need to implement the json.Marshaler/Unmarshaler interface(s).

	// Name is the name of the alias command to use for this CLI.
	Name() string
	// Node returns the command node for the CLI. This is where the CLI's logic lives.
	Node() command.Node
	// Changed indicates whether or not the CLI has changed after execution.
	// If true, the CLI's value will be save to the cache.
	Changed() bool
	// Setup describes a set of commands that will be run in bash prior to the CLI.
	// The output from the commands will be stored in a file whose name will be
	// stored in the `Data` object. See the following methods:
	// `Data.SetupOutputFile()` returns the file name.
	// `Data.SetupOutputString()` returns the file contents as a string.
	// `Data.SetupOutputContents()` returns the file contents as a string slice.
	Setup() []string
}

// Returns if there was an error
func (s *sourcerer) executeExecutor(output command.Output, d *command.Data) error {
	cli := s.cliArg.Get(d)

	sourcingFile := d.String(fileArg.Name())
	args := d.StringList(passthroughArgs.Name())

	// Add the setup arg if relevant. This should be identical to
	// setup in commandtest.go.
	n := cli.Node()
	if len(cli.Setup()) > 0 {
		n = command.PreprendSetupArg(n)
	}

	eData, err := command.Execute(n, command.ParseExecuteArgs(args), output)
	if err != nil {
		if command.IsUsageError(err) && !s.printedUsageError && !s.forAutocomplete && !command.IsExtraArgsError(err) {
			s.printedUsageError = true
			command.ShowUsageAfterError(n, output)
		}
		// Commands are responsible for printing out error messages so
		// we just return if there are any issues here
		return err
	}

	// Save the CLI if it has changed.
	if cli.Changed() {
		if err := save(cli); err != nil {
			return output.Stderrf("failed to save cli data: %v\n", err)
		}
	}

	// Run the executable file if relevant.
	if eData == nil || len(eData.Executable) == 0 {
		return nil
	}

	f, err := os.OpenFile(sourcingFile, os.O_WRONLY|os.O_CREATE, command.CmdOS.DefaultFilePerm())
	if err != nil {
		return output.Stderrf("failed to open file: %v\n", err)
	}

	v := strings.Join(eData.Executable, "\n")

	if eData.FunctionWrap {
		v = strings.Join([]string{
			"function _leep_execute_data_function_wrap {",
			v,
			"}",
			"_leep_execute_data_function_wrap",
			"",
		}, "\n")
	}

	if _, err := f.WriteString(v); err != nil {
		return output.Stderrf("failed to write to execute file: %v\n", err)
	}

	return nil
}

func (s *sourcerer) autocompleteExecutor(o command.Output, d *command.Data) error {
	s.forAutocomplete = true
	cli := s.cliArg.Get(d)

	g, err := command.Autocomplete(cli.Node(), compLineArg.Get(d), autocompletePassthroughArgs.Get(d))
	if err != nil {
		// Only display the error if the user is requesting completion via successive tabs (so distinct completions are guaranteed to be displayed)
		if compTypeArg.Get(d) == 63 { /* code 63 = '?' character */
			// Add newline so we're outputting stderr on a newline (and not line with cursor)
			o.Stderrf("\n%v", err)
			// Suggest non-overlapping strings (one space and one tab) so COMP_LINE is reprinted
			o.Stdoutf("\t\n \n")
		}
		return err
	}
	if len(g) > 0 {
		o.Stdoutf("%s\n", strings.Join(g, "\n"))
	}

	return nil
}

// separate method for testing
var (
	// EnvCacheVar is the environment variable pointing to the path for caching.
	// var so it can be modified for tests
	EnvCacheVar = "LEEP_CLI_CACHE"
	// getCache is a variable function so it can be swapped in tests
	getCache = func() (*cache.Cache, error) {
		return cache.FromEnvVar(EnvCacheVar)
	}
)

func load(cli CLI) error {
	ck := cacheKey(cli)
	cash, err := getCache()
	_, err = cash.GetStruct(ck, cli)
	return err
}

type sourcerer struct {
	clis              map[string]CLI
	cliArg            *command.MapArgument[string, CLI]
	sl                string
	printedUsageError bool
	opts              *compiledOpts
	forAutocomplete   bool
}

func (*sourcerer) UnmarshalJSON(jsn []byte) error { return nil }
func (*sourcerer) Changed() bool                  { return false }
func (*sourcerer) Setup() []string                { return nil }
func (*sourcerer) Name() string {
	return "_internal_sourcerer"
}

var (
	loadOnlyFlag = command.BoolFlag("load-only", 'l', "If set to true, the binaries are assumed to exist and only the aliases and completion setups are generated")
)

func (s *sourcerer) Node() command.Node {
	loadCLIArg := command.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
		// TODO: Test this
		return load(s.cliArg.Get(d))
	})
	return &command.BranchNode{
		Branches: map[string]command.Node{
			"autocomplete": command.SerialNodes(
				s.cliArg,
				loadCLIArg,
				compTypeArg,
				compPointArg,
				compLineArg,
				autocompletePassthroughArgs,
				&command.ExecutorProcessor{F: s.autocompleteExecutor},
			),
			"usage": command.SerialNodes(
				s.cliArg,
				loadCLIArg,
				passthroughArgs,
				command.SimpleProcessor(s.usageExecutor, nil),
			),
			"execute": command.SerialNodes(
				s.cliArg,
				loadCLIArg,
				fileArg,
				passthroughArgs,
				&command.ExecutorProcessor{F: s.executeExecutor},
			),
			"source": command.SerialNodes(
				command.FlagProcessor(
					loadOnlyFlag,
				),
				targetNameArg,
				&command.ExecutorProcessor{F: s.generateFile},
			),
		},
		HideUsage: true,
		Default: command.SerialNodes(
			// Just eat the remaining args
			// command.ListArg[string]("UNUSED", "", 0, command.UnboundedList),
			&command.ExecutorProcessor{func(o command.Output, d *command.Data) error {
				// Add echo so it's a comment if included in sourced output
				o.Stderrf("echo %q", "Executing a sourcerer.CLI directly through `go run` is tricky. Either generate a CLI or use the `goleep` command to directly run the file.")
				return nil
			}},
		),
	}
}

func (s *sourcerer) usageExecutor(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
	cli := s.cliArg.Get(d)
	ed.Executor = append(ed.Executor, func(o command.Output, d *command.Data) error {
		n := cli.Node()
		u, err := command.Use(n, command.ParseExecuteArgs(passthroughArgs.Get(d)))
		if err != nil {
			o.Err(err)
			if command.IsUsageError(err) {
				command.ShowUsageAfterError(n, o)
			}
			return err
		}
		o.Stdout(u.String())
		return nil
	})
	return nil
}

type Option interface {
	modifyCompiledOpts(*compiledOpts)
}

type simpleOption func(*compiledOpts)

func (so *simpleOption) modifyCompiledOpts(co *compiledOpts) {
	(*so)(co)
}

func multiOpts(opts ...Option) Option {
	so := simpleOption(func(co *compiledOpts) {
		for _, o := range opts {
			o.modifyCompiledOpts(co)
		}
	})
	return &so
}

type compiledOpts struct {
	aliasers map[string]*Aliaser
}

// Source generates the bash source file for a list of CLIs.
func Source(clis []CLI, opts ...Option) int {
	o := command.NewOutput()
	defer o.Close()
	if source(clis, os.Args[1:], o, opts...) != nil {
		return 1
	}
	return 0
}

// Separate method used for testing.
func source(clis []CLI, osArgs []string, o command.Output, opts ...Option) error {
	sl, err := getSourceLoc()
	if err != nil {
		return o.Annotate(err, "failed to get source location")
	}

	cos := &compiledOpts{
		aliasers: map[string]*Aliaser{},
	}
	for _, oi := range opts {
		oi.modifyCompiledOpts(cos)
	}

	cliMap := map[string]CLI{}
	for _, c := range clis {
		cliMap[c.Name()] = c
	}

	s := &sourcerer{
		clis:   cliMap,
		cliArg: command.MapArg("CLI", "", cliMap, false),
		sl:     sl,
		opts:   cos,
	}

	// Sourcerer is always executed. Its execution branches into the relevant CLI's
	// execution/autocomplete/usage path.
	d := &command.Data{
		Values: map[string]interface{}{
			s.cliArg.Name(): s,
			// Don't need execute file here
			passthroughArgs.Name(): osArgs,
		},
	}

	return s.executeExecutor(o, d)
}

var (
	// Stubbed out for testing purposes
	getSourceLoc = func() (string, error) {
		_, sourceLocation, _, ok := runtime.Caller(3)
		if !ok {
			return "", fmt.Errorf("failed to fetch caller")
		}
		return sourceLocation, nil
	}
)

var (
	// getExecuteFile returns the name of the file to which execute file logic is written.
	// It is a separte function so it can be stubbed out for testing.
	getExecuteFile = func(filename string) string {
		return fmt.Sprintf("%s/bin/_custom_execute_%s", os.Getenv("GOPATH"), filename)
	}
)

func (s *sourcerer) generateFile(o command.Output, d *command.Data) error {
	filename := "leep-frog-source"
	if d.Has(targetNameArg.Name()) {
		filename = d.String(targetNameArg.Name())
	}

	// cd into the directory of the file that is actually calling this and install dependencies.
	if !loadOnlyFlag.Get(d) {
		o.Stdoutf(generateBinary, s.sl, filename)
	}

	// define the autocomplete function
	o.Stdoutf(autocompleteFunction, filename, filename)

	// The execute logic is put in an actual file so it can be used by other
	// bash environments that don't actually source sourcerer-related commands.
	efc := fmt.Sprintf(executeFileContents, filename, filename, filename)

	f, err := os.OpenFile(getExecuteFile(filename), os.O_WRONLY|os.O_CREATE, command.CmdOS.DefaultFilePerm())
	if err != nil {
		return o.Stderrf("failed to open execute function file: %v\n", err)
	}

	if _, err := f.WriteString(efc); err != nil {
		return o.Stderrf("failed to write to execute function file: %v\n", err)
	}

	cliArr := maps.Values(s.clis)
	sort.SliceStable(cliArr, func(i, j int) bool { return cliArr[i].Name() < cliArr[j].Name() })
	for _, cli := range cliArr {
		alias := cli.Name()

		aliasCommand := fmt.Sprintf(aliasFormat, alias, filename, alias)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			o.Stdoutf(setupFunctionFormat, setupFunctionName, strings.Join(scs, "  \n  "))
			aliasCommand = fmt.Sprintf(aliasWithSetupFormat, alias, setupFunctionName, filename, alias)
		}

		o.Stdoutln(aliasCommand)

		// We sort ourselves, hence the no sort.
		o.Stdoutf("complete -F _custom_autocomplete_%s %s %s\n", filename, NosortString(), alias)
	}

	AliasSourcery(o, maps.Values(s.opts.aliasers)...)

	return nil
}

var (
	// NosortString returns the complete option to ignore sorting.
	// It returns nothing if the IGNORE_NOSORT environment variable is set.
	NosortString = func() string {
		if _, ignore := os.LookupEnv("IGNORE_NOSORT"); ignore {
			return ""
		}
		return "-o nosort"
	}
)

func save(c CLI) error {
	ck := cacheKey(c)
	cash, err := getCache()
	if err != nil {
		return err
	}

	if err := cash.PutStruct(ck, c); err != nil {
		return fmt.Errorf("failed to save cli %q: %v", c.Name(), err)
	}
	return nil
}

func cacheKey(cli CLI) string {
	return fmt.Sprintf("leep-frog-cache-key-%s.json", cli.Name())
}
