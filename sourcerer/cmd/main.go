package main

import (
	"github.com/leep-frog/command/sourcerer"
)

func main() {
	goleep := &GoLeep{}
	sourcerer.Source([]sourcerer.CLI{
		sourcerer.SourcererCLI(),
		&AliaserCommand{},
		&Debugger{},
		goleep,
		&UpdateLeepPackageCommand{},
		&UsageCommand{},
		&Eko{},
	}, /*Ugoleep.Aliasers()*/ EkoAliasers())
}
