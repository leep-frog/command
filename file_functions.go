package command

import (
	"path/filepath"
)

var (
	// This is set to a tmp directory during tests.
	fileRoot = ""
)

// FileTransformer returns a transformer that transforms a string into its full file-path.
func FileTransformer() *Transformer[string] {
	return &Transformer[string]{
		t: func(s string, d *Data) (string, error) {
			if fileRoot == "" {
				return filepathAbs(s)
			}
			return filepath.Join(fileRoot, s), nil
		},
	}
}
