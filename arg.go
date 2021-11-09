package command

import "fmt"

type ArgNode struct {
	name      string
	desc      string
	opt       *argOpt
	minN      int
	optionalN int
	//transform func([]*string) (*Value, error)
	vt        ValueType
	shortName rune
	flag      bool
}

func (an *ArgNode) AddOptions(opts ...ArgOpt) *ArgNode {
	for _, o := range opts {
		o.modifyArgOpt(an.opt)
	}
	return an
}

func (an *ArgNode) Name() string {
	return an.name
}

func (an *ArgNode) Desc() string {
	return an.desc
}

func (an *ArgNode) Set(v *Value, data *Data) {
	if an.opt != nil && an.opt.customSet != nil {
		an.opt.customSet(v, data)
	} else {
		data.Set(an.name, v)
	}
}

func (an *ArgNode) Usage(u *Usage) {
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
}

func (an *ArgNode) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
	an.aliasCheck(i, false)

	sl, enough := i.PopN(an.minN, an.optionalN)

	// Don't set at all if no arguments provided for arg.
	if len(sl) == 0 {
		if !enough {
			return o.Err(an.notEnoughErr(len(sl)))
		}
		return nil
	}

	// Transform from string to value.
	v, err := vtMap.transform(an.vt, sl)
	if err != nil {
		o.Stderr(err.Error())
		return err
	}

	// Run custom transformer.
	if an.opt != nil && an.opt.transformer != nil {
		if !v.IsType(an.opt.transformer.vt) {
			return o.Stderrf("Transformer of type %v cannot be applied to a value with type %v", an.opt.transformer.vt, v.Type())
		}

		newV, err := an.opt.transformer.t(v)
		if err != nil {
			return o.Stderrf("Custom transformer failed: %v", err)
		}
		v = newV
	}

	// Copy values into returned list (required for aliasing)
	newSl := v.ToArgs()
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

func (an *ArgNode) notEnoughErr(got int) error {
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

func (an *ArgNode) aliasCheck(input *Input, complete bool) {
	if an.opt != nil && an.opt.alias != nil {
		if an.optionalN == UnboundedList {
			input.CheckAliases(len(input.remaining), an.opt.alias.AliasCLI, an.opt.alias.AliasName, complete)
		} else {
			input.CheckAliases(an.minN+an.optionalN, an.opt.alias.AliasCLI, an.opt.alias.AliasName, complete)
		}
	}
}

func (an *ArgNode) Complete(input *Input, data *Data) *CompleteData {
	an.aliasCheck(input, true)

	sl, enough := input.PopN(an.minN, an.optionalN)

	// Try to transform from string to value.
	v, err := vtMap.transform(an.vt, sl)
	if err != nil {
		// If we're on the last one, then complete it.
		if !enough || input.FullyProcessed() {
			var lastArg string
			if len(sl) > 0 {
				lastArg = *sl[len(sl)-1]
			}
			return &CompleteData{
				Completion: an.opt.completor.Complete(lastArg, v, data),
			}
		}

		return &CompleteData{
			Error: err,
		}
	}

	// Run custom transformer on a best effor basis (i.e. if the transformer fails,
	// then we just continue with the original value).
	if an.opt != nil && an.opt.transformer != nil && an.opt.transformer.forComplete {
		// Don't return an error because this may not be the last one.
		if v.IsType(an.opt.transformer.vt) {
			newV, err := an.opt.transformer.t(v)
			if err == nil {
				v = newV
			}
		}
	}

	an.Set(v, data)

	// Don't run validations when completing.

	// If we have enough and more needs to be processed.
	if enough && !input.FullyProcessed() {
		return nil
	}

	if an.opt == nil || an.opt.completor == nil {
		// We are completing for this arg so we should return.
		return &CompleteData{}
	}

	var lastArg string
	ta := v.ToArgs()
	if len(ta) > 0 {
		lastArg = ta[len(ta)-1]
	}
	return &CompleteData{
		Completion: an.opt.completor.Complete(lastArg, v, data),
	}
}

func StringListNode(name, desc string, minN, optionalN int, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, minN, optionalN, StringListType, opts...)
}

func IntListNode(name, desc string, minN, optionalN int, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, minN, optionalN, IntListType, opts...)
}

func FloatListNode(name, desc string, minN, optionalN int, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, minN, optionalN, FloatListType, opts...)
}

func StringNode(name, desc string, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, 1, 0, StringType, opts...)
}

func OptionalStringNode(name, desc string, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, 0, 1, StringType, opts...)
}

func IntNode(name, desc string, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, 1, 0, IntType, opts...)
}

func OptionalIntNode(name, desc string, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, 0, 1, IntType, opts...)
}

func FloatNode(name, desc string, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, 1, 0, FloatType, opts...)
}

func OptionalFloatNode(name, desc string, opts ...ArgOpt) *ArgNode {
	return listNode(name, desc, 0, 1, FloatType, opts...)
}

func BoolNode(name, desc string) *ArgNode {
	return listNode(name, desc, 1, 0, BoolType, BoolCompletor())
}

func listNode(name, desc string, minN, optionalN int, vt ValueType, opts ...ArgOpt) *ArgNode {
	return &ArgNode{
		name:      name,
		desc:      desc,
		minN:      minN,
		optionalN: optionalN,
		opt:       newArgOpt(opts...),
		vt:        vt,
		//transform: transformer,
	}
}
