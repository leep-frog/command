// package sourcerer sources CLI commands in a shell environment.
// See the `main` function in github.com/leep-frog/command/examples/source.go
// for an example of how to define a source file that uses this.
package sourcerer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/leep-frog/command/cache"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/internal/spycommander"
	"github.com/leep-frog/command/internal/testutil"
	"golang.org/x/exp/maps"
)

var (
	fileArg         = commander.FileArgument("FILE", "Temporary file for execution")
	targetNameRegex = commander.MatchesRegex("^[a-zA-Z0-9]+$")
	targetNameArg   = commander.Arg[string]("TARGET_NAME", "The name of the created target in $GOPATH/bin", targetNameRegex)
	passthroughArgs = commander.ListArg[string]("ARG", "Arguments that get passed through to relevant CLI command", 0, command.UnboundedList)
	helpFlag        = commander.BoolFlag("help", commander.FlagNoShortName, "Display command's usage doc")
	// See the below link for more details on COMP_* details:
	// https://www.gnu.org/software/bash/manual/html_node/Bash-Variables.html#Bash-Variables
	compTypeArg  = commander.Arg[int]("COMP_TYPE", "COMP_TYPE variable from bash complete function")
	compPointArg = commander.Arg[int]("COMP_POINT", "COMP_POINT variable from bash complete function")
	compLineArg  = commander.Arg[string]("COMP_LINE", "COMP_LINE variable from bash complete function", &commander.Transformer[string]{F: func(s string, d *command.Data) (string, error) {
		if compLineFileFlag.Get(d) {
			b, err := osReadFile(s)
			if err != nil {
				return "", fmt.Errorf("assumed COMP_LINE to be a file, but unable to read it: %v", err)
			}
			s = string(b)
		}

		// We should only consider the string up to where the cursor is (i.e. COMP_POINT)
		cPoint := compPointArg.Get(d)
		if cPoint <= len(s) {
			return s[:cPoint], nil
		}
		// The space isn't included in comp line in windows sometimes, hence the need for this.
		return s + strings.Repeat(" ", cPoint-len(s)), nil
	}})
	compLineFileFlag            = commander.BoolFlag("comp-line-file", commander.FlagNoShortName, "If set, the COMP_LINE arg is taken to be a file that contains the COMP_LINE contents")
	autocompletePassthroughArgs = commander.ListArg[string]("PASSTHROUGH_ARG", "Arguments that get passed through to autocomplete command", 0, command.UnboundedList)

	// Made these methods so they can be stubbed out in tests
	getUuid    = uuid.NewString
	osReadFile = os.ReadFile
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
	cli := (*s.cliArg.Processor).Get(d)

	sourcingFile := d.String(fileArg.Name())
	args := d.StringList(passthroughArgs.Name())

	if helpFlag.Get(d) {
		return s.usageExecutorHelper(cli, args)(output, d)
	}

	// Add the setup arg if relevant. This should be identical to
	// setup in commandtest.go.
	n := cli.Node()
	if len(cli.Setup()) > 0 {
		if s.isRunCLI() {
			return output.Stderrln("Setup() must be empty when running via RunCLI() (supported only via Source())")
		}
		n = commander.SerialNodes(commander.SetupArg, n)
	}

	// We check this error afer saving. It is up to the user to only mark something as
	// changed when it should actually be changed (i.e. check for errors in their logic).
	eData, err := commander.Execute(n, command.ParseExecuteArgs(args), output, CurrentOS)

	// Save the CLI if it has changed.
	if cli.Changed() {
		if err := save(cli); err != nil {
			return output.Stderrf("failed to save cli data: %v\n", err)
		}
	}

	if err != nil {
		if commander.IsUsageError(err) && !s.printedUsageError && !s.forAutocomplete && !command.IsExtraArgsError(err) {
			s.printedUsageError = true
			spycommander.ShowUsageAfterError(n, output)
		}
		// Commands are responsible for printing out error messages so
		// we just return if there are any issues here
		return err
	}

	// Run the executable file if relevant.
	if eData == nil || len(eData.Executable) == 0 {
		return nil
	}

	if s.isRunCLI() {
		return output.Stderrln("ExecuteData.Executable is not supported via RunCLI() (use Source() instead)")
	}

	f, err := os.OpenFile(sourcingFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return output.Stderrf("failed to open file: %v\n", err)
	}

	v := strings.Join(eData.Executable, "\n")

	if eData.FunctionWrap {
		v = CurrentOS.FunctionWrap(fmt.Sprintf("_leep_execute_data_function_wrap_%s", strings.ReplaceAll(getUuid(), "-", "_")), v)
	}

	if _, err := f.WriteString(v); err != nil {
		return output.Stderrf("failed to write to execute file: %v\n", err)
	}

	return nil
}

