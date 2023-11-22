package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/leep-frog/command/glog"
)

// Output defines methods for writing output.
type Output interface {
	// Writes the provided text to stdout.
	Stdout(string)
	// Writes the provided text to stderr and returns an error with the same message.
	Stderr(string) error
	// Writes a formatted string to stdout.
	Stdoutf(string, ...interface{})
	// Writes a formatted string to stderr and returns an error with the same message.
	Stderrf(string, ...interface{}) error
	// Writes interfaces to stdout and appends a newline.
	Stdoutln(...interface{})
	// Writes interfaces to stderr and appends a newline.
	Stderrln(...interface{}) error
	// Writes the provided error to stderr and returns the provided error.
	Err(err error) error
	// Annotate prepends the message to the error
	Annotate(error, string) error
	// Annotatef prepends the message to the error
	Annotatef(error, string, ...interface{}) error
	// Terminate terminates the execution with the provided error (if it's not nil).
	Terminate(error)
	// Terminatef terminates the execution with a formatted error.
	Terminatef(string, ...interface{})
	// Tannotate terminates the execution with the an annotation of provided error (if it's not nil).
	Tannotate(error, string)
	// Tannotatef terminates the execution with an annotation of the provided error (if it's not nil).
	Tannotatef(error, string, ...interface{})
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

// DevNull returns an io.Writer that ignores all output.
func DevNull() io.Writer {
	return &outputWriter{}
}

// StdoutWriter returns an io.Writer that writes to stdout.
func StdoutWriter(o Output) io.Writer {
	return &outputWriter{o.Stdout}
}

// StderrWriter returns an io.Writer that writes to stderr.
func StderrWriter(o Output) io.Writer {
	return &outputWriter{func(s string) { o.Stderr(s) }}
}

type output struct {
	stdoutChan chan string
	stderrChan chan string
	wg         *sync.WaitGroup
	termErr    error
}

func (o *output) Stdout(s string) {
	o.stdoutChan <- s
}

func (o *output) Stdoutf(s string, a ...interface{}) {
	o.stdoutChan <- fmt.Sprintf(s, a...)
}

func (o *output) Stdoutln(a ...interface{}) {
	o.stdoutChan <- fmt.Sprintln(a...)
}

func (o *output) Stderr(s string) error {
	return o.writeStderr(s)
}

func (o *output) Stderrf(s string, a ...interface{}) error {
	return o.writeStderr(fmt.Sprintf(s, a...))
}

func (o *output) Stderrln(a ...interface{}) error {
	return o.writeStderr(fmt.Sprintln(a...))
}

func (o *output) Annotate(err error, s string) error {
	if err == nil {
		return nil
	}
	return o.Stderrf("%s: %v\n", s, err)
}

func (o *output) Annotatef(err error, s string, a ...interface{}) error {
	if err == nil {
		return nil
	}
	return o.Stderrf("%s: %v\n", fmt.Sprintf(s, a...), err)
}

func (o *output) Terminate(err error) {
	if err != nil {
		o.terminate(o.Stderrln(err.Error()))
	}
}

func (o *output) Terminatef(s string, a ...interface{}) {
	o.terminate(o.Stderrf(s, a...))
}

func (o *output) Tannotate(err error, s string) {
	if err != nil {
		o.Terminate(fmt.Errorf("%s: %v", s, err))
	}
}

func (o *output) Tannotatef(err error, s string, a ...interface{}) {
	if err != nil {
		o.Terminate(fmt.Errorf("%s: %v", fmt.Sprintf(s, a...), err))
	}
}

// terminator is a custom type that is passed to panic
// when running `o.Terminate`
type terminator struct {
	terminationError error
}

func (o *output) terminate(err error) {
	panic(&terminator{err})
}

func (o *output) writeStderr(s string) error {
	o.stderrChan <- s
	return errors.New(strings.TrimSpace(s))
}

func (o *output) Err(err error) error {
	if err != nil {
		o.Stderrf("%s\n", err.Error())
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
	// The built-in go `log` package automatically appends a newline
	// to all inputs. To avoid this, I created a separate glog
	// package which is identical aside from that rule.
	stdout := glog.New(os.Stdout, "", 0)
	stderr := glog.New(os.Stderr, "", 0)
	so := func(s string) {
		stdout.Print(s)
	}
	se := func(s string) {
		stderr.Print(s)
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

// NewIgnoreErrOutput is an output that ignores errors that satisfy any
// of the provided functions.
func NewIgnoreErrOutput(o Output, fs ...func(error) bool) Output {
	return &ignoreErrOutput{o, fs}
}

// NewIgnoreAllOutput is an output that ignores all output.
func NewIgnoreAllOutput() Output {
	return osFromChan(func(s string) {}, func(s string) {})
}

// so it can be a field name in Output wrapper implementors
type fo Output

type ignoreErrOutput struct {
	fo
	fs []func(error) bool
}

func (ieo *ignoreErrOutput) Err(err error) error {
	// Don't output the error if it matches a filter.
	for _, f := range ieo.fs {
		if f(err) {
			return err
		}
	}
	// Regular output functionality if no filter matched.
	return ieo.fo.Err(err)
}

// FakeOutput is a fake `Output` object that can be used for testing.
type FakeOutput struct {
	fo
	stdout []string
	stderr []string
	closed bool
}

// NewFakeOutput returns a new `FakeOutput` object.
func NewFakeOutput() *FakeOutput {
	tcos := &FakeOutput{}
	so := func(s string) {
		tcos.stdout = append(tcos.stdout, s)
	}
	se := func(s string) {
		tcos.stderr = append(tcos.stderr, s)
	}
	cos := osFromChan(so, se)
	tcos.fo = cos
	return tcos
}

// Close closes the fake output channel.
func (fo *FakeOutput) Close() {
	if !fo.closed {
		fo.fo.Close()
		fo.closed = true
	}
}

// GetStdout returns all of the data that was written to the stdout channel.
func (fo *FakeOutput) GetStdout() string {
	fo.Close()
	return strings.Join(fo.stdout, "")
}

// GetStdoutByCalls returns all of the individual calls made to stdout.
func (fo *FakeOutput) GetStdoutByCalls() []string {
	fo.Close()
	return fo.stdout
}

// GetStderr returns all of the data that was written to the stderr channel.
func (fo *FakeOutput) GetStderr() string {
	fo.Close()
	return strings.Join(fo.stderr, "")
}

// GetStderrByCalls returns all of the individual calls made to stdout.
func (fo *FakeOutput) GetStderrByCalls() []string {
	fo.Close()
	return fo.stderr
}
