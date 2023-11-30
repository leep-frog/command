package commander

import (
	"fmt"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/operator"
	"golang.org/x/exp/slices"
)

// Argument is a type that implements `command.Processor`. It can be
// created via `Arg[T]` and `ListArg[T]` functions.
type Argument[T any] struct {
	name      string
	desc      string
	opt       *argumentOption[T]
	minN      int
	optionalN int
	shortName rune
	flag      bool
}

// AddOptions adds options to an `Argument`. Although chaining isn't conventional
// in go, it is done here because args are usually declared as package-level
// variables.
func (an *Argument[T]) AddOptions(opts ...ArgumentOption[T]) *Argument[T] {
	for _, o := range opts {
		o.modifyArgumentOption(an.opt)
	}
	return an
}

// Name returns the name of the argument.
func (an *Argument[T]) Name() string {
	return an.name
}

// Desc returns the description of the argument.
func (an *Argument[T]) Desc() string {
	return an.desc
}

// Get fetches the arguments value from the `command.Data` object.
func (an *Argument[T]) Get(data *command.Data) T {
	return command.GetData[T](data, an.name)
}

// Provided returns whether or not the argument has been set in `command.Data`.
func (an *Argument[T]) Provided(data *command.Data) bool {
	return data.Has(an.name)
}

// GetOrDefault fetches the arguments value from the `command.Data` object.
func (an *Argument[T]) GetOrDefault(data *command.Data, dflt T) T {
	if data.Has(an.name) {
		return command.GetData[T](data, an.name)
	}
	return dflt
}

// Set sets the argument key in the given `command.Data` object.
func (an *Argument[T]) Set(v T, data *command.Data) {
	if an.opt != nil && an.opt.customSet != nil && an.opt.customSet.F != nil {
		an.opt.customSet.F(v, data)
	} else {
		data.Set(an.name, v)
	}
}

// command.Usage adds the command info to the provided `command.Usage` object.
func (an *Argument[T]) Usage(i *command.Input, d *command.Data, u *command.Usage) error {
	noneRemaining := i.NumRemaining() == 0
	err := an.Execute(i, command.NewIgnoreAllOutput(), d, nil)
	// If enough arguments have been provided, then no need to print usage.
	if err == nil && !noneRemaining {
		return nil
	}

	// If actual error, then return
	if err != nil && !IsNotEnoughArgsError(err) {
		return err
	}

	// Otherwise, there weren't enough args, so generate usage info.
	if an.opt != nil && an.opt.hideUsage {
		return nil
	}

	if an.desc != "" {
		u.UsageSection.Add(command.ArgSection, an.name, an.desc)
		if an.opt != nil {
			for _, v := range an.opt.validators {
				u.UsageSection.Add(command.ArgSection, an.name, v.Usage)
			}
		}
	}

	for i := 0; i < an.minN; i++ {
		u.Usage = append(u.Usage, an.name)
	}
	if an.optionalN == command.UnboundedList {
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
		if err := b.Usage(d, u); err != nil {
			return fmt.Errorf("InputBreaker usage failed: %v", err)
		}
	}
	return nil
}

// Execute fulfills the `command.Processor` interface for `Argument`.
func (an *Argument[T]) Execute(i *command.Input, o command.Output, data *command.Data, eData *command.ExecuteData) error {
	an.shortcutCheck(i, o, data, false)

	sl, enough := i.PopN(an.minN, an.optionalN, an.opt.inputValidators(), data)

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
			data.Complexecute = true
			compl, err := RunArgumentCompleter(an.opt.completer, v, data)
			data.Complexecute = false
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
			suggestions := compl.Process(lastArg, nil, true)
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
	newSl := operator.GetOperator[T]().ToArgs(v)
	if len(newSl) != len(sl) {
		// We enforce this for Arg transformers. The change around `command.Input` are too complicated
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
			if err := validator.RunValidation(an, v, data); err != nil {
				return o.Err(err)
			}
		}
	}

	if !enough {
		return o.Err(an.notEnoughErr(len(sl)))
	}
	return nil
}

