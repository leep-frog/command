package commander

import "github.com/leep-frog/command/commondels"

// SimpleProcessor creates a `commondels.Processor` from execution and completion functions.
func SimpleProcessor(e func(*commondels.Input, commondels.Output, *commondels.Data, *commondels.ExecuteData) error, c func(*commondels.Input, *commondels.Data) (*commondels.Completion, error)) commondels.Processor {
	return &simpleProcessor{
		e: e,
		c: c,
	}
}

// SuperSimpleProcessor returns a processor from a single function that is run in both
// the execution and completion contexts.
func SuperSimpleProcessor(f func(*commondels.Input, *commondels.Data) error) commondels.Processor {
	return &simpleProcessor{
		e: func(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
			return o.Err(f(i, d))
		},
		c: func(i *commondels.Input, d *commondels.Data) (*commondels.Completion, error) {
			return nil, f(i, d)
		},
	}
}

type simpleProcessor struct {
	e    func(*commondels.Input, commondels.Output, *commondels.Data, *commondels.ExecuteData) error
	c    func(*commondels.Input, *commondels.Data) (*commondels.Completion, error)
	desc string
}

func (sp *simpleProcessor) Usage(i *commondels.Input, d *commondels.Data, u *commondels.Usage) error {
	if sp.desc != "" {
		u.Description = sp.desc
	}
	return nil
}

func (sp *simpleProcessor) Execute(i *commondels.Input, o commondels.Output, d *commondels.Data, e *commondels.ExecuteData) error {
	if sp.e == nil {
		return nil
	}
	return sp.e(i, o, d, e)
}

func (sp *simpleProcessor) Complete(i *commondels.Input, d *commondels.Data) (*commondels.Completion, error) {
	if sp.c == nil {
		return nil, nil
	}
	return sp.c(i, d)
}
