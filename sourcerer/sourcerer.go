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
	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/internal/spycommander"
	"github.com/leep-frog/command/internal/testutil"
	"golang.org/x/exp/maps"
)

const (
	// RootDirectoryEnvVar is the directory in which all artifact files needed will be created and stored.
	RootDirectoryEnvVar = "COMMAND_CLI_OUTPUT_DIR"
)

var (
	rootDirectoryArg = &commander.EnvArg{
		Name: RootDirectoryEnvVar,
		Validators: []*commander.ValidatorOption[string]{
			commander.IsDir(),
		},
		Transformers: []*commander.Transformer[string]{
			commander.FileTransformer(),
		},
	}
	fileArg         = commander.FileArgument("FILE", "Temporary file for execution")
	targetNameRegex = commander.MatchesRegex("^[a-zA-Z0-9]+$")
	passthroughArgs = commander.ListArg[string]("ARG", "Arguments that get passed through to relevant CLI command", 0, command.UnboundedList)
	helpFlag        = commander.BoolFlag("help", commander.FlagNoShortName, "Display command's usage doc")
	quietFlag       = commander.BoolFlag("quiet", commander.FlagNoShortName, "Hide unnecessary output")
	// See the below link for more details on COMP_* details:
	// https://www.gnu.org/software/bash/manual/html_node/Bash-Variables.html#Bash-Variables
	compTypeArg  = commander.Arg[int]("COMP_TYPE", "COMP_TYPE variable from bash complete function")
	compPointArg = commander.Arg[int]("COMP_POINT", "COMP_POINT variable from bash complete function")
	compLineArg  = commander.Arg("COMP_LINE", "COMP_LINE variable from bash complete function", &commander.Transformer[string]{F: func(s string, d *command.Data) (string, error) {
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
	getUuid     = uuid.NewString
	osReadFile  = os.ReadFile
	osWriteFile = os.WriteFile
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
		if err := save(cli, d); err != nil {
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

var (
	// getCacheStub is a variable function so it can be swapped in tests
	getCacheStub = func(dir string) (*cache.Cache, error) {
		return cache.FromDir(dir)
	}
)

func getCache(d *command.Data) (*cache.Cache, error) {
	// Note that the cache.FromDir will create the directory if it does not exist
	// This will only result in making the `cache` dir and no parent dirs because
	// the env arg for the root dir enforces that the root dir exists.
	return getCacheStub(filepath.Join(rootDirectoryArg.Get(d), "cache"))
}

func load(cli CLI, d *command.Data) error {
	ck := cacheKey(cli)
	cash, err := getCache(d)
	if err != nil {
		return fmt.Errorf("failed to load cache from environment variable: %v", err)
	}
	_, err = cash.GetStruct(ck, cli)
	return err
}

type sourcerer struct {
	targetName           string
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
	loadCLIArg := commander.SerialNodes(
		rootDirectoryArg,
		commander.SuperSimpleProcessor(func(i *command.Input, d *command.Data) error {
			return load((*s.cliArg.Processor).Get(d), d)
		}),
	)

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
					rootDirectoryArg,
					commander.FlagProcessor(
						quietFlag,
					),
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
	if source(true, "ignoreTargetName", []CLI{cli}, os.Args[0], os.Args[1:], o) != nil {
		return 1
	}
	return 0
}

// Source generates the bash source file for a list of CLIs.
func Source(targetName string, clis []CLI, opts ...Option) int {
	o := command.NewOutput()
	defer o.Close()
	if source(false, targetName, clis, os.Args[0], os.Args[1:], o, opts...) != nil {
		return 1
	}
	return 0
}

func (s *sourcerer) initBuiltInSourcerer() error {
	return s.initSourcerer(false, true, "leepFrogCLIBuiltIns", []CLI{
		&AliaserCommand{s.goExecutableFilePath},
		&Debugger{},
		&GoLeep{},
		&UpdateLeepPackageCommand{},
	}, s.sourceLocation, []Option{})
}

func (s *sourcerer) initSourcerer(runCLI, builtin bool, targetName string, clis []CLI, sourceLocation string, opts []Option) error {
	cos := &compiledOpts{
		aliasers: map[string]*Aliaser{},
	}
	for _, oi := range opts {
		oi.modifyCompiledOpts(cos)
	}

	cliMap := map[string]CLI{}

	if !runCLI {
		clis = append(clis, &topLevelCLI{
			name:           targetName,
			sourceLocation: sourceLocation,
		})
	}

	for i, c := range clis {
		if c == nil {
			return fmt.Errorf("nil CLI provided at index %d", i)
		}

		name := c.Name()
		if _, ok := cliMap[name]; ok {
			return fmt.Errorf("multiple CLIs with the same name (%q); note: a top-level CLI is generated with the provided targetName, so that must be different than all names of the provided CLIs", name)
		}
		cliMap[name] = c
	}

	s.clis = cliMap
	ma := commander.MapArg("CLI", "", cliMap, false)
	s.cliArg.Processor = &ma

	s.sourceLocation = sourceLocation
	s.targetName = targetName
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
func source(runCLI bool, targetName string, clis []CLI, goExecutableFilePath string, osArgs []string, o command.Output, opts ...Option) error {
	if err := targetNameRegex.Validate(targetName, nil); err != nil {
		return o.Annotatef(err, "Invalid target name")
	}

	sl, err := getSourceLoc()
	if err != nil {
		return o.Annotate(err, "failed to get source location")
	}

	s := &sourcerer{
		goExecutableFilePath: goExecutableFilePath,
		cliArg:               commander.NewMutableProcessor[*commander.MapFlargument[string, CLI]](nil),
	}
	if err := s.initSourcerer(runCLI, false, targetName, clis, sl, opts); err != nil {
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
	osMkdirAll = os.MkdirAll
)

func (s *sourcerer) generateFile(o command.Output, d *command.Data) error {
	loud := !quietFlag.Get(d)

	// Create the artifacts directory
	rootDir := rootDirectoryArg.Get(d)
	artifactsDir := filepath.Join(rootDir, "artifacts")
	sourcerersDir := filepath.Join(rootDir, "sourcerers")
	// Note: this will only result in making the `artifacts/sourcerers` dir and no parent dirs because
	// the env arg for the root dir enforces that the root dir exists.
	if err := osMkdirAll(artifactsDir, 0777); err != nil {
		return o.Annotatef(err, "failed to make artifacts directory")
	}
	if err := osMkdirAll(sourcerersDir, 0777); err != nil {
		return o.Annotatef(err, "failed to make sourcerers directory")
	}

	b, err := osReadFile(s.goExecutableFilePath)
	if err != nil {
		return o.Annotatef(err, "failed to read executable file")
	}

	newExecutableFilePath := filepath.Join(artifactsDir, fmt.Sprintf("%s%s", s.targetName, CurrentOS.ExecutableFileSuffix()))
	if err := osWriteFile(newExecutableFilePath, b, 0744); err != nil {
		return o.Annotatef(err, "failed to copy executable file")
	}
	s.goExecutableFilePath = newExecutableFilePath

	if loud {
		o.Stdoutf("Binary file created: %q\n", newExecutableFilePath)
	}

	fileData := CurrentOS.RegisterCLIs(s.builtin, s.goExecutableFilePath, s.targetName, maps.Values(s.clis))

	fileData = append(fileData, AliasSourcery(s.goExecutableFilePath, maps.Values(s.opts.aliasers)...)...)

	fileContents := CurrentOS.FunctionWrap(fmt.Sprintf("_%s_wrap_function", s.targetName), strings.Join(fileData, "\n"))

	sourceableFile := filepath.Join(sourcerersDir, CurrentOS.SourceableFile(s.targetName))
	if err := osWriteFile(sourceableFile, []byte(fileContents), 0644); err != nil {
		return o.Annotatef(err, "failed to write sourceable file contents")
	}

	var builtinArg string
	if s.builtin {
		builtinArg = fmt.Sprintf(" %s", BuiltInCommandParameter)
	}
	goRunSourceCommand := fmt.Sprintf(`go run .%s source`, builtinArg)

	if loud {
		o.Stdoutln(fmt.Sprintf("Sourceable file created: %q\n", sourceableFile))
	}

	cliNames := maps.Keys(s.clis)
	slices.Sort(cliNames)
	o.Stdoutln(color.Apply(fmt.Sprintf("Successfully generated CLIs (%s) from %s!", strings.Join(cliNames, ", "), s.targetName), color.Green, color.Bold))
	if loud {
		o.Stdoutln(strings.Join([]string{
			``,
			"Run the following (and/or add it to your terminal profile) to load your CLIs in your current terminal:",
			``,
			color.Apply(strings.Join(CurrentOS.SourceSetup(sourceableFile, s.targetName, goRunSourceCommand, filepath.Dir(s.sourceLocation)), "\n"), color.Blue),
		}, "\n"))
	}

	return nil
}

func save(c CLI, d *command.Data) error {
	ck := cacheKey(c)
	cash, err := getCache(d)
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

type topLevelCLI struct {
	name           string
	sourceLocation string
}

func (t *topLevelCLI) Name() string    { return t.name }
func (t *topLevelCLI) Setup() []string { return nil }
func (t *topLevelCLI) Changed() bool   { return false }

func (t *topLevelCLI) Node() command.Node {
	builtinFlag := commander.BoolFlag("builtin", 'b', "Whether or not the built-in CLIs should be used instead of user-defined ones")
	return commander.SerialNodes(
		commander.Description("This is a CLI for running generic cross-CLI utility commands for all CLIs generated with this target name"),
		&commander.BranchNode{
			Branches: map[string]command.Node{
				"reload": commander.SerialNodes(
					commander.Description("Regenerate all CLI artifacts and executables using the current go source code"),
					commander.FlagProcessor(builtinFlag),
					rootDirectoryArg,
					commander.PrintlnProcessor("HERIO 1"),
					commander.SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
						temp, err := os.MkdirTemp(rootDirectoryArg.Get(d), "top-level-cli-*")
						if err != nil {
							return fmt.Errorf("failed to create temp directory")
						}

						fmt.Println("TEMP", temp)
						d.Set("TEMP_DIR", temp)
						return nil
					}, nil),

					commander.PrintlnProcessor("HERIO 2"),
					commander.ClosureProcessor(func(i *command.Input, d *command.Data) command.Processor {
						args := []string{"run", ".", "source"}
						if builtinFlag.Get(d) {
							args = []string{"run", ".", "builtin", "source"}
						}

						fmt.Println("WTHf", filepath.Dir(t.sourceLocation), "PRE", d.String("TEMP_DIR"), "POST")
						return &commander.ShellCommand[string]{
							// Dir:               filepath.Dir(t.sourceLocation),
							// CommandName:       "go",
							CommandName:       "echo",
							Args:              args,
							DontRunOnComplete: true,
							ForwardStdout:     true,
							Env: []string{
								fmt.Sprintf("%s=%s", RootDirectoryEnvVar, d.String("TEMP_DIR")),
							},
						}
					}),
					commander.PrintlnProcessor("HERIO 3"),
					// commander.ClosureProcessor(func(i *command.Input, d *command.Data) command.Processor {
					// 	// TODO: Use os.CopyFS in go 1.23
					// 	fmt.Println("here too")
					// 	return &commander.ShellCommand[string]{
					// 		CommandName: "cp",
					// 		Args: []string{
					// 			ValueByOS(map[string]string{
					// 				Windows().Name(): "-Recurse",
					// 				Linux().Name():   "-a",
					// 			}),
					// 			filepath.Join(d.String("TEMP_DIR"), "*"),
					// 			rootDirectoryArg.Get(d),
					// 		},
					// 		DontRunOnComplete: true,
					// 		ForwardStdout:     true,
					// 	}
					// }),
				),
			},
		},
	)
}
