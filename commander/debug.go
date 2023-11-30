package commander

import (
	"os"

	"github.com/leep-frog/command/command"
)

const (
	DebugEnvVar = "LEEP_FROG_DEBUG"
)

// DebugMode returns whether or not debug mode is active.
func DebugMode() bool {
	return os.Getenv(DebugEnvVar) != ""
}

func Debugf(o command.Output, s string, i ...interface{}) {
	if DebugMode() {
		o.Stderrf(s, i)
	}
}
