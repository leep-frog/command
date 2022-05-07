package command

// ArgOpt is a type for modifying `Arg` nodes.
type ArgOpt[T any] interface {
	modifyArgOpt(*argOpt[T])
}

type argOpt[T any] struct {
	validators   []*ValidatorOption[T]
	completor    Completor[T]
	transformers []*Transformer[T]
	shortcut     *shortcutOpt[T]
	customSet    customSetter[T]
	_default     *T
	breaker      *ListBreaker

	hiddenUsage bool
}

func newArgOpt[T any](opts ...ArgOpt[T]) *argOpt[T] {
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

// Transformer is an `ArgOpt` that transforms an argument.
// TODO: make from and to different types?
type Transformer[T any] struct {
	t func(T, *Data) (T, error)
	// forComplete is whether or not the value
	// should be transformed during completions.
	forComplete bool
}

func (t *Transformer[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.transformers = append(ao.transformers, t)
}

// TransformerList changes a single-arg transformer (`Transformer[T]`) to a list-arg transformer (`Transformer[[]T]`).
func TransformerList[T any](t *Transformer[T]) *Transformer[[]T] {
	return NewTransformer(func(vs []T, data *Data) ([]T, error) {
		l := make([]T, 0, len(vs))
		for i, v := range vs {
			nv, err := t.t(v, data)
			if err != nil {
				return append(l, vs[i:]...), err
			}
			l = append(l, nv)
		}
		return l, nil
	}, t.forComplete)
}

// NewTransformer creates a new `Transformer`.
func NewTransformer[T any](f func(T, *Data) (T, error), forComplete bool) *Transformer[T] {
	return &Transformer[T]{
		t:           f,
		forComplete: forComplete,
	}
}

// ValidatorOption is an `ArgOpt` for validating arguments.
type ValidatorOption[T any] struct {
	validate func(T) error
}

// ValidatorList changes a single-arg validator (`Validator[T]`) to a list-arg validator (`Validator[[]T]`).
func ValidatorList[T any](vo *ValidatorOption[T]) *ValidatorOption[[]T] {
	return &ValidatorOption[[]T]{
		validate: func(ts []T) error {
			for _, t := range ts {
				if err := vo.validate(t); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func (vo *ValidatorOption[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.validators = append(ao.validators, vo)
}

func (vo *ValidatorOption[T]) modifyBashNode(bn *BashCommand[T]) {
	bn.validators = append(bn.validators, vo)
}

// Validate validates the argument and returns an error if the validation fails.
func (vo *ValidatorOption[T]) Validate(v T) error {
	return vo.validate(v)
}

// Default is an `ArgOpt` that sets a default value for an `Arg` node.
func Default[T any](v T) ArgOpt[T] {
	return &defaultArgOpt[T]{v}
}

type defaultArgOpt[T any] struct {
	v T
}

func (dao *defaultArgOpt[T]) modifyArgOpt(ao *argOpt[T]) {
	ao._default = &dao.v
}

// HiddenArg is an `ArgOpt` that hides an argument from a command's usage text.
func HiddenArg[T any]() ArgOpt[T] {
	return &hiddenArg[T]{}
}

type hiddenArg[T any] struct{}

func (ha *hiddenArg[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.hiddenUsage = true
}
