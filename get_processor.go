package command

// GetProcessor is a simple interface that extends the `Processor` interface
// and allows users to use a single object both as a Processor and to retrieve
// data (similar to the `Arg` and `Flag` types).
type GetProcessor[T any] struct {
	Processor
	Name string
}

// Provided returns whether or not the argument has been set in `Data`.
func (gp *GetProcessor[T]) Provided(d *Data) bool {
	return d.Has(gp.Name)
}

// Get returns the value, if it has been set in `Data`; panics otherwise.
// Use `GetProcessor.Provided(*Data)` before calling this when it is not guaranteed to have been set.
func (gp *GetProcessor[T]) Get(d *Data) T {
	return GetData[T](d, gp.Name)
}
