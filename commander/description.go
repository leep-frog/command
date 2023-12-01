package commander

import (
	"fmt"

	"github.com/leep-frog/command/command"
)

// Description creates a `command.Processor` that adds a command description to the usage text.
func Description(desc string) command.Processor {
	return &descNode{desc}
}

// Descriptionf is like `Description`, but with formatting options.
func Descriptionf(s string, a ...interface{}) command.Processor {
	return &descNode{fmt.Sprintf(s, a...)}
}

type descNode struct {
	desc string
}

func (dn *descNode) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	u.SetDescription(dn.desc)
	return nil
}

func (dn *descNode) Execute(*command.Input, command.Output, *command.Data, *command.ExecuteData) error {
	return nil
}

func (dn *descNode) Complete(*command.Input, *command.Data) (*command.Completion, error) {
	return nil, nil
}
