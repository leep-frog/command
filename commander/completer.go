package commander

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/constants"
	"github.com/leep-frog/command/internal/operator"
	"github.com/leep-frog/command/internal/stubs"
)

var (
	// Used for testing.
	filepathAbs = filepath.Abs
)

// SimpleCompleter returns a completer that suggests the provided strings for command autocompletion.
func SimpleCompleter[T any](s ...string) Completer[T] {
	return AsCompleter[T](
		&command.Completion{
			Suggestions: s,
		},
	)
}

// SimpleDistinctCompleter returns a completer that distinctly suggests the provided strings for command autocompletion.
func SimpleDistinctCompleter[T any](s ...string) Completer[T] {
	return AsCompleter[T](
		&command.Completion{
			Distinct:    true,
			Suggestions: s,
		},
	)
}

// CompleterList changes a single arg completer (`Completer[T]`) into a list arg completer (`Completer[[]T]`).
func CompleterList[T any](c Completer[T]) Completer[[]T] {
	return &simpleCompleter[[]T]{
		f: func(ts []T, d *command.Data) (*command.Completion, error) {
			var t T
			if len(ts) > 0 {
				t = ts[len(ts)-1]
			}
			return c.Complete(t, d)
		},
	}
}

// CompleterFromFunc returns a `Completer` object from the provided function.
func CompleterFromFunc[T any](f func(T, *command.Data) (*command.Completion, error)) Completer[T] {
	return &simpleCompleter[T]{f}
}

type simpleCompleter[T any] struct {
	f func(T, *command.Data) (*command.Completion, error)
}

func (sc *simpleCompleter[T]) Complete(t T, d *command.Data) (*command.Completion, error) {
	return sc.f(t, d)
}

