// package sourcerer sources CLI commands in a shell environment.
// See the `main` function in github.com/leep-frog/command/examples/source.go
// for an example of how to define a source file that uses this.
package sourcerer

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/cache"
)

// TODO: test this package or move as much as relevant into the command
// package and test from there.

const (
	// The file that was used to create the source file will also
	// be used for executing and autocompleting cli commands.
	generateBinary = `
	pushd . > /dev/null
	cd "$(dirname %s)"
	# TODO: this won't work if two separate source files are used.
	go build -o $GOPATH/bin/leep-frog-source 
	popd > /dev/null
	`

	// autocompleteFunction defines a bash function for CLI autocompletion.
	autocompleteFunction = `
	function _custom_autocomplete {
		tFile=$(mktemp)
	
		$GOPATH/bin/leep-frog-source autocomplete $COMP_POINT ${COMP_WORDS[0]} "$COMP_LINE" > $tFile
		local IFS=$'\n'
		COMPREPLY=( $(cat $tFile) )
		rm $tFile
	}
	`

	// executeFunction defines a bash function for CLI execution.
	// TODO: write this to file, so if a user wants to use this without
	// sourcing, they can just run the file. At least for generic stuff
	// like temp file creation and debug checking.
	// Some way for a user to simply run things like:
	// $GOPATH/bin/leep-frog-execute rp|e|...
	// $GOPATH/bin/leep-frog-ls
	executeFunction = `
	function _custom_execute {
		# tmpFile is the file to which we write ExecuteData.Executable
		tmpFile=$(mktemp)
		$GOPATH/bin/leep-frog-source execute $tmpFile "$@"
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
		$GOPATH/bin/leep-frog-source usage ignoreThis "$@"
	}
	`

	// setupFunctionFormat is used to run setup functions prior to a CLI command execution.
	setupFunctionFormat = `
	function %s {
		%s
	}
	`

	// aliasWithSetupFormat is an alias definition template for commands that require a setup function.
	aliasWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && _custom_execute %s $o'\n"
	// aliasFormat is an alias definition template for commands that don't require a setup function.
	aliasFormat = "alias %s='_custom_execute %s'\n"
)

// CLI provides a way to construct CLIs in go, with tab-completion.
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

func execute(cli CLI, executeFile string, args []string) {
	output := command.NewOutput()
	eData, err := command.Execute(cli.Node(), command.ParseExecuteArgs(args), output)
	output.Close()
	if err != nil {
		if command.IsUsageError(err) {
			u := command.GetUsage(cli.Node())
			output.Stderr(u.String())
		}
		// Commands are responsible for printing out error messages so
		// we just return if there are any issues here
		os.Exit(1)
	}

	// Save the CLI if it has changed.
	if cli.Changed() {
		if err := save(cli); err != nil {
			log.Fatalf("failed to save cli data: %v", err)
		}
	}

	// Run the executable file if relevant.
	if eData == nil || len(eData.Executable) == 0 {
		return
	}

	f, err := os.OpenFile(executeFile, os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}

	if command.DebugMode() {
		fmt.Println("# Executable Contents")
	}
	v := strings.Join(eData.Executable, "\n")
	if command.DebugMode() {
		fmt.Println(v)
	}

	if _, err := f.WriteString(v); err != nil {
		log.Fatalf("failed to write to execute file: %v", err)
	}
}

func autocomplete(cli CLI, args string) {
	g := command.Autocomplete(cli.Node(), args)
	fmt.Printf("%s\n", strings.Join(g, "\n"))

	if len(os.Getenv("LEEP_FROG_DEBUG")) > 0 {
		debugFile, err := os.Create("leepFrogDebug.txt")
		if err != nil {
			log.Fatalf("Unable to create file: %v", err)
		}
		if _, err := debugFile.WriteString(fmt.Sprintf("%d %s\n", len(args), strings.ReplaceAll(args, " ", "_"))); err != nil {
			log.Fatalf("Unable to write to file: %v", err)
		}
		if _, err := debugFile.WriteString(fmt.Sprintf("%d %s\n", len(g), strings.Join(g, "_"))); err != nil {
			log.Fatalf("Unable to write to file: %v", err)
		}
		debugFile.Close()
	}
}

