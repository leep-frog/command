// package sourcerer sources CLI commands in a shell environment.
// See the `main` function in github.com/leep-frog/command/examples/source.go
// for an example of how to define a source file that uses this.
package sourcerer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
)

var (
	// The file that was used to create the source file will also
	// be used for executing and autocompleting cli commands.
	generateBinary = strings.Join([]string{
		"pushd . > /dev/null",
		`cd "$(dirname %s)"`,
		"go build -o $GOPATH/bin/_%s_runner",
		"popd > /dev/null",
	}, "\n")

	// autocompleteFunction defines a bash function for CLI autocompletion.
	autocompleteFunction = strings.Join([]string{
		"function _custom_autocomplete_%s {",
		`  tFile=$(mktemp)`,
		`  $GOPATH/bin/_%s_runner autocomplete ${COMP_WORDS[0]} $COMP_POINT "$COMP_LINE" > $tFile`,
		`  local IFS=$'\n'`,
		`  COMPREPLY=( $(cat $tFile) )`,
		`  rm $tFile`,
		"}",
	}, "\n")

	// executeFunction defines a bash function for CLI execution.
	executeFunction = strings.Join([]string{
		`function _custom_execute_%s {`,
		`  # tmpFile is the file to which we write ExecuteData.Executable`,
		`  tmpFile=$(mktemp)`,
		`  $GOPATH/bin/_%s_runner execute $tmpFile "$@"`,
		`  source $tmpFile`,
		`  if [ -z "$LEEP_FROG_DEBUG" ]`,
		`  then`,
		`    rm $tmpFile`,
		`  else`,
		`    echo $tmpFile`,
		`  fi`,
		`}`,
	}, "\n")

	usageFunction = strings.Join([]string{
		"function mancli {",
		"  # Extract the custom execute function so that this function",
		"  # can work regardless of file name",
		`  file="$(type $1 | head -n 1 | grep "is aliased to ._custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
		`  "$GOPATH/bin/_${file}_runner" usage $@`,
		"}",
	}, "\n")

	// setupFunctionFormat is used to run setup functions prior to a CLI command execution.
	setupFunctionFormat = strings.Join([]string{
		`function %s {`,
		`  %s`,
		"}",
	}, "\n")

	// aliasWithSetupFormat is an alias definition template for commands that require a setup function.
	aliasWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && _custom_execute_%s %s $o'"
	// aliasFormat is an alias definition template for commands that don't require a setup function.
	aliasFormat = "alias %s='_custom_execute_%s %s'"
)

var (
	cliArg          = command.Arg[string]("CLI", "Name of the CLI command to use")
	fileArg         = command.FileNode("FILE", "Temporary file for execution")
	targetNameArg   = command.OptionalArg[string]("TARGET_NAME", "The name of the created target in $GOPATH/bin")
	passthroughArgs = command.ListArg[string]("ARG", "Arguments that get passed through to relevant CLI command", 0, command.UnboundedList)
	compPointArg    = command.Arg[int]("COMP_POINT", "COMP_POINT variable from bash complete function")
	compLineArg     = command.Arg[string]("COMP_LINE", "COMP_LINE variable from bash complete function")
)

// CLI provides a way to construct CLIs in go, with tab-completion.
// Note, this has to be an interface (as opposed to a struct) because of the Load function.
type CLI interface {
	// We use json unmarshaling so if a CLI type wants custom marshaling/unmarshaling,
	// they just need to implement the json.Marshaler/Unmarshaler interface(s).

	// Name is the name of the alias command to use for this CLI.
	Name() string
	// Node returns the command node for the CLI. This is where the CLI's logic lives.
	Node() *command.Node
	// Changed indicates whether or not the CLI has changed after execution.
	// If true, the CLI's value will be save to the cache.
	Changed() bool
	// Setup describes a set of commands that will be run in bash prior to the CLI.
	// The output from the commands will be stored in a file whose name will be
	// passed in as data[command.SetupArgName]
	Setup() []string
}

// Returns if there was an error
func (s *sourcerer) executeExecutor(output command.Output, d *command.Data) error {
	cli, err := s.getCLI(d.String(cliArg.Name()))
	if err != nil {
		return output.Err(err)
	}

	executeFile := d.String(fileArg.Name())
	args := d.StringList(passthroughArgs.Name())

	eData, err := command.Execute(cli.Node(), command.ParseExecuteArgs(args), output)
	if err != nil {
		if command.IsUsageError(err) && !s.printedUsageError {
			s.printedUsageError = true
			u := command.GetUsage(cli.Node())
			output.Stderr(u.String())
		}
		// Commands are responsible for printing out error messages so
		// we just return if there are any issues here
		return err
	}

	// Save the CLI if it has changed.
	if cli.Changed() {
		if err := save(cli); err != nil {
			return output.Stderrf("failed to save cli data: %v", err)
		}
	}

	// Run the executable file if relevant.
	if eData == nil || len(eData.Executable) == 0 {
		return nil
	}

	for i, line := range eData.Executable {
		eData.Executable[i] = strings.ReplaceAll(line, `\`, `\\`)
	}

	f, err := os.OpenFile(executeFile, os.O_WRONLY, 0644)
	if err != nil {
		return output.Stderrf("failed to open file: %v", err)
	}

	if command.DebugMode() {
		return output.Stderrf("# Executable Contents")
	}
	v := strings.Join(eData.Executable, "\n")
	if command.DebugMode() {
		output.Stderr(v)
	}

	if _, err := f.WriteString(v); err != nil {
		return output.Stderrf("failed to write to execute file: %v", err)
	}

	return nil
}

func (s *sourcerer) autocompleteExecutor(o command.Output, d *command.Data) error {
	cli, err := s.getCLI(d.String(cliArg.Name()))
	if err != nil {
		return o.Err(err)
	}

	cpoint := d.Int(compPointArg.Name())
	args := d.String(compLineArg.Name())[:cpoint]

	g := command.Autocomplete(cli.Node(), args)
	o.Stdoutf("%s\n", strings.Join(g, "\n"))

	if len(os.Getenv("LEEP_FROG_DEBUG")) > 0 {
		debugFile, err := os.Create("leepFrogDebug.txt")
		if err != nil {
			return o.Stderrf("Unable to create file: %v", err)
		}
		if _, err := debugFile.WriteString(fmt.Sprintf("%d %s\n", len(args), strings.ReplaceAll(args, " ", "_"))); err != nil {
			return o.Stderrf("Unable to write to file: %v", err)
		}
		if _, err := debugFile.WriteString(fmt.Sprintf("%d %s\n", len(g), strings.Join(g, "_"))); err != nil {
			return o.Stderrf("Unable to write to file: %v", err)
		}
		debugFile.Close()
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
		return cache.New(EnvCacheVar)
	}
)

func load(cli CLI) error {
	ck := cacheKey(cli)
	cash, err := getCache()
	if err != nil {
		return err
	} else if bytes, fileExists, err := cash.GetBytes(ck); err != nil {
		return fmt.Errorf("failed to load cli %q: %v", cli.Name(), err)
	} else if fileExists && bytes != nil {
		return json.Unmarshal(bytes, cli)
	}
	return nil
}

type sourcerer struct {
	clis              []CLI
	sl                string
	printedUsageError bool
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

func (s *sourcerer) Node() *command.Node {
	generateBinaryNode := command.SerialNodes(
		targetNameArg,
		command.ExecutorNode(s.generateFile),
	)

	return command.BranchNode(map[string]*command.Node{
		"autocomplete": command.SerialNodes(
			cliArg,
			compPointArg,
			compLineArg,
			command.ExecuteErrNode(s.autocompleteExecutor),
		),
		"usage": command.SerialNodes(
			cliArg,
			command.ExecuteErrNode(s.usageExecutor),
		),
		"execute": command.SerialNodes(
			fileArg,
			cliArg,
			passthroughArgs,
			command.ExecuteErrNode(s.executeExecutor),
		),
	}, generateBinaryNode)
}

func (s *sourcerer) usageExecutor(o command.Output, d *command.Data) error {
	cli, err := s.getCLI(d.String(cliArg.Name()))
	if err != nil {
		return o.Err(err)
	}
	o.Stdout(command.GetUsage(cli.Node()).String())
	return nil
}

// Source generates the bash source file for a list of CLIs.
func Source(clis ...CLI) int {
	o := command.NewOutput()
	defer o.Close()
	if source(clis, os.Args[1:], o) != nil {
		return 1
	}
	return 0
}

// Separate method used for testing.
func source(clis []CLI, osArgs []string, o command.Output) error {
	sl, err := getSourceLoc()
	if err != nil {
		return o.Annotate(err, "failed to get source location")
	}
	s := &sourcerer{
		clis: clis,
		sl:   sl,
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

func (s *sourcerer) generateFile(o command.Output, d *command.Data) {
	filename := "leep-frog-source"
	if d.Has(targetNameArg.Name()) {
		filename = d.String(targetNameArg.Name())
	}

	// cd into the directory of the file that is actually calling this and install dependencies.
	o.Stdoutf(generateBinary, s.sl, filename)

	// define the autocomplete function
	o.Stdoutf(autocompleteFunction, filename, filename)

	// define the execute function
	o.Stdoutf(executeFunction, filename, filename)

	// define the usage function
	o.Stdout(usageFunction)

	sort.SliceStable(s.clis, func(i, j int) bool { return s.clis[i].Name() < s.clis[j].Name() })
	for _, cli := range s.clis {
		alias := cli.Name()

		aliasCommand := fmt.Sprintf(aliasFormat, alias, filename, alias)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			o.Stdoutf(setupFunctionFormat, setupFunctionName, strings.Join(scs, "  \n  "))
			aliasCommand = fmt.Sprintf(aliasWithSetupFormat, alias, setupFunctionName, filename, alias)
		}

		o.Stdout(aliasCommand)

		// We sort ourselves, hence the no sort.
		o.Stdoutf("complete -F _custom_autocomplete_%s -o nosort %s", filename, alias)
	}
}

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
	return fmt.Sprintf("leep-frog-cache-key-%s", cli.Name())
}

// TODO: add these to clis.go and look into (potential) performance issues
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
func (bc *bashCLI) Node() *command.Node {
	return command.SerialNodes(command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
		cmd := exec.Command("bash", "-c", bc.commandString)
		cmd.Stdout = command.StdoutWriter(o)
		cmd.Stderr = command.StderrWriter(o)
		return cmd.Run()
	}))
}
