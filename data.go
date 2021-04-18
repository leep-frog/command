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

type ExecuteData struct {
	// Executable is a list of commands to run after execution in the commands package.
	Executable [][]string
	Executor   func(Output, *Data) error
}

type CompleteData struct {
	// Since printing out data during a completion command causes issues,
	// any error encountered will be stored here.
	Completion *Completion
	Error      error
}
