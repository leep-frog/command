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

func BashCompletor[T any](command []string) *Completor[T] {
	return &Completor[T]{
		SuggestionFetcher: BashFetcher[T](command),
	}
}

func BashFetcher[T any](command []string) Fetcher[T] {
	return &bashFetcher[T]{command}
}

type bashFetcher[T any] struct {
	command []string
}

func (bf *bashFetcher[T]) Fetch(T, *Data) (*Completion, error) {
	resp, err := BashCommand[T]("", bf.command).Run(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch autocomplete suggestions with bash command: %v", err)
	}
	return &Completion{
		Suggestions: getOperator[T]().toArgs(resp),
	}, nil
}

// BashCommand runs the provided command in bash and stores the response as
// a value in data as a value with the provided type and argument name.
func BashCommand[T any](argName string, command []string, opts ...BashOption[T]) *bashCommand[T] {
	bc := &bashCommand[T]{
		argName:  argName,
		contents: command,
	}
	for _, o := range opts {
		o.modifyBashNode(bc)
	}
	return bc
}

type hideStderr[T any] struct{}

func (*hideStderr[T]) modifyBashNode(bc *bashCommand[T]) {
	bc.hideStderr = true
}

func HideStderr[T any]() BashOption[T] {
	return &hideStderr[T]{}
}

type forwardStdout[T any] struct{}

func (*forwardStdout[T]) modifyBashNode(bc *bashCommand[T]) {
	bc.forwardStdout = true
}

func ForwardStdout[T any]() BashOption[T] {
	return &forwardStdout[T]{}
}

type bashCommand[T any] struct {
	argName  string
	contents []string
	desc     string

	validators    []*ValidatorOption[T]
	hideStderr    bool
	forwardStdout bool
}

type BashOption[T any] interface {
	modifyBashNode(*bashCommand[T])
}

func (bn *bashCommand[T]) Get(d *Data) T {
	return GetData[T](d, bn.argName)
}

func (bn *bashCommand[T]) Complete(input *Input, data *Data) (*Completion, error) {
	return nil, nil
}

func (bn *bashCommand[T]) Usage(u *Usage) {
	u.Description = bn.desc
}

func (bn *bashCommand[T]) set(v T, d *Data) {
	d.Set(bn.argName, v)
}

func (bn *bashCommand[T]) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	err := bn.execute(input, output, data, eData)
	if bn.hideStderr {
		return err
	}
	return output.Err(err)
}

func (bn *bashCommand[T]) execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	v, err := bn.Run(output)
	if err != nil {
		return err
	}

	bn.set(v, data)
	return nil
}

func DebugMode() bool {
	return os.Getenv("LEEP_FROG_DEBUG") != ""
}

func (bn *bashCommand[T]) Run(output Output) (T, error) {
	var nill T
	// Create temp file.
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return nill, fmt.Errorf("failed to create file for execution: %v", err)
	}

	contents := []string{
		// Exit when any command fails.
		"set -e",
		// Exit if any command in a pipeline fails.
		// https://stackoverflow.com/questions/32684119/exit-when-one-process-in-pipe-fails
		"set -o pipefail",
	}
	contents = append(contents, bn.contents...)

	// Write contents to temp file.
	if _, err := f.WriteString(strings.Join(contents, "\n")); err != nil {
		return nill, fmt.Errorf("failed to write contents to execution file: %v", err)
	}
	if err := f.Close(); err != nil {
		return nill, fmt.Errorf("failed to cleanup temporary execution file: %v", err)
	}

	if DebugMode() {
		output.Stdoutf("Bash execution file: %s\n", f.Name())
	}

	// Execute the contents of the file.
	var rawOut bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	if bn.forwardStdout {
		cmd.Stdout = io.MultiWriter(StdoutWriter(output), &rawOut)
	} else {
		cmd.Stdout = &rawOut
	}
	if bn.hideStderr {
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
		if err := validator.Validate(v); err != nil {
			return nill, fmt.Errorf("validation failed: %v", err)
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
