package command

// GetProcessor is a simple interface that extends the `Processor` interface
// and allows users to use a single object both as a Processor and to retrieve
// data (similar to the `Arg` and `Flag` types).
type GetProcessor[T any] struct {
	Processor
	get func(*Data) T
}

func (gp *GetProcessor[T]) Get(d *Data) T {
	return gp.get(d)
}
