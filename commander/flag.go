package commander

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/constants"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const (
	// FlagNoShortName is the rune value for flags that indicates no short flag should be included.
	FlagNoShortName rune = constants.FlagNoShortName // Runes are actually int32. Negative values indicate unknown rune

	// FlagStop is a string indicating that no further flag values should be processed.
	FlagStop = "--"
)

var (
	// MultiFlagRegex is the regex used to determine a multi-flag (`-qwer -> -q -w -e -r`).
	// It explicitly doesn't allow short number flags.
	MultiFlagRegex = regexp.MustCompile("^-[a-zA-Z]{2,}$")
	ShortFlagRegex = regexp.MustCompile("^[a-zA-Z0-9]$")
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
	// command.Processor returns a node `command.Processor` that processes arguments after the flag indicator.
	Processor() command.Processor

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
	// an empty `command.Input` object.
	Combinable bool
	// AllowsMultiple returns whether or not the flag can be provided multiple times.
	AllowsMultiple bool
	// ProcessMissing processes the flag when it is not provided
	ProcessMissing func(*command.Data) error
	// PostProcess runs after the entire flag processor has been processed.
	PostProcess func(*command.Input, command.Output, *command.Data, *command.ExecuteData) error
}

func (fo *FlagOptions) combinable() bool {
	return fo != nil && fo.Combinable
}

func (fo *FlagOptions) allowsMultiple() bool {
	return fo != nil && fo.AllowsMultiple
}

func (fo *FlagOptions) processMissing(d *command.Data) error {
	if fo == nil || fo.ProcessMissing == nil {
		return nil
	}
	return fo.ProcessMissing(d)
}

func (fo *FlagOptions) postProcess(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
	if fo == nil || fo.PostProcess == nil {
		return nil
	}
	return fo.PostProcess(i, o, d, ed)
}

type FlagWithType[T any] interface {
	FlagInterface
	// Get returns the flags value from a `command.Data` object.
	Get(*command.Data) T
	// GetOrDefault returns the flags value from a `command.Data` object, if the flag was set.
	// Otherwise, it returns the provided input.
	GetOrDefault(*command.Data, T) T
	// Provided returns whether or not the flag was provided
	Provided(*command.Data) bool

	// AddOptions adds options to a `FlagWithType`. Although chaining isn't
	// conventional in go, it is done here because flags are usually declared as
	// package-level variables.
	AddOptions(...ArgumentOption[T]) FlagWithType[T]
}

func flagName(f FlagInterface) string {
	return fmt.Sprintf("--%s", f.Name())
}

func flagShortName(f FlagInterface) string {
	return fmt.Sprintf("-%c", f.ShortName())
}

// FlagProcessor returns a `command.Processor` that iterates over the remaining command line
// arguments and processes any flags that are present.
func FlagProcessor(fs ...FlagInterface) *flagProcessor {
	m := map[string]FlagInterface{}
	for _, f := range fs {
		// We explicitly don't check for duplicate keys to give more freedom to users
		// For example, if they wanted to override a flag from a separate package
		m[flagName(f)] = f
		sn := f.ShortName()
		if sn == FlagNoShortName {
			continue
		}
		if !ShortFlagRegex.MatchString(string(f.ShortName())) {
			panic(fmt.Sprintf("Short flag name %q must match regex %v", f.ShortName(), ShortFlagRegex))
		}
		m[flagShortName(f)] = f
	}
	return &flagProcessor{
		flagMap: m,
	}
}

type flagProcessor struct {
	flagMap map[string]FlagInterface
}

// ListBreaker returns a `ListBreaker` that breaks a list at any
// string that would be considered a flag (short/full flag name, multi-flag)
// in this flag processor.
func (fn *flagProcessor) ListBreaker() *ListBreaker[[]string] {
	return ListUntil[string](
		// Don't eat any full flags (e.g. --my-flag)
		&ValidatorOption[string]{
			func(s string, d *command.Data) error {
				if _, ok := fn.flagMap[s]; ok {
					return fmt.Errorf("value %q is a flag in the flag map", s)
				}
				return nil
			},
			"",
		},
		// Don't eat any multi-flags where all flags are in the FlagProcessor.
		&ValidatorOption[string]{
			func(s string, d *command.Data) error {
				if !MultiFlagRegex.MatchString(s) {
					return nil
				}
				for j := 1; j < len(s); j++ {
					shortCode := fmt.Sprintf("-%s", string(s[j]))
					if _, ok := fn.flagMap[shortCode]; !ok {
						// This isn't a multi-flag for this FlagProcessor, so eat the arg.
						return nil
					}
				}
				return fmt.Errorf("value %q is a multi-flag argument for the FlagProcessor", s)
			},
			"",
		},
	)
}

