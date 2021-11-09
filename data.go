package command

import "regexp"

type Data map[string]*Value

func (d *Data) Set(s string, v *Value) {
	(*d)[s] = v
}

func (d *Data) Keys() []string {
	var keys []string
	for k := range *d {
		keys = append(keys, k)
	}
	return keys
}

func (d *Data) get(s string) *Value {
	return (*d)[s]
}

func (d *Data) HasArg(s string) bool {
	_, ok := (*d)[s]
	return ok
}

// Str returns a string representation of the arg's value
// regardless if the arg is actually a string or not.
func (d *Data) Str(s string) string {
	return d.get(s).Str()
}

func (d *Data) String(s string) string {
	return d.get(s).String()
}

func (d *Data) StringList(s string) []string {
	return d.get(s).StringList()
}

// Regexp returns a regexp.Regexp object that is created from the corresponding string node.
// This function should only be used with string nodes that use the IsRegex valiator.
func (d *Data) Regexp(s string) *regexp.Regexp {
	return regexp.MustCompile(d.String(s))
}

// RegexpList returns a slice of regexp.Regexp objects that is created from the corresponding string list node.
// This function should only be used with string list nodes that use the ListIsRegex valiator.
func (d *Data) RegexpList(s string) []*regexp.Regexp {
	var rs []*regexp.Regexp
	for _, s := range d.StringList(s) {
		rs = append(rs, regexp.MustCompile(s))
	}
	return rs
}

func (d *Data) Int(s string) int {
	return d.get(s).Int()
}

func (d *Data) IntList(s string) []int {
	return d.get(s).IntList()
}

func (d *Data) Float(s string) float64 {
	return d.get(s).Float()
}

func (d *Data) FloatList(s string) []float64 {
	return d.get(s).FloatList()
}

func (d *Data) Bool(s string) bool {
	return d.get(s).Bool()
}

type ExecuteData struct {
	// Executable is a list of commands to run after execution in the commands package.
	Executable []string
	// TODO: make this a list of functions.
	Executor func(Output, *Data) error
}

type CompleteData struct {
	// Since printing out data during a completion command causes issues,
	// any error encountered will be stored here.
	Completion *Completion
	Error      error
}
