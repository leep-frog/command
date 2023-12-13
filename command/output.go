package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/glog"
	"github.com/leep-frog/command/internal/spycommand"
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
	// Color changes the format of stdout to the provided formats.
	Color(fs ...color.Format)
	// Colerr (color + err hehe) changes the format of stderr to the provided formats.
	Colerr(fs ...color.Format)
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

func (o *output) Color(fs ...color.Format) {
	o.Stdout(color.OutputCode(fs...))
}

func (o *output) Colerr(fs ...color.Format) {
	o.Stderr(color.OutputCode(fs...))
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

func (o *output) terminate(err error) {
	spycommand.Terminate(err)
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
	return OutputFromFuncs(so, se)
}

// OutputFromFuncs returns an Output object that forwards data to the provided stdout
// and stderr functions.
//
// If you need an Output object for testing purposes, consider using commandtest.NewOutput()
// which provides the `GetStdout/GetStderr` and `GetStdoutByCalls/GetStderrByCalls` functions.
func OutputFromFuncs(stdoutFunc, stderrFunc func(string)) Output {
	stdoutChan := make(chan string)
	stderrChan := make(chan string)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		for s := range stdoutChan {
			stdoutFunc(s)
		}
		wg.Done()
	}()

	go func() {
		for s := range stderrChan {
			stderrFunc(s)
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
	return OutputFromFuncs(func(s string) {}, func(s string) {})
}

type ignoreErrOutput struct {
	Output
	fs []func(error) bool
}

func (ieo *ignoreErrOutput) Err(err error) error {
	// Don't output the error (but still return it) if it matches a filter.
	for _, f := range ieo.fs {
		if f(err) {
			return err
		}
	}
	// Regular output functionality if no filter matched.
	return ieo.Output.Err(err)
}
