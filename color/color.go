// Package color makes it easy to output formatted text (via tput). See the tput
// documentation for more info:
// https://linuxcommand.org/lc3_adv_tput.php
// This package is specifically implemented to work well with the base `command`
// package, both in terms of usage and testing.
package color

import (
	"strconv"

	"github.com/leep-frog/command"
)

// Format is a format (bold, color, etc.) that can be applied to output.
type Format []string

// TputColorCode is the tput code for specific colors.
type TputColorCode int

const (
	Black TputColorCode = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	unused
	ResetColor
)

var (
	// TputCommand is a function that applies a format via tput. It is a variable
	// so it can be stubbed out by tests in other packages.
	TputCommand = func(output command.Output, args ...interface{}) error {
		// TODO:
		return nil
		// return sh.Command("tput", args...).Run()
	}
)

func newF(args ...string) *Format {
	f := Format(args)
	return &f
}

// Apply applies the `Format`.
func (f *Format) Apply(output command.Output) {
	var i []interface{}
	for _, j := range *f {
		i = append(i, j)
	}
	TputCommand(output, i...)
}

// MultiFormat combines multiple formats into one format.
func MultiFormat(fs ...*Format) *Format {
	var s []string
	for _, f := range fs {
		s = append(s, []string(*f)...)
	}
	return newF(s...)
}

// Background is a `Format` that applies color to the background.
func Background(color TputColorCode) *Format {
	return newF("setab", strconv.Itoa(int(color)))
}

// Text is a `Format` that applies color to text.
func Text(color TputColorCode) *Format {
	return newF("setaf", strconv.Itoa(int(color)))
}

// Bold is a `Format` that applies bold.
func Bold() *Format {
	return newF("bold")
}

// Underline is a `Format` that applies underline.
func Underline() *Format {
	return newF("smul")
}

// EndUnderline is a `Format` that un-applies underline.
func EndUnderline() *Format {
	return newF("rmul")
}

// Reset is a `Format` that resets all tput formatting and clears the terminal screen.
func Reset() *Format {
	return newF("reset")
}

// Init is a `Format` that resets all tput formatting.
func Init() *Format {
	return newF("init")
}
