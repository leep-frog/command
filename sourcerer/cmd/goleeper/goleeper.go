package main

import (
	"os"

	"github.com/leep-frog/command/sourcerer"
)

func main() {
	os.Exit(sourcerer.Source(&sourcerer.GoLeep{}))
}
