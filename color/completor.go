package color

import (
	"github.com/leep-frog/command"
)

type fetcher struct{}

// TODO: add existing stuff in here so don't display already present format.
func (f *fetcher) Fetch(value *command.Value, data *command.Data) *command.Completion {
	return &command.Completion{
		Suggestions: Attributes(),
	}
}

func Completor() *command.Completor {
	return &command.Completor{
		Distinct:          true,
		SuggestionFetcher: &fetcher{},
	}
}

var (
	ArgName = "format"
	Arg     = command.StringListNode(ArgName, 1, command.UnboundedList, &command.ArgOpt{Completor: Completor()})
)

func ApplyCodes(f *Format, output command.Output, data *command.Data) (*Format, error) {
	if f == nil {
		f = &Format{}
	}
	codes := data.Values[ArgName].StringList()
	for _, c := range codes {
		if err := f.AddAttribute(c); err != nil {
			return nil, output.Err(err)
		}
	}
	return f, nil
}
