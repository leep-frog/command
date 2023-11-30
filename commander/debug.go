package commander

import (
	"os"

	"github.com/leep-frog/command/commondels"
)

const (
	DebugEnvVar = "LEEP_FROG_DEBUG"
)

// DebugMode returns whether or not debug mode is active.
func DebugMode() bool {
	return os.Getenv(DebugEnvVar) != ""
}

func Debugf(o commondels.Output, s string, i ...interface{}) {
	if DebugMode() {
		o.Stderrf(s, i)
	}
}
