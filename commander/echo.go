package commander

import (
	"fmt"
	"strings"

	"github.com/leep-frog/command/command"
)

// DryRun is a `command.Processor` that outputs what would happen at a point
// in the graph execution, without actually executing it. Note that any nodes
// after this one will still be executed as normal.
//
// Note that this will always be run. To run this conditionally, consider using
// this in conjunction with `IfData` and `BoolFlag`.
//
//	commander.SerialNodes(
//		FlagProcessor(BoolFlag("dry-run", 'd', "Dry-run mode")),
//		IfData("dry-run", DryRun()),
//	)
func DryRun() command.Processor {
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {

		summary := append([]string{
			"# Dry Run Summary",
			fmt.Sprintf("# Number of executor functions: %d", len(ed.Executor)),
			"# Shell executables:"},
			ed.Executable...,
		)

		o.Stdoutln(strings.Join(summary, "\n"))
		ed.Executable = nil
		ed.Executor = nil
		return nil
	}, nil)
}

// EchoExecuteDataProcessor is a `command.Processor` that outputs the current command.ExecuteData contents.
type EchoExecuteDataProcessor struct {
	// Stderr is whether or not the output should be written to Stderr instead.
	Stderr bool
	// Format
	Format string
}

func (e *EchoExecuteDataProcessor) Execute(_ *command.Input, o command.Output, _ *command.Data, ed *command.ExecuteData) error {
	if e.Format != "" && len(ed.Executable) > 0 {
		if e.Stderr {
			o.Stderrf(e.Format, strings.Join(ed.Executable, "\n"))
		} else {
			o.Stdoutf(e.Format, strings.Join(ed.Executable, "\n"))
		}
		return nil
	}

	for _, s := range ed.Executable {
		if e.Stderr {
			o.Stderrln(s)
		} else {
			o.Stdoutln(s)
		}
	}
	return nil
}

func (e *EchoExecuteDataProcessor) Complete(*command.Input, *command.Data) (*command.Completion, error) {
	return nil, nil
}

func (e *EchoExecuteDataProcessor) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	return nil
}

// EchoExecuteData returns a `command.Processor` that sends the `command.ExecuteData` contents
// to stdout.
func EchoExecuteData() *EchoExecuteDataProcessor {
	return &EchoExecuteDataProcessor{}
}

// EchoExecuteDataf returns a `command.Processor` that sends the `command.ExecuteData` contents
// to stdout with the provided format
func EchoExecuteDataf(format string) command.Processor {
	return &EchoExecuteDataProcessor{Format: format}
}

// PrintlnProcessor returns a `command.Processor` that runs `output.Stdoutln(v)`.
func PrintlnProcessor(v string) command.Processor {
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		o.Stdoutln(v)
		return nil
	}, nil)
}
