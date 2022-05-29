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
	// ProcessMissing processes the flag when it is not provided
	ProcessMissing(d *Data)
}

type FlagWithType[T any] interface {
	Flag
	// Get returns the flags value from a `Data` object.
	Get(*Data) T
}

func flagName(f Flag) string {
	return fmt.Sprintf("--%s", f.Name())
}

func flagShortName(f Flag) string {
	return fmt.Sprintf("-%c", f.ShortName())
}

// NewFlagNode returns a node that iterates over the remaining command line
// arguments and processes any flags that are present.
func NewFlagNode(fs ...Flag) Processor {
	m := map[string]Flag{}
	for _, f := range fs {
		// We explicitly don't check for duplicate keys to give more freedom to users
		// For example, if they wanted to override a flag from a separate package
		m[flagName(f)] = f
		m[flagShortName(f)] = f
	}
	return &flagNode{
		flagMap: m,
	}
}

type flagNode struct {
	flagMap map[string]Flag
}

func (fn *flagNode) Complete(input *Input, data *Data) (*Completion, error) {
	unprocessed := map[string]Flag{}
	for _, f := range fn.flagMap {
		unprocessed[f.Name()] = f
	}
	for i := 0; i < len(input.remaining); {
		a, _ := input.PeekAt(i)
		f, ok := fn.flagMap[a]
		if !ok {
			i++
			continue
		}
		delete(unprocessed, f.Name())

		input.offset = i
		// Remove flag argument (e.g. --flagName).
		input.Pop()
		c, err := f.Processor().Complete(input, data)
		if c != nil || err != nil {
			input.offset = 0
			return c, err
		}
		input.offset = 0
	}

	if lastArg, ok := input.PeekAt(len(input.remaining) - 1); ok && len(lastArg) > 0 && lastArg[0] == '-' {
		k := make([]string, 0, len(fn.flagMap))
		for n := range fn.flagMap {
			k = append(k, n)
		}
		sort.Strings(k)
		return &Completion{
			Suggestions: k,
		}, nil
	}
	for _, f := range unprocessed {
		f.ProcessMissing(data)
	}
	return nil, nil
}

func (fn *flagNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	unprocessed := map[string]Flag{}
	for _, f := range fn.flagMap {
		unprocessed[f.Name()] = f
	}
	for i := 0; i < len(input.remaining); {
		a, _ := input.PeekAt(i)
		f, ok := fn.flagMap[a]
		if !ok {
			i++
			continue
		}
		delete(unprocessed, f.Name())

		input.offset = i
		// Remove flag argument (e.g. --flagName).
		input.Pop()
		if err := f.Processor().Execute(input, output, data, eData); err != nil {
			input.offset = 0
			return err
		}
		input.offset = 0
	}
	for _, f := range unprocessed {
		f.ProcessMissing(data)
	}
	return nil
}

func (fn *flagNode) Usage(u *Usage) {
	var flags []Flag
	for k, f := range fn.flagMap {
		// flagMap contains entries for name and short name, so ensure we only do each one once.
		if k == flagName(f) {
			flags = append(flags, f)
		}
	}

	sort.SliceStable(flags, func(i, j int) bool { return flags[i].Name() < flags[j].Name() })

	for _, f := range flags {
		if f.Desc() != "" {
			u.UsageSection.Add(FlagSection, fmt.Sprintf("[%c] %s", f.ShortName(), f.Name()), f.Desc())
		}

		u.Flags = append(u.Flags, fmt.Sprintf("%s|%s", flagName(f), flagShortName(f)))
	}
}

type flag[T any] struct {
	name      string
	desc      string
	shortName rune
	argNode   *ArgNode[T]
}

func (f *flag[T]) Desc() string {
	return f.desc
}

func (f *flag[T]) Processor() Processor {
	return f.argNode
}

func (f *flag[T]) ProcessMissing(d *Data) {
	if f.argNode.opt != nil && f.argNode.opt._default != nil {
		f.argNode.Set(*f.argNode.opt._default, d)
	}
}

func (f *flag[T]) Name() string {
	return f.name
}

func (f *flag[T]) ShortName() rune {
	return f.shortName
}

func (f *flag[T]) Get(d *Data) T {
	return GetData[T](d, f.name)
}

// NewFlag creates a `Flag` from argument info.
func NewFlag[T any](name string, shortName rune, desc string, opts ...ArgOpt[T]) FlagWithType[T] {
	return listFlag(name, desc, shortName, 1, 0, opts...)
}

// BoolFlag creates a `Flag` for a booean argument.
func BoolFlag(name string, shortName rune, desc string) FlagWithType[bool] {
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

func (bf *boolFlag) ProcessMissing(*Data) {}

func (bf *boolFlag) Complete(input *Input, data *Data) (*Completion, error) {
	data.Set(bf.name, true)
	return nil, nil
}

func (bf *boolFlag) Usage(u *Usage) {
	// Since flag nodes are added at the beginning, the usage statements can be a bit awkward
	// Instead add another row for supported flags
	u.UsageSection.Add(FlagSection, bf.name, bf.desc)
}

func (bf *boolFlag) Execute(_ *Input, _ Output, data *Data, _ *ExecuteData) error {
	data.Set(bf.name, true)
	return nil
}

func (bf *boolFlag) Get(d *Data) bool {
	return GetData[bool](d, bf.name)
}

// NewListFlag creates a `Flag` from list argument info.
func NewListFlag[T any](name string, shortName rune, desc string, minN, optionalN int, opts ...ArgOpt[[]T]) FlagWithType[[]T] {
	return listFlag(name, desc, shortName, minN, optionalN, opts...)
}

func listFlag[T any](name, desc string, shortName rune, minN, optionalN int, opts ...ArgOpt[T]) FlagWithType[T] {
	return &flag[T]{
		name:      name,
		desc:      desc,
		shortName: shortName,
		argNode: &ArgNode[T]{
			flag:      true,
			name:      name,
			minN:      minN,
			optionalN: optionalN,
			opt:       newArgOpt(opts...),
		},
	}
}
