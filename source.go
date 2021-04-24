package command

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	generateBinary = `
	pushd . > /dev/null
	cd "$(dirname %s)"
	go install 
	popd > /dev/null
	`
	autocompleteFunction = `
	function _custom_autocomplete {
		tFile=$(mktemp)

		# Do we need to add a "shift" command since we are now passing in the binary name?
	
		# autocomplete might only need to just print newline-separated items to the file
		$1 autocomplete $COMP_CWORD $COMP_LINE > $tFile
		local IFS=$'\n'
		COMPREPLY=( $(cat $tFile) )
		rm $tFile
	}

	`

	executeFunction = `
	function _custom_execute {
		# TODO: might not be named source (if provided by user)
		binary="$1"
		shift

		tmpFile=$(mktemp)
		chmod +x $tmpFile
		$binary execute $tmpFile "$@"
		if [[ ! -z $LEEP_DEBUG ]]; then
			echo Executing: $(cat $tmpFile)
		fi
		source $tmpFile
	}
	`

	setupFunctionFormat = `
	function %s {
		%s
	}
	`

	aliasWithSetupFormat = "alias %s='o=$(mktemp) && %s > $o && _custom_execute %s $o'\n"
	aliasFormat          = "alias %s='_custom_execute %s'\n"
)

// CLI provides a way to construct CLIs in go, with tab-completion.
type CLI interface {
	Name() string
	Alias() string
	Load(string) error
	Node() *Node
	Changed() bool
	// Setup describes a set of commands that will be run in bash prior to the CLI.
	// The output from the commands will be stored in a file whose name will be
	// passed in as data.Values[command.SetupArgName]
	Setup() []string
}

func Sourcerer(cli CLI) {
	if os.Args[1] == "autocomplete" {
		SourceAutocomplete(cli)
		return
	}

	if os.Args[1] == "execute" {
		SourceExecute(cli)
		return
	}

	log.Fatalf("unknown process: %v", os.Args)
}

func SourceExecute(cli CLI) {
	// Args:
	// No setup command:   [binary execute $tmpFile $alias]
	// With setup command: [binary execute $tmpFile $alias $outputFile]
	executeFile := os.Args[2]
	args := os.Args[3:]

	if err := SourceLoad(cli); err != nil {
		log.Fatalf("failed to load cli: %v", err)
	}

	// TODO: add oi.
	output := NewOutput()
	eData, err := Execute(getNode(cli), ParseExecuteArgs(args), output)
	output.Close()
	if err != nil {
		// commands are responsible for printing out error messages so
		// we just return if there are any issues here
		os.Exit(1)
	}

	if cli.Changed() {
		if err := SourceSave(cli); err != nil {
			log.Fatalf("failed to save cli data: %v", err)
		}
	}

	if eData == nil || len(eData.Executable) == 0 {
		return
	}

	f, err := os.OpenFile(executeFile, os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	for _, ex := range eData.Executable {
		if _, err := f.WriteString(strings.ReplaceAll(strings.Join(ex, " "), "\\", "\\\\")); err != nil {
			log.Fatalf("failed to write to execute file: %v", err)
		}
	}
}

func SourceAutocomplete(cli CLI) {
	cursorWord := os.Args[2]

	cursorIdx, err := strconv.Atoi(cursorWord)
	if err != nil {
		log.Fatalf("failed to parse cursor position: %v", err)
	}

	if err := SourceLoad(cli); err != nil {
		log.Fatalf("failed to load cli: %v", err)
	}

	// Remove this file's name, the command alias, and the cursor word and pass to autocomplete.
	// TODO: actually use cursorIdx here.
	_ = cursorIdx
	g := Autocomplete(getNode(cli), os.Args[3:])
	fmt.Printf("%s\n", strings.Join(g, "\n"))
}

func getNode(c CLI) *Node {
	if len(c.Setup()) == 0 {
		return c.Node()
	}
	return SerialNodesTo(c.Node(), SetupArg)
}

func SourceLoad(cli CLI) error {
	ck := cacheKey(cli)
	cash := &Cache{}
	s, err := cash.Get(ck)
	if err != nil {
		return fmt.Errorf("failed to load cli %q: %v", cli.Name(), err)
	}

	return cli.Load(s)
}

func createSourceFile(cli CLI) string {
	return "$GOPATH/bin/tmp"
	/*

			_, sourceLocation, _, ok := runtime.Caller(2)
			if !ok {
				return "", fmt.Errorf("failed to fetch caller")
			}

			// cd into the directory of the file that is actually calling this and install dependencies.
		if _, err := f.WriteString(fmt.Sprintf(generateBinary, sourceLocation)); err != nil {
			return "", fmt.Errorf("failed to write binary generator code to file: %v", err)
		}
	*/
}

// Source generates the bash source file for a list of CLIs.
func SourceSource(clis ...CLI) {
	f, err := ioutil.TempFile("", "golang-cli-source")
	if err != nil {
		log.Fatalf("failed to create tmp file: %v", err)
	}

	// TODO:
	// - create a temporary dir/file under the source location with a main package
	// - build the file and store it with a unique (per-alias) name in $GOPATH/bin/leep-frog/...
	// - clean up temporary directory

	// define the autocomplete function
	if _, err := f.WriteString(autocompleteFunction); err != nil {
		log.Fatalf("failed to write autocomplete function to file: %v", err)
	}

	// define the execute function
	if _, err := f.WriteString(executeFunction); err != nil {
		log.Fatalf("failed to write autocomplete function to file: %v", err)
	}

	for _, cli := range clis {
		sourceFile := createSourceFile(cli)

		alias := cli.Alias()

		// TODO: aliasFormat ANNNDDD aliasWithSetupFormat should pass in custom binary name as input to _custom_execute.
		aliasCommand := fmt.Sprintf(aliasFormat, alias, sourceFile)
		if scs := cli.Setup(); len(scs) > 0 {
			setupFunctionName := fmt.Sprintf("_setup_for_%s_cli", alias)
			if _, err := f.WriteString(fmt.Sprintf(setupFunctionFormat, setupFunctionName, strings.Join(scs, "  \n"))); err != nil {
				log.Fatalf("failed to write setup command to file: %v", err)
			}
			aliasCommand = fmt.Sprintf(aliasWithSetupFormat, alias, setupFunctionName, sourceFile)
		}

		if _, err := f.WriteString(aliasCommand); err != nil {
			log.Fatalf("failed to write alias to file: %v", err)
		}

		// We sort ourselves, hence the no sort.
		if _, err := f.WriteString(fmt.Sprintf("complete -F _custom_autocomplete -o nosort %s %s\n", sourceFile, alias)); err != nil {
			log.Fatalf("failed to write autocomplete command to file: %v", err)
		}
	}

	f.Close()
	fmt.Printf(f.Name())
}

func SourceSave(c CLI) error {
	ck := cacheKey(c)
	cash := &Cache{}
	if err := cash.PutStruct(ck, c); err != nil {
		return fmt.Errorf("failed to save cli %q: %v", c.Name(), err)
	}
	return nil
}

func cacheKey(cli CLI) string {
	return fmt.Sprintf("cache-key-%s", cli.Name())
}
