package commander

import "github.com/leep-frog/command/commondels"

// IfElse runs `commondels.Processor` t if the function argunment returns true
// in the relevant complete and execute contexts. Otherwise, `commondels.Processor` f
// is run.
func IfElse(t, f commondels.Processor, fn func(i *commondels.Input, d *commondels.Data) bool) commondels.Processor {
	return SimpleProcessor(
		func(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
			if fn(i, d) {
				return processOrExecute(t, i, o, d, ed)
			}
			if f == nil {
				return nil
			}
			return processOrExecute(f, i, o, d, ed)
		},
		func(i *commondels.Input, d *commondels.Data) (*commondels.Completion, error) {
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
func If(p commondels.Processor, fn func(i *commondels.Input, d *commondels.Data) bool) commondels.Processor {
	return IfElse(p, nil, fn)
}

// IfElseData runs `commondels.Processor` t if the argument name is present in commondels.Data.
// If the argument's type is a boolean, then it also must not be false.
// Otherwise, `commondels.Processor` f is run.
func IfElseData(dataArg string, t, f commondels.Processor) commondels.Processor {
	return IfElse(t, f, func(i *commondels.Input, d *commondels.Data) bool {
		// If the arg is not in data, return false.
		if !d.Has(dataArg) {
			return false
		}

		// Return true if the value is not a boolean. If it is a boolean, return its value.
		b, ok := (d.Get(dataArg)).(bool)
		return !ok || b
	})
}

// IfData runs `commondels.Processor` p if the argument name is present in commondels.Data.
// If the argument's type is a boolean, then it also must not be false.
func IfData(dataArg string, p commondels.Processor) commondels.Processor {
	return IfElseData(dataArg, p, nil)
}
