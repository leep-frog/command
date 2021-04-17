package command

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// Output defines methods for writing output.
type Output interface {
	// Writes a line to stdout.
	Stdout(string, ...interface{})
	// Writes a line to stderr.
	Stderr(string, ...interface{})
	// Close informs the os that no more data will be written.
	Close()
}

type output struct {
	stdoutChan chan string
	stderrChan chan string
	wg         *sync.WaitGroup
}

func (o *output) Stdout(s string, a ...interface{}) {
	o.stdoutChan <- fmt.Sprintf(s, a...)
}

func (o *output) Stderr(s string, a ...interface{}) {
	o.stderrChan <- fmt.Sprintf(s, a...)
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
		stdout.Println(stdout)
	}
	se := func(s string) {
		stderr.Println(stderr)
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

func (fo *FakeOutput) Stderr(s string, a ...interface{}) {
	fo.c.Stderr(s, a...)
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
