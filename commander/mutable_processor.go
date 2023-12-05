package commander

import "github.com/leep-frog/command/command"

type MutableProcessor[P command.Processor] struct {
	Processor *P
}

func NewMutableProcessor[P command.Processor](p P) *MutableProcessor[P] {
	return &MutableProcessor[P]{&p}
}

func (rp *MutableProcessor[P]) Execute(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
	return (*rp.Processor).Execute(i, o, d, ed)
}

func (rp *MutableProcessor[P]) Complete(i *command.Input, d *command.Data) (*command.Completion, error) {
	return (*rp.Processor).Complete(i, d)
}

func (rp *MutableProcessor[P]) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	return (*rp.Processor).Usage(i, d, u)
}
