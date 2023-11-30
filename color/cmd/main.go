package main

import (
	"fmt"

	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/command"
)

func main() {
	o := command.NewOutput()
	o.Stdoutln("Output 1")
	o.Stderrln("Errput 1")
	fmt.Println("Fmt 1")

	color.Text(color.Blue).Apply(nil)

	o.Stdoutln("Output 2")
	o.Stderrln("Errput 2")
	fmt.Println("Fmt 2")

	color.MultiFormat(color.Bold(), color.Text(color.Green)).Apply(nil)

	o.Stdoutln("Output 3")
	o.Stderrln("Errput 3")
	fmt.Println("Fmt 3")

	color.Init().Apply(nil)

	o.Stdoutln("Output 4")
	o.Stderrln("Errput 4")
	fmt.Println("Fmt 4")

	o.Close()
}