func (sc *simpleCompleter[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.completer = sc
}

// Completer is an autocompletion object that can be used as an `ArgumentOption`.
type Completer[T any] interface {
	Complete(T, *command.Data) (*command.Completion, error)
	modifyArgumentOption(*argumentOption[T])
}

// DeferredCompleter returns an argument/flag `Completer` that defers completion
// until after the provided graph is run. See the `DeferredCompletion` object
// for more info.
func DeferredCompleter[T any](graph command.Node, completer Completer[T]) Completer[T] {
	return CompleterFromFunc(func(t T, d *command.Data) (*command.Completion, error) {
		return &command.Completion{
			DeferredCompletion: &command.DeferredCompletion{
				graph,
				func(c *command.Completion, d *command.Data) (*command.Completion, error) {
					return RunArgumentCompleter(completer, t, d)
				},
			},
		}, nil
	})
}

// CompleterWithOpts sets the relevant options in the `command.Completion` object
// returned by the `Completer`.
func CompleterWithOpts[T any](cr Completer[T], cn *command.Completion) Completer[T] {
	return &cmplWithOpts[T]{cr, cn}
}

type cmplWithOpts[T any] struct {
	cr Completer[T]
	cn *command.Completion
}

func (cwo *cmplWithOpts[T]) Complete(t T, d *command.Data) (*command.Completion, error) {
	c, err := cwo.cr.Complete(t, d)
	if c != nil {
		s := c.Suggestions
		c = cwo.cn.Clone()
		c.Suggestions = s
	}
	return c, err
}

func (cwo *cmplWithOpts[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.completer = cwo
}

// AsCompleter converts the `command.Completion` object into a `Completer` interface.
// This function is useful for constructing simple completers. To create a simple list,
// for example:
// ```go
// &command.Completion{Suggestions: []{"abc", "def", ...}}.AsCompleter
// ```
func AsCompleter[T any](c *command.Completion) Completer[T] {
	return &completionCompleter[T]{c}
}

type completionCompleter[T any] struct {
	c *command.Completion
}

func (sc *completionCompleter[T]) Complete(t T, d *command.Data) (*command.Completion, error) {
	return sc.c, nil
}

func (sc *completionCompleter[T]) modifyArgumentOption(c *argumentOption[T]) {
	c.completer = sc
}

// BoolCompleter is a completer for all boolean strings.
func BoolCompleter() Completer[bool] {
	return SimpleCompleter[bool](constants.BoolStringValues...)
}

// RunArgumentCompleter generates a `command.Completion` object from the provided
// `Completer` and inputs.
func RunArgumentCompleter[T any](c Completer[T], value T, data *command.Data) (*command.Completion, error) {
	if c == nil {
		return nil, nil
	}

	completion, err := c.Complete(value, data)
	if completion == nil || err != nil {
		return nil, err
	}

	return RunArgumentCompletion(completion, value, data)
}

// RunArgumentCompletion generates a `command.Completion` object from the provided
// `command.Completion` and inputs.
func RunArgumentCompletion[T any](completion *command.Completion, value T, data *command.Data) (*command.Completion, error) {

	op := operator.GetOperator[T]()

	if completion.Distinct {
		existingValues := map[string]bool{}
		// Don't include the last element because sometimes we want to just add a
		// a space to the command. For example,
		// "e commands.go" should return ["commands.go"]
		sl := op.ToArgs(value)
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

// FileCompleter is a `Completer` implementer specifically for file args.
type FileCompleter[T any] struct {
	// Regexp is the regexp that all suggested files must satisfy.
	Regexp *regexp.Regexp
	// Directory is the directory in which to search for files.
	Directory string
	// Distinct is whether or not each argument has to be unique.
	// Separate from command.Completion.Distinct because file completion
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
	IgnoreFunc func(fullPath string, basename string, data *command.Data) bool
	// ExcludePwd is whether or not the current working directory path should be excluded
	// from completions.
	ExcludePwd bool
	// MaxDepth is the maximum depth for files allowed. If less than or equal to zero,
	// then no limit is applied.
	MaxDepth int
}

func (ff *FileCompleter[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.completer = ff
}

var (
	// osReadDir is a var so it can be stubbed out for tests.
	osReadDir = os.ReadDir
	// filepathRel is a var so it can be stubbed out for tests.
	filepathRel = filepath.Rel
)

// filepathDepth returns the depth of the provided directory
func filepathDepth(path string) int {
	return len(strings.Split(strings.TrimLeft(filepath.FromSlash(path), string(os.PathSeparator)), string(os.PathSeparator)))
}

func isAbs(dir string) bool {
	return filepath.IsAbs(dir) || (len(dir) > 0 && (dir[0] == '/' || dir[0] == '\\'))
}

// Complete creates a `command.Completion` object with the relevant set of files.
func (ff *FileCompleter[T]) Complete(value T, data *command.Data) (*command.Completion, error) {
	var lastArg string
	op := operator.GetOperator[T]()
	if args := op.ToArgs(value); len(args) > 0 {
		lastArg = args[len(args)-1]
	}

	laDir, laFile := filepath.Split(filepath.FromSlash(lastArg))
	tooDeep := ff.MaxDepth > 0 && filepathDepth(lastArg) >= ff.MaxDepth
	var dir string
	// Use extra check for mingw on windows
	if isAbs(laDir) {
		dir = laDir
	} else {
		var err error
		dir, err = filepathAbs(filepath.Join(ff.Directory, laDir))
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute filepath: %v", err)
		}
	}

	if data.Complexecute && len(laFile) == 0 {
		// If complexecuting and we are at a full directory (no basename),
		// then just return that.
		return &command.Completion{
			Suggestions: []string{lastArg},
		}, nil
	}

	files, err := osReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %v", err)
	}

	var ignoreDir *string
	if ff.ExcludePwd {
		pwd, err := stubs.OSGetwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %v", err)
		}

		// pwd and dir are both absolute paths, so an error should never be returned
		rel, err := filepathRel(dir, pwd)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative directory: %v", err)
		}
		if rel[0] != '.' {
			ignoreDir = &(strings.Split(rel, string(os.PathSeparator))[0])
		}
	}

	onlyDir := true
	suggestions := make([]string, 0, len(files))
	allowedFileTypes := map[string]bool{}
	for _, ft := range ff.FileTypes {
		allowedFileTypes[ft] = true
	}
	for _, f := range files {
		isDir := f.IsDir() || (f.Type()&fs.ModeSymlink != 0)
		if (isDir && ff.IgnoreDirectories) || (!isDir && ff.IgnoreFiles) {
			continue
		}

		// ExcludePwd check
		if isDir && ignoreDir != nil && *ignoreDir == f.Name() {
			continue
		}

		if ff.Regexp != nil && !ff.Regexp.MatchString(f.Name()) {
			continue
		}

		if !strings.HasPrefix(strings.ToLower(f.Name()), strings.ToLower(laFile)) {
			continue
		}

		if isDir {
			if tooDeep {
				suggestions = append(suggestions, f.Name())
			} else {
				suggestions = append(suggestions, filepath.FromSlash(fmt.Sprintf("%s/", f.Name())))
			}
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
		args := op.ToArgs(value)
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

	c := &command.Completion{
		Suggestions:         suggestions,
		IgnoreFilter:        true,
		CaseInsensitiveSort: true,
	}

	// If only 1 suggestion matching, then we want it to autocomplete the whole thing.
	if len(c.Suggestions) == 1 {
		// Want to autocomplete the full path
		// Note: we can't use filepath.Join here because it cleans up the path
		c.Suggestions[0] = fmt.Sprintf("%s%s", laDir, c.Suggestions[0])

		// If complexecuting, then just complete to the directory
		// Also, if the command.OS does not add a space, then no need to do this.
		if onlyDir && !data.Complexecute && !tooDeep {
			// This does "dir1/" and "dir1/_" so that the user's command is
			// autocompleted to "dir1/" without a space after it.
			c.SpacelessCompletion = true
		}
		return c, nil
	}

	// If here, then there are multiple suggestions, which means complexecute should fail.
	// So, we don't need to try to autofill.
	if data.Complexecute {
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
	}
	c.SpacelessCompletion = true

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

// FileArgument creates an `Argument` processor for a file object. The `Argument` returned
// by this function only relates to existing files (for execution and completion).
// For more granular control of the specifics, make your own `Arg(...)` with file-relevant
// `ArgumentOptions` (such as `FileCompleter`, `FileExists`, `IsDir`, `FileTransformer`, etc.)
func FileArgument(argName, desc string, opts ...ArgumentOption[string]) *Argument[string] {

	// Defaults must go first so they can be overriden by provided opts
	// For example, the last `Completer` opt in the slice will be the one
	// set in the `ArgumentOption` object.
	return Arg(argName, desc, append([]ArgumentOption[string]{
		&FileCompleter[string]{},
		FileTransformer(),
		FileExists(),
	}, opts...)...)
}

// FileListArgument creates an `ArgList` node for file objects.
func FileListArgument(argName, desc string, minN, optionalN int, opts ...ArgumentOption[[]string]) *Argument[[]string] {
	opts = append(opts,
		&FileCompleter[[]string]{},
		TransformerList(FileTransformer()),
	)
	return ListArg(argName, desc, minN, optionalN, opts...)
}
