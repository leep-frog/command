package command

import "os"

const (
	DebugEnvVar = "LEEP_FROG_DEBUG"
)

// DebugMode returns whether or not debug mode is active.
func DebugMode() bool {
	return os.Getenv(DebugEnvVar) != ""
}

func Debugf(o Output, s string, i ...interface{}) {
	if DebugMode() {
		o.Stderrf(s, i)
	}
}
