package commander

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/leep-frog/command/commondels"
	"github.com/leep-frog/command/internal/operator"
	"github.com/leep-frog/command/internal/stubs"
)

// ShellCommandCompleter creates a completer object that completes a command
// graph with the output from the provided shell command.
func ShellCommandCompleter[T any](name string, args ...string) Completer[T] {
	return ShellCommandCompleterWithOpts[T](nil, name, args...)
}

// ShellCommandCompleterWithOpts creates a completer object that completes a command graph
// with the output from the provided command info.
func ShellCommandCompleterWithOpts[T any](opts *commondels.Completion, name string, args ...string) Completer[T] {
	return &simpleCompleter[T]{func(t T, d *commondels.Data) (*commondels.Completion, error) {
		bc := &ShellCommand[[]string]{
			CommandName: name,
			Args:        args,
		}
		resp, err := bc.Run(nil, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch autocomplete suggestions with shell command: %v", err)
		}
		var filtered []string
		for _, r := range resp {
			f := strings.TrimSpace(r)
			if f != "" {
				filtered = append(filtered, f)
			}
		}
		if opts == nil {
			return &commondels.Completion{
				Suggestions: filtered,
			}, nil
		}
		c := opts.Clone()
		c.Suggestions = filtered
		return c, nil
	}}
}

// ShellCommand can run the provided command `Contents` in the shell and stores
// the response as a value in data with the provided type and `ArgName`.
type ShellCommand[T any] struct {
	// ArgName is the argument name to use if stored in `commondels.Data`.
	ArgName string
	// Command is the command to forward to `exec.Command`.
	CommandName string
	// Args are the args to forward to `exec.Command`.
	Args []string
	// Desc is the description of this shell command. Used for the CLI usage doc.
	Desc string
	// Dir is the directory in which to run the command. Defaults to the current directory.
	Dir string
	// Stdin is the `io.Reader` to forward for use in `exec.Command`
	Stdin io.Reader

	// Validators contains a list of validators to run with the shell command output.
	Validators []*ValidatorOption[T]
	// HideStderr is whether or not the stderr of the command should be sent to actual stderr or not.
	HideStderr bool
	// ForwardStdout indicates whether the output should also be displayed (originally, it is just parsed into a value).
	ForwardStdout bool
	// DontRunOnComplete indicates whether or not the shell command should be run when we are completing a command arg.
	DontRunOnComplete bool
	// OutputStreamProcessor is a function that will be run with every item written to stdout.
	OutputStreamProcessor func(commondels.Output, *commondels.Data, []byte) error
	// EchoCommand, if true, forwards the command being run (with args) to Stdout.
	EchoCommand bool
}

type ShellCommandDataStringer[T any] interface {
	ToString(d *commondels.Data) (string, error)
}

func NewShellCommandDataStringer[T, A any](arg *Argument[A], delimiter string) ShellCommandDataStringer[T] {
	return CustomShellCommandDataStringer[T](func(d *commondels.Data) (string, error) {
		return strings.Join(operator.GetOperator[A]().ToArgs(arg.Get(d)), delimiter), nil
	})
}

func CustomShellCommandDataStringer[T any](f func(*commondels.Data) (string, error)) ShellCommandDataStringer[T] {
	return &shellDataStringer[T]{f}
}

type shellDataStringer[T any] struct {
	op func(*commondels.Data) (string, error)
}

func (df *shellDataStringer[T]) ToString(d *commondels.Data) (string, error) {
	return df.op(d)
}

// Name returns the arg name of the `ShellCommand`
func (bn *ShellCommand[T]) Name() string {
	return bn.ArgName
}

// Get fetches the relevant shell output from the provided `commondels.Data` object.
func (bn *ShellCommand[T]) Get(d *commondels.Data) T {
	return commondels.GetData[T](d, bn.Name())
}

// Complete fulfills the `commondels.Processor` interface for `ShellCommand`.
func (bn *ShellCommand[T]) Complete(input *commondels.Input, data *commondels.Data) (*commondels.Completion, error) {
	if bn.DontRunOnComplete {
		return nil, nil
	}
	return nil, bn.execute(&commondels.FakeOutput{}, data)
}

// commondels.Usage fulfills the `commondels.Processor` interface for `ShellCommand`.
func (bn *ShellCommand[T]) Usage(i *commondels.Input, d *commondels.Data, u *commondels.Usage) error {
	u.Description = bn.Desc
	return nil
}

func (bn *ShellCommand[T]) set(v T, d *commondels.Data) {
	if bn.Name() != "" {
		d.Set(bn.Name(), v)
	}
}

// Execute fulfills the `commondels.Processor` interface for `ShellCommand`.
func (bn *ShellCommand[T]) Execute(input *commondels.Input, output commondels.Output, data *commondels.Data, eData *commondels.ExecuteData) error {
	err := bn.execute(output, data)
	if bn.HideStderr {
		return err
	}
	return output.Err(err)
}

func (bn *ShellCommand[T]) execute(output commondels.Output, data *commondels.Data) error {
	v, err := bn.Run(output, data)
	if err != nil {
		return err
	}

	bn.set(v, data)
	return nil
}

type outputStreamer struct {
	f func(commondels.Output, *commondels.Data, []byte) error
	d *commondels.Data
	o commondels.Output
}

func (os *outputStreamer) Write(b []byte) (int, error) {
	return len(b), os.f(os.o, os.d, b)
}

// Run runs the `ShellCommand` with the provided `commondels.Output` object.
func (bn *ShellCommand[T]) Run(output commondels.Output, data *commondels.Data) (T, error) {
	var nill T

	// Execute the command
	var rawOut bytes.Buffer
	stdoutWriters := []io.Writer{&rawOut}
	cmd := exec.Command(bn.CommandName, bn.Args...)
	cmd.Stdin = bn.Stdin
	cmd.Dir = bn.Dir
	if bn.ForwardStdout && output != nil {
		stdoutWriters = append(stdoutWriters, commondels.StdoutWriter(output))
	}
	if bn.OutputStreamProcessor != nil {
		stdoutWriters = append(stdoutWriters, &outputStreamer{bn.OutputStreamProcessor, data, output})
	}
	cmd.Stdout = io.MultiWriter(stdoutWriters...)

	if bn.HideStderr || output == nil {
		cmd.Stderr = commondels.DevNull()
	} else {
		cmd.Stderr = commondels.StderrWriter(output)
	}

	if bn.EchoCommand {
		output.Stdoutf("%s %s\n", bn.CommandName, strings.Join(bn.Args, " "))
	}
	if err := stubs.Run(cmd); err != nil {
		return nill, fmt.Errorf("failed to execute shell command: %v", err)
	}

	sl, err := outToSlice(rawOut)
	if err != nil {
		return nill, err
	}

	op := operator.GetOperator[T]()
	v, err := op.FromArgs(sl)
	if err != nil {
		return nill, err
	}

	for _, validator := range bn.Validators {
		if err := validator.RunValidation(bn, v, data); err != nil {
			return v, err
		}
	}

	return v, nil
}

func outToSlice(rawOut bytes.Buffer) ([]*string, error) {
	var err error
	var sl []*string
	var s string
	for s, err = rawOut.ReadString('\n'); err != io.EOF; s, err = rawOut.ReadString('\n') {
		k := strings.TrimSpace(s)
		sl = append(sl, &k)
	}
	if err != io.EOF {
		return nil, fmt.Errorf("failed to read output: %v", err)
	}
	if s != "" {
		s = strings.TrimSpace(s)
		sl = append(sl, &s)
	}
	return sl, nil
}
