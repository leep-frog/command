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

// FlagInterface defines a flag argument that is parsed regardless of it's position in
// the provided command line arguments.
type FlagInterface interface {
	// Name is the name of the flag. "--name" is the flags indicator
	Name() string
	// Desc is the description of the flag.
	Desc() string
	// ShortName indicates the shorthand version of the flag. "-s" is the short hand flag indicator.
	ShortName() rune
	// Processor returns a node processor that processes arguments after the flag indicator.
	Processor() Processor

	// Options returns the set of additional options for this flag.
	// Returning a separate type (rather than enumerating functions here)
	// allows us to update the options without breaking existing code.
	Options() *FlagOptions
}

// FlagOptions contains optional data for flags
type FlagOptions struct {
	// Combinable indicates whether or not the short flag can be combined
	// with other flags (`-qwer` = `-q -w -e -r`, for example).
	// When used as a combinable flag, the flag will be evaluated with
	// an empty `Input` object.
	Combinable bool
	// AllowsMultiple returns whether or not the flag can be provided multiple times.
	AllowsMultiple bool
	// ProcessMissing processes the flag when it is not provided
	ProcessMissing func(*Data) error
	// PostProcess runs after the entire flag node has been processed.
	PostProcess func(*Input, Output, *Data, *ExecuteData) error
}

func (fo *FlagOptions) combinable() bool {
	return fo != nil && fo.Combinable
}

func (fo *FlagOptions) allowsMultiple() bool {
	return fo != nil && fo.AllowsMultiple
}

func (fo *FlagOptions) processMissing(d *Data) error {
	if fo == nil || fo.ProcessMissing == nil {
		return nil
	}
	return fo.ProcessMissing(d)
}

func (fo *FlagOptions) postProcess(i *Input, o Output, d *Data, ed *ExecuteData) error {
	if fo == nil || fo.PostProcess == nil {
		return nil
	}
	return fo.PostProcess(i, o, d, ed)
}

type FlagWithType[T any] interface {
	FlagInterface
	// Get returns the flags value from a `Data` object.
	Get(*Data) T
	// AddOptions adds options to a `FlagWithType`. Although chaining isn't
	// conventional in go, it is done here because flags are usually declared as
	// package-level variables.
	AddOptions(...ArgOpt[T]) FlagWithType[T]
}

func flagName(f FlagInterface) string {
	return fmt.Sprintf("--%s", f.Name())
}

func flagShortName(f FlagInterface) string {
	return fmt.Sprintf("-%c", f.ShortName())
}

// FlagNode returns a node that iterates over the remaining command line
// arguments and processes any flags that are present.
func FlagNode(fs ...FlagInterface) Processor {
	m := map[string]FlagInterface{}
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
	flagMap map[string]FlagInterface
}

