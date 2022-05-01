package command

import (
	"fmt"
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

// SimpleCompletor returns a completor that suggests the provided strings for command autocompletion.
func SimpleCompletor[T any](s ...string) *Completor[T] {
	return &Completor[T]{
		Fetcher: &ListFetcher[T]{
			Options: s,
		},
	}
}

// ListFetcher is a fetcher that suggests a static set of strings.
type ListFetcher[T any] struct {
	Options []string
}

func (lf *ListFetcher[T]) Fetch(T, *Data) (*Completion, error) {
	return &Completion{Suggestions: lf.Options}, nil
}

// SimpleDistinctCompletor is the same as `SimpleCompletor` except it requires distinct arguments.
func SimpleDistinctCompletor[T any](s ...string) *Completor[T] {
	return &Completor[T]{
		Distinct: true,
		Fetcher: &ListFetcher[T]{
			Options: s,
		},
	}
}

// CompletorList changes a single arg completor (`Completor[T]`) into a list arg completor (`Completor[[]T]`).
func CompletorList[T any](c *Completor[T]) *Completor[[]T] {
	return &Completor[[]T]{
		c.Distinct,
		c.CaseInsensitive,
		SimpleFetcher(func(ts []T, d *Data) (*Completion, error) {
			var t T
			if len(ts) > 0 {
				t = ts[len(ts)-1]
			}
			return c.Fetcher.Fetch(t, d)
		}),
	}
}

type simpleFetcher[T any] struct {
	f func(T, *Data) (*Completion, error)
}

func (sf *simpleFetcher[T]) Fetch(v T, d *Data) (*Completion, error) {
	return sf.f(v, d)
}

// SimpleFetcher creates a `Fetcher` object from the provided function.
func SimpleFetcher[T any](f func(T, *Data) (*Completion, error)) Fetcher[T] {
	return &simpleFetcher[T]{f: f}
}

// Fetcher is an interface for fetching command suggestions.
type Fetcher[T any] interface {
	Fetch(T, *Data) (*Completion, error)
}

// Completor is an autocompletion object that can be used as an `ArgOpt`.
type Completor[T any] struct {
	// Distinct is whether or not we should return only distinct suggestions (specifically to prevent duplicates in list args).
	Distinct bool
	// CaseInsensitve is whether or not case should be considered when filtering out suggestions.
	CaseInsensitive bool
	// Fetcher is object that fetches all of the suggestions.
	Fetcher Fetcher[T]
}

func (c *Completor[T]) modifyArgOpt(ao *argOpt[T]) {
	ao.completor = c
}

// Completion is the object returned by a fetcher.
type Completion struct {
	// Suggestions is the set of autocomplete suggestions.
	Suggestions []string
	// IgnoreFilter indicates whether prefixes that don't match should be filtered out or not.
	IgnoreFilter bool
	// DontComplete indicates whether or not we should fill in partial completions.
	// This is achieved by adding a " " suggestion.
	DontComplete bool
	// CaseInsensitiveSort returns whether or not we should sort irrespective of case.
	// TODO: do we need this?
	CaseInsensitiveSort bool
	// CaseInsensitve is whether or not case should be considered when filtering out suggestions.
	CaseInsensitive bool
}

// BoolCompletor is a completor for all boolean strings.
func BoolCompletor() *Completor[bool] {
	return &Completor[bool]{
		Fetcher: &boolFetcher{},
	}
}

type boolFetcher struct{}

func (*boolFetcher) Fetch(bool, *Data) (*Completion, error) {
	var keys []string
	for k := range boolStringMap {
		keys = append(keys, k)
	}
	return &Completion{
		Suggestions: keys,
	}, nil
}

// Complete generates the `Completion` object from the provided inputs.
func (c *Completor[T]) Complete(rawValue string, value T, data *Data) (*Completion, error) {
	if c == nil || c.Fetcher == nil {
		return nil, nil
	}

	completion, err := c.Fetcher.Fetch(value, data)
	if completion == nil || err != nil {
		return nil, err
	}

	op := getOperator[T]()

	if c.Distinct {
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

	completion.CaseInsensitive = completion.CaseInsensitive || c.CaseInsensitive

	return completion, nil
}

// Process processes a `Completion` object against a given input.
func (c *Completion) Process(input *Input) []string {
	var lastArg string
	if input != nil && len(input.args) > 0 {
		lastArg = input.args[len(input.args)-1].value
	}
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
		sort.Slice(results, func(i, j int) bool {
			return strings.ToLower(results[i]) < strings.ToLower(results[j])
		})
	} else {
		sort.Strings(results)
	}

	for i, result := range results {
		if strings.Contains(result, " ") {
			if input.delimiter == nil {
				results[i] = strings.ReplaceAll(result, " ", "\\ ")
			} else {
				results[i] = fmt.Sprintf("%s%s%s", string(*input.delimiter), result, string(*input.delimiter))
			}
		}
	}

	if c.DontComplete {
		results = append(results, " ")
	}
	return results
}

// FileFetcher is a `Fetcher` implementer specific to file fetching.
type FileFetcher[T any] struct {
	// Regexp is the regexp that all suggested files must satisfy.
	Regexp *regexp.Regexp
	// Directory is the directory in which to search for files.
	Directory string
	// Distinct is whether or not each argument has to be unique.
	// Separate from Completor.Distinct because file fetching
	// does more complicated custom logic (like only comparing
	// base names even though other arguments may have folder paths too).
	Distinct bool
	// FileTypes is the set of file suffixes to allow.
	FileTypes []string
	// IgnoreFiles indicates whether we should only consider directories.
	IgnoreFiles bool
	// IgnoreDirectories indicates whether we should only consider files.
	IgnoreDirectories bool
	// IgnoreFunc is a custom function that returns the set of strings to ignore.
	// TODO: should this just return a boolean and have []string as an input?
	IgnoreFunc func(T, *Data) []string
}

// Fetch fetches the set of files for autocompletion.
func (ff *FileFetcher[T]) Fetch(value T, data *Data) (*Completion, error) {
	var lastArg string
	op := getOperator[T]()
	if args := op.toArgs(value); len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	laDir, laFile := filepath.Split(lastArg)
	dir, err := filepathAbs(filepath.Join(ff.Directory, laDir))
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute filepath: %v", err)
	}

	files, err := ioutil.ReadDir(dir)
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
		if (f.Mode().IsDir() && ff.IgnoreDirectories) || (f.Mode().IsRegular() && ff.IgnoreFiles) {
			continue
		}

		if ff.Regexp != nil && !ff.Regexp.MatchString(f.Name()) {
			continue
		}

		if !strings.HasPrefix(strings.ToLower(f.Name()), strings.ToLower(laFile)) {
			continue
		}

		if f.Mode().IsDir() {
			suggestions = append(suggestions, fmt.Sprintf("%s/", f.Name()))
		} else if len(allowedFileTypes) == 0 || allowedFileTypes[filepath.Ext(f.Name())] {
			onlyDir = false
			suggestions = append(suggestions, f.Name())
		}
	}

	if len(suggestions) == 0 {
		return nil, nil
	}

	ignorable := map[string]bool{}
	if ff.IgnoreFunc != nil {
		for _, s := range ff.IgnoreFunc(value, data) {
			ignorable[s] = true
		}
	}
	// Remove any non-distinct matches, if relevant.
	if ff.Distinct {
		// TODO: should this just go up to everything but the last value?
		for _, v := range op.toArgs(value) {
			ignorable[v] = true
		}
	}

	relevantSuggestions := make([]string, 0, len(suggestions))
	for _, s := range suggestions {
		if !ignorable[fmt.Sprintf("%s%s", laDir, s)] {
			relevantSuggestions = append(relevantSuggestions, s)
		}
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

		if onlyDir {
			// This does dir1/ and dir1// so that the user's command is autocompleted to dir1/
			// without a space after it.
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
	opts = append(opts,
		&Completor[string]{Fetcher: &FileFetcher[string]{}},
		FileTransformer(),
		FileExists(),
	)
	return Arg(argName, desc, opts...)
}

// FileListNode creates an `ArgList` node for file objects.
func FileListNode(argName, desc string, minN, optionalN int, opts ...ArgOpt[[]string]) *ArgNode[[]string] {
	opts = append(opts,
		&Completor[[]string]{Fetcher: &FileFetcher[[]string]{}},
		TransformerList(FileTransformer()),
	)
	return ListArg(argName, desc, minN, optionalN, opts...)
}
