package commander

import (
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/operator"
)

// ListUntilSymbol returns an unbounded list node that ends when a specific symbol is parsed.
func ListUntilSymbol[T comparable](symbol T) *ListBreaker[[]T] {
	return &ListBreaker[[]T]{
		Validators: []*ValidatorOption[[]T]{
			ListifyValidatorOption(NEQ[T](symbol)),
		},
		UsageFunc: func(d *command.Data, u *command.Usage) error {
			arg := operator.GetOperator[T]().ToArgs(symbol)[0] // slices aren't comparable, so this will only ever be length 1
			u.Usage = append(u.Usage, arg)
			u.UsageSection.Add(command.SymbolSection, arg, "List breaker")
			return nil
		},
	}
}

// ListUntil returns a `ListBreaker` node that breaks when any of the provided `ValidatorOptions` are not satisfied.
func ListUntil[T any](validators ...*ValidatorOption[T]) *ListBreaker[[]T] {
	var listValidators []*ValidatorOption[[]T]
	for _, v := range validators {
		listValidators = append(listValidators, ListifyValidatorOption(v))
	}
	return &ListBreaker[[]T]{
		Validators: listValidators,
	}
}

// ListBreaker is a type that implements `command.InputBreaker` as well as `ArgumentOtion[T]`.
type ListBreaker[T any] struct {
	// Validators is the list of validators
	Validators []*ValidatorOption[T]
	// Discard is whether the culprit character should be removed
	Discard bool
	// UsageFunc modifies the usage doc
	UsageFunc func(*command.Data, *command.Usage) error
}

func (lb *ListBreaker[T]) DiscardBreak(s string, d *command.Data) bool {
	return lb.Discard
}

func (lb *ListBreaker[T]) Break(s string, d *command.Data) bool {
	for _, v := range lb.Validators {
		op := operator.GetOperator[T]()
		args, err := operator.FromArgs(op, s)
		if err != nil {
			continue
		}
		if err := v.Validate(args, d); err != nil {
			return true
		}
	}
	return false
}

func (lb *ListBreaker[T]) Validate(t T, d *command.Data) error {
	for _, v := range lb.Validators {
		if err := v.Validate(t, d); err != nil {
			return err
		}
	}
	return nil
}

func (lb *ListBreaker[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.breakers = append(ao.breakers, lb)
}

// command.Usage updates the provided `command.Usage` object.
func (lb *ListBreaker[T]) Usage(d *command.Data, u *command.Usage) error {
	if lb.UsageFunc != nil {
		return lb.UsageFunc(d, u)
	}
	return nil
}
