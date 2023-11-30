package commander

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/leep-frog/command/command"
)

var (
	// This is set to a tmp directory during tests.
	fileRoot = ""
)

// FileTransformer returns a transformer that transforms a string into its full file-path.
func FileTransformer() *Transformer[string] {
	return &Transformer[string]{
		F: func(s string, d *command.Data) (string, error) {
			if fileRoot == "" {
				return filepathAbs(s)
			}
			return filepath.Join(fileRoot, s), nil
		},
	}
}

// Below are all file helper functions

// ReadFile reads the file into a slice of strings
func ReadFile(name string) ([]string, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n"), nil
}

// CreateFile creates a file from the provided string slice.
func CreateFile(name string, contents []string, ps fs.FileMode) error {
	return ioutil.WriteFile(name, []byte(strings.Join(contents, "\n")), ps)
}

// Stat runs os.Stat on the provided file and returns (nil, nil) if the file
// does not exist.
func Stat(name string) (os.FileInfo, error) {
	fi, err := os.Stat(name)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return fi, nil
}

// FileContents converts a filename into the file's contents.
func FileContents(name, desc string, opts ...ArgumentOption[string]) command.Processor {
	fc := FileArgument(name, desc, opts...)
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		if err := processOrExecute(fc, i, o, d, ed); err != nil {
			return err
		}
		b, err := os.ReadFile(d.String(name))
		if err != nil {
			return o.Annotatef(err, "failed to read fileee")
		}
		d.Set(name, strings.Split(strings.TrimSpace(string(b)), "\n"))
		return nil
	}, func(i *command.Input, d *command.Data) (*command.Completion, error) {
		return processOrComplete(fc, i, d)
	})
}
