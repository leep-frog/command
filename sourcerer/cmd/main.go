package main

import "github.com/leep-frog/command/sourcerer"

func main() {
	goleep := &GoLeep{}
	sourcerer.Source([]sourcerer.CLI{
		&AliaserCommand{},
		&Debugger{},
		goleep,
		&UpdateLeepPackageCommand{},
		&UsageCommand{},
	}, goleep.Aliasers())
}
