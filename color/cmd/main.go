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

	color.Text(color.Blue).Apply()

	o.Stdoutln("Output 2")
	o.Stderrln("Errput 2")
	fmt.Println("Fmt 2")

	color.MultiFormat(color.Bold(), color.Text(color.Green)).Apply()

	o.Stdoutln("Output 3")
	o.Stderrln("Errput 3")
	fmt.Println("Fmt 3")

	color.Init().Apply()

	o.Stdoutln("Output 4")
	o.Stderrln("Errput 4")
	fmt.Println("Fmt 4")

	o.Close()
}
