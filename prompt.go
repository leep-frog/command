package command

import (
	"bufio"
	"os"
)

type Prompt struct {
	Question string
	Chan     chan string
}

func (p *Prompt) Prompt(output Output) {
	reader := bufio.NewReader(os.Stdin)
	output.Stdout(p.Question)

	go func() {
		for {
			text, err := reader.ReadString('\n')
			if err == nil {
				p.Chan <- text
				return
			}
			output.Stderr("failed to read prompt input (%v); trying again", err)
			continue
		}
	}()
}
