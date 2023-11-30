package commander

import "github.com/leep-frog/command/command"

// GetProcessor is a simple interface that extends the `command.Processor` interface
// and allows users to use a single object both as a command.Processor and to retrieve
// data (similar to the `Arg` and `Flag` types).
type GetProcessor[T any] struct {
	command.Processor
	Name string
}

// Provided returns whether or not the argument has been set in `command.Data`.
func (gp *GetProcessor[T]) Provided(d *command.Data) bool {
	return d.Has(gp.Name)
}

// Get returns the value, if it has been set in `command.Data`; panics otherwise.
// Use `GetProcessor.Provided(*command.Data)` before calling this when it is not guaranteed to have been set.
func (gp *GetProcessor[T]) Get(d *command.Data) T {
	return command.GetData[T](d, gp.Name)
}
