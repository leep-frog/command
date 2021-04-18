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

func (an *argNode) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
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

	data.Set(an.name, v)

	if an.opt != nil {
		for _, validator := range an.opt.Validators {
			if err := validator.Validate(v); err != nil {
				return o.Stderr("validation failed: %v", err)
			}
		}
	}

	if !enough {
		return o.Stderr("not enough arguments")
	}
	return nil
}

func (an *argNode) Complete(input *Input, data *Data) *CompleteData {
	sl, enough := input.PopN(an.minN, an.optionalN)

	// Try to transform from string to value.
	v, err := an.transform(sl)
	if err != nil {
		return &CompleteData{
			Error: err,
		}
	}

	// Run custom transformer.
	if an.opt != nil && an.opt.Transformer != nil {
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

	data.Set(an.name, v)

	// Don't run validations when completing.

	// If we have enough and more needs to be processed.
	if enough && !input.FullyProcessed() {
		return nil
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

/*func (lan *listArgNode) Complete(ws *WorldState) bool {
	return ws.ProcessMultiple(lan.minN, lan.optionalN, func(args []string, _ bool) ([]string, error) {
		v, _ := lan.transform(args)
		if lan.opt != nil && lan.opt.Transformer != nil {
			if v.IsType(lan.opt.Transformer.ValueType()) {
				newV, err := lan.opt.Transformer.Transform(v)
				if err == nil {
					v = newV
				}
			}
		}
		lan.set(v, ws)
		if len(ws.RawArgs) == 0 {
			if len(args) > 0 {
				lan.completeArg(ws, args[len(args)-1], v)
			} else {
				lan.completeArg(ws, "", v)
			}
			return v.ToArgs(), fmt.Errorf("terminate")
		}
		return v.ToArgs(), nil
	}) == nil
}

func (lan *listArgNode) completeArg(ws *WorldState, rawValue string, v *Value) {
	if lan.opt != nil && lan.opt.Completor != nil {
		ws.CompleteResponse = lan.opt.Completor.Complete(rawValue, v, ws.Values)
	}
}

func StringListNode(name string, minN, optionalN int, opt *ArgOpt) NodeProcessor {
	t := func(s []string) (*Value, error) { return StringListValue(s...), nil }
	return listNode(name, minN, optionalN, StringListType, t, opt)
}

func IntListNode(name string, minN, optionalN int, opt *ArgOpt) NodeProcessor {
	return listNode(name, minN, optionalN, IntListType, intListTransform, opt)
}

func FloatListNode(name string, minN, optionalN int, opt *ArgOpt) NodeProcessor {
	return listNode(name, minN, optionalN, FloatListType, floatListTransform, opt)
}

func listNode(name string, minN, optionalN int, vt ValueType, transformer func([]string) (*Value, error), opt *ArgOpt) NodeProcessor {
	return &listArgNode{
		name:      name,
		minN:      minN,
		optionalN: optionalN,
		opt:       opt,
		vt:        vt,
		transform: transformer,
	}
}
*/
