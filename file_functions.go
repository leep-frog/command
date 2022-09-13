package command

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	// This is set to a tmp directory during tests.
	fileRoot = ""
)

// FileTransformer returns a transformer that transforms a string into its full file-path.
func FileTransformer() *Transformer[string] {
	return &Transformer[string]{
		F: func(s string, d *Data) (string, error) {
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
