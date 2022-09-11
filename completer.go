package command

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	// Used for testing.
	filepathAbs = filepath.Abs
)

const (
	suffixChar = "_"
)

// SimpleCompleter returns a completer that suggests the provided strings for command autocompletion.
func SimpleCompleter[T any](s ...string) Completer[T] {
	return AsCompleter[T](
		&Completion{
			Suggestions: s,
		},
	)
}

// SimpleDistinctCompleter returns a completer that distinctly suggests the provided strings for command autocompletion.
func SimpleDistinctCompleter[T any](s ...string) Completer[T] {
	return AsCompleter[T](
		&Completion{
			Distinct:    true,
			Suggestions: s,
		},
	)
}

// CompleterList changes a single arg completer (`Completer[T]`) into a list arg completer (`Completer[[]T]`).
func CompleterList[T any](c Completer[T]) Completer[[]T] {
	return &simpleCompleter[[]T]{
		f: func(ts []T, d *Data) (*Completion, error) {
			var t T
			if len(ts) > 0 {
				t = ts[len(ts)-1]
			}
			return c.Complete(t, d)
		},
	}
}

// CompleterFromFunc returns a `Completer` object from the provided function.
func CompleterFromFunc[T any](f func(T, *Data) (*Completion, error)) Completer[T] {
	return &simpleCompleter[T]{f}
}

type simpleCompleter[T any] struct {
	f func(T, *Data) (*Completion, error)
}

func (sc *simpleCompleter[T]) Complete(t T, d *Data) (*Completion, error) {
	return sc.f(t, d)
}

