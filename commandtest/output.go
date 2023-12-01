package commandtest

import (
	"strings"

	"github.com/leep-frog/command/command"
)

// Output is a fake `Output` object that can be used for testing.
type Output struct {
	command.Output
	stdout []string
	stderr []string
	closed bool
}

// NewOutput returns a new `Output` object.
func NewOutput() *Output {
	tcos := &Output{}
	so := func(s string) {
		tcos.stdout = append(tcos.stdout, s)
	}
	se := func(s string) {
		tcos.stderr = append(tcos.stderr, s)
	}
	tcos.Output = command.OutputFromFuncs(so, se)
	return tcos
}

// Close closes the fake output channel.
func (fo *Output) Close() {
	if !fo.closed {
		fo.Output.Close()
		fo.closed = true
	}
}

// GetStdout returns all of the data that was written to the stdout channel.
func (fo *Output) GetStdout() string {
	fo.Close()
	return strings.Join(fo.stdout, "")
}

// GetStdoutByCalls returns all of the individual calls made to stdout.
func (fo *Output) GetStdoutByCalls() []string {
	fo.Close()
	return fo.stdout
}

// GetStderr returns all of the data that was written to the stderr channel.
func (fo *Output) GetStderr() string {
	fo.Close()
	return strings.Join(fo.stderr, "")
}

// GetStderrByCalls returns all of the individual calls made to stdout.
func (fo *Output) GetStderrByCalls() []string {
	fo.Close()
	return fo.stderr
}
