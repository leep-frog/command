package command

type unprocessedArg struct {
	originalIdx int
}

type Input struct {
	args      []string
	remaining []int
	delimiter *rune
	offset    int
}

const (
	UnboundedList = -1
)

func (i *Input) FullyProcessed() bool {
	return len(i.remaining) <= i.offset
}

func (i *Input) Remaining() []string {
	r := make([]string, 0, len(i.remaining))
	for _, v := range i.remaining {
		r = append(r, i.args[v])
	}
	return r
}

func (i *Input) Peek() (string, bool) {
	return i.PeekAt(0)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (i *Input) CheckAliases(upTo int, ac AliasCLI, name string, complete bool) {
	k := 0
	if complete {
		k = -1
	}

	// TODO: test this works with offset (specifically push front and replace near end of function).
	for j := i.offset; j < len(i.remaining)+k && j < i.offset+upTo; {
		// TODO: make func (input) Get(j) { return i.args[i.remaining[j]] }
		// A couple silly errors caused by forgetting to do nested lookup in places.
		sl, ok := getAlias(ac, name, i.args[i.remaining[j]])
		if !ok {
			j++
			continue
		}

		// TODO: Verify this works for empty aliases
		end := len(sl) - 1
		// TODO: do these functions need to be public at all anymore?
		i.args[i.remaining[j]] = sl[end]
		i.PushFrontAt(j, sl[:end]...)
		j += len(sl)
	}
}

func (i *Input) PushFront(sl ...string) {
	i.PushFrontAt(0, sl...)
}

func (i *Input) PushFrontAt(idx int, sl ...string) {
	tmpOffset := i.offset + idx
	// Update remaining.
	startIdx := len(i.args)
	if len(i.remaining) > 0 {
		if tmpOffset < len(i.remaining) {
			startIdx = i.remaining[tmpOffset]
		}
	}
	i.args = append(i.args[:startIdx], append(sl, i.args[startIdx:]...)...)
	// increment all remaining after offset.
	for j := tmpOffset; j < len(i.remaining); j++ {
		i.remaining[j] += len(sl)
	}
	insert := make([]int, 0, len(sl))
	for j := 0; j < len(sl); j++ {
		insert = append(insert, j+startIdx)
	}
	i.remaining = append(i.remaining[:tmpOffset], append(insert, i.remaining[tmpOffset:]...)...)
}

func (i *Input) PeekAt(idx int) (string, bool) {
	if idx < 0 {
		return "", false
	}
	if idx >= len(i.remaining) {
		return "", false
	}
	return i.args[i.remaining[idx]], true
}

func (i *Input) Pop() (string, bool) {
	sl, ok := i.PopN(1, 0)
	if !ok {
		return "", false
	}
	return *sl[0], true
}

func (i *Input) PopN(n, optN int) ([]*string, bool) {
	shift := n + optN
	if optN == UnboundedList || shift+i.offset > len(i.remaining) {
		shift = len(i.remaining) - i.offset
	}

	if shift <= 0 {
		return nil, n == 0
	}

	ret := make([]*string, 0, shift)
	for idx := 0; idx < shift; idx++ {
		ret = append(ret, &i.args[i.remaining[idx+i.offset]])
	}
	i.remaining = append(i.remaining[:i.offset], i.remaining[i.offset+shift:]...)
	return ret, len(ret) >= n
}

var (
	// Default is string list
	quotationChars = map[rune]bool{
		'"':  true,
		'\'': true,
	}
)

func ParseExecuteArgs(args []string) *Input {
	r := make([]int, 0, len(args))
	for i := range args {
		r = append(r, i)
	}
	return &Input{
		args:      args,
		remaining: r,
	}
}

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

	return NewInput(parsedArgs, delimiter)
}

func NewInput(args []string, delimiter *rune) *Input {
	r := make([]int, len(args))
	for i := range args {
		r[i] = i
	}

	return &Input{
		args:      args,
		delimiter: delimiter,
		remaining: r,
	}
}
