package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

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
// executed via "go run"). While this is useful to use in conjunction
// with the `goleep` command, consider using `sourcerer.Run` to ensure
// your CLI data is also load from persistent memory.
func RunNodes(n Node) error {
	o := NewOutput()
	err := RunNodesWithOutput(n, o)
	o.Close()
	return err
}

// RunNodesWithOutput is similar to `RunNodes`, but accepts the output to use.
// Note: this function does *not* close the output channel.
func RunNodesWithOutput(n Node, o Output) error {
	return runNodes(n, o, &Data{}, os.Args[1:])
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
