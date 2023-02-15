package command

// SimpleProcessor creates a `Processor` from execution and completion functions.
func SimpleProcessor(e func(*Input, Output, *Data, *ExecuteData) error, c func(*Input, *Data) (*Completion, error)) Processor {
	return &simpleProcessor{
		e: e,
		c: c,
	}
}

// SuperSimpleProcessor returns a processor from a single function that is run in both
// the execution and completion contexts.
func SuperSimpleProcessor(f func(*Input, *Data) error) Processor {
	return &simpleProcessor{
		e: func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			return o.Err(f(i, d))
		},
		c: func(i *Input, d *Data) (*Completion, error) {
			return nil, f(i, d)
		},
	}
}

type simpleProcessor struct {
	e    func(*Input, Output, *Data, *ExecuteData) error
	c    func(*Input, *Data) (*Completion, error)
	desc string
}

func (sp *simpleProcessor) Usage(u *Usage) {
	if sp.desc != "" {
		u.Description = sp.desc
	}
}

func (sp *simpleProcessor) Execute(i *Input, o Output, d *Data, e *ExecuteData) error {
	if sp.e == nil {
		return nil
	}
	return sp.e(i, o, d, e)
}

func (sp *simpleProcessor) Complete(i *Input, d *Data) (*Completion, error) {
	if sp.c == nil {
		return nil, nil
	}
	return sp.c(i, d)
}
