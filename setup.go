package command

var (
	// SetupArg is an argument that points to the filename containing the output of the Setup command.
	// Note: for some reason in windows (for history at least), this has a ton of null characters (\x00)
	// that need to be removed in the CLI itself.
	SetupArg = FileArgument("SETUP_FILE", "file used to run setup for command", HiddenArg[string]())
)
