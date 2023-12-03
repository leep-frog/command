# Flags

## Overview

Another common way to provide input to a CLI is through flags. This package supports flag definitions, similar to [positional arguments](./args.md), except they are initiated with a flag (e.g. `--name` or `-n`).

Flags are declared similar to positional arguments and are [retrieved the way](./args.md#retrieving-an-arguments-value).

## Examples

### Simple, Single-Value Flag

```go
var (
  // This will create a flag that can be set by either `--flag` or `-f`
  myFlag = commander.Flag[string]("flag", 'f', "My first flag", /* options... */)
)
```

### List Flags

The `ListFlag` function requires a few more inputs. Specifically, the minimum number of required arguments, and the maximum number of optional arguments.

> Note: the argument minimum and maximum are only enforced if the flag is set.

```go
var (
  // This will create a list flag that requires between two and five arguments
  myListFlag = commander.ListFlag[string]("flag", 'f', "My first flag", 2, 3)
)
```

### Boolean Flags

It is also useful to have flags that simply set a boolean bit based on the presence of the `--name` or `-n` identifer. This package provides a simple `BoolFlag` function as well as a few other useful boolean flag modifications:

```go
var (
  // Set `Data.Values["one"]` to `true` if `--one` or `-o` is provided.
  b1 = commander.BoolFlag("one", 'o', "description")

  // (Singular) Set `Data.Values["two"]` to `123` if `--one` or `-o` is provided.
  b1 = command.BoolValueFlag[int]("one", 'o', "description", 123)

  // (Plural) Set `Data.Values["two"]` to "hi" if `--one` or `-o` is provided; otherwise sets it to "hello"
  b1 = command.BoolValuesFlag[string]("one", 'o', "description", 123, "hi", "hello")
)
```

## Flag Options

Flags accept the same [options as arguments](./args.md#argument-options). See [the options doc](./options.md) for more details.
