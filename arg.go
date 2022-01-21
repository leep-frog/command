package command

import "fmt"

type ArgNode[T any] struct {
	name      string
	desc      string
	opt       *argOpt[T]
	minN      int
	optionalN int
	shortName rune
	flag      bool
}

func (an *ArgNode[T]) AddOptions(opts ...ArgOpt[T]) *ArgNode[T] {
	for _, o := range opts {
		o.modifyArgOpt(an.opt)
	}
	return an
}

func (an *ArgNode[T]) Name() string {
	return an.name
}

func (an *ArgNode[T]) Desc() string {
	return an.desc
}

func (an *ArgNode[T]) Set(v T, data *Data) {
	if an.opt != nil && an.opt.customSet != nil {
		an.opt.customSet(v, data)
	} else {
		data.Set(an.name, v)
	}
}

func (an *ArgNode[T]) Usage(u *Usage) {
	if an.opt != nil && an.opt.hiddenUsage {
		return
	}

	if an.desc != "" {
		u.UsageSection.Add(ArgSection, an.name, an.desc)
	}

	for i := 0; i < an.minN; i++ {
		u.Usage = append(u.Usage, an.name)
	}
	if an.optionalN == UnboundedList {
		u.Usage = append(u.Usage, fmt.Sprintf("[ %s ... ]", an.name))
	} else {
		if an.optionalN > 0 {
			u.Usage = append(u.Usage, "[")
			for i := 0; i < an.optionalN; i++ {
				u.Usage = append(u.Usage, an.name)
			}
			u.Usage = append(u.Usage, "]")
		}
	}

	if an.opt.breaker != nil {
		an.opt.breaker.Usage(u)
	}
}

func (an *ArgNode[T]) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
	an.aliasCheck(i, false)

	sl, enough := i.PopN(an.minN, an.optionalN, an.opt.breaker)

	// Don't set at all if no arguments provided for arg.
	if len(sl) == 0 {
		if !enough {
			return o.Err(an.notEnoughErr(len(sl)))
		}
		if an.opt != nil && an.opt._default != nil {
			an.Set(*an.opt._default, data)
		}
		return nil
	}

	// Transform from string to value.
	op := getOperator[T]()
	v, err := op.fromArgs(sl)
	if err != nil {
		return o.Err(err)
	}

	// Run custom transformer.
	if an.opt != nil && an.opt.transformer != nil {
		newV, err := an.opt.transformer.t(v)
		if err != nil {
			return o.Stderrf("Custom transformer failed: %v", err)
		}
		v = newV
	}

	// Copy values into returned list (required for aliasing)
	newSl := op.toArgs(v)
	for i := 0; i < len(sl); i++ {
		*sl[i] = newSl[i]
	}

	an.Set(v, data)

	if an.opt != nil {
		for _, validator := range an.opt.validators {
			if err := validator.Validate(v); err != nil {
				return o.Stderrf("validation failed: %v", err)
			}
		}
	}

	if !enough {
		return o.Err(an.notEnoughErr(len(sl)))
	}
	return nil
}

func (an *ArgNode[T]) notEnoughErr(got int) error {
	return NotEnoughArgs(an.name, an.minN, got)
}

func IsNotEnoughArgsError(err error) bool {
	_, ok := err.(*notEnoughArgs)
	return ok
}

func IsUsageError(err error) bool {
	return IsNotEnoughArgsError(err) || IsExtraArgsError(err) || IsBranchingError(err)
}

func NotEnoughArgs(name string, req, got int) error {
	return &notEnoughArgs{name, req, got}
}

type notEnoughArgs struct {
	name string
	req  int
	got  int
}

func (ne *notEnoughArgs) Error() string {
	plural := "s"
	if ne.req == 1 {
		plural = ""
	}
	return fmt.Sprintf("Argument %q requires at least %d argument%s, got %d", ne.name, ne.req, plural, ne.got)
}

func (an *ArgNode[T]) aliasCheck(input *Input, complete bool) {
	if an.opt != nil && an.opt.alias != nil {
		if an.optionalN == UnboundedList {
			input.CheckAliases(len(input.remaining), an.opt.alias.AliasCLI, an.opt.alias.AliasName, complete)
		} else {
			input.CheckAliases(an.minN+an.optionalN, an.opt.alias.AliasCLI, an.opt.alias.AliasName, complete)
		}
	}
}

func (an *ArgNode[T]) Complete(input *Input, data *Data) (*Completion, error) {
	an.aliasCheck(input, true)

	sl, enough := input.PopN(an.minN, an.optionalN, an.opt.breaker)

	// If this is the last arg, we want the node walkthrough to stop (which
	// doesn't happen if c and err are nil).
	c, err := an.complete(sl, enough, input, data)
	if (!enough || input.FullyProcessed()) && c == nil {
		c = &Completion{}
	}
	return c, err
}

func (an *ArgNode[T]) complete(sl []*string, enough bool, input *Input, data *Data) (*Completion, error) {
	// Try to transform from string to value.
	op := getOperator[T]()
	v, err := op.fromArgs(sl)
	if err != nil {
		// If we're on the last one, then complete it.
		if !enough || input.FullyProcessed() {
			var lastArg string
			if len(sl) > 0 {
				lastArg = *sl[len(sl)-1]
			}
			return an.opt.completor.Complete(lastArg, v, data)
		}

		return nil, err
	}

	// Run custom transformer on a best effor basis (i.e. if the transformer fails,
	// then we just continue with the original value).
	if an.opt != nil && an.opt.transformer != nil && an.opt.transformer.forComplete {
		// Don't return an error because this may not be the last one.
		newV, err := an.opt.transformer.t(v)
		if err == nil {
			v = newV
		}
	}

	an.Set(v, data)

	// Don't run validations when completing.

	// If we have enough and more needs to be processed.
	if enough && !input.FullyProcessed() {
		return nil, nil
	}

	if an.opt == nil || an.opt.completor == nil {
		return nil, nil
	}

	var lastArg string
	ta := op.toArgs(v)
	if len(ta) > 0 {
		lastArg = ta[len(ta)-1]
	}
	return an.opt.completor.Complete(lastArg, v, data)
}

func Arg[T any](name, desc string, opts ...ArgOpt[T]) *ArgNode[T] {
	return listNode[T](name, desc, 1, 0, opts...)
}

func OptionalArg[T any](name, desc string, opts ...ArgOpt[T]) *ArgNode[T] {
	return listNode[T](name, desc, 0, 1, opts...)
}

func ListArg[T any](name, desc string, minN, optionalN int, opts ...ArgOpt[[]T]) *ArgNode[[]T] {
	return listNode[[]T](name, desc, minN, optionalN, opts...)
}

func BoolNode(name, desc string) *ArgNode[bool] {
	return listNode[bool](name, desc, 1, 0, BoolCompletor())
}

func listNode[T any](name, desc string, minN, optionalN int, opts ...ArgOpt[T]) *ArgNode[T] {
	return &ArgNode[T]{
		name:      name,
		desc:      desc,
		minN:      minN,
		optionalN: optionalN,
		opt:       newArgOpt(opts...),
		//transform: transformer,
	}
}
