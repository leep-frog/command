package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
)

// BashCommand runs the provided command in bash and stores the response as
// a value in data as a value with the provided type and argument name.
func BashCommand(vt ValueType, argName string, command ...string) *bashCommand {
	return &bashCommand{
		vt:       vt,
		argName:  argName,
		contents: command,
	}
}

type bashCommand struct {
	vt       ValueType
	argName  string
	contents []string
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
	// Create temp file.
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return fmt.Errorf("failed to create file for execution: %v", err)
	}

	// Write contents to temp file.
	if _, err := f.WriteString(strings.Join(bn.contents, "\n")); err != nil {
		return fmt.Errorf("failed to write contents to execution file: %v", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to cleanup temporary execution file: %v", err)
	}

	// Execute the contents of the file.
	var rawOut, rawErr bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	cmd.Stdout = &rawOut
	cmd.Stderr = &rawErr

	if err := cmd.Run(); err != nil {
		return err
	}

	switch bn.vt {
	case StringType:
		bn.set(StringValue(rawOut.String()), data)
		return nil
	case IntType:
		i, err := strconv.Atoi(strings.TrimSpace(rawOut.String()))
		bn.set(IntValue(i), data)
		return err
	case FloatType:
		f, err := strconv.ParseFloat(strings.TrimSpace(rawOut.String()), 64)
		bn.set(FloatValue(f), data)
		return err
	case BoolType:
		f, err := strconv.ParseBool(strings.TrimSpace(rawOut.String()))
		bn.set(BoolValue(f), data)
		return err
	case StringListType:
		var sl []string
		for s, err := rawOut.ReadString('\n'); err != nil; s, err = rawOut.ReadString('\n') {
			sl = append(sl, s)
		}
		if err != io.EOF {
			return fmt.Errorf("failed to read output: %v", err)
		}
		bn.set(StringListValue(sl...), data)
		return nil
	case IntListType:
		var il []int
		for s, err := rawOut.ReadString('\n'); err != nil; s, err = rawOut.ReadString('\n') {
			i, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				return err
			}
			il = append(il, i)
		}
		if err != io.EOF {
			return fmt.Errorf("failed to read output: %v", err)
		}
		bn.set(IntListValue(il...), data)
		return nil
	case FloatListType:
		var fl []float64
		for s, err := rawOut.ReadString('\n'); err != nil; s, err = rawOut.ReadString('\n') {
			f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return err
			}
			fl = append(fl, f)
		}
		if err != io.EOF {
			return fmt.Errorf("failed to read output: %v", err)
		}
		bn.set(FloatListValue(fl...), data)
		return nil
	}
	return fmt.Errorf("unknown value type for bash execution: %v", bn.vt)
}
