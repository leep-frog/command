package command

import (
	"fmt"
	"strings"
)

const (
	// UnboundedList is used to indicate that an argument list should allow an unbounded amount of arguments.
	UnboundedList = -1
)

var (
	quotationChars = map[rune]bool{
		'"':  true,
		'\'': true,
	}

	wordBreakChars = map[rune]bool{
		' ': true,
	}
)

type inputSnapshot int

// Input is a type that tracks the entire input and how much of the input
// has been parsed. It also takes care of input snapshots (i.e. snapshots for
// shortcuts and caching purposes).
type Input struct {
	args          []*inputArg
	remaining     []int
	delimiter     *rune
	offset        int
	snapshotCount inputSnapshot
}

type inputArg struct {
	value     string
	snapshots map[inputSnapshot]bool
}

func (ia *inputArg) addSnapshots(is ...inputSnapshot) {
	if ia.snapshots == nil {
		ia.snapshots = map[inputSnapshot]bool{}
	}
	for _, i := range is {
		ia.snapshots[i] = true
	}
}

// Snapshot takes a snapshot of the remaining input arguments.
func (i *Input) Snapshot() inputSnapshot {
	i.snapshotCount++
	for j := i.offset; j < len(i.remaining); j++ {
		i.get(j).addSnapshots(i.snapshotCount)
	}
	return i.snapshotCount
}

// GetSnapshot retrieves the snapshot.
func (i *Input) GetSnapshot(is inputSnapshot) []string {
	var r []string
	for _, arg := range i.args {
		if arg.snapshots[is] {
			r = append(r, arg.value)
		}
	}
	return r
}

// FullyProcessed returns whether or not the input has been fully processed.
func (i *Input) FullyProcessed() bool {
	return i.offset >= len(i.remaining)
}

func (i *Input) NumRemaining() int {
	return len(i.remaining)
}

// Remaining returns the remaining arguments.
func (i *Input) Remaining() []string {
	r := make([]string, 0, len(i.remaining))
	for _, v := range i.remaining {
		r = append(r, i.args[v].value)
	}
	return r
}

// Used returns the input arguments that have already been processed.
func (i *Input) Used() []string {
	r := map[int]bool{}
	for _, v := range i.remaining {
		r[v] = true
	}
	var v []string
	for idx := 0; idx < len(i.args); idx++ {
		if !r[idx] {
			v = append(v, i.args[idx].value)
		}
	}
	return v
}

// Peek returns the next argument and whether or not there is another argument.
func (i *Input) Peek() (string, bool) {
	return i.PeekAt(0)
}

func (i *Input) get(j int) *inputArg {
	return i.args[i.remaining[j]]
}

// PushFront pushes arguments to the front of the remaining input.
func (i *Input) PushFront(sl ...string) {
	i.PushFrontAt(0, sl...)
}

