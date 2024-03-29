package commander

import (
	"bufio"
	"os"

	"github.com/leep-frog/command/command"
)

// Prompt prompts the user for input.
func Prompt(output command.Output, question string) chan string {
	reader := bufio.NewReader(os.Stdin)
	output.Stdoutln(question)
	c := make(chan string)

	go func() {
		for {
			text, err := reader.ReadString('\n')
			if err == nil {
				c <- text
				return
			}
			output.Stderrf("failed to read prompt input (%v); trying again\n", err)
		}
	}()

	return c
}
