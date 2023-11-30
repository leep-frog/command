package commander

import (
	"fmt"
	"os"
	"strings"

	"github.com/leep-frog/command/commondels"
)

var (
	// SetupArg is an argument that points to the filename containing the output of the Setup command.
	// Note: for some reason in windows (for history at least), this has a ton of null characters (\x00)
	// that need to be removed in the CLI itself.
	SetupArg = FileArgument("SETUP_FILE", "file used to run setup for command", HiddenArg[string]())
)

// SetupOutputFile returns the name of the setup file for the command.
func SetupOutputFile(d *commondels.Data) string {
	return d.String(SetupArg.Name())
}

// SetupOutputString returns the file contents, as a string, of the setup file for the command.
func SetupOutputString(d *commondels.Data) (string, error) {
	b, err := os.ReadFile(SetupOutputFile(d))
	if err != nil {
		return "", fmt.Errorf("failed to read setup file (%s): %v", SetupOutputFile(d), err)
	}
	return strings.TrimSpace(string(b)), nil
}

// SetupOutputString returns the file contents, as a string slice, of the setup file for the command.
func SetupOutputContents(d *commondels.Data) ([]string, error) {
	s, err := SetupOutputString(d)
	return strings.Split(s, "\n"), err
}
