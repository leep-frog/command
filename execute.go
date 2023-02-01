package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
)

// Processor defines the logic that should be executed at a `Node`.
type Processor interface {
	// Execute is the function called when a command graph is
	// being executed.
	Execute(*Input, Output, *Data, *ExecuteData) error
	// Complete is the function called when a command graph is
	// being autocompleted. If it returns a non-nil `Completion` object,
	// then the graph traversal stops and uses the returned object
	// to construct the command completion suggestions.
	Complete(*Input, *Data) (*Completion, error)
	// Usage is the function called when the usage data for a command
	// graph is being constructed. The input `Usage` object should be
	// updated for each `Node`.
	Usage(*Usage)
}

// Edge determines which `Node` to execute next.
type Edge interface {
	// Next fetches the next node in the command graph based on
	// the provided `Input` and `Data`.
	Next(*Input, *Data) (Node, error)
	// UsageNext fetches the next node in the command graph when
	// command graph usage is being constructed. This is separate from
	// the `Next` function because `Next` is input-dependent whereas `UsageNext`
	// receives no input arguments.
	UsageNext() Node
}

// Node defines a cohesive node in the command graph. It is simply a combination
// of a `Processor` and an `Edge`.
type Node interface {
	Processor
	Edge
}

type SimpleNode struct {
	Processor Processor
	Edge      Edge
}

func (sn *SimpleNode) Next(i *Input, d *Data) (Node, error) {
	if sn.Edge == nil {
		return nil, nil
	}
	return sn.Edge.Next(i, d)
}

func (sn *SimpleNode) UsageNext() Node {
	if sn.Edge == nil {
		return nil
	}
	return sn.Edge.UsageNext()
}

func (sn *SimpleNode) Execute(input *Input, output Output, data *Data, exData *ExecuteData) error {
	if sn.Processor == nil {
		return nil
	}
	return processOrExecute(sn.Processor, input, output, data, exData)
}

func (sn *SimpleNode) Complete(input *Input, data *Data) (*Completion, error) {
	if sn.Processor == nil {
		return nil, nil
	}
	return processOrComplete(sn.Processor, input, data)
}

func (sn *SimpleNode) Usage(usage *Usage) {
	if sn.Processor != nil {
		processOrUsage(sn.Processor, usage)
	}
}

const (
	// ArgSection is the title of the arguments usage section.
	ArgSection = "Arguments"
	// FlagSection is the title of the flags usage section.
	FlagSection = "Flags"
	// SymbolSection is the title of the symbols usage section.
	SymbolSection = "Symbols"
)

// Usage contains all data needed for constructing a command's usage text.
type Usage struct {
	// UsageSection is a map from section name to key phrase for that section to description for that key.
	// TODO: Only displayed when --help flag is provided
	UsageSection *UsageSection
	// Description is the usage doc for the command.
	Description string
	// Usage is arg usage string.
	Usage []string
	// Flags is the flag usage string.
	Flags []string

	// Subsections is a set of `Usage` objects that are indented from the current `Usage` object.
	SubSections []*Usage
}

// UsageSection is a map from section name to key phrase for that section to description for that key.
type UsageSection map[string]map[string][]string

// Add adds a usage section.
func (us *UsageSection) Add(section, key string, value ...string) {
	if (*us)[section] == nil {
		(*us)[section] = map[string][]string{}
	}
	(*us)[section][key] = append((*us)[section][key], value...)
}

func (us *UsageSection) Set(section, key string, value ...string) {
	if (*us)[section] == nil {
		(*us)[section] = map[string][]string{}
	}
	(*us)[section][key] = value
}

// String crates the full usage text as a single string.
func (u *Usage) String() string {
	var r []string
	r = u.string(r, 0)

	var sections []string
	if u.UsageSection != nil {
		for s := range *u.UsageSection {
			sections = append(sections, s)
		}
		sort.Strings(sections)
		for _, sk := range sections {
			r = append(r, fmt.Sprintf("%s:", sk))
			kvs := (*u.UsageSection)[sk]
			var keys []string
			for k := range kvs {
				keys = append(keys, k)
			}
			if sk == FlagSection {
				// We want to sort flags by full name, not short flags.
				// So, we trim "  [c]" from each flag description.
				sort.SliceStable(keys, func(i, j int) bool {
					return keys[i][4:] < keys[j][4:]
				})
			} else {
				sort.Strings(keys)
			}
			for _, k := range keys {
				for idx, kv := range kvs[k] {
					if idx == 0 {
						r = append(r, fmt.Sprintf("  %s: %s", k, kv))
					} else {
						r = append(r, fmt.Sprintf("    %s", kv))
					}
				}

			}

			// Since already split by newlines, this statement actually adds one newline.
			r = append(r, "")
		}
	}

	i := len(r) - 1
	for ; i > 0 && r[i-1] == ""; i-- {
	}
	r = r[:i]

	return strings.Join(r, "\n")
}

