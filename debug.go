package command

import "os"

const (
	DebugEnvVar = "LEEP_FROG_DEBUG"
)

// DebugMode returns whether or not debug mode is active.
// TODO: have debug mode point to directory or file
//       and all output can be written there.
func DebugMode() bool {
	return os.Getenv(DebugEnvVar) != ""
}

func Debugf(o Output, s string, i ...interface{}) {
	if DebugMode() {
		o.Stdoutf(s, i)
	}
}
