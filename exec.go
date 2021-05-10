package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
)

// TODO: change this to RunValue(contents []string, vt ValueType) (*Value, error)
func RunInt(contents []string) (int, error) {
	result, err := RunOne(contents)
	if err != nil {
		return 0, err
	}
	i, err := strconv.Atoi(result)
	if err != nil {
		return 0, fmt.Errorf("failed to convert Run result to int: %v", err)
	}
	return i, nil
}

func RunOne(contents []string) (string, error) {
	result, err := Run(contents)
	if err != nil {
		return "", err
	}
	if len(result) != 2 || result[0] != "" {
		return "", fmt.Errorf("unexpected number of results (%d): %s", len(result), strings.Join(result, "_"))
	}
	// result is [value, "" (because of newline)]
	return result[0], nil
}

func Run(contents []string) ([]string, error) {
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return nil, fmt.Errorf("failed to create file for execution: %v", err)
	}

	if _, err := f.WriteString(strings.Join(contents, "\n")); err != nil {
		return nil, fmt.Errorf("failed to write contents to execution file: %v", err)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("failed to cleanup temporary execution file: %v", err)
	}

	var rawOut, rawErr bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	cmd.Stdout = &rawOut
	cmd.Stderr = &rawErr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run command: %v", err)
	}

	sOut := string(rawOut.Bytes())
	if len(sOut) == 0 {
		return nil, nil
	}

	return strings.Split(sOut, "\n"), nil
}
