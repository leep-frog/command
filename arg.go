package command

import (
	"fmt"

	"golang.org/x/exp/slices"
)

// ArgNode is a type that implements `Processor`. It can be
// created via `Arg[T]` and `ListArg[T]` functions.
type ArgNode[T any] struct {
	name      string
	desc      string
	opt       *argOpt[T]
	minN      int
	optionalN int
	shortName rune
	flag      bool
}

// AddOptions adds options to an `ArgNode`. Although chaining isn't conventional
// in go, it is done here because args are usually declared as package-level
// variables.
func (an *ArgNode[T]) AddOptions(opts ...ArgOpt[T]) *ArgNode[T] {
	for _, o := range opts {
		o.modifyArgOpt(an.opt)
	}
	return an
}

// Name returns the name of the argument.
func (an *ArgNode[T]) Name() string {
	return an.name
}

// Desc returns the description of the argument.
func (an *ArgNode[T]) Desc() string {
	return an.desc
}

// Get fetches the arguments value from the `Data` object.
func (an *ArgNode[T]) Get(data *Data) T {
	return GetData[T](data, an.name)
}

func (an *ArgNode[T]) Has(data *Data) bool {
	return data.Has(an.name)
}

// GetOrDefault fetches the arguments value from the `Data` object.
func (an *ArgNode[T]) GetOrDefault(data *Data, dflt T) T {
	if data.Has(an.name) {
		return GetData[T](data, an.name)
	}
	return dflt
}

// Set sets the argument key in the given `Data` object.
func (an *ArgNode[T]) Set(v T, data *Data) {
	if an.opt != nil && an.opt.customSet != nil && an.opt.customSet.F != nil {
		an.opt.customSet.F(v, data)
	} else {
		data.Set(an.name, v)
	}
}

// Usage adds the command info to the provided `Usage` object.
func (an *ArgNode[T]) Usage(u *Usage) {
	if an.opt != nil && an.opt.hideUsage {
		return
	}

	if an.desc != "" {
		u.UsageSection.Add(ArgSection, an.name, an.desc)
		if an.opt != nil {
			for _, v := range an.opt.validators {
				u.UsageSection.Add(ArgSection, an.name, v.Usage)
			}
		}
	}

	for i := 0; i < an.minN; i++ {
		u.Usage = append(u.Usage, an.name)
	}
	if an.optionalN == UnboundedList {
		u.Usage = append(u.Usage, fmt.Sprintf("[ %s ... ]", an.name))
	} else {
		if an.optionalN > 0 {
			u.Usage = append(u.Usage, "[")
			for i := 0; i < an.optionalN; i++ {
				u.Usage = append(u.Usage, an.name)
			}
			u.Usage = append(u.Usage, "]")
		}
	}

	for _, b := range an.opt.breakers {
		b.Usage(u)
	}
}

// Execute fulfills the `Processor` interface for `ArgNode`.
func (an *ArgNode[T]) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
	an.shortcutCheck(i, o, data, false)

	sl, enough := i.PopN(an.minN, an.optionalN, an.opt.breakers)

	// Don't set at all if no arguments provided for arg.
	if len(sl) == 0 {
		if !enough {
			return o.Err(an.notEnoughErr(len(sl)))
		}
		if an.opt != nil && an.opt._default != nil {
			def, err := an.opt._default.f(data)
			if err != nil {
				return o.Annotatef(err, "failed to get default")
			}
			an.Set(def, data)
		}
		return nil
	}

	if an.opt != nil && an.opt.complexecute != nil && an.opt.complexecute.enabled {
		strict := an.opt.complexecute.strict

		// Iteratively complete arguments
		for i := 1; i <= len(sl); i++ {
			tsl := sl[:i]
			v, err := an.convertStringValue(tsl, data, false)
			data.complexecute = true
			compl, err := RunCompletion(an.opt.completer, *tsl[len(tsl)-1], v, data)
			data.complexecute = false
			if err != nil {
				if strict {
					return o.Annotatef(err, "[Complexecute] failed to fetch completion for %q", an.name)
				}
				continue
			} else if compl == nil {
				if strict {
					return o.Stderrf("[Complexecute] nil completion returned for %q\n", an.name)
				}
				continue
			}

			lastArg := *tsl[len(tsl)-1]
			suggestions := compl.process(lastArg, nil, true)
			if len(suggestions) == 1 || (an.opt.complexecute.exactMatch && slices.Contains(suggestions, lastArg)) {
				*tsl[len(tsl)-1] = suggestions[0]
			} else if strict {
				return o.Stderrf("[Complexecute] requires exactly one suggestion to be returned for %q, got %d: %v\n", an.name, len(suggestions), suggestions)
			}
		}
	}

	v, err := an.convertStringValue(sl, data, true)
	if err != nil {
		return o.Err(err)
	}

	// Copy values into returned list (required for shortcutting)
	newSl := getOperator[T]().toArgs(v)
	if len(newSl) != len(sl) {
		// We enforce this for Arg transformers. The change around `Input` are too complicated
		// to warrant enabling this functionality here, when users can easily just make a
		// separate processor that transforms the input args later on.
		return o.Stderrf("[%s] Transformers must return a value that is the same length as the original arguments\n", an.name)
	}
	for i := 0; i < len(sl); i++ {
		*sl[i] = newSl[i]
	}

	an.Set(v, data)

	if an.opt != nil {
		for _, validator := range an.opt.validators {
			if err := validator.RunValidation(an, v); err != nil {
				return o.Err(err)
			}
		}
	}

	if !enough {
		return o.Err(an.notEnoughErr(len(sl)))
	}
	return nil
}

