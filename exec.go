package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const (
	NoExitCode = -1
)

// TODO: change this to RunValue(contents []string, vt ValueType) (*Value, error)
func RunInt(contents []string) (int, error, int) {
	result, err, exitCode := RunOne(contents)
	if err != nil {
		return 0, err, exitCode
	}
	i, err := strconv.Atoi(result)
	if err != nil {
		return 0, fmt.Errorf("failed to convert Run result to int: %v", err), exitCode
	}
	return i, nil, exitCode
}

func RunOne(contents []string) (string, error, int) {
	result, err, exitCode := Run(contents)
	if err != nil {
		return "", err, exitCode
	}
	if len(result) != 2 || result[1] != "" {
		return "", fmt.Errorf("unexpected number of results (%d): %s", len(result), result), exitCode
	}
	// result is [value, "" (because of newline)]
	return result[0], nil, exitCode
}

// returns output, error and exit status.
func Run(contents []string) ([]string, error, int) {
	f, err := ioutil.TempFile("", "leepFrogCommandExecution")
	if err != nil {
		return nil, fmt.Errorf("failed to create file for execution: %v", err), NoExitCode
	}

	if _, err := f.WriteString(strings.Join(contents, "\n")); err != nil {
		return nil, fmt.Errorf("failed to write contents to execution file: %v", err), NoExitCode
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("failed to cleanup temporary execution file: %v", err), NoExitCode
	}

	var rawOut, rawErr bytes.Buffer
	// msys/mingw doesn't work if "bash" is excluded.
	cmd := exec.Command("bash", f.Name())
	cmd.Stdout = &rawOut
	cmd.Stderr = &rawErr

	if err := cmd.Run(); err != nil {
		exitCode := NoExitCode
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
		return nil, fmt.Errorf("failed to run command: %v", err), exitCode
	}

	sOut := string(rawOut.Bytes())
	if len(sOut) == 0 {
		return nil, nil, 0
	}

	return strings.Split(sOut, "\n"), nil, 0
}
