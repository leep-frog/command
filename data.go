package command

type Data struct {
	Values map[string]*Value
}

func (d *Data) Set(s string, v *Value) {
	if d.Values == nil {
		d.Values = map[string]*Value{}
	}
	d.Values[s] = v
}

// Str returns a string representation of the arg's value
// regardless if the arg is actually a string or not.
func (d *Data) Str(s string) string {
	return d.Values[s].Str()
}

func (d *Data) HasArg(s string) bool {
	_, ok := d.Values[s]
	return ok
}

func (d *Data) String(s string) string {
	return d.Values[s].String()
}

func (d *Data) StringList(s string) []string {
	return d.Values[s].StringList()
}

func (d *Data) Int(s string) int {
	return d.Values[s].Int()
}

func (d *Data) IntList(s string) []int {
	return d.Values[s].IntList()
}

func (d *Data) Float(s string) float64 {
	return d.Values[s].Float()
}

func (d *Data) FloatList(s string) []float64 {
	return d.Values[s].FloatList()
}

func (d *Data) Bool(s string) bool {
	return d.Values[s].Bool()
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
