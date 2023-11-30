package commander

import (
	"strings"

	"github.com/leep-frog/command/commondels"
)

// EchoExecuteDataProcessor is a `commondels.Processor` that outputs the current commondels.ExecuteData contents.
type EchoExecuteDataProcessor struct {
	// Stderr is whether or not the output should be written to Stderr instead.
	Stderr bool
	// Format
	Format string
}

func (e *EchoExecuteDataProcessor) Execute(_ *commondels.Input, o commondels.Output, _ *commondels.Data, ed *commondels.ExecuteData) error {
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

func (e *EchoExecuteDataProcessor) Complete(*commondels.Input, *commondels.Data) (*commondels.Completion, error) {
	return nil, nil
}

func (e *EchoExecuteDataProcessor) Usage(i *commondels.Input, d *commondels.Data, u *commondels.Usage) error {
	return nil
}

// EchoExecuteData returns a `commondels.Processor` that sends the `commondels.ExecuteData` contents
// to stdout.
func EchoExecuteData() *EchoExecuteDataProcessor {
	return &EchoExecuteDataProcessor{}
}

// EchoExecuteDataf returns a `commondels.Processor` that sends the `commondels.ExecuteData` contents
// to stdout with the provided format
func EchoExecuteDataf(format string) commondels.Processor {
	return &EchoExecuteDataProcessor{Format: format}
}

// PrintlnProcessor returns a `commondels.Processor` that runs `output.Stdoutln(v)`.
func PrintlnProcessor(v string) commondels.Processor {
	return SimpleProcessor(func(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
		o.Stdoutln(v)
		return nil
	}, nil)
}
