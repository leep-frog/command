package color

import (
	"github.com/leep-frog/command"
)

type fetcher struct{}

func (f *fetcher) Fetch(value string, data *command.Data) (*command.Completion, error) {
	return &command.Completion{
		Suggestions: Attributes(),
	}, nil
}

func Completor() *command.Completor[string] {
	return &command.Completor[string]{
		Distinct: true,
		Fetcher:  &fetcher{},
	}
}

var (
	ArgName = "format"
	Arg     = command.ListArg[string](ArgName, "color", 1, command.UnboundedList, command.CompletorList(Completor()))
)

func ApplyCodes(f *Format, output command.Output, data *command.Data) (*Format, error) {
	if f == nil {
		f = &Format{}
	}
	codes := data.StringList(ArgName)
	for _, c := range codes {
		if err := f.AddAttribute(c); err != nil {
			return nil, output.Err(err)
		}
	}
	return f, nil
}
