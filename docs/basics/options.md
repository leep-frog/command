# Argument and Flag Options

[Arguments](./args.md) and [Flags](./flags.md) both accept a handful of options. Comon option types are enumerated below.

## Completers

To activate autocompletion for arguments and flags, `Completer` objects can be passed in as options. See the [autocompletion doc](../features/autocompletion.md) for more details.

## `commander.Default`

This argument option sets the argument's value (only useful for optional arguments and flags).

```go
var (
  optionalFloat = command.OptionalArg[float64]("N", "An optional float argument", commander.Default[float64](12.3))

  stringFlag = commander.Flag[string]("STRING", 's', "A string flag", commander.Default[string]("default value"))
```

## `command.CustomSetter`

By default, the value provided is simply stored in the `command.Data.Values` map with the argument/flag name as the key. If you wish to override this behavior, you can provide a `CustomSetter` to do custom logic with the value.

## `command.Transformer`

A `Transformer` transforms the argument/flag's value. For example, `FileTransformer()` transforms a relative filepath into it's full absolute path (which is useful for [shortcuts](../features/shortcuts.md)).

## `command.Validator`

A `Validator` validates an input. Each argument/flag can have as many valiadators as is necessary. If validation fails, a useful error message will be sent to stderr. See below for validator examples:

```go
var (
  // An integer that must be positive
  positiveInt = command.Arg[int]("N", "An int argument", command.Positive())
  // A string argument that must be at least 8 characters long.
  username = command.Arg[string]("USERNAME", "Your username", command.MinLength(8))
)
```
