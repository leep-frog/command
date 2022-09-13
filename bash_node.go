package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
)

var (
	// Used to stub out tests.
	run = func(cmd *exec.Cmd) error {
		return cmd.Run()
	}
)

// BashCompleter creates a completer object that completes a command graph
// with the output from the provided bash `command`.
func BashCompleter[T any](command ...string) Completer[T] {
	return BashCompleterWithOpts[T](nil, command...)
}

// BashCompleterWithOpts creates a completer object that completes a command graph
// with the output from the provided bash `command`.
func BashCompleterWithOpts[T any](opts *Completion, command ...string) Completer[T] {
	return &simpleCompleter[T]{func(t T, d *Data) (*Completion, error) {
		bc := &BashCommand[[]string]{Contents: command}
		resp, err := bc.Run(nil, d)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch autocomplete suggestions with bash command: %v", err)
		}
		if opts == nil {
			return &Completion{
				Suggestions: resp,
			}, nil
		}
		c := opts.Clone()
		c.Suggestions = resp
		return c, nil
	}}
}

// BashCommand can run the provided command `Contents` in bash and stores the response as
// a value in data with the provided type and `ArgName`.
type BashCommand[T any] struct {
	// ArgName is the argument name to use if stored in `Data`.
	ArgName string
	// Contents contains the bash commands to run.
	Contents []string
	// Desc is the description of this bash command. Used for the CLI usage doc.
	Desc string

	// Validators contains a list of validators to run with the bash output.
	Validators []*ValidatorOption[T]
	// HideStderr is whether or not the stderr of the command should be sent to actual stderr or not.
	HideStderr bool
	// ForwardStdout indicates whether the output should also be displayed (originally, it is just parsed into a value).
	ForwardStdout bool
	// DontRunOnComplete indicates whether or not the bash command should be run when we are completing a command arg.
	DontRunOnComplete bool
	// FormatArgs contains a list of values that will be formatted against the contents.
	// This is used to use data values that are populated during execution.
	FormatArgs []BashDataStringer[T]
}

type BashDataStringer[T any] interface {
	ToString(d *Data) (string, error)
}

func NewBashDataStringer[T, A any](arg *ArgNode[A], delimiter string) BashDataStringer[T] {
	return CustomBashDataStringer[T](func(d *Data) (string, error) {
		return strings.Join(getOperator[A]().toArgs(arg.Get(d)), delimiter), nil
	})
}

func CustomBashDataStringer[T any](f func(*Data) (string, error)) BashDataStringer[T] {
	return &bashDataStringer[T]{f}
}

type bashDataStringer[T any] struct {
	op func(*Data) (string, error)
}

func (df *bashDataStringer[T]) ToString(d *Data) (string, error) {
	return df.op(d)
}

// Name returns the arg name of the `BashCommand`
func (bn *BashCommand[T]) Name() string {
	return bn.ArgName
}

// Get fetches the relevant bash output from the provided `Data` object.
func (bn *BashCommand[T]) Get(d *Data) T {
	return GetData[T](d, bn.Name())
}

// Complete fulfills the `Processor` interface for `BashCommand`.
func (bn *BashCommand[T]) Complete(input *Input, data *Data) (*Completion, error) {
	if bn.DontRunOnComplete {
		return nil, nil
	}
	return nil, bn.execute(&FakeOutput{}, data)
}

// Usage fulfills the `Processor` interface for `BashCommand`.
func (bn *BashCommand[T]) Usage(u *Usage) {
	u.Description = bn.Desc
}

func (bn *BashCommand[T]) set(v T, d *Data) {
	d.Set(bn.Name(), v)
}

// Execute fulfills the `Processor` interface for `BashCommand`.
func (bn *BashCommand[T]) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	err := bn.execute(output, data)
	if bn.HideStderr {
		return err
	}
	return output.Err(err)
}

func (bn *BashCommand[T]) execute(output Output, data *Data) error {
	v, err := bn.Run(output, data)
	if err != nil {
		return err
	}

	bn.set(v, data)
	return nil
}

// Run runs the `BashCommand` with the provided `Output` object.
func (bn *BashCommand[T]) Run(output Output, data *Data) (T, error) {
	var nill T
	// Create temp file.
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return nill, fmt.Errorf("failed to create file for execution: %v", err)
	}

	contents := strings.Join(append([]string{
		// Exit when any command fails.
		"set -e",
		// Exit if any command in a pipeline fails.
		// https://stackoverflow.com/questions/32684119/exit-when-one-process-in-pipe-fails
		"set -o pipefail",
	}, bn.Contents...), "\n")

	if len(bn.FormatArgs) > 0 {
		var args []any
		for _, fa := range bn.FormatArgs {
			s, err := fa.ToString(data)
			if err != nil {
				return nill, fmt.Errorf("failed to get string for bash formatting: %v", err)
			}
			args = append(args, s)
		}
		contents = fmt.Sprintf(contents, args...)
	}

	// Write contents to temp file.
	if _, err := f.WriteString(contents); err != nil {
		return nill, fmt.Errorf("failed to write contents to execution file: %v", err)
	}
	if err := f.Close(); err != nil {
		return nill, fmt.Errorf("failed to cleanup temporary execution file: %v", err)
	}

	Debugf(output, "Bash execution file: %s\n", f.Name())

	// Execute the contents of the file.
	var rawOut bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	if !bn.ForwardStdout || output == nil {
		cmd.Stdout = &rawOut
	} else {
		cmd.Stdout = io.MultiWriter(StdoutWriter(output), &rawOut)
	}
	if bn.HideStderr || output == nil {
		cmd.Stderr = DevNull()
	} else {
		cmd.Stderr = StderrWriter(output)
	}

	if err := run(cmd); err != nil {
		return nill, fmt.Errorf("failed to execute bash command: %v", err)
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
		if err := validator.Validate(bn, v); err != nil {
			return nill, err
		}
	}

	return v, nil
}

func outToSlice(rawOut bytes.Buffer) ([]*string, error) {
	var err error
	var sl []*string
	var s string
	var atLeastOnce bool
	for s, err = rawOut.ReadString('\n'); err != io.EOF; s, err = rawOut.ReadString('\n') {
		atLeastOnce = true
		k := strings.TrimSpace(s)
		sl = append(sl, &k)
	}
	if err != io.EOF {
		return nil, fmt.Errorf("failed to read output: %v", err)
	}
	if atLeastOnce || s != "" {
		s = strings.TrimSpace(s)
		sl = append(sl, &s)
	}
	return sl, nil
}
