// package sourcerer sources CLI commands in a shell environment.
// See the `main` function in github.com/leep-frog/command/examples/source.go
// for an example of how to define a source file that uses this.
package sourcerer

import (
	"fmt"
	"os"
	"os/exec"
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

	GlobalAutocompleteForAliasFunction = strings.Join([]string{
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
		`  $GOPATH/bin/_%s_runner execute $tmpFile "$@"`,
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
	cliArg          = command.Arg[string]("CLI", "Name of the CLI command to use")
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
	cli, err := s.getCLI(d.String(cliArg.Name()))
	if err != nil {
		return output.Err(err)
	}

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
		if command.IsUsageError(err) && !s.printedUsageError && !s.forAutocomplete {
			s.printedUsageError = true
			output.Stderrln(command.ShowUsageAfterError(n))
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
	cli, err := s.getCLI(d.String(cliArg.Name()))
	if err != nil {
		return o.Err(err)
	}

	g, err := command.Autocomplete(cli.Node(), compLineArg.Get(d), autocompletePassthroughArgs.Get(d))
	if err != nil {
		// Only display the error if the user is requesting completion via successive tabs (so distinct completions are guaranteed to be displayed)
		if compTypeArg.Get(d) == 63 { /* code 63 = '?' character */
			// Add newline so we're outputting stderr on a newline (and not line with cursor)
			o.Stderrf("\n%v", err)
			// Also suggest non-overlapping strings so comp line is reprinted
			o.Stdoutf(" \n\t\n")
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

// Run loads and runs the provided CLI. This is especially useful
// when used in conjunction with the `goleep` tool. The return value is an exit status.
func Run(cli CLI) int {
	o := command.NewOutput()
	if err := load(cli); err != nil {
		o.Err(err)
		return 1
	}
	o.Close()
	return 0
}

func load(cli CLI) error {
	ck := cacheKey(cli)
	cash, err := getCache()
	_, err = cash.GetStruct(ck, cli)
	return err
}

type sourcerer struct {
	clis              []CLI
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

func (s *sourcerer) getCLI(cli string) (CLI, error) {
	if cli == s.Name() {
		return s, nil
	}

	for _, c := range s.clis {
		if c.Name() == cli {
			if err := load(c); err != nil {
				return nil, fmt.Errorf("failed to load cli: %v", err)
			}
			return c, nil
		}
	}
	return nil, fmt.Errorf("unknown CLI %q", cli)
}

func (s *sourcerer) Node() command.Node {
	generateBinaryNode := command.SerialNodes(
		targetNameArg,
		&command.ExecutorProcessor{F: s.generateFile},
	)

	return &command.BranchNode{
		Branches: map[string]command.Node{
			"autocomplete": command.SerialNodes(
				cliArg,
				compTypeArg,
				compPointArg,
				compLineArg,
				autocompletePassthroughArgs,
				&command.ExecutorProcessor{F: s.autocompleteExecutor},
			),
			"usage": command.SerialNodes(
				cliArg,
				&command.ExecutorProcessor{F: s.usageExecutor},
			),
			"execute": command.SerialNodes(
				fileArg,
				cliArg,
				passthroughArgs,
				&command.ExecutorProcessor{F: s.executeExecutor},
			),
		},
		Default: generateBinaryNode,
	}
}

func (s *sourcerer) usageExecutor(o command.Output, d *command.Data) error {
	cli, err := s.getCLI(d.String(cliArg.Name()))
	if err != nil {
		return o.Err(err)
	}
	o.Stdoutln(command.GetUsage(cli.Node()).String())
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

	s := &sourcerer{
		clis: clis,
		sl:   sl,
		opts: cos,
	}

	// Sourcerer is always executed. Its execution branches into the relevant CLI's
	// execution/autocomplete/usage path.
	d := &command.Data{
		Values: map[string]interface{}{
			cliArg.Name(): s.Name(),
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
	o.Stdoutf(generateBinary, s.sl, filename)

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

	sort.SliceStable(s.clis, func(i, j int) bool { return s.clis[i].Name() < s.clis[j].Name() })
	for _, cli := range s.clis {
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

// SimpleCommands returns a list of CLIs that are simply aliased
// to a bash command.
func SimpleCommands(m map[string]string) []CLI {
	cs := []CLI{}
	for name, cmd := range m {
		cs = append(cs, &bashCLI{name, cmd})
	}
	return cs
}

type bashCLI struct {
	name          string
	commandString string
}

func (bc *bashCLI) Changed() bool              { return false }
func (bc *bashCLI) Setup() []string            { return nil }
func (bc *bashCLI) UnmarshalJSON([]byte) error { return nil }
func (bc *bashCLI) Name() string               { return bc.name }
func (bc *bashCLI) Node() command.Node {
	return command.SerialNodes(&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
		cmd := exec.Command("bash", "-c", bc.commandString)
		cmd.Stdout = command.StdoutWriter(o)
		cmd.Stderr = command.StderrWriter(o)
		return cmd.Run()
	}})
}
