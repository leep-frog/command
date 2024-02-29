package commander

import "github.com/leep-frog/command/command"

// ArgumentOption is an interface for modifying `Argument` objects.
type ArgumentOption[T any] interface {
	modifyArgumentOption(*argumentOption[T])
}

type argumentOption[T any] struct {
	validators   []*ValidatorOption[T]
	completer    Completer[T]
	transformers []*Transformer[T]
	shortcut     *shortcutOpt[T]
	customSet    *CustomSetter[T]
	_default     *defaultArgumentOption[T]
	breakers     []*ListBreaker[T]
	complexecute *Complexecute[T]
	hideUsage    bool
}

func (ao *argumentOption[T]) inputValidators() []command.InputBreaker {
	var ibs []command.InputBreaker
	for _, v := range ao.breakers {
		ibs = append(ibs, v)
	}
	return ibs
}

type simpleArgumentOption[T any] func(*argumentOption[T])

func (sao *simpleArgumentOption[T]) modifyArgumentOption(ao *argumentOption[T]) {
	(*sao)(ao)
}

func newArgumentOption[T any](f func(*argumentOption[T])) ArgumentOption[T] {
	sao := simpleArgumentOption[T](f)
	return &sao
}

func multiArgumentOptions[T any](opts ...ArgumentOption[T]) *argumentOption[T] {
	ao := &argumentOption[T]{}
	for _, opt := range opts {
		opt.modifyArgumentOption(ao)
	}
	return ao
}

// ShortcutOpt is an `ArgumentOption` that checks for shortcut substitution.
func ShortcutOpt[T any](name string, ac ShortcutCLI) ArgumentOption[T] {
	return &shortcutOpt[T]{
		ShortcutName: name,
		ShortcutCLI:  ac,
	}
}

type shortcutOpt[T any] struct {
	ShortcutName string
	ShortcutCLI  ShortcutCLI
}

func (so *shortcutOpt[T]) modifyArgumentOption(argO *argumentOption[T]) {
	argO.shortcut = so
}

// CustomSetter is an `ArgumentOption` to specify a custom setting function when setting
// argument data.
type CustomSetter[T any] struct {
	F func(T, *command.Data)
}

func (cs *CustomSetter[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.customSet = cs
}

// Complexecute (Complete for Execute) is an arg option for arg execution.
// If a command execution is run, then the last value for this arg
// will be completed using its `Complete` logic. Exactly one suggestion
// must be returned.
//
// The type parameter is needed because it implements `ArgumentOption[T]`.
type Complexecute[T any] struct {
	// Lenient indicates whether a no-match should result in error or not.
	// Default behavior (false) means that an error will be thrown if the completion
	// argument doesn't exactly match one of the completion values and if the number
	// of completion suggestions isn't exactly one.
	Lenient bool
}

func (c *Complexecute[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.complexecute = c
}

// Transformer is an `ArgumentOption` that transforms an argument.
type Transformer[T any] struct {
	F func(T, *command.Data) (T, error)
}

func (t *Transformer[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.transformers = append(ao.transformers, t)
}

// TransformerList changes a single-arg transformer (`Transformer[T]`) to a list-arg transformer (`Transformer[[]T]`).
func TransformerList[T any](t *Transformer[T]) *Transformer[[]T] {
	return &Transformer[[]T]{F: func(vs []T, data *command.Data) ([]T, error) {
		l := make([]T, 0, len(vs))
		for i, v := range vs {
			nv, err := t.F(v, data)
			if err != nil {
				return append(l, vs[i:]...), err
			}
			l = append(l, nv)
		}
		return l, nil
	}}
}

// Default is an `ArgumentOption` that sets a default value for an `Arg` node.
// Note, this package explicitly does not support a `DefaultFunc` `ArgumentOption`. Instead,
// use the `Argument.GetOrDefaultFunc` method inside of your `Node`'s executor logic.
func Default[T any](v T) ArgumentOption[T] {
	return &defaultArgumentOption[T]{v}
}

type defaultArgumentOption[T any] struct {
	v T
}

func (dao *defaultArgumentOption[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao._default = dao
}

// HiddenArg is an `ArgumentOption` that hides an argument from a command's usage text.
func HiddenArg[T any]() ArgumentOption[T] {
	return &hiddenArg[T]{}
}

type hiddenArg[T any] struct{}

func (ha *hiddenArg[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.hideUsage = true
}
