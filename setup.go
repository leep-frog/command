package command

var (
	// SetupArg is an argument that points to the filename containing the output of the Setup command.
	SetupArg = FileArgument("SETUP_FILE", "file used to run setup for command", HiddenArg[string]())
)
