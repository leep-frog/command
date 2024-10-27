# Command

`github.com/leep-frog/command` is a Go package for writing custom bash commands in Go! Some of the most valuable benefits of this package include:

- [Simple autocomplete implementation](./docs/features/autocompletion.md)

- [Automatic help documentation](./docs/features/automated_documentation.md)

- [Persistent CLI data](./docs/features/persistent_data.md)

- [CLI shortcutting](./docs/features/shortcuts.md)

- [Testing framework for command execution, completion, and usage](./docs/features/testing.md)

- [Caching previous command executions](./docs/features/caching.md)

See the following docs folders for more info:

- [`docs/basics` folder](./docs/basics/): Core concepts and types for this package.

- [`docs/features` folder](./docs/features/): Exhaustive list of features.

## Installation

1. Create a new go project (`go mod init`)

1. Include this in the project (`go get github.com/leep-frog/command`)

1. Create a `main.go` with the following contents:

```golang
package main

import (
  "os"

  "github.com/leep-frog/command/sourcerer"
)

func main() {
  clis := []sourcerer.CLI{
    // Put the CLIs you want here
  }

  opts := []sourcerer.Option{
    // Put CLI options (e.g. Aliasers) here
  }

  os.Exit(sourcerer.Source(clis, opts...))
}
```

4. `cd` into your project and run the following onetime setup:

```bash
go run . source myCLIs $FOLDER_FOR_BINARY_OUTPUT > $FILE_TO_SOURCE
```

> Note: if you ever make a change to your CLIs or above go file, you'll need to
> re-run the above command. If you do this frequently, consider making a helper
> function and adding it to your bash profile:

```bash
function reload_clis() {
  pushd . > /dev/null
  cd $PATH_TO_YOUR_GO_PROJECT
  go run . source myCLIs $FOLDER_FOR_BINARY_OUTPUT > $FILE_TO_SOURCE
  popd . > /dev/null
}
```

5. Add the following to your bash profile:

```bash
source $FILE_TO_SOURCE # <-- this must match the output file from the previous step
```

6. Either run the above command or start a new terminal session for your changes to take effect.

## Writing Your First Command

There are a couple of ways to get started. Some people love reading through a good doc, while others prefer to get their hands on examples right away. The following two sub-sections should hopefully satiate both of those approaches.

### Building Blocks Docs

- [Nodes, Processors, and Edges](./docs/basics/nodes_processors_and_edges.md)

- [Argument Nodes and Processors](./docs/basics/args.md)

- [Flag Processors](./building_blocks/flag_processors.md)

- [Execution Nodes](./building_blocks/execution_nodes.md)

- [Other Node Types](./building_blocks/other_nodes.md)

### Example

The steps from the [Installation section](#installation) will create a new bash
CLI called `sourcerer`. You can use this command to generate entirely new bash
CLIs written completely in Go!

To get an idea of how to write your own commands, start with an example:

1. Download the [example_main.go file](./cmd/example_main.go) locally and read through the file to become familiar with CLI setup (the file is thoroughly commented).

1. `cd` into the local directory containing the `example_main.go` file.

1. Run `sourcerer . my_custom_clis` (and
   add this line to your bash profile to automatically load this command from now on).

You are now able to use the `mfc` command in your bash shell! Try out the following runs and see what happens:

- ```bash
  mfc
  ```

- ```bash
  mfc $USER
  ```

- ```bash
  mfc <tab><tab>
  ```

- ```bash
  mfc B<tab><tab>
  ```

- ```bash
  mfc Br<tab>
  ```

- ```bash
  mfc there 10
  ```

- ```bash
  mfc World -f 10
  ```

To explore a more thorough explanation of all this package can do,
check out the [`docs/features` folder](./docs/features/)