func (sc *simpleCompleter[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.completer = sc
}

// Completer is an autocompletion object that can be used as an `ArgOpt`.
type Completer[T any] interface {
	Complete(T, *Data) (*Completion, error)
	modifyArgOpt(*argOpt[T])
}

// Completion is the object constructed by a completer.
type Completion struct {
	// Suggestions is the set of autocomplete suggestions.
	Suggestions []string
	// IgnoreFilter indicates whether prefixes that don't match should be filtered out or not.
	IgnoreFilter bool
	// DontComplete indicates whether or not we should fill in partial completions.
	// This is achieved by adding a " " suggestion.
	DontComplete bool
	// CaseInsensitiveSort returns whether or not we should sort irrespective of case.
	CaseInsensitiveSort bool
	// CaseInsensitve is whether or not case should be considered when filtering out suggestions.
	CaseInsensitive bool
	// Distinct is whether or not we should return only distinct suggestions (specifically to prevent duplicates in list args).
	Distinct bool
}

func (c *Completion) Clone() *Completion {
	return &Completion{
		c.Suggestions,
		c.IgnoreFilter,
		c.DontComplete,
		c.CaseInsensitiveSort,
		c.CaseInsensitive,
		c.Distinct,
	}
}

// CompleterWithOpts sets the relevant options in the `Completion` object
// returned by the `Completer`.
func CompleterWithOpts[T any](cr Completer[T], cn *Completion) Completer[T] {
	return &cmplWithOpts[T]{cr, cn}
}

type cmplWithOpts[T any] struct {
	cr Completer[T]
	cn *Completion
}

func (cwo *cmplWithOpts[T]) Complete(t T, d *Data) (*Completion, error) {
	c, err := cwo.cr.Complete(t, d)
	if c != nil {
		s := c.Suggestions
		c = cwo.cn.Clone()
		c.Suggestions = s
	}
	return c, err
}

func (cwo *cmplWithOpts[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.completer = cwo
}

// AsCompleter converts the `Completion` object into a `Completer` interface.
// This function is useful for constructing simple completers. To create a simple list,
// for example:
// ```go
// &Completion{Suggestions: []{"abc", "def", ...}}.AsCompleter
// ```
func AsCompleter[T any](c *Completion) Completer[T] {
	return &completionCompleter[T]{c}
}

type completionCompleter[T any] struct {
	c *Completion
}

func (sc *completionCompleter[T]) Complete(t T, d *Data) (*Completion, error) {
	return sc.c, nil
}

func (sc *completionCompleter[T]) modifyArgOpt(c *argOpt[T]) {
	c.completer = sc
}

// BoolCompleter is a completer for all boolean strings.
func BoolCompleter() Completer[bool] {
	return SimpleCompleter[bool](boolStringValues...)
}

// RunCompletion generates the `Completion` object from the provided inputs.
func RunCompletion[T any](c Completer[T], rawValue string, value T, data *Data) (*Completion, error) {
	if c == nil {
		return nil, nil
	}

	completion, err := c.Complete(value, data)
	if completion == nil || err != nil {
		return nil, err
	}

	op := getOperator[T]()

	if completion.Distinct {
		existingValues := map[string]bool{}
		// Don't include the last element because sometimes we want to just add a
		// a space to the command. For example,
		// "e commands.go" should return ["commands.go"]
		sl := op.toArgs(value)
		for i := 0; i < len(sl)-1; i++ {
			existingValues[sl[i]] = true
		}

		filtered := make([]string, 0, len(completion.Suggestions))
		for _, s := range completion.Suggestions {
			if !existingValues[s] {
				filtered = append(filtered, s)
			}
		}
		completion.Suggestions = filtered
	}

	return completion, nil
}

// ProcessInput processes a `Completion` object against a given `Input` object.
func (c *Completion) ProcessInput(input *Input) []string {
	var lastArg string
	if input != nil && len(input.args) > 0 {
		lastArg = input.args[len(input.args)-1].value
	}
	return c.process(lastArg, input.delimiter, false)
}

// process processes a `Completion` object using the provided `lastArg` and `delimiter`.
// If skipDelimiter is true, then no delimiter changes are done.
func (c *Completion) process(lastArg string, delimiter *rune, skipDelimiter bool) []string {
	results := c.Suggestions

	// Filter out prefixes.
	if !c.IgnoreFilter {
		filterFunc := func(s string) bool { return strings.HasPrefix(s, lastArg) }
		if c.CaseInsensitive {
			lowerLastArg := strings.ToLower(lastArg)
			filterFunc = func(s string) bool { return strings.HasPrefix(strings.ToLower(s), lowerLastArg) }
		}
		var filteredOpts []string
		for _, o := range results {
			if filterFunc(o) {
				filteredOpts = append(filteredOpts, o)
			}
		}
		results = filteredOpts
	}

	if c.CaseInsensitiveSort {
		sort.SliceStable(results, func(i, j int) bool {
			return strings.ToLower(results[i]) < strings.ToLower(results[j])
		})
	} else {
		sort.Strings(results)
	}

	if !skipDelimiter {
		for i, result := range results {
			if strings.Contains(result, " ") {
				if delimiter == nil {
					results[i] = strings.ReplaceAll(result, " ", "\\ ")
				} else {
					results[i] = fmt.Sprintf("%s%s%s", string(*delimiter), result, string(*delimiter))
				}
			}
		}
	}

	if c.DontComplete {
		results = append(results, " ")
	}
	return results
}

// FileCompleter is a `Completer` implementer specifically for file args.
type FileCompleter[T any] struct {
	// Regexp is the regexp that all suggested files must satisfy.
	Regexp *regexp.Regexp
	// Directory is the directory in which to search for files.
	Directory string
	// Distinct is whether or not each argument has to be unique.
	// Separate from Completion.Distinct because file completion
	// does more complicated custom logic (like only comparing
	// base names even though other arguments may have folder paths too).
	Distinct bool
	// FileTypes is the set of file suffixes to allow.
	FileTypes []string
	// IgnoreFiles indicates whether we should only consider directories.
	IgnoreFiles bool
	// IgnoreDirectories indicates whether we should only consider files.
	IgnoreDirectories bool
	// IgnoreFunc is a function that indicates whether a suggestion should be ignored.
	IgnoreFunc func(fullPath string, basename string, data *Data) bool
}

func (ff *FileCompleter[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.completer = ff
}

var (
	// ioutilReadDir is a var so it can be stubbed out for tests.
	ioutilReadDir = ioutil.ReadDir
)

// Complete creates a `Completion` object with the relevant set of files.
func (ff *FileCompleter[T]) Complete(value T, data *Data) (*Completion, error) {
	var lastArg string
	op := getOperator[T]()
	if args := op.toArgs(value); len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	laDir, laFile := filepath.Split(lastArg)
	var dir string
	// Use extra check for mingw on windows
	if cmdos.isAbs(laDir) {
		dir = laDir
	} else {
		var err error
		dir, err = filepathAbs(filepath.Join(ff.Directory, laDir))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute filepath: %v", err)
		}
	}

	if data.completeForExecute && len(laFile) == 0 {
		// If completing for execute and we are at a full directory (no basename),
		// then just return that.
		return &Completion{
			Suggestions: []string{lastArg},
		}, nil
	}

	files, err := ioutilReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %v", err)
	}

	onlyDir := true
	suggestions := make([]string, 0, len(files))
	allowedFileTypes := map[string]bool{}
	for _, ft := range ff.FileTypes {
		allowedFileTypes[ft] = true
	}
	for _, f := range files {
		isDir := f.IsDir() || (f.Mode()&fs.ModeSymlink != 0)
		if (isDir && ff.IgnoreDirectories) || (!isDir && ff.IgnoreFiles) {
			continue
		}

		if ff.Regexp != nil && !ff.Regexp.MatchString(f.Name()) {
			continue
		}

		if !strings.HasPrefix(strings.ToLower(f.Name()), strings.ToLower(laFile)) {
			continue
		}

		if isDir {
			suggestions = append(suggestions, fmt.Sprintf("%s/", f.Name()))
		} else if len(allowedFileTypes) == 0 || allowedFileTypes[filepath.Ext(f.Name())] {
			onlyDir = false
			suggestions = append(suggestions, f.Name())
		}
	}

	if len(suggestions) == 0 {
		return nil, nil
	}

	// Ignore any non-distinct matches, if relevant.
	argSet := map[string]bool{}
	absSet := map[string]bool{}
	if ff.Distinct {
		args := op.toArgs(value)
		for i := 0; i < len(args)-1; i++ {
			argSet[args[i]] = true
			if absArg, err := filepathAbs(args[i]); err == nil {
				absSet[absArg] = true
			}
		}
	}
	relevantSuggestions := make([]string, 0, len(suggestions))
	for _, s := range suggestions {
		fullPath := fmt.Sprintf("%s%s", laDir, s)
		if argSet[fullPath] {
			continue
		}

		if absFP, err := filepathAbs(filepath.Join(ff.Directory, fullPath)); err == nil && absSet[absFP] {
			continue
		}

		if ff.IgnoreFunc != nil && ff.IgnoreFunc(fullPath, s, data) {
			continue
		}

		relevantSuggestions = append(relevantSuggestions, s)
	}
	if len(relevantSuggestions) == 0 {
		return nil, nil
	}
	suggestions = relevantSuggestions

	c := &Completion{
		Suggestions:         suggestions,
		IgnoreFilter:        true,
		CaseInsensitiveSort: true,
	}

	// If only 1 suggestion matching, then we want it to autocomplete the whole thing.
	if len(c.Suggestions) == 1 {
		// Want to autocomplete the full path
		// Note: we can't use filepath.Join here because it cleans up the path
		c.Suggestions[0] = fmt.Sprintf("%s%s", laDir, c.Suggestions[0])

		// If completing for execute, then just complete to the directory
		if onlyDir && !data.completeForExecute {
			// This does "dir1/" and "dir1/_" so that the user's command is
			// autocompleted to "dir1/" without a space after it.
			c.Suggestions = append(c.Suggestions, fmt.Sprintf("%s%s", c.Suggestions[0], suffixChar))
		}
		return c, nil
	}

	autoFill, ok := getAutofillLetters(laFile, c.Suggestions)
	if !ok {
		// Nothing can be autofilled so we just return file names
		// Don't autocomplete because all suggestions have the same
		// prefix so this would actually autocomplete to the prefix
		// without the directory name
		c.DontComplete = true
		return c, nil
	}

	// Otherwise, we should complete all of the autofill letters
	c.DontComplete = false
	autoFill = laDir + autoFill
	c.Suggestions = []string{
		autoFill,
		autoFill + suffixChar,
	}

	return c, nil
}

func getAutofillLetters(laFile string, suggestions []string) (string, bool) {
	nextLetterPos := len(laFile)
	for proceed := true; proceed; nextLetterPos++ {
		var nextLetter *rune
		var lowerNextLetter rune
		for _, s := range suggestions {
			if len(s) <= nextLetterPos {
				// If a remaining suggestion has run out of letters, then
				// we can't autocomplete more than that.
				proceed = false
				break
			}

			char := rune(s[nextLetterPos])
			if nextLetter == nil {
				nextLetter = &char
				lowerNextLetter = unicode.ToLower(char)
				continue
			}

			if unicode.ToLower(char) != lowerNextLetter {
				proceed = false
				break
			}
		}
	}

	completeUpTo := nextLetterPos - 1
	if completeUpTo <= len(laFile) {
		return "", false
	}

	caseToCompleteWith := suggestions[0]
	for _, s := range suggestions {
		if strings.HasPrefix(s, laFile) {
			caseToCompleteWith = s
			break
		}
	}
	return caseToCompleteWith[:completeUpTo], true
}

// FileNode creates an `Arg` node for a file object.
func FileNode(argName, desc string, opts ...ArgOpt[string]) *ArgNode[string] {

	// Defaults must go first so they can be overriden by provided opts
	// For example, the last `Completer` opt in the slice will be the one
	// set in the `ArgOpt` object.
	return Arg(argName, desc, append([]ArgOpt[string]{
		&FileCompleter[string]{},
		FileTransformer(),
		FileExists(),
	}, opts...)...)
}

// FileListNode creates an `ArgList` node for file objects.
func FileListNode(argName, desc string, minN, optionalN int, opts ...ArgOpt[[]string]) *ArgNode[[]string] {
	opts = append(opts,
		&FileCompleter[[]string]{},
		TransformerList(FileTransformer()),
	)
	return ListArg(argName, desc, minN, optionalN, opts...)
}
