// Package color makes it easy to wrap strings with bash coloring.
package color

import (
	"fmt"
	"sort"
	"strings"
)

type Format struct {
	Color     Color
	Thickness Thickness
}

type Color string
type Thickness bool

var (
	Bold Thickness = true
	Shy  Thickness = false
)

func (t Thickness) ToString() string {
	if t {
		return "bold"
	}
	return "shy"
}

var (
	reset          = "\033[0m"
	bold           = "\033[1m"
	Default  Color = "default"
	Red      Color = "red"
	Green    Color = "green"
	Yellow   Color = "yellow"
	Blue     Color = "blue"
	Purple   Color = "purple"
	Cyan     Color = "cyan"
	Gray     Color = "gray"
	White    Color = "white"
	colorMap       = map[Color]string{
		Default: "",
		Red:     "\033[31m",
		Green:   "\033[32m",
		Yellow:  "\033[33m",
		Blue:    "\033[34m",
		Purple:  "\033[35m",
		Cyan:    "\033[36m",
		Gray:    "\033[37m",
		White:   "\033[97m",
	}
)

func (f *Format) AddAttribute(s string) error {
	s = strings.ToLower(s)
	if s == "bold" {
		f.Thickness = Bold
		return nil
	}
	if s == "shy" {
		f.Thickness = Shy
		return nil
	}
	c := Color(s)
	if !c.Valid() {
		return fmt.Errorf("invalid attribute: %s", s)
	}
	f.Color = c
	return nil
}

func (f *Format) RemoveAttribute(s string) error {
	s = strings.ToLower(s)
	if s == "bold" {
		f.Thickness = Shy
	}
	if s == "shy" {
		f.Thickness = Bold
		return nil
	}
	c := Color(s)
	if c != f.Color {
		return fmt.Errorf("format has color %q, not %q", f.Color, c)
	}
	return nil
}

func (f *Format) Attributes() []string {
	r := make([]string, 0, 2)
	if f.Thickness {
		r = append(r, "bold")
	}
	if f.Color != "" {
		r = append(r, string(f.Color))
	}
	return r
}

func Attributes() []string {
	r := make([]string, 0, len(colorMap)+1)
	r = append(r, "bold")
	r = append(r, "shy")
	for c := range colorMap {
		r = append(r, string(c))
	}
	sort.Strings(r)
	return r
}

func (f *Format) Format(s string) string {
	if f == nil {
		return s
	}
	return f.Color.Format(f.Thickness.Format(s))
}

func (c Color) Format(s string) string {
	code := Color(strings.ToLower(string(c)))
	if colorPrefix, ok := colorMap[code]; ok {
		return colorPrefix + s + reset
	}
	return s
}

func (t Thickness) Format(s string) string {
	if t {
		return bold + s + reset
	}
	return s
}

func (c Color) Valid() bool {
	_, ok := colorMap[c]
	return ok
}
