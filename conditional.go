package command

// IfElse runs `Processor` t if the function argunment returns true
// in the relevant complete and execute contexts. Otherwise, `Processor` f
// is run.
func IfElse(t, f Processor, fn func(i *Input, d *Data) bool) Processor {
	return SimpleProcessor(
		func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			if fn(i, d) {
				return processOrExecute(t, i, o, d, ed)
			}
			if f == nil {
				return nil
			}
			return processOrExecute(f, i, o, d, ed)
		},
		func(i *Input, d *Data) (*Completion, error) {
			if fn(i, d) {
				return processOrComplete(t, i, d)
			}
			if f == nil {
				return nil, nil
			}
			return processOrComplete(f, i, d)
		},
	)
}

// If runs the provided processor if the function argunment returns true
// in the relevant complete and execute contexts.
func If(p Processor, fn func(i *Input, d *Data) bool) Processor {
	return IfElse(p, nil, fn)
}

// IfElseData runs `Processor` t if the argument name is present in Data.
// If the argument's type is a boolean, then it also must not be false.
// Otherwise, `Processor` f is run.
func IfElseData(dataArg string, t, f Processor) Processor {
	return IfElse(t, f, func(i *Input, d *Data) bool {
		// If the arg is not in data, return false.
		if !d.Has(dataArg) {
			return false
		}

		// Return true if the value is not a boolean. If it is a boolean, return its value.
		b, ok := (d.Get(dataArg)).(bool)
		return !ok || b
	})
}

// IfData runs `Processor` p if the argument name is present in Data.
// If the argument's type is a boolean, then it also must not be false.
func IfData(dataArg string, p Processor) Processor {
	return IfElseData(dataArg, p, nil)
}
