// Package color makes it easy to output formatted text (via tput). See the tput
// documentation for more info:
// https://linuxcommand.org/lc3_adv_tput.php
package color

import (
	"strconv"

	"github.com/codeskyblue/go-sh"
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
	Reset
)

var (
	// TputCommand is a function that applies a format via tput. It is a variable
	// so it can be stubbed out by tests in other packages.
	TputCommand = func(name string, args ...interface{}) error {
		return sh.Command(name, args...).Run()
	}
)

func newF(args ...string) *Format {
	f := Format(args)
	return &f
}

// Apply applies the `Format`.
func (f *Format) Apply() {
	var i []interface{}
	for _, j := range *f {
		i = append(i, j)
	}
	TputCommand("tput", i...)
}

// BackgroundColor is a `Format` that applies color to the background.
func BackgroundColor(color TputColorCode) *Format {
	return newF("setab", strconv.Itoa(int(color)))
}

// Color is a `Format` that applies color to text.
func Color(color TputColorCode) *Format {
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

// Underline is a `Format` that un-applies underline.
func EndUnderline() *Format {
	return newF("rmul")
}
