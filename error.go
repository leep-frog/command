package command

import "fmt"

// ExtraArgsErr returns an error for when too many arguments are provided to a command.
func ExtraArgsErr(input *Input) error {
	return input.extraArgsErr()
}

func (i *Input) extraArgsErr() error {
	return &extraArgsErr{i}
}

type extraArgsErr struct {
	input *Input
}

func (eae *extraArgsErr) Error() string {
	return fmt.Sprintf("Unprocessed extra args: %v", eae.input.Remaining())
}

// IsExtraArgs returns whether or not the provided error is an `ExtraArgsErr`.
// TODO: error.go file.
func IsExtraArgsError(err error) bool {
	_, ok := err.(*extraArgsErr)
	return ok
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
	return IsNotEnoughArgsError(err) || IsBranchingError(err) || IsExtraArgsError(err)
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
