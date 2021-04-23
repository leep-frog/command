package command

import (
	"strconv"
)

type argNode struct {
	name      string
	opt       *ArgOpt
	minN      int
	optionalN int
	transform func([]*string) (*Value, error)
	vt        ValueType
	shortName rune
	flag      bool
}

func (an *argNode) Set(v *Value, data *Data) {
	if an.opt != nil && an.opt.CustomSet != nil {
		an.opt.CustomSet(v, data)
	} else {
		data.Set(an.name, v)
	}
}

func (an *argNode) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
	an.aliasCheck(i, false)

	// TODO: If not enough for single, don't do validation and transforming.
	sl, enough := i.PopN(an.minN, an.optionalN)

	// Transform from string to value.
	v, err := an.transform(sl)
	if err != nil {
		o.Stderr(err.Error())
		return err
	}

	// Run custom transformer.
	if an.opt != nil && an.opt.Transformer != nil {
		if !v.IsType(an.opt.Transformer.ValueType()) {
			return o.Stderr("Transformer of type %v cannot be applied to a value with type %v", an.opt.Transformer.ValueType(), v.Type())
		}

		newV, err := an.opt.Transformer.Transform(v)
		if err != nil {
			return o.Stderr("Custom transformer failed: %v", err)
		}
		v = newV
	}

	v.provided = len(sl) > 0

	// Copy values into returned list (required for aliasing)
	newSl := v.ToArgs()
	for i := 0; i < len(sl); i++ {
		*sl[i] = newSl[i]
	}

	// TODO: move this after validators
	an.Set(v, data)

	if an.opt != nil {
		for _, validator := range an.opt.Validators {
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

func IsNotEnoughArgsErr(err error) bool {
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
	if an.opt != nil && an.opt.Alias != nil {
		if an.optionalN == UnboundedList {
			input.CheckAliases(len(input.remaining), an.opt.Alias.AliasCLI, an.opt.Alias.AliasName, complete)
		} else {
			input.CheckAliases(an.minN+an.optionalN, an.opt.Alias.AliasCLI, an.opt.Alias.AliasName, complete)
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
				Completion: an.opt.Completor.Complete(lastArg, v, data),
			}
		}

		return &CompleteData{
			Error: err,
		}
	}

	// Run custom transformer on a best effor basis (i.e. if the transformer fails,
	// then we just continue with the original value).
	if an.opt != nil && an.opt.Transformer != nil && an.opt.Transformer.ForComplete() {
		// Don't return an error because this may not be the last one.
		if v.IsType(an.opt.Transformer.ValueType()) {
			newV, err := an.opt.Transformer.Transform(v)
			if err == nil {
				v = newV
			}
		}
	}

	v.provided = len(sl) > 0

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

	if an.opt == nil || an.opt.Completor == nil {
		// We are completing for this arg so we should return.
		return &CompleteData{}
	}

	var lastArg string
	ta := v.ToArgs()
	if len(ta) > 0 {
		lastArg = ta[len(ta)-1]
	}
	return &CompleteData{
		Completion: an.opt.Completor.Complete(lastArg, v, data),
	}
}

func StringListNode(name string, minN, optionalN int, opt *ArgOpt) Processor {
	return listNode(name, minN, optionalN, StringListType, stringListTransform, opt)
}

func stringListTransform(sl []*string) (*Value, error) {
	r := make([]string, 0, len(sl))
	for _, s := range sl {
		r = append(r, *s)
	}
	return StringListValue(r...), nil
}

func IntListNode(name string, minN, optionalN int, opt *ArgOpt) Processor {
	return listNode(name, minN, optionalN, IntListType, intListTransform, opt)
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

func FloatListNode(name string, minN, optionalN int, opt *ArgOpt) Processor {
	return listNode(name, minN, optionalN, FloatListType, floatListTransform, opt)
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

func StringNode(name string, opt *ArgOpt) Processor {
	return listNode(name, 1, 0, StringType, stringTransform, opt)
}

func OptionalStringNode(name string, opt *ArgOpt) Processor {
	return listNode(name, 0, 1, StringType, stringTransform, opt)
}

func stringTransform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return StringValue(""), nil
	}
	return StringValue(*sl[0]), nil
}

func IntNode(name string, opt *ArgOpt) Processor {
	return listNode(name, 1, 0, IntType, intTransform, opt)
}

func OptionalIntNode(name string, opt *ArgOpt) Processor {
	return listNode(name, 0, 1, IntType, intTransform, opt)
}

func intTransform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return IntValue(0), nil
	}
	i, err := strconv.Atoi(*sl[0])
	return IntValue(i), err
}

func FloatNode(name string, opt *ArgOpt) Processor {
	return listNode(name, 1, 0, FloatType, floatTransform, opt)
}

func OptionalFloatNode(name string, opt *ArgOpt) Processor {
	return listNode(name, 0, 1, FloatType, floatTransform, opt)
}

func floatTransform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return FloatValue(0), nil
	}
	f, err := strconv.ParseFloat(*sl[0], 64)
	return FloatValue(f), err
}

func listNode(name string, minN, optionalN int, vt ValueType, transformer func([]*string) (*Value, error), opt *ArgOpt) Processor {
	return &argNode{
		name:      name,
		minN:      minN,
		optionalN: optionalN,
		opt:       opt,
		vt:        vt,
		transform: transformer,
	}
}
