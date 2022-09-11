package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
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
		resp, err := NewBashCommand[[]string]("", command).Run(nil, d)
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

// TODO: Change things to just structs and remove constructor functions.
// NewBashCommand runs the provided command in bash and stores the response as
// a value in data with the provided type and argument name.
func NewBashCommand[T any](argName string, command []string, opts ...BashOption[T]) *BashCommand[T] {
	bc := &BashCommand[T]{
		argName:  argName,
		contents: command,
	}
	for _, o := range opts {
		o.modifyBashNode(bc)
	}
	return bc
}

type hideStderr[T any] struct{}

func (*hideStderr[T]) modifyBashNode(bc *BashCommand[T]) {
	bc.hideStderr = true
}

// HideStderr is a `BashOption` that hides stderr output when
// running the bash command.
func HideStderr[T any]() BashOption[T] {
	return &hideStderr[T]{}
}

type forwardStdout[T any] struct{}

func (*forwardStdout[T]) modifyBashNode(bc *BashCommand[T]) {
	bc.forwardStdout = true
}

// ForwardStdout is a `BashOption` that forwards stdout to the terminal (rather than just parsing it).
func ForwardStdout[T any]() BashOption[T] {
	return &forwardStdout[T]{}
}

type dontRunOnComplete[T any] struct{}

func (*dontRunOnComplete[T]) modifyBashNode(bc *BashCommand[T]) {
	bc.dontRunOnComplete = true
}

// DontRunOnComplete is a `BashOption` that skips the node's execution when in the `Complete` context.
func DontRunOnComplete[T any]() BashOption[T] {
	return &dontRunOnComplete[T]{}
}

type BashCommand[T any] struct {
	argName  string
	contents []string
	desc     string

	validators        []*ValidatorOption[T]
	hideStderr        bool
	forwardStdout     bool
	dontRunOnComplete bool
	formatArgs        []BashDataStringer[T]
}

type BashDataStringer[T any] interface {
	BashOption[T]
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

func (df *bashDataStringer[T]) modifyBashNode(bc *BashCommand[T]) {
	bc.formatArgs = append(bc.formatArgs, df)
}

// BashOption is an option type for modifying `BashNode` objects
type BashOption[T any] interface {
	modifyBashNode(*BashCommand[T])
}

// Name returns the arg name of the `BashCommand`
func (bn *BashCommand[T]) Name() string {
	return bn.argName
}

// Get fetches the relevant bash output from the provided `Data` object.
func (bn *BashCommand[T]) Get(d *Data) T {
	return GetData[T](d, bn.argName)
}

// Complete fulfills the `Processor` interface for `BashCommand`.
func (bn *BashCommand[T]) Complete(input *Input, data *Data) (*Completion, error) {
	if bn.dontRunOnComplete {
		return nil, nil
	}
	return nil, bn.execute(&FakeOutput{}, data)
}

// Usage fulfills the `Processor` interface for `BashCommand`.
func (bn *BashCommand[T]) Usage(u *Usage) {
	u.Description = bn.desc
}

func (bn *BashCommand[T]) set(v T, d *Data) {
	d.Set(bn.argName, v)
}

// Execute fulfills the `Processor` interface for `BashCommand`.
func (bn *BashCommand[T]) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	err := bn.execute(output, data)
	if bn.hideStderr {
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

// DebugMode returns whether or not debug mode is active.
// TODO: Separate debug.go file that contains all info like this.
// TODO: have debug mode point to directory or file
//       and all output can be written there.
func DebugMode() bool {
	return os.Getenv("LEEP_FROG_DEBUG") != ""
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
	}, bn.contents...), "\n")

	if len(bn.formatArgs) > 0 {
		var args []any
		for _, fa := range bn.formatArgs {
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

	if DebugMode() {
		// TODO: global "mode" variable (execute, complete, usage)
		//       maybe store it in data?
		output.Stdoutf("Bash execution file: %s\n", f.Name())
	}

	// Execute the contents of the file.
	var rawOut bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	if !bn.forwardStdout || output == nil {
		cmd.Stdout = &rawOut
	} else {
		cmd.Stdout = io.MultiWriter(StdoutWriter(output), &rawOut)
	}
	if bn.hideStderr || output == nil {
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

	for _, validator := range bn.validators {
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
