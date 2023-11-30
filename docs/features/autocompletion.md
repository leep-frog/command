# Autocompletion

One of the most useful features of this package is the built-in autocompletion capabilities. Simply by providing a `Completer` option to your arguments/flags, you can have dynamic autocompletion to your CLIs. See below for specific examples.

## `command.SimpleCompleter`

The simplest completer is one that autocompletes a set of hardcoded options:

```go
var (
  myArg = command.Arg[string]("CHARACTER", "Choose a character", command.SimpleCompleter("Mario", "Kirby", "Link")
)
```

## `command.SimpleDistincCompleter` (for `ListArg/ListFlag`)

This completer is identical to the `SimpleCompleter` except it won't include items that are included earlier in the list.

```go
var (
  // `cli Mario [tab]` will only suggest "Kirby" and "Link".
  myArgs = command.ListArg[string]("CHARACTERS", "Choose two characters", 2, 0 command.SimpleCompleter("Mario", "Kirby", "Link")

  // Also works for list flags.
  myFlags = command.ListFlags[string]("TEAMMATE", "Choose two characters", 2, 0 command.SimpleCompleter("Luigi", "Metaknight", "Zelda")
)
```

## `command.FileCompleter`

This struct completes file/directory names. See the [go doc](TODO) for more info.

## Writing Your Own Completer (`command.CompleterFromFunc`)

This completer runs the provided function and uses the `Completion/error` returned from that function as the completion object.

This function is most useful for writing your own completion logic:

```go
var (
  myArg = command.Arg[string]("ARG", "Description", command.CompleterFromFunc(func(s string, d *commondels.Data) (*command.Completion, error) {
    var sl []string

    // Run whatever logic you want

    return &command.Completion{
      Suggestions: sl,
    }, nil
  }))
)
```
