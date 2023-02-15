package command

import "fmt"

// ExtraArgsErr returns an error for when too many arguments are provided to a command.
func ExtraArgsErr(input *Input) error {
	return &extraArgsErr{input}
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
