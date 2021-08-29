package command

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

// Output defines methods for writing output.
type Output interface {
	// Writes a line to stdout.
	Stdout(string, ...interface{})
	// Writes a line to stderr and returns an error with the same message.
	Stderr(string, ...interface{}) error
	// Writes the provided error to stderr and returns the provided error.
	Err(err error) error
	// Close informs the os that no more data will be written.
	Close()
}

type output struct {
	stdoutChan chan string
	stderrChan chan string
	wg         *sync.WaitGroup
}

func (o *output) Stdout(s string, a ...interface{}) {
	if len(a) == 0 {
		o.stdoutChan <- s
	} else {
		o.stdoutChan <- fmt.Sprintf(s, a...)
	}
}

func (o *output) Stderr(s string, a ...interface{}) error {
	var err error
	if len(a) == 0 {
		err = errors.New(s)
	} else {
		err = fmt.Errorf(s, a...)
	}
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

func (ieo *ignoreErrOutput) Stdout(s string, a ...interface{}) {
	ieo.o.Stdout(s, a...)
}

func (ieo *ignoreErrOutput) Stderr(s string, a ...interface{}) error {
	return ieo.o.Stderr(s, a...)
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

func (fo *FakeOutput) Stdout(s string, a ...interface{}) {
	fo.c.Stdout(s, a...)
}

func (fo *FakeOutput) Stderr(s string, a ...interface{}) error {
	return fo.c.Stderr(s, a...)
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