func (an *Argument[T]) convertStringValue(sl []*string, data *command.Data, transform bool) (T, error) {
	var nill T
	// Transform from string to value.
	op := operator.GetOperator[T]()
	v, err := op.FromArgs(sl)
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

func (an *Argument[T]) notEnoughErr(got int) error {
	return NotEnoughArgs(an.name, an.minN, got)
}

func (an *Argument[T]) shortcutCheck(input *command.Input, output command.Output, data *command.Data, complete bool) error {
	if an.opt == nil || an.opt.shortcut == nil {
		return nil
	}

	upTo := an.minN + an.optionalN
	if an.optionalN == command.UnboundedList {
		upTo = input.NumRemaining()
	}

	return shortcutInputTransformer(an.opt.shortcut.ShortcutCLI, an.opt.shortcut.ShortcutName, upTo-1).Transform(input, output, data, complete)
}

func shortcutInputTransformer(sc ShortcutCLI, name string, upToIndex int) *command.InputTransformer {
	return &command.InputTransformer{F: func(o command.Output, d *command.Data, s string) ([]string, error) {
		sl, ok := getShortcut(sc, name, s)
		if !ok {
			return []string{s}, nil
		}
		return sl, nil
	}, UpToIndex: upToIndex}
}

// Complete fulfills the `command.Processor` interface for `Argument`.
func (an *Argument[T]) Complete(input *command.Input, data *command.Data) (*command.Completion, error) {
	an.shortcutCheck(input, command.NewIgnoreAllOutput(), data, true)

	sl, enough := input.PopN(an.minN, an.optionalN, an.opt.inputValidators(), data)

	// If this is the last arg, we want the node walkthrough to stop (which
	// doesn't happen if c and err are nil).
	c, err := an.complete(sl, enough, input, data)
	if (!enough || input.FullyProcessed()) && c == nil {
		c = &command.Completion{}
	}
	return c, err
}

func (an *Argument[T]) complete(sl []*string, enough bool, input *command.Input, data *command.Data) (*command.Completion, error) {
	// Try to transform from string to value.
	op := operator.GetOperator[T]()
	v, err := op.FromArgs(sl)
	if err != nil {
		// If we're on the last one, then complete it.
		if !enough || input.FullyProcessed() {
			return RunArgumentCompleter(an.opt.completer, v, data)
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

	return RunArgumentCompleter(an.opt.completer, v, data)
}

// Arg creates an argument `command.Processor` that requires exactly one input.
func Arg[T any](name, desc string, opts ...ArgumentOption[T]) *Argument[T] {
	return listArgument(name, desc, 1, 0, opts...)
}

// OptionalArg creates an argument `command.Processor` that accepts zero or one input arguments.
func OptionalArg[T any](name, desc string, opts ...ArgumentOption[T]) *Argument[T] {
	return listArgument(name, desc, 0, 1, opts...)
}

// ListArg creates a list argument that requires at least `minN` arguments and
// at most `minN`+`optionalN` arguments. Use UnboundedList for `optionalN` to
// allow an unlimited number of arguments.
func ListArg[T any](name, desc string, minN, optionalN int, opts ...ArgumentOption[[]T]) *Argument[[]T] {
	return listArgument(name, desc, minN, optionalN, opts...)
}

// BoolArg creates a boolean argument.
func BoolArg(name, desc string) *Argument[bool] {
	return listArgument[bool](name, desc, 1, 0, BoolCompleter())
}

func listArgument[T any](name, desc string, minN, optionalN int, opts ...ArgumentOption[T]) *Argument[T] {
	return &Argument[T]{
		name:      name,
		desc:      desc,
		minN:      minN,
		optionalN: optionalN,
		opt:       multiArgumentOptions(opts...),
	}
}
