package command

// ArgOpt is a type for modifying `Arg` nodes.
type ArgOpt[T any] interface {
	modifyArgOpt(*argOpt[T])
}

type argOpt[T any] struct {
	validators         []*ValidatorOption[T]
	completer          Completer[T]
	transformers       []*Transformer[T]
	shortcut           *shortcutOpt[T]
	customSet          customSetter[T]
	_default           *defaultArgOpt[T]
	breaker            *ListBreaker
	completeForExecute *completeForExecute

	hiddenUsage bool
}

type simpleArgOpt[T any] func(*argOpt[T])

func (sao *simpleArgOpt[T]) modifyArgOpt(ao *argOpt[T]) {
	(*sao)(ao)
}

func newArgOpt[T any](f func(*argOpt[T])) ArgOpt[T] {
	sao := simpleArgOpt[T](f)
	return &sao
}

func multiArgOpts[T any](opts ...ArgOpt[T]) *argOpt[T] {
	ao := &argOpt[T]{}
	for _, opt := range opts {
		opt.modifyArgOpt(ao)
	}
	return ao
}

// ShortcutOpt is an `ArgOpt` that checks for shortcut substitution.
func ShortcutOpt[T any](name string, ac ShortcutCLI) ArgOpt[T] {
	return &shortcutOpt[T]{
		ShortcutName: name,
		ShortcutCLI:  ac,
	}
}

type shortcutOpt[T any] struct {
	ShortcutName string
	ShortcutCLI  ShortcutCLI
}

func (so *shortcutOpt[T]) modifyArgOpt(argO *argOpt[T]) {
	argO.shortcut = so
}

// CustomSetter is an `ArgOpt` to specify a custom setting function when setting
// argument data.
func CustomSetter[T any](f func(T, *Data)) ArgOpt[T] {
	cs := customSetter[T](f)
	return &cs
}

type customSetter[T any] func(T, *Data)

func (cs *customSetter[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.customSet = *cs
}

// CompleteForExecute is an arg option for arg execution.
// If a command execution is run, then the last value for this arg
// will be completed using its `Complete` logic. Exactly one suggestion
// must be returned.
func CompleteForExecute[T any](opts ...CompleteForExecuteOption) ArgOpt[T] {
	cfe := &completeForExecute{
		enabled: true,
		strict:  true,
	}
	for _, o := range opts {
		o(cfe)
	}
	return newArgOpt(func(ao *argOpt[T]) {
		ao.completeForExecute = cfe
	})
}

type CompleteForExecuteOption func(*completeForExecute)

type completeForExecute struct {
	// Whether or not to actually complete it
	enabled    bool
	strict     bool
	exactMatch bool
}

// CompleteForExecuteBestEffort runs CompleteForExecute on a best effort basis.
// If zero or multiple completions are suggested, then the argument isn't altered.
func CompleteForExecuteBestEffort() CompleteForExecuteOption {
	return func(cfe *completeForExecute) { cfe.strict = false }
}

// CompleteForExecuteAllowExactMatch allows exact matches even if multiple
// completions were returned. For example, if the arg is "Hello", and the resulting
// completions are ["Hello", "HelloThere", "Hello!"], then we won't error.
func CompleteForExecuteAllowExactMatch() CompleteForExecuteOption {
	return func(cfe *completeForExecute) { cfe.exactMatch = true }
}

// Transformer is an `ArgOpt` that transforms an argument.
type Transformer[T any] struct {
	F func(T, *Data) (T, error)
}

func (t *Transformer[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.transformers = append(ao.transformers, t)
}

// TransformerList changes a single-arg transformer (`Transformer[T]`) to a list-arg transformer (`Transformer[[]T]`).
func TransformerList[T any](t *Transformer[T]) *Transformer[[]T] {
	return &Transformer[[]T]{F: func(vs []T, data *Data) ([]T, error) {
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

// Default is an `ArgOpt` that sets a default value for an `Arg` node.
func Default[T any](v T) ArgOpt[T] {
	return DefaultFunc(func(d *Data) (T, error) { return v, nil })
}

// DefaultFunc is an `ArgOpt` that sets a default value (obtained from the provided function) for an `Arg` node.
func DefaultFunc[T any](f defaultFunc[T]) ArgOpt[T] {
	return &defaultArgOpt[T]{f}
}

type defaultFunc[T any] func(d *Data) (T, error)

type defaultArgOpt[T any] struct {
	f defaultFunc[T]
}

func (dao *defaultArgOpt[T]) modifyArgOpt(ao *argOpt[T]) {
	ao._default = dao
}

// HiddenArg is an `ArgOpt` that hides an argument from a command's usage text.
func HiddenArg[T any]() ArgOpt[T] {
	return &hiddenArg[T]{}
}

type hiddenArg[T any] struct{}

func (ha *hiddenArg[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.hiddenUsage = true
}