func (u *Usage) string(r []string, depth int) []string {
	prefix := strings.Repeat(" ", depth*2)
	if u.Description != "" {
		r = append(r, prefix+u.Description)
	}
	r = append(r, prefix+strings.Join(append(u.Usage, u.Flags...), " "))
	r = append(r, "") // since we join with newlines, this just adds one extra newline

	for _, su := range u.SubSections {
		r = su.string(r, depth+1)

		if su.UsageSection != nil {
			for section, m := range *su.UsageSection {
				for k, v := range m {
					// Subsections override the higher-level section
					// Mostly needed for duplicate sections (like nested branch nodes).
					u.UsageSection.Set(section, k, v...)
				}
			}
		}
	}
	return r
}

type descNode struct {
	desc string
}

// Description creates a `Processor` that adds a command description to the usage text.
func Description(desc string) Processor {
	return &descNode{desc}
}

// Descriptionf is like `Description`, but with formatting options.
func Descriptionf(s string, a ...interface{}) Processor {
	return &descNode{fmt.Sprintf(s, a...)}
}

func (dn *descNode) Usage(u *Usage) {
	u.Description = dn.desc
}

func (dn *descNode) Execute(*Input, Output, *Data, *ExecuteData) error {
	return nil
}

func (dn *descNode) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

// ExecuteData contains operations to resolve after all nodes have been processed.
// This separation is needed for caching and shortcuts nodes.
type ExecuteData struct {
	// Executable is a list of bash commands to run after all nodes have been processed.
	Executable []string
	// Executor is a set of functions to run after all nodes have been processed.
	Executor []func(Output, *Data) error
	// FunctionWrap is whether or not to wrap the Executable contents
	// in a function. This allows Executable to use things like "return" and "local".
	FunctionWrap bool
}

// GetUsage constructs a `Usage` object from the head `Node` of a command graph.
func GetUsage(n Node) *Usage {
	return getUsage(n, &Usage{
		UsageSection: &UsageSection{},
	})
}

// ShowUsageAfterError returns a string containing the the provided
// Node's usage doc along with a prefix to separate it from the printed error.
func ShowUsageAfterError(n Node) string {
	return fmt.Sprintf("\n======= Command Usage =======\n%s", GetUsage(n).String())
}

func getUsage(n Node, u *Usage) *Usage {
	for n != nil {
		n.Usage(u)
		n = n.UsageNext()
	}
	return u
}

// Execute executes a node with the provided `Input` and `Output`.
func Execute(n Node, input *Input, output Output) (*ExecuteData, error) {
	return execute(n, input, output, &Data{})
}

// RunNodes executes the provided node. This function can be used when nodes
// aren't used for CLI tools (such as in regular main.go files that are
// executed via "go run"). Using in this in conjunction with the `goleep`
// command is incredibly useful.
func RunNodes(n Node) error {
	o := NewOutput()
	err := runNodes(n, o, &Data{}, os.Args[1:])
	o.Close()
	return err
}

const (
	// PassthroughArgs is the argument name for args that are passed through
	// to autocomplete. Used for `aliaser` command.
	PassthroughArgs = "PASSTHROUGH_ARGS"
)

