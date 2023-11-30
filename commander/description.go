package commander

import (
	"fmt"

	"github.com/leep-frog/command/commondels"
)

// Description creates a `commondels.Processor` that adds a command description to the usage text.
func Description(desc string) commondels.Processor {
	return &descNode{desc}
}

// Descriptionf is like `Description`, but with formatting options.
func Descriptionf(s string, a ...interface{}) commondels.Processor {
	return &descNode{fmt.Sprintf(s, a...)}
}

type descNode struct {
	desc string
}

func (dn *descNode) Usage(i *commondels.Input, d *commondels.Data, u *commondels.Usage) error {
	u.Description = dn.desc
	return nil
}

func (dn *descNode) Execute(*commondels.Input, commondels.Output, *commondels.Data, *commondels.ExecuteData) error {
	return nil
}

func (dn *descNode) Complete(*commondels.Input, *commondels.Data) (*commondels.Completion, error) {
	return nil, nil
}
