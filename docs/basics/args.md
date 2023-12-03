# Positional Args

## Overview

The most common way to provide arguments to a bash command is through positional arguments. This package provides users the capability to define arguments with the `command.Arg` function. This function creates a `Node` object from an argument name and description (which are used when creating the command's usage doc). See below for examples:

### Creating an Argument object

An argument can be created from a handful of argument functions. See the [examples section](./args.md#examples) for details.

### Retrieving an Argument's value

An argument's value is stored in the `command.Data` object. Retrieve an argument's value as follows:

```go
var (
  strArg = command.Arg[string]("STR_ARG", "A string")
  intArg = command.Arg[int]("INT_ARG", "An int")
)

// (data *command.Data)
myStr := strArg.Get(data) // returns a string
myInt := intArg.Get(data) // returns an int
```

## Examples

### Simple Positional Argument

A simple `string` argument.

```go
command.Arg[string]("ARG_NAME", "A simple string argument")
```

### Optional Argument

An optional `int` argument.

```go
command.OptionalArg[int]("N", "An optional integer argument")
```

### List Argument

The `ListArg` function requires a few more inputs. Specifically, the minimum number of required arguments, and the maximum number of optional arguments.

```go
var (
  // A list that requires exactly three arguments
  listOne := commander.ListArg[int]("LIST_ONE", "Desc", 3, 0)
  // A list that requires between two and five arguments.
  listOne := commander.ListArg[float64]("LIST_ONE", "Desc", 2, 3)
  // A list that requires at least four arguments
  listOne := commander.ListArg[int]("LIST_ONE", "Desc", 4, command.UnboundedList)
)
```

## Argument Options

Arguments also accept a handful of `ArgumentOption` modifiers. See [the options doc](./options.md) for more details.
