package command

type Input struct {
	args      []string
	pos       int
	delimiter *rune
}

const (
	UnboundedList = -1
)

func (i *Input) FullyProcessed() bool {
	return i.pos >= len(i.args)
}

func (i *Input) Remaining() []string {
	return i.args[i.pos:]
}

func (i *Input) Peek() (string, bool) {
	if i.FullyProcessed() {
		return "", false
	}
	return i.args[i.pos], true
}

func (i *Input) Pop() (string, bool) {
	sl, ok := i.PopN(1, 0)
	if !ok {
		return "", false
	}
	return sl[0], true
}

func (i *Input) PopN(n, optN int) ([]string, bool) {
	remaining := len(i.args) - i.pos
	shift := n + optN
	if optN == UnboundedList || shift+i.pos > len(i.args) {
		shift = remaining
	}
	ret := i.args[i.pos:(i.pos + shift)]
	i.pos += shift
	return ret, len(ret) >= n
}

var (
	// Default is string list
	quotationChars = map[rune]bool{
		'"':  true,
		'\'': true,
	}
)

// TODO: this should just take in a "string" argument so there is
// absolutely no parsing required by the CLI side of it.
// TODO: make this a state machine via interfaces.
// ParseArgs parses raw, unsplit text.
func ParseArgs(unparsedArgs []string) *Input {
	if len(unparsedArgs) == 0 {
		return &Input{}
	}

	// Ignore if the last charater is just a quote
	var delimiterOverride *rune
	lastArg := unparsedArgs[len(unparsedArgs)-1]
	if len(lastArg) == 1 && quotationChars[rune(lastArg[0])] {
		r := rune(lastArg[0])
		delimiterOverride = &r
		unparsedArgs[len(unparsedArgs)-1] = ""
	}

	// Words might be combined so parsed args will be less than or equal to unparsedArgs length.
	parsedArgs := make([]string, 0, len(unparsedArgs))

	// Max length of the string can be all characters (including spaces).
	totalLen := len(unparsedArgs)
	for _, arg := range unparsedArgs {
		totalLen += len(arg)
	}
	currentString := make([]rune, 0, totalLen)

	var currentQuote *rune
	// Note: "one"two is equivalent to (onetwo) as opposed to (one two).
	for i, arg := range unparsedArgs {
		for j := 0; j < len(arg); j++ {
			char := rune(arg[j])

			if currentQuote != nil {
				if char == *currentQuote {
					// Break out of quote state.
					currentQuote = nil
				} else {
					// Still in quote, just add character.
					currentString = append(currentString, char)
				}
			} else if quotationChars[char] {
				// Start of a new quoted section.
				currentQuote = &char
			} else if char == '\\' && j < len(arg)-1 && rune(arg[j+1]) == ' ' {
				// "\ " in a string.
				currentString = append(currentString, ' ')
				j++
			} else {
				// Regular word.
				currentString = append(currentString, char)
			}
		}

		if currentQuote != nil && i != len(unparsedArgs)-1 {
			// If we're in a quoted section and still have more words to parse.
			currentString = append(currentString, ' ')
		} else if len(arg) > 0 && rune(arg[len(arg)-1]) == '\\' {
			// If last character of argument is a backslash, then it's just a space.
			currentString[len(currentString)-1] = ' '
		} else {
			// Start the next word.
			parsedArgs = append(parsedArgs, string(currentString))
			currentString = currentString[0:0]
		}
	}

	var delimiter *rune
	if delimiterOverride != nil {
		delimiter = delimiterOverride
	} else if currentQuote != nil {
		delimiter = currentQuote
	}

	return &Input{
		args:      parsedArgs,
		delimiter: delimiter,
	}
}
