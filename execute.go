package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
)

// Node is a type containing the relevant processing that should be done
// when the node is reached (`Processor`) as well as what `Node` should be
// visited next (`Edge`). `Node` also implements the `Processor` interface,
// so a single root `Node` (and its entire underlying graph) can also be treated
// as an individual `Processor` element.
type Node struct {
	// Processor is used to process the node when it is visited.
	Processor Processor
	// Edge determines the next node to visit.
	Edge Edge
}

type NodeInterface interface {
	Processor
	Edge
}

func AsNode(n NodeInterface) *Node {
	return &Node{
		n,
		n,
	}
}

func (n *Node) Execute(input *Input, output Output, data *Data, exData *ExecuteData) error {
	// TODO: maybe the caller (sourcerer) should be required to check
	// if the input has been fully processed or not?
	ieo := NewIgnoreErrOutput(output, IsExtraArgsError)
	err := iterativeExecute(n, input, ieo, data, exData)
	if IsExtraArgsError(err) {
		return nil
	}
	return err
}

func (n *Node) Complete(input *Input, data *Data) (*Completion, error) {
	c, err := getCompleteData(n, input, data)
	if IsExtraArgsError(err) {
		return c, nil
	}
	return c, err
}

func (n *Node) Usage(usage *Usage) {
	getUsage(n, usage)
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
type UsageSection map[string]map[string]string

// Add adds a usage section.
func (us *UsageSection) Add(section, key, value string) {
	if (*us)[section] == nil {
		(*us)[section] = map[string]string{}
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
				r = append(r, fmt.Sprintf("  %s: %s", k, kvs[k]))
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
					u.UsageSection.Add(section, k, v)
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

// Processor is the interface for nodes in the command graph.
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

// Edge is the interface for edges in the command graph.
type Edge interface {
	// Next fetches the next node in the command graph based on
	// the provided `Input` and `Data`.
	Next(*Input, *Data) (*Node, error)
	// UsageNext fetches the next node in the command graph when
	// command graph usage is being constructed. This is separate from
	// the `Next` function because `Next` is input-dependent whereas `UsageNext`
	// receives no input arguments.
	UsageNext() *Node
}

// GetUsage constructs a `Usage` object from the head `Node` of a command graph.
func GetUsage(n *Node) *Usage {
	return getUsage(n, &Usage{
		UsageSection: &UsageSection{},
	})
}

// ShowUsageAfterError returns a string containing the the provided
// Node's usage doc along with a prefix to separate it from the printed error.
func ShowUsageAfterError(n *Node) string {
	return fmt.Sprintf("\n======= Command Usage =======\n%s", GetUsage(n).String())
}

func getUsage(n *Node, u *Usage) *Usage {
	for n != nil {
		n.Processor.Usage(u)

		if n.Edge == nil {
			return u
		}

		n = n.Edge.UsageNext()
	}
	return u
}

// Execute executes a node with the provided `Input` and `Output`.
func Execute(n *Node, input *Input, output Output) (*ExecuteData, error) {
	return execute(n, input, output, &Data{})
}

// RunNodes executes the provided node. This function can be used when nodes
// aren't used for CLI tools (such as in regular main.go files that are
// executed via "go run"). Using in this in conjunction with the `goleep`
// command is incredibly useful.
func RunNodes(n *Node) error {
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
func runNodes(n *Node, o Output, d *Data, args []string) error {
	// We set default node to n in case user tries to run with "go run", but using goleep
	// is better because "go run main.go autocomplete" won't work as expected.
	var filename string
	nrf := "NODE_RUNNER_FILE"
	exNode := SerialNodes(
		FileNode(nrf, "Temporary file for execution"),
		SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			filename = d.String(nrf)
			return nil
		}, nil),
		n,
	)

	bn := AsNode(&BranchNode{
		Branches: map[string]*Node{
			"execute": exNode,
			"usage": SerialNodes(
				ExecutorNode(func(o Output, d *Data) {
					o.Stdoutln(GetUsage(n).String())
				}),
			),
			"autocomplete": SerialNodes(
				// Don't need comp point because input will have already been trimmed by goleep processing.
				Arg[string](PassthroughArgs, ""),
				ExecuteErrNode(func(o Output, d *Data) error {
					sl, err := Autocomplete(n, d.String(PassthroughArgs), nil)
					if err != nil {
						return o.Stderrln(err)
					}
					for _, s := range sl {
						o.Stdoutln(s)
					}
					return nil
				})),
		},
		Default:           n,
		DefaultCompletion: true,
	})

	eData, err := execute(bn, ParseExecuteArgs(args), o, d)
	if err != nil {
		if IsUsageError(err) {
			o.Stderrln(ShowUsageAfterError(n))
		}
		return err
	}
	if filename != "" && len(eData.Executable) > 0 {
		if err := ioutil.WriteFile(filename, []byte(strings.Join(eData.Executable, "\n")), 0644); err != nil {
			return o.Stderrf("failed to write eData.Executable to file: %v\n", err)
		}
	}
	return nil
}

// Separate method for testing purposes.
func execute(n *Node, input *Input, output Output, data *Data) (*ExecuteData, error) {
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
		if err := iterativeExecute(n, input, output, data, eData); err != nil {
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

func iterativeExecute(n *Node, input *Input, output Output, data *Data, eData *ExecuteData) error {
	for n != nil {
		if n.Processor != nil {
			if err := n.Processor.Execute(input, output, data, eData); err != nil {
				return err
			}
		}

		if n.Edge == nil {
			break
		}

		var err error
		if n, err = n.Edge.Next(input, data); err != nil {
			return err
		}
	}

	if !input.FullyProcessed() {
		return output.Err(ExtraArgsErr(input))
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
