package command

import (
	"strconv"
)

type argNode struct {
	name string
	//opt       *ArgOpt
	minN      int
	optionalN int
	transform func([]string) (*Value, error)
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
	// TODO
	/*if lan.opt != nil && lan.opt.Transformer != nil {
		if !v.IsType(lan.opt.Transformer.ValueType()) {
			err := fmt.Errorf("Transformer of type %v cannot be applied to a value with type %v", lan.opt.Transformer.ValueType(), v.Type())
			ws.Cos.Stderr(err.Error())
			return nil, err
		}

		newV, err := lan.opt.Transformer.Transform(v)
		if err != nil {
			ws.Cos.Stderr("Custom transformer failed: %v", err)
			return nil, err
		}
		v = newV
	}*/

	v.provided = len(sl) > 0

	// Copy values into returned list (required for aliasing)
	newSl := v.ToArgs()
	for i := 0; i < len(sl); i++ {
		sl[i] = newSl[i]
	}

	data.Set(an.name, v)

	if !enough {
		return o.Stderr("not enough arguments")
	}
	return nil
}

func (an *argNode) Complete(input *Input, output Output, data *Data, cData *CompleteData) error {
	return nil
}

func StringListNode(name string, minN, optionalN int) Processor {
	t := func(s []string) (*Value, error) { return StringListValue(s...), nil }
	return listNode(name, minN, optionalN, StringListType, t)
}

func IntListNode(name string, minN, optionalN int) Processor {
	return listNode(name, minN, optionalN, IntListType, intListTransform)
}

func intListTransform(sl []string) (*Value, error) {
	var err error
	var is []int
	for _, s := range sl {
		i, e := strconv.Atoi(s)
		if e != nil {
			// TODO: add failed to load field to values.
			// These can be used in autocomplete if necessary.
			err = e
		}
		is = append(is, i)
	}
	return IntListValue(is...), err
}

func FloatListNode(name string, minN, optionalN int) Processor {
	return listNode(name, minN, optionalN, FloatListType, floatListTransform)
}

func floatListTransform(sl []string) (*Value, error) {
	var err error
	var fs []float64
	for _, s := range sl {
		f, e := strconv.ParseFloat(s, 64)
		if e != nil {
			err = e
		}
		fs = append(fs, f)
	}
	return FloatListValue(fs...), err
}

func StringNode(name string) Processor {
	return listNode(name, 1, 0, StringType, stringTransform)
}

func stringTransform(sl []string) (*Value, error) {
	if len(sl) == 0 {
		return StringValue(""), nil
	}
	return StringValue(sl[0]), nil
}

func IntNode(name string) Processor {
	return listNode(name, 1, 0, IntType, intTransform)
}

func intTransform(sl []string) (*Value, error) {
	if len(sl) == 0 {
		return IntValue(0), nil
	}
	i, err := strconv.Atoi(sl[0])
	return IntValue(i), err
}

func FloatNode(name string) Processor {
	return listNode(name, 1, 0, FloatType, floatTransform)
}

func floatTransform(sl []string) (*Value, error) {
	if len(sl) == 0 {
		return FloatValue(0), nil
	}
	f, err := strconv.ParseFloat(sl[0], 64)
	return FloatValue(f), err
}

func listNode(name string, minN, optionalN int, vt ValueType, transformer func([]string) (*Value, error)) Processor { //opt *ArgOpt) //NodeProcessor {
	return &argNode{
		name:      name,
		minN:      minN,
		optionalN: optionalN,
		//opt:       opt,
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
