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

// BashCommand runs the provided command in bash and stores the response as
// a value in data as a value with the provided type and argument name.
func BashCommand(vt ValueType, argName string, command []string, opts ...BashOption) *bashCommand {
	bc := &bashCommand{
		vt:       vt,
		argName:  argName,
		contents: command,
	}
	for _, o := range opts {
		o.modifyBashNode(bc)
	}
	return bc
}

type bashCommand struct {
	vt       ValueType
	argName  string
	contents []string

	validators []*validatorOption
}

type BashOption interface {
	modifyBashNode(*bashCommand)
}

func (bn *bashCommand) Get(d *Data) *Value {
	return (*d)[bn.argName]
}

func (bn *bashCommand) Complete(input *Input, data *Data) *CompleteData {
	return nil
}

func (bn *bashCommand) set(v *Value, d *Data) {
	d.Set(bn.argName, v)
}

func (bn *bashCommand) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	v, err := bn.getValue(data, output)
	if err != nil {
		return output.Err(err)
	}

	for _, validator := range bn.validators {
		if err := validator.Validate(v); err != nil {
			return output.Stderrf("validation failed: %v", err)
		}
	}

	bn.set(v, data)
	return nil
}

func DebugMode() bool {
	return os.Getenv("LEEP_FROG_DEBUG") != ""
}

func (bn *bashCommand) getValue(data *Data, output Output) (*Value, error) {
	// Create temp file.
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return nil, fmt.Errorf("failed to create file for execution: %v", err)
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
		return nil, fmt.Errorf("failed to write contents to execution file: %v", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("failed to cleanup temporary execution file: %v", err)
	}

	if DebugMode() {
		output.Stdoutf("Bash execution file: %s\n", f.Name())
	}

	// Execute the contents of the file.
	var rawOut, rawErr bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	cmd.Stdout = &rawOut
	cmd.Stderr = &rawErr

	if err := run(cmd); err != nil {
		fmt.Println("yup")
		retErr := fmt.Errorf("failed to execute bash command: %v", err)

		sl, sliceErr := outToSlice(rawErr)
		if sliceErr != nil {
			output.Stderrf("failed to read stderr: %v", sliceErr)
			return nil, retErr
		}

		v, tErr := vtMap.transform(StringListType, sl, "ugh")
		if tErr != nil {
			output.Stderrf("failed to convert stderr to string slice: %v", tErr)
		}

		for _, s := range v.StringList() {
			output.Stderr(s)
		}
		return nil, retErr
	}

	sl, err := outToSlice(rawOut)
	if err != nil {
		return nil, err
	}

	return vtMap.transform(bn.vt, sl, "bn")
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
