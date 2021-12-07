// package sourcerer sources CLI commands in a shell environment.
// See the `main` function in github.com/leep-frog/command/examples/source.go
// for an example of how to define a source file that uses this.
package sourcerer

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
)

const (
	// The file that was used to create the source file will also
	// be used for executing and autocompleting cli commands.
	generateBinary = `
	pushd . > /dev/null
	cd "$(dirname %s)"
	go build -o $GOPATH/bin/%s
	popd > /dev/null
	`

	// autocompleteFunction defines a bash function for CLI autocompletion.
	autocompleteFunction = `
	function _custom_autocomplete_%s {
		tFile=$(mktemp)
		$GOPATH/bin/%s autocomplete ${COMP_WORDS[0]} $COMP_POINT "$COMP_LINE" > $tFile
		local IFS=$'\n'
		COMPREPLY=( $(cat $tFile) )
		rm $tFile
	}
	`

	// executeFunction defines a bash function for CLI execution.
	executeFunction = `
	function _custom_execute_%s {
		# tmpFile is the file to which we write ExecuteData.Executable
		tmpFile=$(mktemp)
		$GOPATH/bin/%s execute $tmpFile "$@"
		source $tmpFile
		if [ -z "$LEEP_FROG_DEBUG" ]
		then
		  rm $tmpFile
		else
		  echo $tmpFile
		fi
	}
	`

	usageFunction = `
	function mancli {
		$GOPATH/bin/%s usage "$@"
	}
	`

	// setupFunctionFormat is used to run setup functions prior to a CLI command execution.
	setupFunctionFormat = `
	function %s {
		%s
	}
	`

	// aliasWithSetupFormat is an alias definition template for commands that require a setup function.
	aliasWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && _custom_execute_%s %s $o'\n"
	// aliasFormat is an alias definition template for commands that don't require a setup function.
	aliasFormat = "alias %s='_custom_execute_%s %s'\n"
)

// ParseArgs executes the provided CLI
func RunNodes(n *command.Node) error {
	o := command.NewOutput()
	// Don't care about execute data
	if _, err := command.Execute(n, command.ParseExecuteArgs(os.Args[1:]), o); err != nil {
		if command.IsUsageError(err) {
			o.Stderr(command.GetUsage(n).String())
		}
		return err
	}
	return nil
}

var (
	cliArg          = command.StringNode("CLI", "Name of the CLI command to use")
	fileArg         = command.FileNode("FILE", "Temporary file for execution")
	targetNameArg   = command.OptionalStringNode("TARGET_NAME", "The name of the created target in $GOPATH/bin")
	passthroughArgs = command.StringListNode("ARG", "Arguments that get passed through to relevant CLI command", 0, command.UnboundedList)
	compPointArg    = command.IntNode("COMP_POINT", "COMP_POINT variable from bash complete function")
	compLineArg     = command.StringNode("COMP_LINE", "COMP_LINE variable from bash complete function")
)

// CLI provides a way to construct CLIs in go, with tab-completion.
// Note, this has to be an interface (as opposed to a struct) because of the Load function.
type CLI interface {
	// Name is the name of the alias command to use for this CLI.
	Name() string
	// Load loads a json string into the CLI object.
	Load(json string) error
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
	getCache = func() *cache.Cache {
		return cache.NewCache()
	}
)

func load(cli CLI) error {
	ck := cacheKey(cli)
	cash := getCache()
	s, err := cash.Get(ck)
	if err != nil {
		return fmt.Errorf("failed to load cli %q: %v", cli.Name(), err)
	}
	return cli.Load(s)
}

type sourcerer struct {
	clis              []CLI
	printedUsageError bool
}

func (*sourcerer) Load(jsn string) error { return nil }
func (*sourcerer) Changed() bool         { return false }
func (*sourcerer) Setup() []string       { return nil }
func (*sourcerer) Name() string {
	return "sb"
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
			command.ExecutorNode(s.autocompleteExecutor),
		),
		"usage": command.SerialNodes(
			cliArg,
			command.ExecutorNode(s.usageExecutor),
		),
		"execute": command.SerialNodes(
			fileArg,
			cliArg,
			passthroughArgs,
			command.ExecutorNode(s.executeExecutor),
		),
	}, generateBinaryNode, true)
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
func Source(clis ...CLI) {
	o := command.NewOutput()
	source(clis, os.Args[1:], o)
	o.Close()
}

