package command

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
