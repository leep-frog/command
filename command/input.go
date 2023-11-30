package command

import (
	"fmt"

	"github.com/leep-frog/command/internal/spycommand"
	"github.com/leep-frog/command/internal/spyinput"
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

// Input is a type that tracks the entire input and how much of the input
// has been parsed. It also takes care of input snapshots (i.e. snapshots for
// shortcuts and caching purposes).
type Input struct {
	// We use `spyinput.SpyInput` to hold all the arguments so we can test against
	// an internally public, but externally hidden type.
	si *spyinput.SpyInput[InputBreaker]
}

func (i *Input) PushBreakers(vs ...InputBreaker) {
	i.si.Breakers = append(i.si.Breakers, vs...)
}

func (i *Input) PopBreakers(n int) {
	i.si.Breakers = i.si.Breakers[:len(i.si.Breakers)-n]
}

func (i *Input) ConvertedArgs() []string {
	var r []string
	for _, a := range i.si.Args {
		r = append(r, a.Value)
	}
	return r
}

func InputRunAtOffset[T any](i *Input, atOffset int, f func(*Input) T) T {
	oldOffset := i.si.Offset
	i.si.Offset = i.si.Offset + atOffset
	defer func() { i.si.Offset = oldOffset }()

	return f(i)
}

func addSnapshots(ia *spycommand.InputArg, is ...spycommand.InputSnapshot) {
	if ia.Snapshots == nil {
		ia.Snapshots = map[spycommand.InputSnapshot]bool{}
	}
	for _, i := range is {
		ia.Snapshots[i] = true
	}
}

// Snapshot takes a snapshot of the remaining input arguments.
func (i *Input) Snapshot() spycommand.InputSnapshot {
	i.si.SnapshotCount++
	for j := i.si.Offset; j < len(i.si.Remaining); j++ {
		addSnapshots(i.get(j), i.si.SnapshotCount)
	}
	return i.si.SnapshotCount
}

// NumSnapshots returns the number of snapshots
func (i *Input) NumSnapshots() int {
	return int(i.si.SnapshotCount)
}

// GetSnapshot retrieves the snapshot.
func (i *Input) GetSnapshot(is spycommand.InputSnapshot) []string {
	var r []string
	for _, arg := range i.si.Args {
		if arg.Snapshots[is] {
			r = append(r, arg.Value)
		}
	}
	return r
}

// FullyProcessed returns whether or not the input has been fully processed.
func (i *Input) FullyProcessed() bool {
	return i.si.Offset >= len(i.si.Remaining)
}

func (i *Input) NumRemaining() int {
	return len(i.si.Remaining) - i.si.Offset
}

// Remaining returns the remaining arguments.
func (i *Input) Remaining() []string {
	r := make([]string, 0, len(i.si.Remaining)-i.si.Offset)
	for _, v := range i.si.Remaining[i.si.Offset:] {
		r = append(r, i.si.Args[v].Value)
	}
	return r
}

// Used returns the input arguments that have already been processed.
func (i *Input) Used() []string {
	r := map[int]bool{}
	for _, v := range i.si.Remaining {
		r[v] = true
	}
	var v []string
	for idx := 0; idx < len(i.si.Args); idx++ {
		if !r[idx] {
			v = append(v, i.si.Args[idx].Value)
		}
	}
	return v
}

// Peek returns the next argument and whether or not there is another argument.
func (i *Input) Peek() (string, bool) {
	return i.PeekAt(0)
}

func (i *Input) get(j int) *spycommand.InputArg {
	return i.si.Args[i.si.Remaining[j]]
}

// PushFront pushes arguments to the front of the remaining input.
func (i *Input) PushFront(sl ...string) {
	i.PushFrontAt(0, sl...)
}

// PushFrontAt pushes arguments starting at a specific spot in the remaining arguments.
func (i *Input) PushFrontAt(atOffset int, sl ...string) {
	// Use bool in place of void
	InputRunAtOffset[bool](i, atOffset, func(i *Input) bool {
		// Update remaining.
		startIdx := len(i.si.Args)
		var snapshots map[spycommand.InputSnapshot]bool
		if len(i.si.Remaining) > 0 && i.si.Offset < len(i.si.Remaining) {
			startIdx = i.si.Remaining[i.si.Offset]

			if len(i.si.Args[startIdx].Snapshots) > 0 {
				snapshots = map[spycommand.InputSnapshot]bool{}
				for s := range i.si.Args[startIdx].Snapshots {
					snapshots[s] = true
				}
			}
		}

		ial := make([]*spycommand.InputArg, len(sl))
		for j := 0; j < len(sl); j++ {
			var sCopy map[spycommand.InputSnapshot]bool
			if snapshots != nil {
				sCopy = map[spycommand.InputSnapshot]bool{}
				for s := range snapshots {
					sCopy[s] = true
				}
			}

			ial[j] = &spycommand.InputArg{
				Value:     sl[j],
				Snapshots: sCopy,
			}
		}
		i.si.Args = append(i.si.Args[:startIdx], append(ial, i.si.Args[startIdx:]...)...)
		// increment all remaining after offset.
		for j := i.si.Offset; j < len(i.si.Remaining); j++ {
			i.si.Remaining[j] += len(sl)
		}
		insert := make([]int, 0, len(sl))
		for j := 0; j < len(sl); j++ {
			insert = append(insert, j+startIdx)
		}
		i.si.Remaining = append(i.si.Remaining[:i.si.Offset], append(insert, i.si.Remaining[i.si.Offset:]...)...)

		// Return arbitrary dummy value
		return false
	})
}

// PeekAt peeks at a specific argument and returns whether or not there are at least that many arguments.
func (i *Input) PeekAt(idx int) (string, bool) {
	if idx < 0 || idx >= len(i.si.Remaining) {
		return "", false
	}
	return i.get(idx).Value, true
}

// Pop removes the next argument from the input and returns if there is at least one more argument.
func (i *Input) Pop(d *Data) (string, bool) {
	return i.PopAt(0, d)
}

func (i *Input) PopAt(offset int, d *Data) (string, bool) {
	sl, ok := i.PopNAt(offset, 1, 0, nil, d)
	if !ok {
		return "", false
	}
	return *sl[0], true
}

// InputBreaker is an interface used to break a list of values returned by `Input.Pop` functions.
type InputBreaker interface {
	// Break returns whether the input processing should stop.
	Break(string, *Data) bool
	// DiscardBreak returns whether the value responsible for breaking the input shoud be popped or not.
	DiscardBreak(string, *Data) bool
}

// PopN pops the next `n` arguments from the input and returns whether or not there are enough arguments left.
func (i *Input) PopN(n, optN int, breakers []InputBreaker, d *Data) ([]*string, bool) {
	return i.PopNAt(0, n, optN, breakers, d)
}

// PopNAt pops the `n` arguments starting at the provided offset.
func (outerInput *Input) PopNAt(atOffset, n, optN int, breakers []InputBreaker, d *Data) ([]*string, bool) {
	type retVal struct {
		ss []*string
		b  bool
	}

	rv := InputRunAtOffset[*retVal](outerInput, atOffset, func(i *Input) *retVal {
		shift := n + optN
		if optN == UnboundedList || shift+i.si.Offset > len(i.si.Remaining) {
			shift = len(i.si.Remaining) - i.si.Offset
		}

		if shift <= 0 {
			return &retVal{nil, n == 0}
		}

		ret := make([]*string, 0, shift)
		idx := 0
		var broken, discardBreak bool
		for ; idx < shift; idx++ {
			for _, b := range append(breakers, i.si.Breakers...) {
				s := i.get(idx + i.si.Offset).Value
				if b.Break(s, d) {
					broken = true
					discardBreak = b.DiscardBreak(s, d)
					goto LOOP_END
				}
			}
			ret = append(ret, &i.get(idx+i.si.Offset).Value)
		}
	LOOP_END:
		i.si.Remaining = append(i.si.Remaining[:i.si.Offset], i.si.Remaining[i.si.Offset+idx:]...)

		if broken && discardBreak {
			i.PopAt(atOffset, d)
		}
		return &retVal{ret, len(ret) >= n}
	})
	return rv.ss, rv.b
}

// ParseExecuteArgs converts a list of strings into an Input struct.
func ParseExecuteArgs(strArgs []string) *Input {
	r := make([]int, len(strArgs))
	args := make([]*spycommand.InputArg, len(strArgs))
	for i := range strArgs {
		r[i] = i
		args[i] = &spycommand.InputArg{
			Value: strArgs[i],
		}
	}
	return &Input{
		&spyinput.SpyInput[InputBreaker]{
			Args:      args,
			Remaining: r,
		},
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
	// endState w *wordsr the string to append if the current state is the last state
	endState(w *words)
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

func (*wordState) endState(w *words) {}

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

func (*backslashState) endState(w *words) {
	w.addChar('\\')
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

func (*whitespaceState) endState(w *words) {}

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

func (*quoteState) endState(w *words) {}

// ParseCompLine parses the COMP_LINE value provided by the shell
func ParseCompLine(compLine string, passthroughArgs ...string) *Input {
	w := &words{}
	state := parserState(&whitespaceState{})
	for _, c := range compLine {
		state = state.parseChar(c, w)
	}
	state.endState(w)

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
	i.si.Delimiter = delimiter
	return i
}

// InputTransformer checks the next input argument (as a string), runs `F` on that
// argument, and inserts the values returned from `F` in its place.
// See `FileNumberInputTransformer` for a useful example.
//
// Note: `InputTransformer` should only be used
// when the number of arguments or the argument type is expected to change.
// If the number of arguments and type will remain the same, use an `Argument`
// with a `Transformer` option.
type InputTransformer struct {
	// F is the function that will be run on each element in Input.
	F func(Output, *Data, string) ([]string, error)
	// UpToIndex is the input argument index that F will be run through.
	// This is zero-indexed so default behavior (UpToIndex: 0) will run on the
	// first argument. If UpToIndex is less than zero, then this will run
	// on all remaining arguments.
	UpToIndex int // TODO: Test this
}

func (it *InputTransformer) Transform(input *Input, output Output, data *Data, complete bool) error {
	k := 0
	if complete {
		// Don't check the last argument (i.e. the completion argument)
		k = -1
	}

	for j := input.si.Offset; j < input.NumRemaining()+k && (it.UpToIndex < 0 || j <= input.si.Offset+it.UpToIndex); {
		sl, err := it.F(output, data, input.get(j).Value)
		if err != nil {
			return err
		}

		if len(sl) == 0 {
			return fmt.Errorf("shortcut has empty value")
		}
		// TODO: Inserted args should be added to the input snapshot
		end := len(sl) - 1
		input.get(j).Value = sl[end]
		input.PushFrontAt(j, sl[:end]...)
		j += len(sl)
	}
	return nil
}

func (it *InputTransformer) Execute(i *Input, o Output, data *Data, eData *ExecuteData) error {
	return it.Transform(i, o, data, false)
}

func (it *InputTransformer) Complete(input *Input, data *Data) (*Completion, error) {
	return nil, it.Transform(input, NewIgnoreAllOutput(), data, true)
}

func (it *InputTransformer) Usage(*Input, *Data, *Usage) error { return nil }

// ExtraArgsErr returns an error for when too many arguments are provided to a command.
func ExtraArgsErr(input *Input) error {
	return input.extraArgsErr()
}

func (i *Input) extraArgsErr() error {
	return &extraArgsErr{i}
}

type extraArgsErr struct {
	input *Input
}

func (eae *extraArgsErr) Error() string {
	return fmt.Sprintf("Unprocessed extra args: %v", eae.input.Remaining())
}

// IsExtraArgs returns whether or not the provided error is an `ExtraArgsErr`.
// TODO: error.go file.
func IsExtraArgsError(err error) bool {
	_, ok := err.(*extraArgsErr)
	return ok
}
