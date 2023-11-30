package commander

import "github.com/leep-frog/command/commondels"

// GetProcessor is a simple interface that extends the `commondels.Processor` interface
// and allows users to use a single object both as a commondels.Processor and to retrieve
// data (similar to the `Arg` and `Flag` types).
type GetProcessor[T any] struct {
	commondels.Processor
	Name string
}

// Provided returns whether or not the argument has been set in `commondels.Data`.
func (gp *GetProcessor[T]) Provided(d *commondels.Data) bool {
	return d.Has(gp.Name)
}

// Get returns the value, if it has been set in `commondels.Data`; panics otherwise.
// Use `GetProcessor.Provided(*commondels.Data)` before calling this when it is not guaranteed to have been set.
func (gp *GetProcessor[T]) Get(d *commondels.Data) T {
	return commondels.GetData[T](d, gp.Name)
}