// Separate method for testing purposes.
func runNodes(n Node, o Output, d *Data, args []string) error {
	// We set default node to n in case user tries to run with "go run", but using goleep
	// is better because "go run main.go autocomplete" won't work as expected.
	var filename string
	nrf := "NODE_RUNNER_FILE"
	exNode := SerialNodes(
		FileArgument(nrf, "Temporary file for execution"),
		SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			filename = d.String(nrf)
			return nil
		}, nil),
		n,
	)

	bn := &BranchNode{
		Branches: map[string]Node{
			"execute": exNode,
			"usage": SerialNodes(
				&ExecutorProcessor{func(o Output, d *Data) error {
					o.Stdoutln(GetUsage(n).String())
					return nil
				}},
			),
			"autocomplete": SerialNodes(
				// Don't need comp point because input will have already been trimmed by goleep processing.
				Arg[string](PassthroughArgs, ""),
				&ExecutorProcessor{func(o Output, d *Data) error {
					sl, err := Autocomplete(n, d.String(PassthroughArgs), nil)
					if err != nil {
						return o.Stderrln(err)
					}
					for _, s := range sl {
						o.Stdoutln(s)
					}
					return nil
				}}),
		},
		Default:           n,
		DefaultCompletion: true,
	}

	input := ParseExecuteArgs(args)
	eData, err := execute(bn, input, NewIgnoreErrOutput(o, IsExtraArgsError), d)
	if err != nil {
		if IsUsageError(err) {
			if IsExtraArgsError(err) {
				o.Err(err)
			}
			o.Stderrln(ShowUsageAfterError(n))
		}
		return err
	}

	if filename != "" && len(eData.Executable) > 0 {
		if err := ioutil.WriteFile(filename, []byte(strings.Join(eData.Executable, "\n")), CmdOS.DefaultFilePerm()); err != nil {
			return o.Stderrf("failed to write eData.Executable to file: %v\n", err)
		}
	}
	return nil
}

// Separate method for testing purposes.
func execute(n Node, input *Input, output Output, data *Data) (*ExecuteData, error) {
	eData := &ExecuteData{}

	// This threading logic is needed in case the underlying process calls an output.Terminate command.
	var wg sync.WaitGroup
	wg.Add(1)

	var termErr error
	go func() {
		defer func() {
			if termErr == nil {
				termErr = output.terminateError()
			}
			wg.Done()
		}()
		if err := processGraphExecution(n, input, output, data, eData, true); err != nil {
			termErr = err
			return
		}

		if err := input.CheckForExtraArgsError(); err != nil {
			output.Stderrln(err)
			// TODO: Make this the last node we reached?
			ShowUsageAfterError(n)
			termErr = err
			return
		}

		for _, ex := range eData.Executor {
			if err := ex(output, data); err != nil {
				termErr = err
				return
			}
		}
	}()
	wg.Wait()
	return eData, termErr
}

// processOrExecute checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrExecute(p Processor, input *Input, output Output, data *Data, eData *ExecuteData) error {
	if n, ok := p.(Node); ok {
		return processGraphExecution(n, input, output, data, eData, false)
	}
	return p.Execute(input, output, data, eData)
}

// processOrComplete checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or completes the processor accordingly.
func processOrComplete(p Processor, input *Input, data *Data) (*Completion, error) {
	if n, ok := p.(Node); ok {
		return processGraphCompletion(n, input, data, false)
	}
	return p.Complete(input, data)
}

// processOrUsage checks if the provided `Processor` is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrUsage(p Processor, usage *Usage) {
	if n, ok := p.(Node); ok {
		getUsage(n, usage)
	} else {
		p.Usage(usage)
	}
}

// processGraphExecution processes the provided graph
func processGraphExecution(root Node, input *Input, output Output, data *Data, eData *ExecuteData, checkInput bool) error {
	for n := root; n != nil; {
		if err := n.Execute(input, output, data, eData); err != nil {
			return err
		}

		var err error
		if n, err = n.Next(input, data); err != nil {
			return err
		}
	}

	if checkInput {
		return output.Err(input.CheckForExtraArgsError())
	}
	return nil
}

// ExtraArgsErr returns an error for when too many arguments are provided to a command.
func ExtraArgsErr(input *Input) error {
	return &extraArgsErr{input}
}

type extraArgsErr struct {
	input *Input
}

func (eae *extraArgsErr) Error() string {
	return fmt.Sprintf("Unprocessed extra args: %v", eae.input.Remaining())
}

// IsExtraArgs returns whether or not the provided error is an `ExtraArgsErr`.
// TODO: error.go file.
func IsExtraArgsError(err error) bool {
	_, ok := err.(*extraArgsErr)
	return ok
}
