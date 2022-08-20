package color

import (
	"github.com/leep-frog/command"
)

func Completer() command.Completer[string] {
	return command.SimpleDistinctCompleter[string](Attributes()...)
}

var (
	ArgName = "format"
	Arg     = command.ListArg[string](ArgName, "color", 1, command.UnboundedList, command.CompleterList(Completer()))
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
