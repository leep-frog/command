package command

import "strings"

// EchoExecuteDataProcessor is a `Processor` that outputs the current ExecuteData contents.
type EchoExecuteDataProcessor struct {
	// Stderr is whether or not the output should be written to Stderr instead.
	Stderr bool
	// Format
	Format string
}

func (e *EchoExecuteDataProcessor) Execute(_ *Input, o Output, _ *Data, ed *ExecuteData) error {
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

func (e *EchoExecuteDataProcessor) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

func (e *EchoExecuteDataProcessor) Usage(*Usage) {}

// EchoExecuteData returns a `Processor` that sends the `ExecuteData` contents
// to stdout.
func EchoExecuteData() *EchoExecuteDataProcessor {
	return &EchoExecuteDataProcessor{}
}

// EchoExecuteDataf returns a `Processor` that sends the `ExecuteData` contents
// to stdout with the provided format
func EchoExecuteDataf(format string) Processor {
	return &EchoExecuteDataProcessor{Format: format}
}

// PrintlnProcessor returns a `Processor` that runs `output.Stdoutln(v)`.
func PrintlnProcessor(v string) Processor {
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		o.Stdoutln(v)
		return nil
	}, nil)
}
