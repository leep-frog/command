// Package colorreset provides a CLI that clears all color formatting
// done by the github.com/leep-frog/command/color package.
package colorreset

import (
	"bufio"
	"io"
	"os"
	"regexp"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
	"github.com/leep-frog/command/sourcerer"
)

// CLI returns a sourcerer.CLI that clears all formatting added by the github.com/leep-frog/command/color package.
func CLI(name string) sourcerer.CLI {
	return &resetter{name, os.Stdin}
}

var (
	resetRegex = regexp.MustCompile("\033\\[([0-9]+;?)*m")
)

type resetter struct {
	name   string
	reader io.Reader
}

func (r *resetter) Name() string    { return r.name }
func (r *resetter) Setup() []string { return nil }
func (r *resetter) Changed() bool   { return false }
func (r *resetter) Node() command.Node {
	return commander.SerialNodes(
		&commander.ExecutorProcessor{func(o command.Output, d *command.Data) error {
			scanner := bufio.NewScanner(r.reader)
			for scanner.Scan() {
				o.Stdoutln(resetRegex.ReplaceAllString(scanner.Text(), ""))
			}
			return scanner.Err()
		}},
	)
}