func (s *sourcerer) autocompleteExecutor(o command.Output, d *command.Data) error {
	s.forAutocomplete = true
	cli := (*s.cliArg.Processor).Get(d)

	autocompletion, err := commander.Autocomplete(cli.Node(), compLineArg.Get(d), autocompletePassthroughArgs.Get(d), CurrentOS)
	if err != nil {
		CurrentOS.HandleAutocompleteError(o, compTypeArg.Get(d), err)
		return err
	}

	CurrentOS.HandleAutocompleteSuccess(o, autocompletion)
	return nil
}

// separate method for testing
var (
	// EnvCacheVar is the environment variable pointing to the path for caching.
	// var so it can be modified for tests
	EnvCacheVar = "COMMAND_CLI_CACHE"
	// getCache is a variable function so it can be swapped in tests
	getCache = func() (*cache.Cache, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user's home directory: %v", err)
		}
		return cache.FromEnvVarOrDir(EnvCacheVar, filepath.Join(home, ".command-cli-cache"))
	}
)

func load(cli CLI) error {
	ck := cacheKey(cli)
	cash, err := getCache()
	if err != nil {
		return fmt.Errorf("failed to load cache from environment variable: %v", err)
	}
	_, err = cash.GetStruct(ck, cli)
	return err
}

type sourcerer struct {
	goExecutableFilePath string
	clis                 map[string]CLI
	cliArg               *commander.MutableProcessor[*commander.MapFlargument[string, CLI]]
	sourceLocation       string
	printedUsageError    bool
	opts                 *compiledOpts
	forAutocomplete      bool
	builtin              bool
	runCLI               CLI
}

func (s *sourcerer) isRunCLI() bool { return s.runCLI != nil }

func (*sourcerer) UnmarshalJSON(jsn []byte) error { return nil }
func (*sourcerer) Changed() bool                  { return false }
func (*sourcerer) Setup() []string                { return nil }
func (*sourcerer) Name() string {
	return "_internal_sourcerer"
}

const (
	AutocompleteBranchName              = "autocomplete"
	GenerateAutocompleteSetupBranchName = "generate-autocomplete-setup"
	ExecuteBranchName                   = "execute"
	ListBranchName                      = "listCLIs"
	SourceBranchName                    = "source"
	UsageBranchName                     = "usage"

	BuiltInCommandParameter = "builtin"
)

