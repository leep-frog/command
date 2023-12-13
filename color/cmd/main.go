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

	o.Stdout(color.OutputCode(color.Blue))

	o.Stdoutln("Output 2")
	o.Stderrln("Errput 2")
	fmt.Println("Fmt 2")

	o.Stdout(color.OutputCode(color.MultiFormat(color.Bold, color.Green)))

	o.Stdoutln("Output 3")
	o.Stderrln("Errput 3")
	fmt.Println("Fmt 3")

	o.Stdout(color.OutputCode(color.Reset))

	o.Stdoutln("Output 4")
	o.Stderrln("Errput 4")
	fmt.Println("Fmt 4")

	o.Close()
}
