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
	v, err := bn.getValue(data)
	if err != nil {
		return output.Err(err)
	}

	for _, validator := range bn.validators {
		if err := validator.Validate(v); err != nil {
			return output.Stderr("validation failed: %v", err)
		}
	}

	bn.set(v, data)
	return nil
}

func (bn *bashCommand) getValue(data *Data) (*Value, error) {
	// Create temp file.
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return nil, fmt.Errorf("failed to create file for execution: %v", err)
	}

	// Write contents to temp file.
	if _, err := f.WriteString(strings.Join(bn.contents, "\n")); err != nil {
		return nil, fmt.Errorf("failed to write contents to execution file: %v", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("failed to cleanup temporary execution file: %v", err)
	}

	// Execute the contents of the file.
	var rawOut, rawErr bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	cmd.Stdout = &rawOut
	cmd.Stderr = &rawErr

	if err := run(cmd); err != nil {
		return nil, err
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
	for s, err = rawOut.ReadString('\n'); err != io.EOF; s, err = rawOut.ReadString('\n') {
		k := strings.TrimSpace(s)
		sl = append(sl, &k)
	}
	if err != io.EOF {
		return nil, fmt.Errorf("failed to read output: %v", err)
	}
	s = strings.TrimSpace(s)
	sl = append(sl, &s)
	return sl, nil
}