func load(cli CLI) error {
	ck := cacheKey(cli)
	cash := &cache.Cache{}
	s, err := cash.Get(ck)
	if err != nil {
		return fmt.Errorf("failed to load cli %q: %v", cli.Name(), err)
	}
	return cli.Load(s)
}

// Source generates the bash source file for a list of CLIs.
func Source(clis ...CLI) {
	if len(os.Args) <= 1 {
		generateFile(clis...)
		return
	}

	if len(os.Args) < 3 {
		log.Fatalf("Not enough arguments provided to leep-frog function")
	}

	opType := os.Args[1]
	cliName := os.Args[3]

	var cli CLI
	for _, c := range clis {
		if c.Name() == cliName {
			cli = c
			break
		}
	}

	if cli == nil {
		log.Fatalf("attempting to execute unknown CLI %q", cliName)
	}

	if err := load(cli); err != nil {
		log.Fatalf("failed to load cli: %v", err)
	}

	switch opType {
	case "autocomplete":
		cpoint, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Failed to convert COMP_POINT: %v", err)
		}

		autocomplete(cli, os.Args[4][:cpoint])
	case "execute":
		// TODO: change filename to file writer?
		// (cli, filename (for ExecuteData.Exectuable), args)
		execute(cli, os.Args[2], os.Args[4:])
	case "usage":
		fmt.Println(command.GetUsage(cli.Node()))
	default:
		log.Fatalf("unknown process: %v", os.Args)
	}
}

func generateFile(clis ...CLI) {
	f, err := ioutil.TempFile("", "golang-cli-source")
	if err != nil {
		log.Fatalf("failed to create tmp file: %v", err)
	}

	_, sourceLocation, _, ok := runtime.Caller(2)
	if !ok {
		// TODO: return error everywhere so we can test?
		log.Fatalf("failed to fetch caller")
	}

	// cd into the directory of the file that is actually calling this and install dependencies.
	if _, err := f.WriteString(fmt.Sprintf(generateBinary, sourceLocation)); err != nil {
		log.Fatalf("failed to write binary generator code to file: %v", err)
	}

	// define the autocomplete function
	if _, err := f.WriteString(autocompleteFunction); err != nil {
		log.Fatalf("failed to write autocomplete function to file: %v", err)
	}

	// define the execute function
	if _, err := f.WriteString(executeFunction); err != nil {
		log.Fatalf("failed to write autocomplete function to file: %v", err)
	}

	for _, cli := range clis {
		alias := cli.Name()

		aliasCommand := fmt.Sprintf(aliasFormat, alias, alias)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			if _, err := f.WriteString(fmt.Sprintf(setupFunctionFormat, setupFunctionName, strings.Join(scs, "  \n"))); err != nil {
				log.Fatalf("failed to write setup command to file: %v", err)
			}
			aliasCommand = fmt.Sprintf(aliasWithSetupFormat, alias, setupFunctionName, alias)
		}

		if _, err := f.WriteString(aliasCommand); err != nil {
			log.Fatalf("failed to write alias to file: %v", err)
		}

		// We sort ourselves, hence the no sort.
		if _, err := f.WriteString(fmt.Sprintf("complete -F _custom_autocomplete -o nosort %s\n", alias)); err != nil {
			log.Fatalf("failed to write autocomplete command to file: %v", err)
		}
	}

	f.Close()
	fmt.Printf(f.Name())
}

func save(c CLI) error {
	ck := cacheKey(c)
	cash := &cache.Cache{}
	if err := cash.PutStruct(ck, c); err != nil {
		return fmt.Errorf("failed to save cli %q: %v", c.Name(), err)
	}
	return nil
}

func cacheKey(cli CLI) string {
	return fmt.Sprintf("leep-frog-cache-key-%s", cli.Name())
}

// TODO: add these to clis.go and look into (potential) performance issues
// might need to allow for multiple binaries (binary per CLI?)
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
