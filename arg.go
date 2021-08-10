package command

import (
	"strconv"
)

type argNode struct {
	name      string
	opt       *argOpt
	minN      int
	optionalN int
	transform func([]*string) (*Value, error)
	vt        ValueType
	shortName rune
	flag      bool
}

func (an *argNode) Set(v *Value, data *Data) {
	if an.opt != nil && an.opt.customSet != nil {
		an.opt.customSet(v, data)
	} else {
		data.Set(an.name, v)
	}
}

func (an *argNode) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
	an.aliasCheck(i, false)

	sl, enough := i.PopN(an.minN, an.optionalN)

	// Don't set at all if no arguments provided for arg.
	if len(sl) == 0 {
		if !enough {
			return o.Err(NotEnoughArgs())
		}
		return nil
	}

	// Transform from string to value.
	v, err := an.transform(sl)
	if err != nil {
		o.Stderr(err.Error())
		return err
	}

	// Run custom transformer.
	if an.opt != nil && an.opt.transformer != nil {
		if !v.IsType(an.opt.transformer.vt) {
			return o.Stderr("Transformer of type %v cannot be applied to a value with type %v", an.opt.transformer.vt, v.Type())
		}

		newV, err := an.opt.transformer.t(v)
		if err != nil {
			return o.Stderr("Custom transformer failed: %v", err)
		}
		v = newV
	}

	// Copy values into returned list (required for aliasing)
	newSl := v.ToArgs()
	for i := 0; i < len(sl); i++ {
		*sl[i] = newSl[i]
	}

	// TODO: move this after validators
	an.Set(v, data)

	if an.opt != nil {
		for _, validator := range an.opt.validators {
			if err := validator.Validate(v); err != nil {
				return o.Stderr("validation failed: %v", err)
			}
		}
	}

	if !enough {
		return o.Err(NotEnoughArgs())
	}
	return nil
}

func IsNotEnoughArgsError(err error) bool {
	_, ok := err.(*notEnoughArgs)
	return ok
}

func NotEnoughArgs() error {
	return &notEnoughArgs{}
}

type notEnoughArgs struct{}

func (ne *notEnoughArgs) Error() string {
	return "not enough arguments"
}

func (an *argNode) aliasCheck(input *Input, complete bool) {
	if an.opt != nil && an.opt.alias != nil {
		if an.optionalN == UnboundedList {
			input.CheckAliases(len(input.remaining), an.opt.alias.AliasCLI, an.opt.alias.AliasName, complete)
		} else {
			input.CheckAliases(an.minN+an.optionalN, an.opt.alias.AliasCLI, an.opt.alias.AliasName, complete)
		}
	}
}

func (an *argNode) Complete(input *Input, data *Data) *CompleteData {
	an.aliasCheck(input, true)

	sl, enough := input.PopN(an.minN, an.optionalN)

	// Try to transform from string to value.
	v, err := an.transform(sl)
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

	// Copy values into returned list (required for aliasing)
	newSl := v.ToArgs()
	for i := 0; i < len(sl); i++ {
		*sl[i] = newSl[i]
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

func StringListNode(name string, minN, optionalN int, opts ...ArgOpt) Processor {
	return listNode(name, minN, optionalN, StringListType, stringListTransform, opts...)
}

func stringListTransform(sl []*string) (*Value, error) {
	r := make([]string, 0, len(sl))
	for _, s := range sl {
		r = append(r, *s)
	}
	return StringListValue(r...), nil
}

func IntListNode(name string, minN, optionalN int, opts ...ArgOpt) Processor {
	return listNode(name, minN, optionalN, IntListType, intListTransform, opts...)
}

func intListTransform(sl []*string) (*Value, error) {
	var err error
	var is []int
	for _, s := range sl {
		i, e := strconv.Atoi(*s)
		if e != nil {
			// TODO: add failed to load field to values.
			// These can be used in autocomplete if necessary.
			err = e
		}
		is = append(is, i)
	}
	return IntListValue(is...), err
}

func FloatListNode(name string, minN, optionalN int, opts ...ArgOpt) Processor {
	return listNode(name, minN, optionalN, FloatListType, floatListTransform, opts...)
}

func floatListTransform(sl []*string) (*Value, error) {
	var err error
	var fs []float64
	for _, s := range sl {
		f, e := strconv.ParseFloat(*s, 64)
		if e != nil {
			err = e
		}
		fs = append(fs, f)
	}
	return FloatListValue(fs...), err
}

func StringNode(name string, opts ...ArgOpt) Processor {
	return listNode(name, 1, 0, StringType, stringTransform, opts...)
}

func OptionalStringNode(name string, opts ...ArgOpt) Processor {
	return listNode(name, 0, 1, StringType, stringTransform, opts...)
}

func stringTransform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return StringValue(""), nil
	}
	return StringValue(*sl[0]), nil
}

func IntNode(name string, opts ...ArgOpt) Processor {
	return listNode(name, 1, 0, IntType, intTransform, opts...)
}

func OptionalIntNode(name string, opts ...ArgOpt) Processor {
	return listNode(name, 0, 1, IntType, intTransform, opts...)
}

func intTransform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return IntValue(0), nil
	}
	i, err := strconv.Atoi(*sl[0])
	return IntValue(i), err
}

func FloatNode(name string, opts ...ArgOpt) Processor {
	return listNode(name, 1, 0, FloatType, floatTransform, opts...)
}

func OptionalFloatNode(name string, opts ...ArgOpt) Processor {
	return listNode(name, 0, 1, FloatType, floatTransform, opts...)
}

func floatTransform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return FloatValue(0), nil
	}
	f, err := strconv.ParseFloat(*sl[0], 64)
	return FloatValue(f), err
}

func BoolNode(name string) Processor {
	return listNode(name, 1, 0, BoolType, boolTransform, BoolCompletor())
}

func boolTransform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return BoolValue(false), nil
	}
	b, err := strconv.ParseBool(*sl[0])
	return BoolValue(b), err
}

func listNode(name string, minN, optionalN int, vt ValueType, transformer func([]*string) (*Value, error), opts ...ArgOpt) Processor {
	return &argNode{
		name:      name,
		minN:      minN,
		optionalN: optionalN,
		opt:       newArgOpt(opts...),
		vt:        vt,
		transform: transformer,
	}
}
