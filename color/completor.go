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

// TODO: have this accept commandOS and write to stderr with any issues
func ApplyCodes(f *Format, data *command.Data) (*Format, bool) {
	if f == nil {
		f = &Format{}
	}
	codes := data.Values[ArgName].StringList()
	for _, c := range codes {
		f.AddAttribute(c)
	}
	return f, len(codes) != 0
}
