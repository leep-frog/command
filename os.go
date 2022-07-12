package command

import (
	"fmt"
	"path/filepath"
	"runtime"
)

var (
	cmdos = mustGetOS()
)

// TODO: eventually actually implement completions and commands in windows shell.
type commandOS interface {
	isAbs(string) bool
	absStart() string
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

func (*windowsOS) isAbs(dir string) bool {
	return filepath.IsAbs(dir) || (len(dir) > 0 && dir[0] == '/')
}

func (*windowsOS) absStart() string {
	return "/"
}

type windowsMingwOS struct{}

func (*windowsMingwOS) isAbs(dir string) bool {
	return filepath.IsAbs(dir) || (len(dir) > 0 && dir[0] == '/')
}

func (*windowsMingwOS) absStart() string {
	return "/"
}

type linuxOS struct{}

func (*linuxOS) isAbs(dir string) bool {
	return filepath.IsAbs(dir)
}

func (*linuxOS) absStart() string {
	return "/"
}
