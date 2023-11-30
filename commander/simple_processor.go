package commander

import "github.com/leep-frog/command/command"

// SimpleProcessor creates a `command.Processor` from execution and completion functions.
func SimpleProcessor(e func(*command.Input, command.Output, *command.Data, *command.ExecuteData) error, c func(*command.Input, *command.Data) (*command.Completion, error)) command.Processor {
	return &simpleProcessor{
		e: e,
		c: c,
	}
}

// SuperSimpleProcessor returns a processor from a single function that is run in both
// the execution and completion contexts.
func SuperSimpleProcessor(f func(*command.Input, *command.Data) error) command.Processor {
	return &simpleProcessor{
		e: func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			return o.Err(f(i, d))
		},
		c: func(i *command.Input, d *command.Data) (*command.Completion, error) {
			return nil, f(i, d)
		},
		u: func(i *command.Input, d *command.Data, u *command.Usage) error {
			return f(i, d)
		},
	}
}

type simpleProcessor struct {
	e func(*command.Input, command.Output, *command.Data, *command.ExecuteData) error
	c func(*command.Input, *command.Data) (*command.Completion, error)
	u func(*command.Input, *command.Data, *command.Usage) error
}

func (sp *simpleProcessor) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	if sp.u != nil {
		return sp.u(i, d, u)
	}
	return nil
}

func (sp *simpleProcessor) Execute(i *command.Input, o command.Output, d *command.Data, e *command.ExecuteData) error {
	if sp.e == nil {
		return nil
	}
	return sp.e(i, o, d, e)
}

func (sp *simpleProcessor) Complete(i *command.Input, d *command.Data) (*command.Completion, error) {
	if sp.c == nil {
		return nil, nil
	}
	return sp.c(i, d)
}