func (fn *flagProcessor) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	// unprocessed tracks the flags that have not been processed
	unprocessed := map[string]FlagInterface{}
	// available tracks the flags that can still be set (either because they
	// haven't been set yet or because `AllowsMultiple()` returned `true`).
	available := map[string]bool{}
	// Don't define `processed := map[string]bool{}` like we do in Execute
	// because we want to run completion on a best effort basis.
	// Specifically, we will try to complete a flag's value even
	// if the flag was provided twice.
	for _, f := range fn.flagMap {
		unprocessed[f.Name()] = f
		available[f.Name()] = true
	}
	for i := 0; i < input.NumRemaining(); {
		a, _ := input.PeekAt(i)

		// If it's the last arg.
		if i == input.NumRemaining()-1 && len(a) > 0 && a[0] == '-' {
			k := make([]string, 0, len(fn.flagMap))
			for n := range available {
				k = append(k, fmt.Sprintf("--%s", n))
			}
			sort.Strings(k)
			return &command.Completion{
				Suggestions: k,
			}, nil
		}

		// Stop processing flags
		if a == FlagStop {
			input.PopAt(i, data)
			return nil, nil
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
				if !f.Options().allowsMultiple() {
					delete(available, f.Name())
				}

				// Pass an empty input so multiple flags don't compete
				// for the remaining args. Only return if an error is returned,
				// because all multi-flag objects should never be completed.
				if _, err := processOrComplete(f.Processor(), command.NewInput(nil, nil), data); err != nil {
					return nil, err
				}
			}

			// This is outside of the for-loop so we only remove
			// the multi-flag arg (not one arg per flag).
			command.InputRunAtOffset[bool](input, i, func(subInput *command.Input) bool {
				subInput.Pop(data)
				return false
			})
		} else if f, ok := fn.flagMap[a]; ok {
			// If regular flag

			delete(unprocessed, f.Name())
			if !f.Options().allowsMultiple() {
				delete(available, f.Name())
			}

			var c *command.Completion
			var err error
			command.InputRunAtOffset[bool](input, i, func(subInput *command.Input) bool {
				// Remove flag argument (e.g. --flagName).
				subInput.Pop(data)
				subInput.PushBreakers(fn.ListBreaker())
				defer func() { subInput.PopBreakers(1) }()
				c, err = processOrComplete(f.Processor(), subInput, data)
				return false
			})
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

func (fn *flagProcessor) Execute(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	unprocessed := map[string]FlagInterface{}
	processed := map[string]bool{}
	for _, f := range fn.flagMap {
		unprocessed[f.Name()] = f
	}
	for i := 0; i < input.NumRemaining(); {
		a, ok := input.PeekAt(i)
		if !ok {
			break
		}

		// Stop processing flags
		if a == FlagStop {
			input.PopAt(i, data)
			break
		}

		// Check if combinable flag (e.g. `-qwer` -> `-q -w -e -r`).
		if MultiFlagRegex.MatchString(a) {

			var matchCount int
			for j := 1; j < len(a); j++ {
				shortCode := fmt.Sprintf("-%s", string(a[j]))
				if _, ok := fn.flagMap[shortCode]; ok {
					matchCount++
				}
			}
			if matchCount == 0 {
				i++
				continue
			}
			if matchCount != len(a)-1 {
				return output.Stderrln("Either all or no flags in a multi-flag object must be relevant for a FlagProcessor group")
			}

			for j := 1; j < len(a); j++ {
				shortCode := fmt.Sprintf("-%s", string(a[j]))
				f := fn.flagMap[shortCode]
				if !f.Options().combinable() {
					return output.Stderrf("Flag %q is not combinable\n", f.Name())
				}
				delete(unprocessed, f.Name())

				if !f.Options().allowsMultiple() && processed[f.Name()] {
					return output.Stderrf("Flag %q has already been set\n", f.Name())
				}
				processed[f.Name()] = true

				// Pass an empty input so multiple flags don't compete
				// for the remaining args
				if err := processOrExecute(f.Processor(), command.NewInput(nil, nil), output, data, eData); err != nil {
					return err
				}
			}

			// This is outside of the for-loop so we only remove
			// the multi-flag arg (not one arg per flag).
			input.PopAt(i, data)
		} else if f, ok := fn.flagMap[a]; ok {
			// If regular flag
			delete(unprocessed, f.Name())
			if !f.Options().allowsMultiple() && processed[f.Name()] {
				return output.Stderrf("Flag %q has already been set\n", f.Name())
			}
			processed[f.Name()] = true

			// Remove flag argument (e.g. --flagName).
			input.PopAt(i, data)

			// Run processor with fixed offset
			err := command.InputRunAtOffset[error](input, i, func(tmpInput *command.Input) error {
				tmpInput.PushBreakers(fn.ListBreaker())
				defer input.PopBreakers(1)
				return processOrExecute(f.Processor(), tmpInput, output, data, eData)
			})
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

func (fn *flagProcessor) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	if err := fn.Execute(i, command.NewIgnoreAllOutput(), d, nil); err != nil && !IsNotEnoughArgsError(err) {
		return err
	}

	var flags []FlagInterface
	for k, f := range fn.flagMap {
		// flagMap contains entries for name and short name, so ensure we only do each one once.
		if k == flagName(f) {
			flags = append(flags, f)
		}
	}

	sort.SliceStable(flags, func(i, j int) bool { return flags[i].Name() < flags[j].Name() })

	for _, f := range flags {
		u.AddFlag(f.Name(), f.ShortName(), "TODO", f.Desc(), 0, 0) // TODO: numbers here
	}
	return nil
}

type flag[T any] struct {
	name      string
	desc      string
	shortName rune
	argument  *Argument[T]
}

func (f *flag[T]) Desc() string {
	return f.desc
}

func (f *flag[T]) Processor() command.Processor {
	return f.argument
}

func (f *flag[T]) Options() *FlagOptions {
	return &FlagOptions{
		ProcessMissing: func(d *command.Data) error {
			if f.argument.opt == nil || f.argument.opt._default == nil {
				return nil
			}

			f.argument.Set(f.argument.opt._default.v, d)
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

func (f *flag[T]) Get(d *command.Data) T {
	return command.GetData[T](d, f.name)
}

func (f *flag[T]) GetOrDefault(d *command.Data, t T) T {
	if f.Provided(d) {
		return command.GetData[T](d, f.name)
	}
	return t
}

func (f *flag[T]) Provided(d *command.Data) bool {
	return d.Has(f.name)
}

func (f *flag[T]) AddOptions(opts ...ArgumentOption[T]) FlagWithType[T] {
	for _, o := range opts {
		o.modifyArgumentOption(f.argument.opt)
	}
	return f
}

// Flag creates a `FlagInterface` from argument info.
func Flag[T any](name string, shortName rune, desc string, opts ...ArgumentOption[T]) FlagWithType[T] {
	return listFlag(name, desc, shortName, 1, 0, opts...) //.AddOptions(ListUntil[T](MatchesRegex("^-")))
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

func (bf *boolFlag[T]) Processor() command.Processor {
	return bf
}

func (bf *boolFlag[T]) Options() *FlagOptions {
	return &FlagOptions{
		Combinable: true,
		ProcessMissing: func(d *command.Data) error {
			if bf.falseValue != nil {
				d.Set(bf.name, *bf.falseValue)
			}
			return nil
		},
	}
}

func (bf *boolFlag[T]) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	data.Set(bf.name, bf.trueValue)
	return nil, nil
}

func (bf *boolFlag[T]) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	// I believe individual abcFlag.Usage functions aren't ever called (e.g. this, optionalFlag.Usage, etc.)

	// Since flag processors are added at the beginning, the usage statements can be a bit awkward
	// Instead add another row for supported flags
	u.AddFlag(bf.name, bf.shortName, "TODO-BOOL", bf.desc, 0, 0)
	return nil
}

func (bf *boolFlag[T]) Execute(_ *command.Input, _ command.Output, data *command.Data, _ *command.ExecuteData) error {
	data.Set(bf.name, bf.trueValue)
	return nil
}

func (bf *boolFlag[T]) Get(d *command.Data) T {
	return command.GetData[T](d, bf.name)
}

func (bf *boolFlag[T]) GetOrDefault(d *command.Data, t T) T {
	if bf.Provided(d) {
		return command.GetData[T](d, bf.name)
	}
	return t
}

func (bf *boolFlag[T]) Provided(d *command.Data) bool {
	return d.Has(bf.name)
}

func (bf *boolFlag[T]) AddOptions(opts ...ArgumentOption[T]) FlagWithType[T] {
	panic("options cannot be added to a boolean flag")
}

type optionalFlag[T any] struct {
	FlagWithType[T]

	defaultValue T
}

// OptionalFlag is a flag that can accept an optional parameter. Unlike `OptionalArg`, it actually has three different outcomes:
// Example with `OptionalFlag[string]("optStr", 'o', "description", "default-value")`
// 1. `Args=["--optStr"]`: The flag's value is set to "default-value" in data.
// 2. `Args=[]`: The flag's value isn't set (or is set to command.Default(...) option if provided).
// 3. `Args=["--optStr", "custom-value"]`: The flag's value is set to "custom-value" in data.
func OptionalFlag[T any](name string, shortName rune, desc string, defaultValue T, opts ...ArgumentOption[T]) FlagWithType[T] {
	return &optionalFlag[T]{
		listFlag(name, desc, shortName, 0, 1, opts...),
		defaultValue,
	}
}

func (of *optionalFlag[T]) Processor() command.Processor {
	return of
}

func (of *optionalFlag[T]) Execute(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	if err := processOrExecute(of.FlagWithType.Processor(), input, output, data, eData); err != nil {
		return err
	}

	of.setDefault(data)
	return nil
}

func (of *optionalFlag[T]) setDefault(data *command.Data) {
	if !data.Has(of.Name()) {
		data.Set(of.Name(), of.defaultValue)
	}
}

func (of *optionalFlag[T]) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	// Complete flag argument if necessary
	if input.NumRemaining() <= 1 {
		if a, _ := input.Peek(); len(a) > 0 && a[0] == '-' {
			return nil, nil
		}
	}

	// Otherwise just run regular completion.
	c, err := processOrComplete(of.FlagWithType.Processor(), input, data)
	if c != nil || err != nil {
		return c, err
	}

	of.setDefault(data)
	return nil, nil
}

func (of *optionalFlag[T]) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	return of.FlagWithType.Processor().Usage(i, d, u)
}

// TODO: command.Node that populates a struct from arguments.
/*
type structable struct {
	field1 `json:"f1"
}
func StructArg[structable](... the usual ...)
This should create an arg that can be processed with flags or positional arguments:
`v1` -> &struct{"v1"}
`--field1 v1` -> &struct{"v1"}
Make intermediate arg that transforms values into json and then json into struct
*/

// ItemizedListFlag creates a flag that can be set with separate flags (e.g. `cmd -i value-one -i value-two -b other-flag -i value-three`).
func ItemizedListFlag[T any](name string, shortName rune, desc string, opts ...ArgumentOption[[]T]) FlagWithType[[]T] {
	return &itemizedListFlag[T]{
		FlagWithType: ListFlag(name, shortName, desc, 0, command.UnboundedList, opts...),
	}
}

type itemizedListFlag[T any] struct {
	FlagWithType[[]T]

	rawArgs []string
}

func (ilf *itemizedListFlag[T]) Options() *FlagOptions {
	return &FlagOptions{
		// Combinable
		ilf.FlagWithType.Options().combinable(),
		// AllowsMultiple
		true,
		// ProcessMissing
		func(d *command.Data) error {
			return ilf.FlagWithType.Options().processMissing(d)
		},
		// PostProcess
		func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
			return processOrExecute(ilf.FlagWithType.Processor(), command.NewInput(ilf.rawArgs, nil), o, d, ed)
		},
	}
}

func (ilf *itemizedListFlag[T]) Processor() command.Processor {
	return ilf
}

func (ilf *itemizedListFlag[T]) Execute(input *command.Input, output command.Output, data *command.Data, eData *command.ExecuteData) error {
	s, ok := input.Pop(data)
	if !ok {
		return output.Err(NotEnoughArgs(ilf.Name(), 1, 0))
	}
	ilf.rawArgs = append(ilf.rawArgs, s)
	return nil
}

func (ilf *itemizedListFlag[T]) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	if input.FullyProcessed() {
		// Don't think it's possible to get here (because the flag Complete function
		// would complete the flag value ("--ilf") if it was the last value). So,
		// the input will always have at least one more argument.
		return nil, nil
	}
	s, _ := input.Pop(data)
	ilf.rawArgs = append(ilf.rawArgs, s)
	if input.FullyProcessed() {
		c, e := processOrComplete(ilf.FlagWithType.Processor(), command.NewInput(ilf.rawArgs, nil), data)
		return c, e
	}
	return nil, nil
}

func (ilf *itemizedListFlag[T]) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	return ilf.FlagWithType.Processor().Usage(i, d, u)
}

// ListFlag creates a `FlagInterface` from list argument info.
func ListFlag[T any](name string, shortName rune, desc string, minN, optionalN int, opts ...ArgumentOption[[]T]) FlagWithType[[]T] {
	return listFlag(name, desc, shortName, minN, optionalN, opts...)
}

func listFlag[T any](name, desc string, shortName rune, minN, optionalN int, opts ...ArgumentOption[T]) *flag[T] {
	return &flag[T]{
		name:      name,
		desc:      desc,
		shortName: shortName,
		argument: &Argument[T]{
			flag:      true,
			name:      name,
			minN:      minN,
			optionalN: optionalN,
			opt:       multiArgumentOptions(opts...),
		},
	}
}