func (s *sourcerer) Node() command.Node {
	loadCLIArg := commander.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
		return load((*s.cliArg.Processor).Get(d))
	})

	autocompleteBranchNode := func(runCLI bool) command.Node {
		nodes := []command.Processor{
			commander.FlagProcessor(
				compLineFileFlag,
			),
		}

		if !runCLI {
			nodes = append(nodes, s.cliArg)
		}

		return commander.SerialNodes(append(nodes,
			loadCLIArg,
			compTypeArg,
			compPointArg,
			compLineArg,
			autocompletePassthroughArgs,
			&commander.ExecutorProcessor{F: s.autocompleteExecutor},
		)...)
	}

	// Change if runcli
	if s.isRunCLI() {
		aliasFlag := commander.Flag[string]("alias", commander.FlagNoShortName, "")
		return commander.SerialNodes(
			// Set the CLI to runCLI
			commander.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
				(*s.cliArg.Processor).Set(s.runCLI.Name(), d)
				return nil
			}),
			&commander.BranchNode{
				Branches: map[string]command.Node{
					AutocompleteBranchName: autocompleteBranchNode(true),
					GenerateAutocompleteSetupBranchName: commander.SerialNodes(
						commander.FlagProcessor(
							aliasFlag,
						),
						&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
							alias := aliasFlag.GetOrDefault(d, filepath.Base(s.goExecutableFilePath))
							if err := targetNameRegex.Validate(alias, d); err != nil {
								return o.Err(err)
							}

							functionName := fmt.Sprintf("_RunCLI_%s_autocomplete_wrap_function", alias)
							functionContent := strings.Join(CurrentOS.RegisterRunCLIAutocomplete(s.goExecutableFilePath, alias), "\n")
							o.Stdoutln(CurrentOS.FunctionWrap(functionName, functionContent))
							return nil
						}},
					),
				},
				BranchUsageOrder: []string{},
				Default: commander.SerialNodes(
					loadCLIArg,
					commander.FlagProcessor(
						helpFlag,
					),
					passthroughArgs,
					&commander.ExecutorProcessor{F: s.executeExecutor},
				),
			},
		)
	}

	return commander.SerialNodes(
		commander.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
			// If the first argument is the built-in parameter, then only use the built-in commands
			if p, _ := i.Peek(); p == BuiltInCommandParameter {
				i.Pop(d)
				if err := s.initBuiltInSourcerer(); err != nil {
					return err
				}
			}
			return nil
		}),
		&commander.BranchNode{
			Branches: map[string]command.Node{
				AutocompleteBranchName: autocompleteBranchNode(false),
				ListBranchName: commander.SerialNodes(
					commander.SimpleProcessor(s.listCLIExecutor, nil),
				),
				ExecuteBranchName: commander.SerialNodes(
					s.cliArg,
					loadCLIArg,
					fileArg,
					commander.FlagProcessor(
						helpFlag,
					),
					passthroughArgs,
					&commander.ExecutorProcessor{F: s.executeExecutor},
				),
				SourceBranchName: commander.SerialNodes(
					targetNameArg,
					&commander.ExecutorProcessor{F: s.generateFile},
				),
			},
			BranchUsageOrder: []string{},
			Default: commander.SerialNodes(
				// Just eat the remaining args
				// commander.ListArg[string]("UNUSED", "", 0, command.UnboundedList),
				&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
					// Add echo so it's a comment if included in sourced output
					return o.Stderrf("echo %q\n", "Executing a sourcerer.CLI directly through `go run` is tricky. Either generate a CLI or use the `goleep` command to directly run the file.")
				}},
			),
		},
	)
}

func (s *sourcerer) listCLIExecutor(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
	clis := maps.Keys(s.clis)
	slices.Sort(clis)
	o.Stdoutln(strings.Join(clis, "\n"))
	return nil
}

