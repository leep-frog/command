package command

import (
	"bufio"
	"os"
)

func Prompt(output Output, question string) chan string {
	reader := bufio.NewReader(os.Stdin)
	output.Stdout(question)
	c := make(chan string)

	go func() {
		for {
			text, err := reader.ReadString('\n')
			if err == nil {
				c <- text
				return
			}
			output.Stderrf("failed to read prompt input (%v); trying again", err)
		}
	}()

	return c
}
