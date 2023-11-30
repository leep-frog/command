package commander

import (
	"fmt"

	"github.com/leep-frog/command/command"
)

// IsNotEnoughArgsError returns whether or not the provided error
// is a `NotEnoughArgs` error.
func IsNotEnoughArgsError(err error) bool {
	_, ok := err.(*notEnoughArgs)
	return ok
}

// IsUsageError returns whether or not the provided error
// is a usage-related error.
func IsUsageError(err error) bool {
	return IsNotEnoughArgsError(err) || IsBranchingError(err) || command.IsExtraArgsError(err)
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
