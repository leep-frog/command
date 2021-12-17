package command

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// Output defines methods for writing output.
type Output interface {
	// TODO: add Stdoutln and Stderrln
	// Writes a line to stdout.
	Stdout(string)
	// Writes a line to stderr and returns an error with the same message.
	Stderr(string) error
	// Writes a formatted line to stdout.
	Stdoutf(string, ...interface{})
	// Writes a formatted line to stderr and returns an error with the same message.
	Stderrf(string, ...interface{}) error
	// Writes the provided error to stderr and returns the provided error.
	Err(err error) error
	// Close informs the os that no more data will be written.
	Close()
}

type outputWriter struct {
	writeFunc func(string)
}

func (ow *outputWriter) Write(b []byte) (int, error) {
	if ow.writeFunc != nil {
		ow.writeFunc(string(b))
	}
	return len(b), nil
}

func DevNull() io.Writer {
	return &outputWriter{}
}

func StdoutWriter(o Output) io.Writer {
	return &outputWriter{o.Stdout}
}

func StderrWriter(o Output) io.Writer {
	return &outputWriter{func(s string) { o.Stderr(s) }}
}

type output struct {
	stdoutChan chan string
	stderrChan chan string
	wg         *sync.WaitGroup
}

func (o *output) Stdout(s string) {
	o.stdoutChan <- s
}

func (o *output) Stdoutf(s string, a ...interface{}) {
	o.stdoutChan <- fmt.Sprintf(s, a...)
}

func (o *output) Stderr(s string) error {
	err := errors.New(s)
	o.stderrChan <- err.Error()
	return err
}

func (o *output) Stderrf(s string, a ...interface{}) error {
	err := fmt.Errorf(s, a...)
	o.stderrChan <- err.Error()
	return err
}

func (o *output) Err(err error) error {
	if err != nil {
		o.Stderr(err.Error())
	}
	return err
}

func (o *output) Close() {
	close(o.stdoutChan)
	close(o.stderrChan)
	o.wg.Wait()
}

// NewOutput returns an output that points to stdout and stderr.
func NewOutput() Output {
	stdout := log.New(os.Stdout, "", 0)
	stderr := log.New(os.Stderr, "", 0)
	so := func(s string) {
		stdout.Println(s)
	}
	se := func(s string) {
		stderr.Println(s)
	}
	return osFromChan(so, se)
}

func osFromChan(so, se func(string)) Output {
	stdoutChan := make(chan string)
	stderrChan := make(chan string)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		for s := range stdoutChan {
			so(s)
		}
		wg.Done()
	}()

	go func() {
		for s := range stderrChan {
			se(s)
		}
		wg.Done()
	}()
	return &output{
		stdoutChan: stdoutChan,
		stderrChan: stderrChan,
		wg:         &wg,
	}
}

func NewIgnoreErrOutput(o Output, fs ...func(error) bool) Output {
	return &ignoreErrOutput{
		o:  o,
		fs: fs,
	}
}

type ignoreErrOutput struct {
	o  Output
	fs []func(error) bool
}

func (ieo *ignoreErrOutput) Stdout(s string) {
	ieo.o.Stdout(s)
}

func (ieo *ignoreErrOutput) Stdoutf(s string, a ...interface{}) {
	ieo.o.Stdoutf(s, a...)
}

func (ieo *ignoreErrOutput) Stderr(s string) error {
	return ieo.o.Stderr(s)
}

func (ieo *ignoreErrOutput) Stderrf(s string, a ...interface{}) error {
	return ieo.o.Stderrf(s, a...)
}

func (ieo *ignoreErrOutput) Err(err error) error {
	// Don't output the error if it matches a filter.
	for _, f := range ieo.fs {
		if f(err) {
			return err
		}
	}
	// Regular output functionality if no filter matched.
	return ieo.o.Err(err)
}

func (ieo *ignoreErrOutput) Close() {
	ieo.o.Close()
}

type FakeOutput struct {
	stdout []string
	stderr []string
	c      Output
	closed bool
}

func NewFakeOutput() *FakeOutput {
	tcos := &FakeOutput{}
	so := func(s string) {
		tcos.stdout = append(tcos.stdout, s)
	}
	se := func(s string) {
		tcos.stderr = append(tcos.stderr, s)
	}
	cos := osFromChan(so, se)
	tcos.c = cos
	return tcos
}

func (fo *FakeOutput) Stdout(s string) {
	fo.c.Stdout(s)
}

func (fo *FakeOutput) Stdoutf(s string, a ...interface{}) {
	fo.c.Stdoutf(s, a...)
}

func (fo *FakeOutput) Stderr(s string) error {
	return fo.c.Stderr(s)
}

func (fo *FakeOutput) Stderrf(s string, a ...interface{}) error {
	return fo.c.Stderrf(s, a...)
}

func (fo *FakeOutput) Err(err error) error {
	return fo.c.Err(err)
}

func (fo *FakeOutput) Close() {
	if !fo.closed {
		fo.c.Close()
		fo.closed = true
	}
}

func (fo *FakeOutput) GetStdout() []string {
	fo.Close()
	return fo.stdout
}

func (fo *FakeOutput) GetStderr() []string {
	fo.Close()
	return fo.stderr
}
