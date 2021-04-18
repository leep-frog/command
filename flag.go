package command

/*
// Flag defines a flag argument that is parsed regardless of it's position in
// the provided command line arguments.
type Flag interface {
	// Name is the name of the flag. "--name" is the flags indicator
	Name() string
	// ShortName indicates the shorthand version of the flag. "-s" is the short hand flag indicator.
	ShortName() rune
	// Processor returns a node processor that processes arguments after the flag indicator.
	Processor() Processor
}

// NewFlagNode returns a node that iterates over the remaining command line
// arguments and processes any flags that are present.
func NewFlagNode(fs ...Flag) Processor {
	m := map[string]Flag{}
	for _, f := range fs {
		m[fmt.Sprintf("--%s", f.Name())] = f
		m[fmt.Sprintf("-%c", f.ShortName())] = f
	}
	return &flagNode{
		flagMap: m,
	}
}

type flagNode struct {
	// TODO: keep track of duplicate flags.
	flagMap map[string]Flag
}

func (fn *flagNode) Complete(input *Input, output Output) bool {
	for i := 0; i < len(ws.RawArgs); {
		a, _ := ws.Peek()
		f, ok := fn.flagMap[a]
		if !ok {
			//flaglessArgs = append(flaglessArgs, a)
			i++
			continue
		}

		// Remove flag argument
		ws.Process(SimpleProcessor)
		beforeArgs := ws.RawArgs[:i]
		ws.RawArgs = ws.RawArgs[i:]
		if !f.Processor().Complete(ws) {
			//ws.RawArgs = append(flaglessArgs, ws.RawArgs...)
			ws.RawArgs = append(beforeArgs, ws.RawArgs...)
			return false
		}
		ws.RawArgs = append(beforeArgs, ws.RawArgs...)
	}

	// Complete flag arg if last arg looks like beginning of a flag.
	if len(ws.RawArgs) > 0 && len(ws.RawArgs[len(ws.RawArgs)-1]) > 0 && ws.RawArgs[len(ws.RawArgs)-1][:1] == "-" {
		k := make([]string, 0, len(fn.flagMap))
		for n := range fn.flagMap {
			k = append(k, n)
		}
		sort.Strings(k)
		ws.CompleteResponse = &Completion{
			Suggestions: k,
		}
		return false
	}
	return true
}

func (fn *flagNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	for i := 0; i < len(input.remaining); {
		a := input.args[input.remaining[i]]
		f, ok := fn.flagMap[a]
		if !ok {
			i++
			continue
		}

		// Remove flag argument.
		ws.ProcessAt(i, SimpleProcessor)
		fmt.Println("proAt", i, ws.RawArgs)
		beforeArgs := ws.RawArgs[:i]
		ws.RawArgs = ws.RawArgs[i:]
		if err := f.Processor().Execute(ws); err != nil {
			fmt.Println("err", i, ws.RawArgs)
			//ws.RawArgs = append(flaglessArgs, ws.RawArgs...)
			ws.RawArgs = append(beforeArgs, ws.RawArgs...)
			return err
		}
		ws.RawArgs = append(beforeArgs, ws.RawArgs...)
	}
	return nil
}

type flag struct {
	name      string
	shortName rune
	argNode   Processor
}

func (f *flag) Processor() Processor {
	return f.argNode
}

func (f *flag) Name() string {
	return f.name
}

func (f *flag) ShortName() rune {
	return f.shortName
}

func StringFlag(name string, shortName rune, opt *ArgOpt) Flag {
	return listFlag(name, shortName, 1, 0, stringTransform, opt)
}

func IntFlag(name string, shortName rune, opt *ArgOpt) Flag {
	return listFlag(name, shortName, 1, 0, intTransform, opt)
}

func FloatFlag(name string, shortName rune, opt *ArgOpt) Flag {
	return listFlag(name, shortName, 1, 0, floatTransform, opt)
}

func BoolFlag(name string, shortName rune, opts ...ArgValidator) Flag {
	return &flag{
		name:      name,
		shortName: shortName,
		argNode: &boolFlag{
			name: name,
		},
	}
}

type boolFlag struct {
	name string
}

func (bf *boolFlag) Complete(*Input, *Data) *CompleteData {
	return nil
}

func (bf *boolFlag) Execute(_ *Input, _ Output, data *Data, _ *ExecuteData) error {
	data.Set(bf.name, BoolValue(true))
	return nil
}

func StringListFlag(name string, shortName rune, minN, optionalN int, opt *ArgOpt) Flag {
	return listFlag(name, shortName, minN, optionalN, stringListTransform, opt)
}

func IntListFlag(name string, shortName rune, minN, optionalN int, opt *ArgOpt) Flag {
	return listFlag(name, shortName, minN, optionalN, intListTransform, opt)
}

func FloatListFlag(name string, shortName rune, minN, optionalN int, opt *ArgOpt) Flag {
	return listFlag(name, shortName, minN, optionalN, floatListTransform, opt)
}

func listFlag(name string, shortName rune, minN, optionalN int, transform func(s []*string) (*Value, error), opt *ArgOpt) Flag {
	return &flag{
		name:      name,
		shortName: shortName,
		argNode: &argNode{
			flag:      true,
			name:      name,
			minN:      minN,
			optionalN: optionalN,
			opt:       opt,
			transform: transform,
		},
	}
}
*/
