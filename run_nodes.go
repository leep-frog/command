package command

import (
	"io/ioutil"
	"os"
	"strings"
)

const (
	// PassthroughArgs is the argument name for args that are passed through
	// to autocomplete. Used for `aliaser` command.
	PassthroughArgs = "PASSTHROUGH_ARGS"
)

// processOrUsage checks if the provided `Processor` is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func processOrUsage(p Processor, usage *Usage) {
	if n, ok := p.(Node); ok {
		getUsage(n, usage)
	} else {
		p.Usage(usage)
	}
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
