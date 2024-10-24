// Package color makes it easy to output formatted text (via tput or output codes depending on the operating system).
// tput is preferred when possible as that doesn't modify the text passed to downstream commands
// See the tput documentation for more info:
// https://linuxcommand.org/lc3_adv_tput.php
// This package is specifically implemented to work well with the base `command`
// package, both in terms of usage and testing.
package color

import (
	"fmt"
	"strings"

	"slices"
)

// OutputCode gets the output codes for the provided `Formats`.
// Note: the callers of this package are responsible for applying the
// `Reset` format.
func OutputCode(fs ...Format) string {
	var r []string
	for _, f := range fs {
		for _, c := range f.outputCodes() {
			r = append(r, fmt.Sprintf("%d", c))
		}
	}
	if len(r) == 0 {
		return ""
	}

	return fmt.Sprintf("\033[%sm", strings.Join(r, ";"))
}

// Apply applies the provided format to the string and then resets the format.
func Apply(s string, fs ...Format) string {
	return fmt.Sprintf("%s%s%s", OutputCode(fs...), s, OutputCode(Reset))
}

// Format is a format (bold, color, etc.) that can be applied to output.
// This package defines a handful of these for the typical use cases.
type Format interface {
	// tputArgs are the args passed to a `tput` command.
	// this was made private as `outputCodes` is the preferred (and only formally supported)
	// mechanism for now.
	// If we did want to use tputArgs, this would be the way to do it:
	// ```
	// cmd := exec.Command("tput", "bold")
	// cmd.Stdout = os.Stdout // or output.StdoutWriter if in `command` package
	// cmd.Stderr = os.Stderr
	// if err := cmd.Run(); err != nil { ... }
	// fmt.Println("After tput")
	// ```
	tputArgs() [][]string
	// OutputCodes is the list of codes to use in the output string to activate the desired format.
	outputCodes() []int
}

func MultiFormat(fs ...Format) Format {
	mf := multiFormat(slices.Clone(fs))
	return &mf
}

// multiFormat is simply a collection of multiple `Format` objects.
type multiFormat []Format

func (mf *multiFormat) tputArgs() [][]string {
	var r [][]string
	for _, f := range *mf {
		r = append(r, f.tputArgs()...)
	}
	return r
}

func (mf *multiFormat) outputCodes() []int {
	var r []int
	for _, f := range *mf {
		r = append(r, f.outputCodes()...)
	}
	return r
}

// ColorCode is the color code for tput and for the output format `\033[0;(30+X)m`.
type ColorCode struct {
	code       int
	foreground bool
}

var (
	// Can't use iota outside of `const` definition :(

	Black   = &ColorCode{0, true}
	Red     = &ColorCode{1, true}
	Green   = &ColorCode{2, true}
	Yellow  = &ColorCode{3, true}
	Blue    = &ColorCode{4, true}
	Magenta = &ColorCode{5, true}
	Cyan    = &ColorCode{6, true}
	White   = &ColorCode{7, true}
	unused  = &ColorCode{8, true}

	Reset        = newEffect(0, "init")
	Bold         = newEffect(1, "bold")
	Underline    = newEffect(4, "smul")
	EndUnderline = newEffect(24, "rmul")

	foregroundOutputColorCodeOffset = 30
	backgroundOutputColorCodeOffset = 40
)

func (c *ColorCode) Background() *ColorCode {
	return &ColorCode{c.code, false}
}

func (c *ColorCode) tputArgs() [][]string {
	if c.foreground {
		return [][]string{{"setaf", fmt.Sprintf("%d", c.code)}}
	}
	return [][]string{{"setab", fmt.Sprintf("%d", c.code)}}
}

func (c *ColorCode) outputCodes() []int {
	if c.foreground {
		return []int{c.code + foregroundOutputColorCodeOffset}
	}
	return []int{c.code + backgroundOutputColorCodeOffset}
}

// Effect is an effect that defines separate code and tput args.
type Effect struct {
	code          int
	tputArguments []string
}

func newEffect(code int, tputArgs ...string) *Effect {
	return &Effect{code, tputArgs}
}

func (e *Effect) tputArgs() [][]string {
	return [][]string{slices.Clone(e.tputArguments)}
}

func (e *Effect) outputCodes() []int {
	return []int{e.code}
}
