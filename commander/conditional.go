package commander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommander"
)

// IfElse runs `command.Processor` t if the function argunment returns true
// in the relevant complete and execute contexts. Otherwise, `command.Processor` f
// is run.
func IfElse(t, f command.Processor, fn func(i *command.Input, d *command.Data) bool) command.Processor {
	return SimpleProcessor(
		func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			if fn(i, d) {
				return spycommander.ProcessOrExecute(t, i, o, d, ed)
			}
			if f == nil {
				return nil
			}
			return spycommander.ProcessOrExecute(f, i, o, d, ed)
		},
		func(i *command.Input, d *command.Data) (*command.Completion, error) {
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
func If(p command.Processor, fn func(i *command.Input, d *command.Data) bool) command.Processor {
	return IfElse(p, nil, fn)
}

// IfElseData runs `command.Processor` t if the argument name is present in command.Data.
// If the argument's type is a boolean, then it also must not be false.
// Otherwise, `command.Processor` f is run.
func IfElseData(dataArg string, t, f command.Processor) command.Processor {
	return IfElse(t, f, func(i *command.Input, d *command.Data) bool {
		// If the arg is not in data, return false.
		if !d.Has(dataArg) {
			return false
		}

		// Return true if the value is not a boolean. If it is a boolean, return its value.
		b, ok := (d.Get(dataArg)).(bool)
		return !ok || b
	})
}

// IfData runs `command.Processor` p if the argument name is present in command.Data.
// If the argument's type is a boolean, then it also must not be false.
func IfData(dataArg string, p command.Processor) command.Processor {
	return IfElseData(dataArg, p, nil)
}
