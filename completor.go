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

func SimpleCompletor(s ...string) *Completor {
	return &Completor{
		SuggestionFetcher: &ListFetcher{
			Options: s,
		},
	}
}

func SimpleDistinctCompletor(s ...string) *Completor {
	return &Completor{
		Distinct: true,
		SuggestionFetcher: &ListFetcher{
			Options: s,
		},
	}
}

type Fetcher interface {
	Fetch(*Value, *Data) *Completion
}

type Completor struct {
	Distinct          bool
	SuggestionFetcher Fetcher
}

type Completion struct {
	Suggestions []string
	// TODO: each of these can just be option types.
	IgnoreFilter       bool
	DontComplete       bool
	CaseInsenstiveSort bool
}

func BoolCompletor() *Completor {
	return &Completor{
		SuggestionFetcher: &boolFetcher{},
	}
}

type boolFetcher struct{}

func (*boolFetcher) Fetch(*Value, *Data) *Completion {
	var keys []string
	for k := range boolStringMap {
		keys = append(keys, k)
	}
	return &Completion{
		Suggestions: keys,
	}
}

func (c *Completor) Complete(rawValue string, value *Value, data *Data) *Completion {
	if c == nil || c.SuggestionFetcher == nil {
		fmt.Println("NO1")
		return nil
	}

	completion := c.SuggestionFetcher.Fetch(value, data)
	if completion == nil {
		fmt.Println("NO2")
		return nil
	}

	fmt.Println("Completions", completion)

	if c.Distinct {
		existingValues := map[string]bool{}
		// Don't include the last element because sometimes we want to just add a
		// a space to the command. For example,
		// "e commands.go" should return ["commands.go"]
		sl := value.ToArgs()
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
	return completion
}

func (c *Completion) Process(input *Input) []string {
	var lastArg string
	if input != nil && len(input.args) > 0 {
		lastArg = input.args[len(input.args)-1]
	}
	results := c.Suggestions

	// Filter out prefixes.
	if !c.IgnoreFilter {
		var filteredOpts []string
		for _, o := range results {
			if strings.HasPrefix(o, lastArg) {
				filteredOpts = append(filteredOpts, o)
			}
		}
		results = filteredOpts
	}

	if c.CaseInsenstiveSort {
		sort.Slice(results, func(i, j int) bool {
			return strings.ToLower(results[i]) < strings.ToLower(results[j])
		})
	} else {
		sort.Strings(results)
	}

	for i, result := range results {
		if strings.Contains(result, " ") {
			if input.delimiter == nil {
				// TODO: default delimiter behavior should be defined by command?
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

type ListFetcher struct {
	Options []string
}

func (lf *ListFetcher) Fetch(*Value, *Data) *Completion {
	return &Completion{Suggestions: lf.Options}
}

type FileFetcher struct {
	Regexp    *regexp.Regexp
	Directory string
	// Whether or not each argument has to be unique.
	// Separate from Completor.Distinct because file fetching
	// does more complicated custom logic.
	Distinct          bool
	IgnoreFiles       bool
	IgnoreDirectories bool
}

func (ff *FileFetcher) Fetch(value *Value, _ *Data) *Completion {
	var lastArg string
	if value.IsType(StringType) {
		lastArg = value.String()
	} else if value.IsType(StringListType) && len(value.StringList()) > 0 {
		l := value.StringList()
		lastArg = l[len(l)-1]
	}

	laDir, laFile := filepath.Split(lastArg)
	dir, err := filepathAbs(filepath.Join(ff.Directory, laDir))
	if err != nil {
		return nil
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil
	}

	onlyDir := true
	suggestions := make([]string, 0, len(files))
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
		} else {
			onlyDir = false
			suggestions = append(suggestions, f.Name())
		}
	}

	if len(suggestions) == 0 {
		return nil
	}

	// Remove any non-distinct matches, if relevant.
	if ff.Distinct {
		valSet := map[string]bool{}
		// TODO: make validation function so file fetchers can only be
		// assigned to string lists.
		for _, v := range value.StringList() {
			valSet[v] = true
		}

		distinctSuggestions := make([]string, 0, len(suggestions))
		for _, s := range suggestions {
			if !valSet[fmt.Sprintf("%s%s", laDir, s)] {
				distinctSuggestions = append(distinctSuggestions, s)
			}
		}
		if len(distinctSuggestions) == 0 {
			return nil
		}
		suggestions = distinctSuggestions
	}

	c := &Completion{
		Suggestions:        suggestions,
		IgnoreFilter:       true,
		CaseInsenstiveSort: true,
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
		return c
	}

	autoFill, ok := getAutofillLetters(laFile, c.Suggestions)
	if !ok {
		// Nothing can be autofilled so we just return file names
		// Don't autocomplete because all suggestions have the same
		// prefix so this would actually autocomplete to the prefix
		// without the directory name
		c.DontComplete = true
		return c
	}

	// Otherwise, we should complete all of the autofill letters
	c.DontComplete = false
	autoFill = laDir + autoFill
	c.Suggestions = []string{
		autoFill,
		autoFill + suffixChar,
	}

	return c
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
