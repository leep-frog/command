package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

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
