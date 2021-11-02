package command

import (
	"fmt"
	"sort"
)

// Flag defines a flag argument that is parsed regardless of it's position in
// the provided command line arguments.
type Flag interface {
	// Name is the name of the flag. "--name" is the flags indicator
	Name() string
	// Desc is the description of the flag.
	Desc() string
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

func (fn *flagNode) Complete(input *Input, data *Data) *CompleteData {
	for i := 0; i < len(input.remaining); {
		a, _ := input.PeekAt(i)
		f, ok := fn.flagMap[a]
		if !ok {
			i++
			continue
		}

		input.offset = i
		// Remove flag argument (e.g. --flagName).
		input.Pop()
		if cd := f.Processor().Complete(input, data); cd != nil {
			input.offset = 0
			return cd
		}
		input.offset = 0
	}

	if lastArg, ok := input.PeekAt(len(input.remaining) - 1); ok && len(lastArg) > 0 && lastArg[0] == '-' {
		k := make([]string, 0, len(fn.flagMap))
		for n := range fn.flagMap {
			k = append(k, n)
		}
		sort.Strings(k)
		return &CompleteData{
			Completion: &Completion{
				Suggestions: k,
			},
		}
	}
	return nil
}

func (fn *flagNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	for i := 0; i < len(input.remaining); {
		a, _ := input.PeekAt(i)
		f, ok := fn.flagMap[a]
		if !ok {
			i++
			continue
		}

		input.offset = i
		// Remove flag argument (e.g. --flagName).
		input.Pop()
		if err := f.Processor().Execute(input, output, data, eData); err != nil {
			input.offset = 0
			return err
		}
		input.offset = 0
	}
	return nil
}

func (fn *flagNode) Usage(u *Usage) {
	var flags []Flag
	for k, f := range fn.flagMap {
		// flagMap contains entries for name and short name, so ensure we only do each one once.
		if k == fmt.Sprintf("--%s", f.Name()) {
			flags = append(flags, f)
		}
	}

	sort.SliceStable(flags, func(i, j int) bool { return flags[i].Name() < flags[j].Name() })

	for _, f := range flags {
		if f.Desc() != "" {
			u.UsageSection.Add(FlagSection, f.Name(), f.Desc())
		}

		u.Flags = append(u.Flags, fmt.Sprintf("--%s|-%c", f.Name(), f.ShortName()))
	}
}

type flag struct {
	name      string
	desc      string
	shortName rune
	argNode   *ArgNode
}

func (f *flag) Desc() string {
	return f.desc
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

func StringFlag(name string, shortName rune, desc string, opts ...ArgOpt) Flag {
	return listFlag(name, desc, shortName, 1, 0, StringType, opts...)
}

func IntFlag(name string, shortName rune, desc string, opts ...ArgOpt) Flag {
	return listFlag(name, desc, shortName, 1, 0, IntType, opts...)
}

func FloatFlag(name string, shortName rune, desc string, opts ...ArgOpt) Flag {
	return listFlag(name, desc, shortName, 1, 0, FloatType, opts...)
}

func BoolFlag(name string, shortName rune, desc string) Flag {
	/*return &flag{
		name:      name,
		shortName: shortName,
		argNode: &boolFlag{
			name: name,
		},
	}*/
	return &boolFlag{
		name:      name,
		desc:      desc,
		shortName: shortName,
	}
}

type boolFlag struct {
	name      string
	shortName rune
	desc      string
}

func (bf *boolFlag) Desc() string {
	return bf.desc
}

func (bf *boolFlag) Name() string {
	return bf.name
}

func (bf *boolFlag) ShortName() rune {
	return bf.shortName
}

func (bf *boolFlag) Processor() Processor {
	return bf
}

func (bf *boolFlag) Complete(*Input, *Data) *CompleteData {
	return nil
}

func (bf *boolFlag) Usage(u *Usage) {
	// Since flag nodes are added at the beginning, the usage statements can be a bit awkward
	// Instead add another row for supported flags
	u.UsageSection.Add(FlagSection, bf.name, bf.desc)
}

func (bf *boolFlag) Execute(_ *Input, _ Output, data *Data, _ *ExecuteData) error {
	data.Set(bf.name, TrueValue())
	return nil
}

func StringListFlag(name string, shortName rune, desc string, minN, optionalN int, opts ...ArgOpt) Flag {
	return listFlag(name, desc, shortName, minN, optionalN, StringListType, opts...)
}

func IntListFlag(name string, shortName rune, desc string, minN, optionalN int, opts ...ArgOpt) Flag {
	return listFlag(name, desc, shortName, minN, optionalN, IntListType, opts...)
}

func FloatListFlag(name string, shortName rune, desc string, minN, optionalN int, opts ...ArgOpt) Flag {
	return listFlag(name, desc, shortName, minN, optionalN, FloatListType, opts...)
}

func listFlag(name, desc string, shortName rune, minN, optionalN int, vt ValueType, opts ...ArgOpt) Flag {
	return &flag{
		name:      name,
		desc:      desc,
		shortName: shortName,
		argNode: &ArgNode{
			flag:      true,
			name:      name,
			minN:      minN,
			optionalN: optionalN,
			opt:       newArgOpt(opts...),
			vt:        vt,
		},
	}
}
