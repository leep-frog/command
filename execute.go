package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
)

type Node struct {
	Processor Processor
	Edge      Edge
}

const (
	ArgSection    = "Arguments"
	FlagSection   = "Flags"
	SymbolSection = "Symbols"

	UsageDepth = 0
)

type Usage struct {
	// Sections is a map from section name to phrase for that section to description for that key.
	// Only displayed when --help flag is provided
	UsageSection *UsageSection
	// Description is the usage doc for the command
	Description string
	Usage       []string
	Flags       []string

	SubSections []*Usage
}

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

type UsageSection map[string]map[string]string

func (us *UsageSection) Add(section, key, value string) {
	if (*us)[section] == nil {
		(*us)[section] = map[string]string{}
	}
	(*us)[section][key] = value
}

type descNode struct {
	desc string
}

func Description(desc string) Processor {
	return &descNode{desc}
}

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

type Processor interface {
	Execute(*Input, Output, *Data, *ExecuteData) error
	Complete(*Input, *Data) (*Completion, error)
	Usage(*Usage)
}

type Edge interface {
	Next(*Input, *Data) (*Node, error)
	UsageNext() *Node
}

func GetUsage(n *Node) *Usage {
	u := &Usage{
		UsageSection: &UsageSection{},
	}
	for n != nil {
		n.Processor.Usage(u)

		if n.Edge == nil {
			return u
		}

		n = n.Edge.UsageNext()
	}
	return u
}

func Execute(n *Node, input *Input, output Output) (*ExecuteData, error) {
	return execute(n, input, output, &Data{})
}

// RunNodes executes the provided node. This function can be used when nodes
// aren't used for CLI tools (such as in regular main.go files that are
// executed via "go run").
func RunNodes(n *Node) error {
	o := NewOutput()
	err := runNodes(n, o, &Data{}, os.Args[1:])
	o.Close()
	return err
}

const (
	PassthroughArgs = "PASSTHROUGH_ARGS"
)

// Separate method for testing purposes.
func runNodes(n *Node, o Output, d *Data, args []string) error {
	// We set default node to n in case user tries to run with "go run", but using goleep
	// is better because "go run main.go autocomplete" won't work as expected.
	var filename string
	nrf := "NODE_RUNNER_FILE"
	exNode := SerialNodesTo(n,
		FileNode(nrf, "Temporary file for execution"),
		SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			filename = d.String(nrf)
			return nil
		}, nil),
	)

	bn := BranchNode(map[string]*Node{
		"execute": exNode,
		"usage": SerialNodes(
			ExecutorNode(func(o Output, d *Data) {
				o.Stdout(GetUsage(n).String())
			}),
		),
		"autocomplete": SerialNodes(
			// Don't need comp point because input will have already been trimmed by goleep processing.
			Arg[string](PassthroughArgs, ""),
			ExecutorNode(func(o Output, d *Data) {
				for _, s := range Autocomplete(n, d.String(PassthroughArgs)) {
					o.Stdout(s)
				}
			})),
	}, n, DontCompleteSubcommands())

	eData, err := execute(bn, ParseExecuteArgs(args), o, d)
	if err != nil {
		if IsUsageError(err) {
			o.Stderr(GetUsage(n).String())
		}
		return err
	}
	if filename != "" && len(eData.Executable) > 0 {
		if err := ioutil.WriteFile(filename, []byte(strings.Join(eData.Executable, "\n")), 0644); err != nil {
			return o.Stderrf("failed to write eData.Executable to file: %v", err)
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

func ExtraArgsErr(input *Input) error {
	return &extraArgsErr{input}
}

type extraArgsErr struct {
	input *Input
}

func (eae *extraArgsErr) Error() string {
	return fmt.Sprintf("Unprocessed extra args: %v", eae.input.Remaining())
}

func IsExtraArgsError(err error) bool {
	_, ok := err.(*extraArgsErr)
	return ok
}