func (s *sourcerer) usageExecutorHelper(cli CLI, args []string) func(o command.Output, d *command.Data) error {
	return func(o command.Output, d *command.Data) error {
		return spycommander.HelpBehavior(cli.Node(), command.ParseExecuteArgs(args), o, commander.IsUsageError)
	}
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

// RunCLI runs an individual CLI, thus making the go executable file the only
// setup needed.
func RunCLI(cli CLI) int {
	o := command.NewOutput()
	defer o.Close()
	if source(true, []CLI{cli}, os.Args[0], os.Args[1:], o) != nil {
		return 1
	}
	return 0
}

// Source generates the bash source file for a list of CLIs.
func Source(clis []CLI, opts ...Option) int {
	o := command.NewOutput()
	defer o.Close()
	if source(false, clis, os.Args[0], os.Args[1:], o, opts...) != nil {
		return 1
	}
	return 0
}

func (s *sourcerer) initBuiltInSourcerer() error {
	return s.initSourcerer(false, true, []CLI{
		&SourcererCommand{},
		&AliaserCommand{s.goExecutableFilePath},
		&Debugger{},
		&GoLeep{},
		&UpdateLeepPackageCommand{},
	}, s.sourceLocation, []Option{})
}

func (s *sourcerer) initSourcerer(runCLI, builtin bool, clis []CLI, sourceLocation string, opts []Option) error {
	cos := &compiledOpts{
		aliasers: map[string]*Aliaser{},
	}
	for _, oi := range opts {
		oi.modifyCompiledOpts(cos)
	}

	cliMap := map[string]CLI{}
	for i, c := range clis {
		if c == nil {
			return fmt.Errorf("nil CLI provided at index %d", i)
		}
		cliMap[c.Name()] = c
	}

	s.clis = cliMap
	ma := commander.MapArg("CLI", "", cliMap, false)
	s.cliArg.Processor = &ma

	s.sourceLocation = sourceLocation
	s.opts = cos
	s.builtin = builtin
	if runCLI {
		if len(clis) != 1 {
			return fmt.Errorf("%d CLIs provided with RunCLI(); expected exactly one", len(clis))
		}
		s.runCLI = clis[0]
	}
	return nil
}

var (
	osExecutable = os.Executable
)

const (
	ExecutableFileGetProcessorName = "GO_EXECUTABLE_FILE"
)

// StubExecutableFile stubs the executable file returned by ExecutableFileGetProcessor()
func StubExecutableFile(t *testing.T, filepath string, err error) {
	testutil.StubValue(t, &osExecutable, func() (string, error) { return filepath, err })
}

// ExecutableFileGetProcessor returns a `commander.GetProcessor` that sets and gets
// the full go executable file path.
func ExecutableFileGetProcessor() *commander.GetProcessor[string] {
	return &commander.GetProcessor[string]{
		commander.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
			f, err := osExecutable()
			if err != nil {
				return fmt.Errorf("failed to get file from os.Executable(): %v", err)
			}
			d.Set(ExecutableFileGetProcessorName, f)
			return nil
		}),
		ExecutableFileGetProcessorName,
	}
}

// Separate method used for testing.
func source(runCLI bool, clis []CLI, goExecutableFilePath string, osArgs []string, o command.Output, opts ...Option) error {
	sl, err := getSourceLoc()
	if err != nil {
		return o.Annotate(err, "failed to get source location")
	}

	s := &sourcerer{
		goExecutableFilePath: goExecutableFilePath,
		cliArg:               commander.NewMutableProcessor[*commander.MapFlargument[string, CLI]](nil),
	}
	if err := s.initSourcerer(runCLI, false, clis, sl, opts); err != nil {
		return o.Err(err)
	}

	// Sourcerer is always executed. Its execution branches into the relevant CLI's
	// execution/autocomplete/usage path.
	d := &command.Data{
		Values: map[string]interface{}{
			(*s.cliArg.Processor).Name(): s,
			// Don't need execute file here
			passthroughArgs.Name(): osArgs,
		},
	}

	return s.executeExecutor(o, d)
}

var (
	// Stubbed out for testing purposes
	runtimeCaller = runtime.Caller
)

func getSourceLoc() (string, error) {
	_, sourceLocation, _, ok := runtimeCaller(3)
	if !ok {
		return "", fmt.Errorf("failed to fetch runtime.Caller")
	}
	return sourceLocation, nil
}

var (
	commandStat = commander.Stat
)

func (s *sourcerer) generateFile(o command.Output, d *command.Data) error {
	targetName := targetNameArg.Get(d)

	fileData := CurrentOS.RegisterCLIs(s.builtin, s.goExecutableFilePath, targetName, maps.Values(s.clis))

	fileData = append(fileData, AliasSourcery(s.goExecutableFilePath, maps.Values(s.opts.aliasers)...)...)

	o.Stdoutln(CurrentOS.FunctionWrap(fmt.Sprintf("_%s_wrap_function", targetName), strings.Join(fileData, "\n")))

	return nil
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
	return fmt.Sprintf("leep-frog-cache-key-%s.json", cli.Name())
}