// Separate method used for testing.
func source(clis []CLI, osArgs []string, o command.Output) {
	s := &sourcerer{
		clis: clis,
	}

	// Sourcerer is always executed. Its execution branches into the relevant CLI's
	// execution/autocomplete/usage path.
	d := &command.Data{
		Values: map[string]*command.Value{
			cliArg.Name(): command.StringValue(s.Name()),
			// Don't need execute file here
			passthroughArgs.Name(): command.StringListValue(osArgs...),
		},
	}

	s.executeExecutor(o, d)
}

var (
	// Stubbed out for testing purposes
	getSourceLoc = func() (string, error) {
		_, sourceLocation, _, ok := runtime.Caller(7)
		if !ok {
			return "", fmt.Errorf("failed to fetch caller")
		}
		return sourceLocation, nil
	}
)

func (s *sourcerer) generateFile(o command.Output, d *command.Data) error {
	f, err := ioutil.TempFile("", "golang-cli-source")
	if err != nil {
		return o.Stderrf("failed to create tmp file: %v", err)
	}

	filename := "leep-frog-source"
	if d.HasArg(targetNameArg.Name()) {
		filename = d.String(targetNameArg.Name())
	}

	sourceLocation, err := getSourceLoc()
	if err != nil {
		return o.Err(err)
	}

	// cd into the directory of the file that is actually calling this and install dependencies.
	if _, err := f.WriteString(fmt.Sprintf(generateBinary, sourceLocation, filename)); err != nil {
		return o.Stderrf("failed to write binary generator code to file: %v", err)
	}

	// define the autocomplete function
	if _, err := f.WriteString(fmt.Sprintf(autocompleteFunction, filename, filename)); err != nil {
		return o.Stderrf("failed to write autocomplete function to file: %v", err)
	}

	// define the execute function
	if _, err := f.WriteString(fmt.Sprintf(executeFunction, filename, filename)); err != nil {
		return o.Stderrf("failed to write autocomplete function to file: %v", err)
	}

	// define the usage function
	if _, err := f.WriteString(fmt.Sprintf(usageFunction, filename)); err != nil {
		return o.Stderrf("failed to write usage function to file: %v", err)
	}

	sort.SliceStable(s.clis, func(i, j int) bool { return s.clis[i].Name() < s.clis[j].Name() })
	for _, cli := range s.clis {
		alias := cli.Name()

		aliasCommand := fmt.Sprintf(aliasFormat, alias, filename, alias)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			if _, err := f.WriteString(fmt.Sprintf(setupFunctionFormat, setupFunctionName, strings.Join(scs, "  \n  "))); err != nil {
				return o.Stderrf("failed to write setup command to file: %v", err)
			}
			aliasCommand = fmt.Sprintf(aliasWithSetupFormat, alias, setupFunctionName, filename, alias)
		}

		if _, err := f.WriteString(aliasCommand); err != nil {
			return o.Stderrf("failed to write alias to file: %v", err)
		}

		// We sort ourselves, hence the no sort.
		if _, err := f.WriteString(fmt.Sprintf("complete -F _custom_autocomplete_%s -o nosort %s\n", filename, alias)); err != nil {
			return o.Stderrf("failed to write autocomplete command to file: %v", err)
		}
	}

	f.Close()
	o.Stdout(f.Name())
	return nil
}

func save(c CLI) error {
	ck := cacheKey(c)
	cash := getCache()
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

func (bc *bashCLI) Changed() bool     { return false }
func (bc *bashCLI) Setup() []string   { return nil }
func (bc *bashCLI) Load(string) error { return nil }
func (bc *bashCLI) Name() string      { return bc.name }
func (bc *bashCLI) Node() *command.Node {
	return command.SerialNodes(command.ExecutorNode(func(o command.Output, d *command.Data) error {
		cmd := exec.Command("bash", "-c", bc.commandString)
		cmd.Stdout = command.StdoutWriter(o)
		cmd.Stderr = command.StderrWriter(o)
		return cmd.Run()
	}))
}