func (an *ArgNode[T]) convertStringValue(sl []*string, data *Data, transform bool) (T, error) {
	var nill T
	// Transform from string to value.
	op := getOperator[T]()
	v, err := op.fromArgs(sl)
	if err != nil {
		return nill, err
	}

	// Run custom transformer if relevant
	if an.opt == nil || !transform {
		return v, nil
	}

	for _, transformer := range an.opt.transformers {
		newV, err := transformer.F(v, data)
		if err != nil {
			return nill, fmt.Errorf("Custom transformer failed: %v", err)
		}
		v = newV
	}
	return v, nil
}

func (an *ArgNode[T]) notEnoughErr(got int) error {
	return NotEnoughArgs(an.name, an.minN, got)
}

// IsNotEnoughArgsError returns whether or not the provided error
// is a `NotEnoughArgs` error.
func IsNotEnoughArgsError(err error) bool {
	_, ok := err.(*notEnoughArgs)
	return ok
}

// IsUsageError returns whether or not the provided error
// is a usage-related error.
func IsUsageError(err error) bool {
	return IsNotEnoughArgsError(err) || IsExtraArgsError(err) || IsBranchingError(err)
}

// NotEnoughArgs returns a custom error for when not enough arguments are provided to the command.
func NotEnoughArgs(name string, req, got int) error {
	return &notEnoughArgs{name, req, got}
}

type notEnoughArgs struct {
	name string
	req  int
	got  int
}

func (ne *notEnoughArgs) Error() string {
	plural := "s"
	if ne.req == 1 {
		plural = ""
	}
	return fmt.Sprintf("Argument %q requires at least %d argument%s, got %d", ne.name, ne.req, plural, ne.got)
}

func (an *ArgNode[T]) shortcutCheck(input *Input, output Output, data *Data, complete bool) error {
	if an.opt == nil || an.opt.shortcut == nil {
		return nil
	}

	upTo := an.minN + an.optionalN
	if an.optionalN == UnboundedList {
		upTo = len(input.remaining)
	}

	return shortcutInputTransformer(an.opt.shortcut.ShortcutCLI, an.opt.shortcut.ShortcutName, upTo-1).Transform(input, output, data, complete)
}

// Complete fulfills the `Processor` interface for `ArgNode`.
func (an *ArgNode[T]) Complete(input *Input, data *Data) (*Completion, error) {
	an.shortcutCheck(input, NewIgnoreAllOutput(), data, true)

	sl, enough := input.PopN(an.minN, an.optionalN, an.opt.breakers)

	// If this is the last arg, we want the node walkthrough to stop (which
	// doesn't happen if c and err are nil).
	c, err := an.complete(sl, enough, input, data)
	if (!enough || input.FullyProcessed()) && c == nil {
		c = &Completion{}
	}
	return c, err
}

func (an *ArgNode[T]) complete(sl []*string, enough bool, input *Input, data *Data) (*Completion, error) {
	// Try to transform from string to value.
	op := getOperator[T]()
	v, err := op.fromArgs(sl)
	if err != nil {
		// If we're on the last one, then complete it.
		if !enough || input.FullyProcessed() {
			var lastArg string
			if len(sl) > 0 {
				lastArg = *sl[len(sl)-1]
			}
			return RunCompletion(an.opt.completer, lastArg, v, data)
		}

		return nil, err
	}

	// Don't run validations when completing.

	// If we have enough and more needs to be processed, then nothing should
	// be completed, and we should process the arg as if we were executing.
	if enough && !input.FullyProcessed() {
		// Run custom transformer on a best effor basis (i.e. if the transformer fails,
		// then we just continue with the original value).
		if an.opt != nil {
			for _, transformer := range an.opt.transformers {
				// Don't return an error because this may not be the last one.
				newV, err := transformer.F(v, data)
				if err == nil {
					v = newV
				} else {
					break
				}
			}
		}

		an.Set(v, data)
		return nil, nil
	}

	// Otherwise, we are on the last value and should complete it.
	an.Set(v, data)

	// If there isn't a completer, then no work to be done.
	if an.opt == nil || an.opt.completer == nil {
		return nil, nil
	}

	// Otherwise, try to complete arg.
	var lastArg string
	ta := op.toArgs(v)
	if len(ta) > 0 {
		lastArg = ta[len(ta)-1]
	}
	return RunCompletion(an.opt.completer, lastArg, v, data)
}

// Arg creates an argument node that requires exactly one input.
func Arg[T any](name, desc string, opts ...ArgOpt[T]) *ArgNode[T] {
	return listNode(name, desc, 1, 0, opts...)
}

// OptionalArg creates an argument node that accepts zero or one input arguments.
func OptionalArg[T any](name, desc string, opts ...ArgOpt[T]) *ArgNode[T] {
	return listNode(name, desc, 0, 1, opts...)
}

// ListArg creates a list argument that requires at least `minN` arguments and
// at most `minN`+`optionalN` arguments. Use UnboundedList for `optionalN` to
// allow an unlimited number of arguments.
func ListArg[T any](name, desc string, minN, optionalN int, opts ...ArgOpt[[]T]) *ArgNode[[]T] {
	return listNode(name, desc, minN, optionalN, opts...)
}

// BoolNode creates a boolean argument.
func BoolNode(name, desc string) *ArgNode[bool] {
	return listNode[bool](name, desc, 1, 0, BoolCompleter())
}

func listNode[T any](name, desc string, minN, optionalN int, opts ...ArgOpt[T]) *ArgNode[T] {
	return &ArgNode[T]{
		name:      name,
		desc:      desc,
		minN:      minN,
		optionalN: optionalN,
		opt:       multiArgOpts(opts...),
	}
}
