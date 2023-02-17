package command

import "fmt"

// Description creates a `Processor` that adds a command description to the usage text.
func Description(desc string) Processor {
	return &descNode{desc}
}

// Descriptionf is like `Description`, but with formatting options.
func Descriptionf(s string, a ...interface{}) Processor {
	return &descNode{fmt.Sprintf(s, a...)}
}

type descNode struct {
	desc string
}

func (dn *descNode) Usage(i *Input, d *Data, u *Usage) error {
	u.Description = dn.desc
	return nil
}

func (dn *descNode) Execute(*Input, Output, *Data, *ExecuteData) error {
	return nil
}

func (dn *descNode) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}
