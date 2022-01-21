package command

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	// This is set to a tmp directory during tests.
	fileRoot = ""
)

func FileTransformer() *Transformer[string] {
	return &Transformer[string]{
		t: func(s string) (string, error) {
			if fileRoot == "" {
				return filepathAbs(s)
			}
			return filepath.Join(fileRoot, s), nil
		},
	}
}

func relFile(name string) string {
	return filepath.Join(fileRoot, name)
}

func ReadAll(name string) ([]os.FileInfo, error) {
	a, b := ioutil.ReadDir(relFile(name))
	return a, newFileErr(b)
}

func ReadFile(name string) ([]string, error) {
	b, err := os.ReadFile(relFile(name))
	if err != nil {
		return nil, newFileErr(err)
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n"), nil
}

func Mkdir(name string) error {
	return newFileErr(os.Mkdir(relFile(name), 0644))
}

func CreateFile(name string, contents []string) error {
	return newFileErr(ioutil.WriteFile(relFile(name), []byte(strings.Join(contents, "\n")), 0644))
}

func DeleteFile(name string) error {
	return newFileErr(os.Remove(relFile(name)))
}

type fileErr struct {
	err error
}

func (fe *fileErr) Error() string {
	if fileRoot == "" {
		return fe.err.Error()
	}
	return strings.ReplaceAll(fe.err.Error(), fileRoot, "TEST_DIR/")
}

func newFileErr(err error) error {
	if err == nil {
		return nil
	}
	return &fileErr{err}
}

func Stat(name string) (os.FileInfo, error) {
	fi, err := os.Stat(relFile(name))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return fi, nil
}

type FakeFile struct {
	name     string
	contents []string
	files    []*FakeFile
	dir      bool
}

func NewFakeDir(name string, files []*FakeFile) *FakeFile {
	return &FakeFile{
		name:  name,
		files: files,
		dir:   true,
	}
}

func NewFakeFile(name string, contents []string) *FakeFile {
	return &FakeFile{
		name:     name,
		contents: contents,
	}
}

func (ff *FakeFile) create(t *testing.T, parentDirs []string) {
	t.Helper()
	relativeName := append(parentDirs, ff.name)
	if ff.dir {
		if err := Mkdir(filepath.Join(relativeName...)); err != nil {
			t.Fatalf("Failed to create directory %q during file path setup: %v", ff.name, err)
		}
		for _, f := range ff.files {
			f.create(t, relativeName)
		}
	} else {
		if err := CreateFile(filepath.Join(relativeName...), ff.contents); err != nil {
			t.Fatalf("Failed to create file %q during file path setup: %v", ff.name, ff.contents)
		}
	}
}

// Read all of the directory and load into fake files
func toFakeFiles(t *testing.T, name string) *FakeFile {
	t.Helper()

	fi, err := Stat(name)
	if err != nil {
		t.Fatalf("Stat(%s) returned error: %v", name, err)
	}
	if fi == nil {
		t.Fatalf("Stat(%s) returned nil", name)
	}
	if fi.IsDir() {
		var files []*FakeFile
		fs, err := ReadAll(name)
		if err != nil {
			t.Fatalf("ReadAll(%s) returned error: %v", name, err)
		}

		for _, f := range fs {
			files = append(files, toFakeFiles(t, filepath.Join(name, f.Name())))
		}
		return NewFakeDir(filepath.Base(name), files)
	}
	c, err := ReadFile(name)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", name, err)
	}
	return NewFakeFile(filepath.Base(name), c)
}
