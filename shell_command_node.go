package command

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

var (
	// Used to stub out tests.
	run = func(cmd *exec.Cmd) error {
		return cmd.Run()
	}
)

// ShellCommandCompleter creates a completer object that completes a command
// graph with the output from the provided shell command.
func ShellCommandCompleter[T any](name string, args ...string) Completer[T] {
	return ShellCommandCompleterWithOpts[T](nil, name, args...)
}

// ShellCommandCompleterWithOpts creates a completer object that completes a command graph
// with the output from the provided command info.
func ShellCommandCompleterWithOpts[T any](opts *Completion, name string, args ...string) Completer[T] {
	return &simpleCompleter[T]{func(t T, d *Data) (*Completion, error) {
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
			return &Completion{
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
	// ArgName is the argument name to use if stored in `Data`.
	ArgName string
	// Command is the command to forward to `exec.Command`.
	CommandName string
	// Args are the args to forward to `exec.Command`.
	Args []string
	// Desc is the description of this shell command. Used for the CLI usage doc.
	Desc string

	// Validators contains a list of validators to run with the shell command output.
	Validators []*ValidatorOption[T]
	// HideStderr is whether or not the stderr of the command should be sent to actual stderr or not.
	HideStderr bool
	// ForwardStdout indicates whether the output should also be displayed (originally, it is just parsed into a value).
	ForwardStdout bool
	// DontRunOnComplete indicates whether or not the shell command should be run when we are completing a command arg.
	DontRunOnComplete bool
	// OutputStreamProcessor is a function that will be run with every item written to stdout.
	OutputStreamProcessor func(Output, *Data, []byte) error
}

// TODO: This from `ShellCommandFileRunner`
/*func ShellCommandFromScript[T any](sc *ShellCommand[T], script []string) *ShellCommand[T] {
	// Create temp file.
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return nill, fmt.Errorf("failed to create file for execution: %v", err)
	}

	// Write contents to temp file.
	if _, err := f.WriteString(contents); err != nil {
		return nill, fmt.Errorf("failed to write contents to execution file: %v", err)
	}
	if err := f.Close(); err != nil {
		return nill, fmt.Errorf("failed to cleanup temporary execution file: %v", err)
	}

	Debugf(output, "Shell execution file: %s\n", f.Name())
}*/

type ShellCommandDataStringer[T any] interface {
	ToString(d *Data) (string, error)
}

func NewShellCommandDataStringer[T, A any](arg *Argument[A], delimiter string) ShellCommandDataStringer[T] {
	return CustomShellCommandDataStringer[T](func(d *Data) (string, error) {
		return strings.Join(getOperator[A]().toArgs(arg.Get(d)), delimiter), nil
	})
}

func CustomShellCommandDataStringer[T any](f func(*Data) (string, error)) ShellCommandDataStringer[T] {
	return &shellDataStringer[T]{f}
}

type shellDataStringer[T any] struct {
	op func(*Data) (string, error)
}

func (df *shellDataStringer[T]) ToString(d *Data) (string, error) {
	return df.op(d)
}

// Name returns the arg name of the `ShellCommand`
func (bn *ShellCommand[T]) Name() string {
	return bn.ArgName
}

// Get fetches the relevant shell output from the provided `Data` object.
func (bn *ShellCommand[T]) Get(d *Data) T {
	return GetData[T](d, bn.Name())
}

// Complete fulfills the `Processor` interface for `ShellCommand`.
func (bn *ShellCommand[T]) Complete(input *Input, data *Data) (*Completion, error) {
	if bn.DontRunOnComplete {
		return nil, nil
	}
	return nil, bn.execute(&FakeOutput{}, data)
}

// Usage fulfills the `Processor` interface for `ShellCommand`.
func (bn *ShellCommand[T]) Usage(i *Input, d *Data, u *Usage) error {
	u.Description = bn.Desc
	return nil
}

func (bn *ShellCommand[T]) set(v T, d *Data) {
	d.Set(bn.Name(), v)
}

// Execute fulfills the `Processor` interface for `ShellCommand`.
func (bn *ShellCommand[T]) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	err := bn.execute(output, data)
	if bn.HideStderr {
		return err
	}
	return output.Err(err)
}

func (bn *ShellCommand[T]) execute(output Output, data *Data) error {
	v, err := bn.Run(output, data)
	if err != nil {
		return err
	}

	bn.set(v, data)
	return nil
}

type outputStreamer struct {
	f func(Output, *Data, []byte) error
	d *Data
	o Output
}

func (os *outputStreamer) Write(b []byte) (int, error) {
	return len(b), os.f(os.o, os.d, b)
}

// Run runs the `ShellCommand` with the provided `Output` object.
func (bn *ShellCommand[T]) Run(output Output, data *Data) (T, error) {
	var nill T

	// Execute the command
	var rawOut bytes.Buffer
	stdoutWriters := []io.Writer{&rawOut}
	cmd := exec.Command(bn.CommandName, bn.Args...)
	if bn.ForwardStdout && output != nil {
		stdoutWriters = append(stdoutWriters, StdoutWriter(output))
	}
	if bn.OutputStreamProcessor != nil {
		stdoutWriters = append(stdoutWriters, &outputStreamer{bn.OutputStreamProcessor, data, output})
	}
	cmd.Stdout = io.MultiWriter(stdoutWriters...)

	if bn.HideStderr || output == nil {
		cmd.Stderr = DevNull()
	} else {
		cmd.Stderr = StderrWriter(output)
	}

	if err := run(cmd); err != nil {
		return nill, fmt.Errorf("failed to execute shell command: %v", err)
	}

	sl, err := outToSlice(rawOut)
	if err != nil {
		return nill, err
	}

	op := getOperator[T]()
	v, err := op.fromArgs(sl)
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