func (fn *flagNode) Complete(input *Input, data *Data) (*Completion, error) {
	unprocessed := map[string]FlagInterface{}
	// Don't define `processed := map[string]bool{}` like we do in Execute
	// because we want to run completion on a best effort basis.
	// Specifically, we will try to complete a flag's value even
	// if the flag was provided twice.
	for _, f := range fn.flagMap {
		unprocessed[f.Name()] = f
	}
	for i := 0; i < input.NumRemaining(); {
		a, _ := input.PeekAt(i)

		// If it's the last arg.
		if i == input.NumRemaining()-1 && len(a) > 0 && a[0] == '-' {
			// TODO: only complete full flag names
			k := make([]string, 0, len(fn.flagMap))
			for n := range fn.flagMap {
				k = append(k, n)
			}
			sort.Strings(k)
			return &Completion{
				Suggestions: k,
			}, nil
		}

		// Check if combinable flag (e.g. `-qwer` -> `-q -w -e -r`).
		if MultiFlagRegex.MatchString(a) {
			for j := 1; j < len(a); j++ {
				shortCode := fmt.Sprintf("-%s", string(a[j]))
				f, ok := fn.flagMap[shortCode]
				// Run multi-flags on a best-effort basis
				if !ok || !f.Options().combinable() {
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

	for _, f := range unprocessed {
		if err := f.Options().processMissing(data); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (fn *flagNode) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	// TODO: Flag args should check for other flag values.
	unprocessed := map[string]FlagInterface{}
	processed := map[string]bool{}
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
					return output.Stderrf("Unknown flag code %q used in multi-flag\n", shortCode)
				}
				if !f.Options().combinable() {
					return output.Stderrf("Flag %q is not combinable\n", f.Name())
				}
				delete(unprocessed, f.Name())

				if processed[f.Name()] {
					return output.Stderrf("Flag %q has already been set\n", f.Name())
				}
				processed[f.Name()] = true

				// Pass an empty input so multiple flags don't compete
				// for the remaining args
				if err := f.Processor().Execute(NewInput(nil, nil), output, data, eData); err != nil {
					return err
				}
			}

			// This is outside of the for-loop so we only remove
			// the multi-flag arg (not one arg per flag).
			// TODO: PopAt function?
			input.offset = i
			input.Pop()
			input.offset = 0
		} else if f, ok := fn.flagMap[a]; ok {
			// If regular flag
			delete(unprocessed, f.Name())
			if !f.Options().allowsMultiple() && processed[f.Name()] {
				return output.Stderrf("Flag %q has already been set\n", f.Name())
			}
			processed[f.Name()] = true

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
		if err := unprocessed[k].Options().processMissing(data); err != nil {
			return output.Annotatef(err, "failed to get default")
		}
	}
	for _, f := range fn.flagMap {
		f.Options().postProcess(input, output, data, eData)
	}
	return nil
}

func (fn *flagNode) Usage(u *Usage) {
	var flags []FlagInterface
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

func (f *flag[T]) Options() *FlagOptions {
	return &FlagOptions{
		ProcessMissing: func(d *Data) error {
			if f.argNode.opt == nil || f.argNode.opt._default == nil {
				return nil
			}

			def, err := f.argNode.opt._default.f(d)
			if err != nil {
				return err
			}
			f.argNode.Set(def, d)
			return nil
		},
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

func (f *flag[T]) Has(d *Data) bool {
	return d.Has(f.name)
}

func (f *flag[T]) AddOptions(opts ...ArgOpt[T]) FlagWithType[T] {
	for _, o := range opts {
		o.modifyArgOpt(f.argNode.opt)
	}
	return f
}

// Flag creates a `FlagInterface` from argument info.
func Flag[T any](name string, shortName rune, desc string, opts ...ArgOpt[T]) FlagWithType[T] {
	return listFlag(name, desc, shortName, 1, 0, opts...)
	//return listFlag(name, desc, shortName, 1, 0, opts...).AddOptions(ListUntil(MatchesRegex("^-")))
}

// BoolFlag creates a `FlagInterface` for a boolean argument.
func BoolFlag(name string, shortName rune, desc string) FlagWithType[bool] {
	return &boolFlag[bool]{
		name:      name,
		desc:      desc,
		shortName: shortName,
		trueValue: true,
	}
}

// BoolValueFlag creates a boolean `FlagInterface` whose data value gets set to
// `trueValue` if the flag is provided.
func BoolValueFlag[T any](name string, shortName rune, desc string, trueValue T) *boolFlag[T] {
	return &boolFlag[T]{name, shortName, desc, trueValue, nil}
}

// BoolValuesFlag creates a boolean `FlagInterface` whose data value gets set to
// `trueValue` if the flag is provided. Otherwise, it gets set to `falseValue`
func BoolValuesFlag[T any](name string, shortName rune, desc string, trueValue, falseValue T) *boolFlag[T] {
	return &boolFlag[T]{name, shortName, desc, trueValue, &falseValue}
}

type boolFlag[T any] struct {
	name       string
	shortName  rune
	desc       string
	trueValue  T
	falseValue *T
}

// TrueValue returns the value used when the boolean flag is set.
func (bf *boolFlag[T]) TrueValue() T {
	return bf.trueValue
}

// FalseValue returns the value used when the boolean flag is not set.
func (bf *boolFlag[T]) FalseValue() T {
	var t T
	if bf.falseValue == nil {
		return t
	}
	return *bf.falseValue
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

func (bf *boolFlag[T]) Options() *FlagOptions {
	return &FlagOptions{
		Combinable: true,
		ProcessMissing: func(d *Data) error {
			if bf.falseValue != nil {
				d.Set(bf.name, *bf.falseValue)
			}
			return nil
		},
	}
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

func (bf *boolFlag[T]) AddOptions(opts ...ArgOpt[T]) FlagWithType[T] {
	panic("options cannot be added to a boolean flag")
}

func ItemizedListFlag[T any](name string, shortName rune, desc string, opts ...ArgOpt[[]T]) FlagWithType[[]T] {
	return &itemizedListFlag[T]{
		flag: listFlag(name, desc, shortName, 0, UnboundedList, opts...),
	}
}

type itemizedListFlag[T any] struct {
	*flag[[]T]

	rawArgs []string
}

func (ilf *itemizedListFlag[T]) Options() *FlagOptions {
	return &FlagOptions{
		// Combinable
		ilf.flag.Options().combinable(),
		// AllowsMultiple
		true,
		// ProcessMissing
		func(d *Data) error {
			return ilf.flag.Options().processMissing(d)
		},
		// PostProcess
		func(i *Input, o Output, d *Data, ed *ExecuteData) error {
			return ilf.flag.argNode.Execute(NewInput(ilf.rawArgs, nil), o, d, ed)
		},
	}
}

func (ilf *itemizedListFlag[T]) Processor() Processor {
	return ilf
}

func (ilf *itemizedListFlag[T]) Execute(input *Input, output Output, data *Data, eData *ExecuteData) error {
	s, ok := input.Pop()
	if !ok {
		return output.Err(NotEnoughArgs(ilf.Name(), 1, 0))
	}
	ilf.rawArgs = append(ilf.rawArgs, s)
	return nil
}

func (ilf *itemizedListFlag[T]) Complete(input *Input, data *Data) (*Completion, error) {
	if input.FullyProcessed() {
		// Don't think it's possible to get here (because the flag Complete function
		// would complete the flag value ("--ilf") if it was the last value). So,
		// the input will always have at least one more argument.
		return nil, nil
	}
	s, _ := input.Pop()
	ilf.rawArgs = append(ilf.rawArgs, s)
	if input.FullyProcessed() {
		c, e := ilf.flag.argNode.Complete(NewInput(ilf.rawArgs, nil), data)
		return c, e
	}
	return nil, nil
}

func (ilf *itemizedListFlag[T]) Usage(u *Usage) {
	ilf.flag.Processor().Usage(u)
}

// ListFlag creates a `FlagInterface` from list argument info.
func ListFlag[T any](name string, shortName rune, desc string, minN, optionalN int, opts ...ArgOpt[[]T]) FlagWithType[[]T] {
	return listFlag(name, desc, shortName, minN, optionalN, opts...)
}

func listFlag[T any](name, desc string, shortName rune, minN, optionalN int, opts ...ArgOpt[T]) *flag[T] {
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
