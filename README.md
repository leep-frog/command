# Command

`command` is a package for writing custom bash commands in go. Some of the most useful benefits of this package include:

 - [Simple autocomplete implementation](#bash-autocompletion)

 - [Built-in CLI usage documentation](#cli-usage-text)
 
 - [Command shortcuts](#shortcuts)
 
 - [Caching previous command executions](#execution-caching)

 - [Built-in persistent command data](#persistent-cli-data)

 - [Testing command execution, completion, and usage](#command-testing)

## Setup

1. Create a new go package using 

  ```bash
  go mod init example.com/clis
  ```

2. Add the following lines to your bash profile to 1) set up a command cache folder and 2) allow easy installation of command CLI implementers:
  
```bash
export LEEP_CLI_CACHE=/any/path/golang-cli-cache
source /path/to/command/sourcerer/cmd/load_sourcerer.sh
```

This will load a bash command (`gg`) that makes updating `leep-frog` cli packages much simpler:

```bash
gg cd emacs grep # etc.
```

3. Install all of the CLI packages you want to use in your new package.

4. Copy the following into a new file called `clis.go`:

```golang
package main

import (
	"github.com/leep-frog/cd"
	"github.com/leep-frog/command/cache"
	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/emacs"
	"github.com/leep-frog/grep"
	"github.com/leep-frog/replace"
	"github.com/leep-frog/todo"
	"github.com/leep-frog/workspace"
)

func main() {
	acs := []sourcerer.CLI{
		grep.RecursiveCLI(),
		grep.HistoryCLI(),
		grep.FilenameCLI(),
		grep.StdinCLI(),
		replace.CLI(),
		todo.CLI(),
		emacs.CLI(),
		workspace.CLI(),
		sourcerer.GoLeepCLI(),
		&cache.Cache{},
	}

	for i := 0; i < 10; i++ {
		acs = append(acs, cd.DotCLI(i))
	}

  // sourcerer.Source returns the exit code of the operation.
	os.Exit(sourcerer.Source(acs...))
}
```

5. Add the final line to your bash profile:

```bash
sourcerer /path/to/clis my_custom_clis
```

6. Restart your shell and use your commands!

## Benefits

### Bash Autocompletion

TODO

### CLI Usage Text

TODO

### Shortcuts

TODO

### Execution Caching

TODO

### Persistent CLI Data

TODO

### Command Testing

TODO
