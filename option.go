package command

type ArgOpt[T any] interface {
	modifyArgOpt(*argOpt[T])
}

type argOpt[T any] struct {
	validators   []*ValidatorOption[T]
	completor    *Completor[T]
	transformers []*Transformer[T]
	alias        *aliasOpt[T]
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

func AliasOpt[T any](name string, ac AliasCLI) ArgOpt[T] {
	return &aliasOpt[T]{
		AliasName: name,
		AliasCLI:  ac,
	}
}

type aliasOpt[T any] struct {
	AliasName string
	AliasCLI  AliasCLI
}

func (ao *aliasOpt[T]) modifyArgOpt(argO *argOpt[T]) {
	argO.alias = ao
}

func CustomSetter[T any](f func(T, *Data)) ArgOpt[T] {
	cs := customSetter[T](f)
	return &cs
}

type customSetter[T any] func(T, *Data)

func (cs *customSetter[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.customSet = *cs
}

type Transformer[T any] struct {
	t func(T) (T, error)
	// forComplete is whether or not the value
	// should be transformed during completions.
	forComplete bool
}

func (t *Transformer[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.transformers = append(ao.transformers, t)
}

//func (t *Transformer[T]) ForList() *Transformer[[]T] {
func TransformerList[T any](t *Transformer[T]) *Transformer[[]T] {
	return NewTransformer(func(vs []T) ([]T, error) {
		l := make([]T, 0, len(vs))
		for i, v := range vs {
			nv, err := t.t(v)
			if err != nil {
				return append(l, vs[i:]...), err
			}
			l = append(l, nv)
		}
		return l, nil
	}, t.forComplete)
}

func NewTransformer[T any](f func(T) (T, error), forComplete bool) *Transformer[T] {
	return &Transformer[T]{
		t:           f,
		forComplete: forComplete,
	}
}

type ValidatorOption[T any] struct {
	validate func(T) error
}

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

func (vo *ValidatorOption[T]) modifyBashNode(bn *bashCommand[T]) {
	bn.validators = append(bn.validators, vo)
}

func (vo *ValidatorOption[T]) Validate(v T) error {
	return vo.validate(v)
}

// Default arg option
type defaultArgOpt[T any] struct {
	v T
}

func (dao *defaultArgOpt[T]) modifyArgOpt(ao *argOpt[T]) {
	ao._default = &dao.v
}

func Default[T any](v T) ArgOpt[T] {
	return &defaultArgOpt[T]{v}
}
