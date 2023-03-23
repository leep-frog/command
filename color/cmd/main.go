package main

import (
	"fmt"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color"
)

func main() {
	o := command.NewOutput()
	o.Stdoutln("Output 1")
	o.Stderrln("Errput 1")
	fmt.Println("Fmt 1")

	color.Color(color.Blue).Apply()

	o.Stdoutln("Output 2")
	o.Stderrln("Errput 2")
	fmt.Println("Fmt 2")

	color.MultiFormat(color.Bold(), color.Color(color.Green)).Apply()

	o.Stdoutln("Output 3")
	o.Stderrln("Errput 3")
	fmt.Println("Fmt 3")

	o.Close()
}
