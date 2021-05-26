package command

import (
	"bufio"
	"os"
	"sort"
)

func PromptUser(output Output, question string, answerActions map[string]func() error) error {
	p := &Prompt{
		Question:      question,
		AnswerActions: answerActions,
	}
	return p.Prompt(output)
}

type Prompt struct {
	Question      string
	AnswerActions map[string]func() error
}

func (p *Prompt) Prompt(output Output) error {
	reader := bufio.NewReader(os.Stdin)
	output.Stdout(p.Question)

	for {
		output.Stdout(": ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		if f, ok := p.AnswerActions[text]; ok {
			return f()
		}
		var keys []string
		for k := range p.AnswerActions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		output.Stdout("Response must be one of %v", keys)
	}
}
