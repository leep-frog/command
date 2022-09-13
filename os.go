package command

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
)

var (
	CmdOS = mustGetOS()
)

// TODO: eventually actually implement completions and commands in windows shell.
type commandOS interface {
	IsAbs(string) bool
	AbsStart() string
	DefaultFilePerm() fs.FileMode
}

func mustGetOS() commandOS {
	// Can also split on runtime.GOARCH, but that currently
	// wouldn't change any logic.
	switch runtime.GOOS {
	case "windows":
		// This is the only differentiator I could find, but certainly open
		// to alternatives.
		fa, err := filepath.Abs("/")
		if err != nil {
			panic(fmt.Sprintf("failed to load absolute path: %f", err))
		}
		switch fa {
		case `C:\`:
			return &windowsMingwOS{}
		default:
			// This returns lowercase `c:\`:
			return &windowsOS{}
		}
	default:
		return &linuxOS{}
	}
}

type windowsOS struct{}

func (*windowsOS) IsAbs(dir string) bool {
	return filepath.IsAbs(dir) || (len(dir) > 0 && dir[0] == '/')
}

func (*windowsOS) AbsStart() string {
	return "/"
}

func (*windowsOS) DefaultFilePerm() fs.FileMode {
	return 0644
}

type windowsMingwOS struct{}

func (*windowsMingwOS) IsAbs(dir string) bool {
	return filepath.IsAbs(dir) || (len(dir) > 0 && dir[0] == '/')
}

func (*windowsMingwOS) AbsStart() string {
	return "/"
}

func (*windowsMingwOS) DefaultFilePerm() fs.FileMode {
	return 0644
}

type linuxOS struct{}

func (*linuxOS) IsAbs(dir string) bool {
	return filepath.IsAbs(dir)
}

func (*linuxOS) AbsStart() string {
	return "/"
}

func (*linuxOS) DefaultFilePerm() fs.FileMode {
	// This was originally 0644, but that caused an error in GCP Cloud Shell (
	// which is Linux based). Hence why we use 0666 here
	return 0666
}