// PushFrontAt pushes arguments starting at a specific spot in the remaining arguments.
func (i *Input) PushFrontAt(idx int, sl ...string) {
	tmpOffset := i.offset + idx
	// Update remaining.
	startIdx := len(i.args)
	var snapshots map[inputSnapshot]bool
	if len(i.remaining) > 0 && tmpOffset < len(i.remaining) {
		startIdx = i.remaining[tmpOffset]

		if len(i.args[startIdx].snapshots) > 0 {
			snapshots = map[inputSnapshot]bool{}
			for s := range i.args[startIdx].snapshots {
				snapshots[s] = true
			}
		}
	}

	ial := make([]*inputArg, len(sl))
	for j := 0; j < len(sl); j++ {
		var sCopy map[inputSnapshot]bool
		if snapshots != nil {
			sCopy = map[inputSnapshot]bool{}
			for s := range snapshots {
				sCopy[s] = true
			}
		}

		ial[j] = &inputArg{
			value:     sl[j],
			snapshots: sCopy,
		}
	}
	i.args = append(i.args[:startIdx], append(ial, i.args[startIdx:]...)...)
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

// PeekAt peeks at a specific argument and returns whether or not there are at least that many arguments.
func (i *Input) PeekAt(idx int) (string, bool) {
	if idx < 0 || idx >= len(i.remaining) {
		return "", false
	}
	return i.get(idx).value, true
}

// Pop removes the next argument from the input and returns if there is at least one more argument.
func (i *Input) Pop() (string, bool) {
	sl, ok := i.PopN(1, 0, nil)
	if !ok {
		return "", false
	}
	return *sl[0], true
}

// PopN pops the next `n` arguments from the input and returns whether or not there are enough arguments left.
func (i *Input) PopN(n, optN int, breaker *ListBreaker) ([]*string, bool) {
	shift := n + optN
	if optN == UnboundedList || shift+i.offset > len(i.remaining) {
		shift = len(i.remaining) - i.offset
	}

	if shift <= 0 {
		return nil, n == 0
	}

	ret := make([]*string, 0, shift)
	idx := 0
	var broken bool
	var validators []*ValidatorOption[string]
	if breaker != nil {
		validators = breaker.Validators()
	}
	for ; idx < shift; idx++ {
		for _, validator := range validators {
			if err := validator.Validate(i.get(idx + i.offset).value); err != nil {
				//if err := validator.validate(StringValue(i.get(idx + i.offset).value)); err != nil {
				broken = true
				break
			}
		}
		if broken {
			break
		}
		ret = append(ret, &i.get(idx+i.offset).value)
	}
	i.remaining = append(i.remaining[:i.offset], i.remaining[i.offset+idx:]...)

	if broken && breaker.DiscardBreak() {
		i.Pop()
	}
	return ret, len(ret) >= n
}

// ParseExecuteArgs converts a list of strings into an Input struct.
func ParseExecuteArgs(strArgs []string) *Input {
	r := make([]int, len(strArgs))
	args := make([]*inputArg, len(strArgs))
	for i := range strArgs {
		r[i] = i
		args[i] = &inputArg{
			value: strArgs[i],
		}
	}
	return &Input{
		args:      args,
		remaining: r,
	}
}

type words struct {
	inWord      bool
	currentWord []rune
	words       []string
}

func (w *words) endWord() {
	w.words = append(w.words, string(w.currentWord))
	w.currentWord = nil
	w.inWord = false
}

func (w *words) startWord() {
	w.inWord = true
}

func (w *words) addChar(c rune) {
	w.currentWord = append(w.currentWord, c)
}

type parserState interface {
	// parseChar should return the next parserState
	parseChar(curChar rune, w *words) parserState
	// delimiter returns the delimiter for the quote set.
	delimiter() *rune
}

type wordState struct{}

func (ws *wordState) parseChar(curChar rune, w *words) parserState {
	if wordBreakChars[curChar] {
		w.endWord()
		return &whitespaceState{}
	}

	if quotationChars[curChar] {
		return &quoteState{curChar}
	}
	if curChar == '\\' {
		return &backslashState{ws}
	}
	w.addChar(curChar)
	return ws
}

func (*wordState) delimiter() *rune {
	return nil
}

type backslashState struct {
	nextState parserState
}

func (bss *backslashState) parseChar(curChar rune, w *words) parserState {
	if curChar != ' ' {
		w.addChar('\\')
	}
	w.addChar(curChar)
	return bss.nextState
}

func (*backslashState) delimiter() *rune {
	return nil
}

type whitespaceState struct{}

func (wss *whitespaceState) parseChar(curChar rune, w *words) parserState {
	if wordBreakChars[curChar] {
		// Don't end word because we're already between words.
		return wss
	}

	w.startWord()
	if quotationChars[curChar] {
		return &quoteState{curChar}
	}
	if curChar == '\\' {
		return &backslashState{&wordState{}}
	}
	w.addChar(curChar)
	return &wordState{}
}

func (*whitespaceState) delimiter() *rune {
	return nil
}

type quoteState struct {
	quoteChar rune
}

func (qs *quoteState) parseChar(curChar rune, w *words) parserState {
	if curChar == qs.quoteChar {
		return &wordState{}
	}
	w.addChar(curChar)
	return qs
}

func (qs *quoteState) delimiter() *rune {
	return &qs.quoteChar
}

// TODO: Should this belong to the os-type implementer
func ParseCompLine(compLine string, passthroughArgs []string) *Input {
	w := &words{}
	state := parserState(&whitespaceState{})
	for _, c := range compLine {
		state = state.parseChar(c, w)
	}

	var args []string
	if w.inWord {
		w.endWord()
		args = w.words
	} else {
		// Needed for autocomplete.
		args = append(w.words, "")
	}

	if len(args) == 1 {
		args = append(args, "")
	}

	// The first argument is the command so we can ignore that.
	return NewInput(append(passthroughArgs, args[1:]...), state.delimiter())
}

// NewInput creates a new `Input` object from a set of args and quote delimiter.
func NewInput(args []string, delimiter *rune) *Input {
	i := ParseExecuteArgs(args)
	i.delimiter = delimiter
	return i
}

func snapshotsMap(iss ...inputSnapshot) map[inputSnapshot]bool {
	if len(iss) == 0 {
		return nil
	}
	m := map[inputSnapshot]bool{}
	for _, is := range iss {
		m[is] = true
	}
	return m
}

// InputTransformer checks the next input argument (as a string), runs `F` on that
// argument, and inserts the values returned from `F` in its place.
// See `FileNumberInputTransformer` for a useful example.
//
// Note: `InputTransformer` should only be used
// when the number of arguments or the argument type is expected to change.
// If the number of arguments and type will remain the same, use an `ArgNode`
// with a `Transformer` option.
type InputTransformer struct {
	// F is the function that will be run on each element in Input.
	F func(Output, *Data, string) ([]string, error)
	// UpToIndex is the input argument index that F will be run through.
	// This is zero-indexed so default behavior (UpToIndex: 0) will run on the
	// first argument.
	UpToIndex int
}

// FileNumberInputTransformer transforms input arguments of the format "input.go:123"
// into ["input.go" "123"]. This allows CLIs to transform provided arguments and
// use regular string and int `ArgNode`s for parsing arguments.
func FileNumberInputTransformer(upToIndex int) *InputTransformer {
	return &InputTransformer{F: func(o Output, d *Data, s string) ([]string, error) {
		sl := strings.Split(s, ":")
		if len(sl) <= 2 {
			return sl, nil
		}
		return nil, o.Stderrf("Expected either 1 or 2 parts, got %d\n", len(sl))
	}, UpToIndex: upToIndex}
}

func shortcutInputTransformer(sc ShortcutCLI, name string, upToIndex int) *InputTransformer {
	return &InputTransformer{F: func(o Output, d *Data, s string) ([]string, error) {
		sl, ok := getShortcut(sc, name, s)
		if !ok {
			return []string{s}, nil
		}
		return sl, nil
	}, UpToIndex: upToIndex}
}

func (it *InputTransformer) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
	return it.Transform(i, o, data, false)
}

func (it *InputTransformer) Transform(input *Input, output Output, data *Data, complete bool) error {
	k := 0
	if complete {
		// Don't check the last argument (i.e. the completion argument)
		k = -1
	}

	for j := input.offset; j < input.NumRemaining()+k && j <= input.offset+it.UpToIndex; {
		sl, err := it.F(output, data, input.get(j).value)
		if err != nil {
			return err
		}

		if len(sl) == 0 {
			return fmt.Errorf("shortcut has empty value")
		}
		// TODO: Inserted args should be added to the input snapshot
		end := len(sl) - 1
		input.get(j).value = sl[end]
		input.PushFrontAt(j, sl[:end]...)
		j += len(sl)
	}
	return nil
}

func (it *InputTransformer) Complete(input *Input, data *Data) (*Completion, error) {
	return nil, it.Transform(input, NewIgnoreAllOutput(), data, true)
}

func (it *InputTransformer) Usage(*Usage) {}
