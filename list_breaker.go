package command

// TODO: Remove this (or hide this) in favor of ItemizedListFlag?
// ListBreakerOption is an option type for the `ListBreaker` type.
type ListBreakerOption[T any] func(*ListBreaker[T])

func newBreakerOpt[T any](f func(*ListBreaker[T])) ListBreakerOption[T] {
	return f
}

// DiscardBreaker is a `ListBreakerOption` that removes the breaker argument from the input (rather than keeping it for the next node to parse).
func DiscardBreaker[T any]() ListBreakerOption[T] {
	return newBreakerOpt(func(lb *ListBreaker[T]) {
		lb.discard = true
	})
}

// ListBreakerUsage is a `ListBreakerOption` that inlcudes usage info in the command's usage text.
func ListBreakerUsage[T any](uf func(*Usage)) ListBreakerOption[T] {
	return newBreakerOpt(func(lb *ListBreaker[T]) {
		lb.u = uf
	})
}

// ListUntilSymbol returns an unbounded list node that ends when a specific symbol is parsed.
func ListUntilSymbol[T any](symbol string, opts ...ListBreakerOption[T]) *ListBreaker[T] {
	return ListUntil[T](NEQ(symbol)).AddOptions(append(opts, ListBreakerUsage[T](func(u *Usage) {
		u.Usage = append(u.Usage, symbol)
		u.UsageSection.Add(SymbolSection, symbol, "List breaker")
	}))...)
}

// AddOptions adds `ListBreakerOptions` to a `ListBreaker` object.
func (lb *ListBreaker[T]) AddOptions(opts ...ListBreakerOption[T]) *ListBreaker[T] {
	for _, opt := range opts {
		opt(lb)
	}
	return lb
}

// ListUntil returns a `ListBreaker` node that breaks when any of the provided `ValidatorOptions` are not satisfied.
func ListUntil[T any](validators ...*ValidatorOption[string]) *ListBreaker[T] {
	return &ListBreaker[T]{
		validators: validators,
	}
}

// ListBreaker is an `ArgumentOption` for breaking out of lists with an optional number of arguments.
type ListBreaker[T any] struct {
	validators []*ValidatorOption[string]
	discard    bool
	u          func(*Usage)
}

func (lb *ListBreaker[T]) Validate(s string) error {
	for _, v := range lb.validators {
		if err := v.Validate(s); err != nil {
			return err
		}
	}
	return nil
}

func (lb *ListBreaker[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.breakers = append(ao.breakers, lb)
}

// ConvertListBreaker converts a `ListBreaker` for an arbitrary type
// into a `ListBreaker` for the provided generic type `T`.
func ConvertListBreaker[T any](lb *ListBreaker[any]) *ListBreaker[T] {
	return &ListBreaker[T]{lb.validators, lb.discard, lb.u}
}

// Validators returns the `ListBreaker`'s validators.
func (lb *ListBreaker[T]) Validators() []*ValidatorOption[string] {
	return lb.validators
}

// DiscardBreak indicates whether the `ListBreaker` discards the argument that breaks the list.
func (lb *ListBreaker[T]) DiscardBreak() bool {
	return lb.discard
}

// Usage updates the provided `Usage` object.
func (lb *ListBreaker[T]) Usage(u *Usage) {
	if lb.u != nil {
		lb.u(u)
	}
}
