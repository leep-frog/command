package command

import (
	"fmt"
	"regexp"
	"sort"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var (
	// MultiFlagRegex is the regex used to determine a multi-flag (`-qwer -> -q -w -e -r`).
	// It explicitly doesn't allow short number flags.
	MultiFlagRegex = regexp.MustCompile("^-[^-0-9]{2,}$")
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
	ProcessMissing(d *Data) error
	// Combinable indicates whether or not the short flag can be combined
	// with other flags (`-qwer` = `-q -w -e -r`, for example).
	// When used as a combinable flag, the flag will be evaluated with
	// an empty `Input` object.
	Combinable(d *Data) bool
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
		a, ok := input.PeekAt(i)
		if !ok {
			i++
			continue
		}
		// Check if combinable flag (e.g. `-qwer` -> `-q -w -e -r`).
		if MultiFlagRegex.MatchString(a) {
			for j := 1; j < len(a); j++ {
				shortCode := fmt.Sprintf("-%s", string(a[j]))
				f, ok := fn.flagMap[shortCode]
				// Run multi-flags on a best-effort basis
				if !ok || !f.Combinable(data) {
					continue
				}
				delete(unprocessed, f.Name())

				// Pass an empty input so multiple flags don't compete
				// for the remaining args. Only return if an error is returned,
				// because all multi-flag objects should never be completed.
				if _, err := f.Processor().Complete(NewInput(nil, nil), data); err != nil {
					return nil, err
				}
			}

			// This is outside of the for-loop so we only remove
			// the multi-flag arg (not one arg per flag).
			input.offset = i
			input.Pop()
			input.offset = 0
		} else if f, ok := fn.flagMap[a]; ok {
			// If regular flag

			delete(unprocessed, f.Name())

			input.offset = i
			// Remove flag argument (e.g. --flagName).
			input.Pop()
			c, err := f.Processor().Complete(input, data)
			input.offset = 0
			if c != nil || err != nil {
				return c, err
			}
		} else {
			i++
			continue
		}
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
		if err := f.ProcessMissing(data); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (fn *flagNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	unprocessed := map[string]Flag{}
	for _, f := range fn.flagMap {
		unprocessed[f.Name()] = f
	}
	for i := 0; i < len(input.remaining); {
		a, ok := input.PeekAt(i)
		if !ok {
			i++
			continue
		}
		// Check if combinable flag (e.g. `-qwer` -> `-q -w -e -r`).
		if MultiFlagRegex.MatchString(a) {
			for j := 1; j < len(a); j++ {
				shortCode := fmt.Sprintf("-%s", string(a[j]))
				f, ok := fn.flagMap[shortCode]
				if !ok {
					return output.Stderrf("Unknown flag code %q used in multi-flag", shortCode)
				}
				if !f.Combinable(data) {
					return output.Stderrf("Flag %q is not combinable", f.Name())
				}
				delete(unprocessed, f.Name())

				// Pass an empty input so multiple flags don't compete
				// for the remaining args
				if err := f.Processor().Execute(NewInput(nil, nil), output, data, eData); err != nil {
					return err
				}
			}

			// This is outside of the for-loop so we only remove
			// the multi-flag arg (not one arg per flag).
			input.offset = i
			input.Pop()
			input.offset = 0
		} else if f, ok := fn.flagMap[a]; ok {
			// If regular flag
			delete(unprocessed, f.Name())

			input.offset = i
			// Remove flag argument (e.g. --flagName).
			input.Pop()
			err := f.Processor().Execute(input, output, data, eData)
			input.offset = 0
			if err != nil {
				return err
			}
		} else {
			i++
			continue
		}
	}

	// Sort keys for deterministic behavior
	keys := maps.Keys(unprocessed)
	slices.Sort(keys)
	for _, k := range keys {
		if err := unprocessed[k].ProcessMissing(data); err != nil {
			return output.Annotatef(err, "failed to get default")
		}
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

func (f *flag[T]) ProcessMissing(d *Data) error {
	if f.argNode.opt == nil || f.argNode.opt._default == nil {
		return nil
	}

	def, err := f.argNode.opt._default.f(d)
	if err != nil {
		return err
	}
	f.argNode.Set(def, d)
	return nil
}

func (f *flag[T]) Name() string {
	return f.name
}

func (f *flag[T]) ShortName() rune {
	return f.shortName
}

func (f *flag[T]) Combinable(*Data) bool {
	return false
}

func (f *flag[T]) Get(d *Data) T {
	return GetData[T](d, f.name)
}

// NewFlag creates a `Flag` from argument info.
func NewFlag[T any](name string, shortName rune, desc string, opts ...ArgOpt[T]) FlagWithType[T] {
	return listFlag(name, desc, shortName, 1, 0, opts...)
}

// BoolFlag creates a `Flag` for a boolean argument.
func BoolFlag(name string, shortName rune, desc string) FlagWithType[bool] {
	return &boolFlag[bool]{
		name:      name,
		desc:      desc,
		shortName: shortName,
		trueValue: true,
	}
}

// BoolValueFlag creates a boolean `Flag` whose data value gets set to
// `trueValue` if the flag is provided. Otherwise, it is set to `falseValue`.
func BoolValueFlag[T any](name string, shortName rune, desc string, trueValue, falseValue T) FlagWithType[T] {
	return &boolFlag[T]{name, shortName, desc, trueValue, &falseValue}
}

type boolFlag[T any] struct {
	name       string
	shortName  rune
	desc       string
	trueValue  T
	falseValue *T
}

func (bf *boolFlag[T]) Desc() string {
	return bf.desc
}

func (bf *boolFlag[T]) Name() string {
	return bf.name
}

func (bf *boolFlag[T]) ShortName() rune {
	return bf.shortName
}

func (bf *boolFlag[T]) Processor() Processor {
	return bf
}

func (bf *boolFlag[T]) Combinable(*Data) bool {
	return true
}

func (bf *boolFlag[T]) ProcessMissing(d *Data) error {
	if bf.falseValue != nil {
		d.Set(bf.name, *bf.falseValue)
	}
	return nil
}

func (bf *boolFlag[T]) Complete(input *Input, data *Data) (*Completion, error) {
	data.Set(bf.name, bf.trueValue)
	return nil, nil
}

func (bf *boolFlag[T]) Usage(u *Usage) {
	// Since flag nodes are added at the beginning, the usage statements can be a bit awkward
	// Instead add another row for supported flags
	u.UsageSection.Add(FlagSection, bf.name, bf.desc)
}

func (bf *boolFlag[T]) Execute(_ *Input, _ Output, data *Data, _ *ExecuteData) error {
	data.Set(bf.name, bf.trueValue)
	return nil
}

func (bf *boolFlag[T]) Get(d *Data) T {
	return GetData[T](d, bf.name)
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
			opt:       multiArgOpts(opts...),
		},
	}
}
