package commander

import (
	"strings"

	"github.com/leep-frog/command/command"
)

// FileNumberInputTransformer transforms input arguments of the format "input.go:123"
// into ["input.go" "123"]. This allows CLIs to transform provided arguments and
// use regular string and int `Argument`s for parsing arguments.
func FileNumberInputTransformer(upToIndex int) *command.InputTransformer {
	return &command.InputTransformer{F: func(o command.Output, d *command.Data, s string) ([]string, error) {
		sl := strings.Split(s, ":")
		if len(sl) <= 2 {
			return sl, nil
		}
		return nil, o.Stderrf("Expected either 1 or 2 parts, got %d\n", len(sl))
	}, UpToIndex: upToIndex}
}
