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

func StringFlag(name string, shortName rune, opts ...ArgOptt) Flag {
	return listFlag(name, shortName, 1, 0, stringTransform, opts...)
}

func IntFlag(name string, shortName rune, opts ...ArgOptt) Flag {
	return listFlag(name, shortName, 1, 0, intTransform, opts...)
}

func FloatFlag(name string, shortName rune, opts ...ArgOptt) Flag {
	return listFlag(name, shortName, 1, 0, floatTransform, opts...)
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

func StringListFlag(name string, shortName rune, minN, optionalN int, opts ...ArgOptt) Flag {
	return listFlag(name, shortName, minN, optionalN, stringListTransform, opts...)
}

func IntListFlag(name string, shortName rune, minN, optionalN int, opts ...ArgOptt) Flag {
	return listFlag(name, shortName, minN, optionalN, intListTransform, opts...)
}

func FloatListFlag(name string, shortName rune, minN, optionalN int, opts ...ArgOptt) Flag {
	return listFlag(name, shortName, minN, optionalN, floatListTransform, opts...)
}

func listFlag(name string, shortName rune, minN, optionalN int, transform func(s []*string) (*Value, error), opts ...ArgOptt) Flag {
	ao := &argOpt{}
	for _, opt := range opts {
		opt.modifyArgOpt(ao)
	}
	return &flag{
		name:      name,
		shortName: shortName,
		argNode: &argNode{
			flag:      true,
			name:      name,
			minN:      minN,
			optionalN: optionalN,
			opt:       ao,
			transform: transform,
		},
	}
}
