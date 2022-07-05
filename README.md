# Command

`github.com/leep-frog/command` is a Go package for writing custom bash commands in Go! Some of the most valuable benefits of this package include:

 - [Simple autocomplete implementation](./docs/features/autocompletion.md)

 - [Automatic CLI usage documentation](./docs/features/automated_documentation.md)
 
 - [CLI shortcutting](./docs/features/shortcuts.md)

 - [Built-in persistent data capabilities](./docs/features/persistent_data.md)

 - [Testing command execution, completion, and usage](./docs/features/testing.md)

 - [Caching previous command executions](./docs/features/caching.md)

See the following docs folders for more info:

- [`docs/basics folder`](./docs/basics/): Core concepts and types for this package.

- [`docs/features` folder](./docs/features/): Exhaustive list of features.

## Installation

1. Download this repository locally:

   ```bash
   export $LEEP_INSTALLATION_PATH=/some/path
   pushd $LEEP_INSTALLATION_PATH > /dev/null
   git clone https://github.com/leep-frog/command.git
   popd > /dev/null
   ```

1. Add the following to your bash profile:

   ```bash
   source $LEEP_INSTALLATION_PATH/command/sourcerer/cmd/load_sourcerer.sh
   ```

1. Reload your shell and you're done!

## Writing Your First Command

There are a couple of ways to get started. Some people love reading through a good doc, while otherwise like to just get their hands on examples right away. The following two sub-sections should hopefully satiate both of those approaches.

### Building Blocks Docs

To read through the building blocks of this package, see the following docs:

- [Nodes, Processors, and Edges](./building_blocks/nodes_processors_edges.md)

- [Argument Processors](./building_blocks/arg_processors.md)

- [Flag Nodes and Processors](./building_blocks/flag_nodes_and_processors.md)

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