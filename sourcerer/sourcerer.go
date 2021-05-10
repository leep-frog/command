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
	
		$GOPATH/bin/leep-frog-source autocomplete $COMP_CWORD.$COMP_POINT $COMP_LINE > $tFile
		local IFS=$'\n'
		COMPREPLY=( $(cat $tFile) )
		rm $tFile
	}
	`

	// executeFunction defines a bash function for CLI execution.
	executeFunction = `
	function _custom_execute {
		# tmpFile is the file to which we write ExecuteData.Executable
		#tmpFile=$(mktemp)
		#chmod +x $tmpFile
		#$GOPATH/bin/leep-frog-source execute $tmpFile "$@"
		#if [[ ! -z $LEEP_FROG_DEBUG ]]; then
		#	echo Executing: $(cat $tmpFile)
		#fi
		#source $tmpFile
		$GOPATH/bin/leep-frog-source execute "$@"
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
	// passed in as data.Values[command.SetupArgName]
	Setup() []string
}

func execute(cli CLI, args []string) {
	output := command.NewOutput()
	eData, err := command.Execute(cli.Node(), command.ParseExecuteArgs(args), output)
	output.Close()
	if err != nil {
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

	f, err := ioutil.TempFile("", "leep-frog-executable")
	defer f.Close()
	if err != nil {
		log.Fatalf("failed to create temporary file")
	}

	for _, ex := range eData.Executable {
		if _, err := f.WriteString(strings.ReplaceAll(strings.Join(ex, " "), "\\", "\\\\")); err != nil {
			log.Fatalf("failed to write to execute file: %v", err)
		}
	}

	cmd := exec.Command("source", f.Name())
	if err := cmd.Run(); err != nil {
		log.Fatalf("failed to run executable command: %v", err)
	}
}

func autocomplete(cli CLI, cword, cpoint int, args []string) {
	// TODO: should cword/cpoint happen here or in command.Autocomplete function?
	// Probably the latter so it can handle what happens if the cursor info isn't
	// at the end?
	if cword > len(args) {
		args = append(args, "")
	}
	// TODO: use cpoint to determine if we're completing in the middle of a word.
	// careful about spaces though.p
	g := command.Autocomplete(cli.Node(), args)
	fmt.Printf("%s\n", strings.Join(g, "\n"))

	if len(os.Getenv("LEEP_FROG_DEBUG")) > 0 {
		debugFile, err := os.Create("leepFrogDebug.txt")
		if err != nil {
			log.Fatalf("Unable to create file: %v", err)
		}
		if _, err := debugFile.WriteString(fmt.Sprintf("%d %d %d %s\n", len(args), cword, cpoint, strings.Join(args, "_"))); err != nil {
			log.Fatalf("Unable to write to file: %v", err)
		}
		if _, err := debugFile.WriteString(fmt.Sprintf("%d %d %d %s\n", len(g), cword, cpoint, strings.Join(g, "_"))); err != nil {
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
	var cliName string
	switch opType {
	case "autocomplete":
		cliName = os.Args[3]
	case "execute":
		cliName = os.Args[2]
	}

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
		cursorInfo := strings.Split(os.Args[2], ".")
		cword, err := strconv.Atoi(cursorInfo[0])
		if err != nil {
			log.Fatalf("Failed to convert COMP_CWORD: %v", err)
		}
		cpoint, err := strconv.Atoi(cursorInfo[1])
		if err != nil {
			log.Fatalf("Failed to convert COMP_POINT: %v", err)
		}
		autocomplete(cli, cword, cpoint, os.Args[4:])
	case "execute":
		// TODO: change filename to file writer?
		// (cli, filename (for ExecuteData.Exectuable), args)
		execute(cli, os.Args[3:])
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
