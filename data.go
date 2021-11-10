package command

import "regexp"

type Data struct {
	Values     map[string]*Value
	Interfaces map[string]interface{}
}

func (d *Data) Set(s string, v *Value) {
	if d.Values == nil {
		d.Values = map[string]*Value{}
	}
	d.Values[s] = v
}

func (d *Data) SetI(s string, i interface{}) {
	if d.Interfaces == nil {
		d.Interfaces = map[string]interface{}{}
	}
	d.Interfaces[s] = i
}

func (d *Data) Keys() []string {
	var keys []string
	for k := range d.Values {
		keys = append(keys, k)
	}
	return keys
}

func (d *Data) KeysI() []string {
	var keys []string
	for k := range d.Interfaces {
		keys = append(keys, k)
	}
	return keys
}

func (d *Data) Get(s string) *Value {
	return d.Values[s]
}

func (d *Data) GetI(s string) interface{} {
	return d.Interfaces[s]
}

func (d *Data) HasArg(s string) bool {
	_, ok := d.Values[s]
	return ok
}

func (d *Data) HasArgI(s string) bool {
	_, ok := d.Interfaces[s]
	return ok
}

func (d *Data) String(s string) string {
	return d.Get(s).ToString()
}

func (d *Data) StringList(s string) []string {
	return d.Get(s).ToStringList()
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
	return d.Get(s).ToInt()
}

func (d *Data) IntList(s string) []int {
	return d.Get(s).ToIntList()
}

func (d *Data) Float(s string) float64 {
	return d.Get(s).ToFloat()
}

func (d *Data) FloatList(s string) []float64 {
	return d.Get(s).ToFloatList()
}

func (d *Data) Bool(s string) bool {
	return d.Get(s).ToBool()
}

type ExecuteData struct {
	// Executable is a list of commands to run after execution in the commands package.
	Executable []string
	Executor   []func(Output, *Data) error
}
