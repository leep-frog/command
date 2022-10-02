# Shortcuts

One of the most powerful tools of this package is shortcuts. Shortcuts allow you to store a set of arguments using a shortcut alias. See the example below for more details

## Example

```go
type myCLI struct {
  Shortcuts map[string]map[string][]string
  changed bool
}

// Other sourcerer.CLI functions not included for brevity.

func (mc *myCLI) Name() string {
  return "mc"
}

// MarkChanged ensures that the cache is saved to persistent data storage.
func (mc *myCLI) MarkChanged() {
  mc.changed = true
}

func (mc *myCLI) ShortcutMap() map[string]map[string][]string {
  if mc.Shortcuts == nil {
    mc.Shortcuts = map[string]map[string][]string{}
  }
  return mc.Shortcuts
}

func (mc *myCLI) Node() *command.Node {
  fileArg := command.Arg[string]("FILE", "filename to print", command.FileTransformer())

  printArgs := command.ListArg[string]("PRINT", "args to print", 1, command.UnboundedList)

  regularNode := command.SerialNodes(
    fileArg,
    printArgs,
    &command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
      o.Stdoutln(fileArg.Get(d))
      o.Stdoutln(strings.Join(printArgs.Get(d), " "))
      return nil
    }},
  )

  return command.ShortcutNode("shortcut-name", mc, regularNode)
}
```

The `ShortcutNode` adds the following branching arguments to your command:

- `a`: Add a shortcut (name first, followed by arguments).
- `d`: Delete an existing shortcut.
- `g`: Show the arguments for an existing shortcut.
- `l`: List all shortcuts and their arguments.
- `s`: Search shortcuts.

The following shell command sequence should give you a good idea of what this can do.

```shell
# Regular command execution
> mc someFile.txt echo contents
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs

# Add two shortcuts
> mc a someFile someFile.txt echo contents
> mc a otherFile otherFile.go other contents

# Run the first shortcut
> mc someFile
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs

# Run the second shortcut
> mc otherFile
/full/path/to/otherFile.txt # output from fileArg
other contents              # output from printArgs
```

## Stores Transformed Arguments

Similar to [caching](./caching.md), shortcuts stores the final, transformed value of the arguments in the CLI's [persistent data](./persistent_data.md), so this can work from any directory (provided the `command.FileTransformer()` option is used).

```shell
# Execute the shortcut normally
> mc someFile
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs

# Change directory
> cd ../..

# Execute mc using the cached arguments
> mc someFile
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs
```

## Padding Arguments

Arguments can also be added after a shortcut. What makes this particularly powerful is that all other package features (autocompletion, validation, transformation, etc.) will still work on the latter arguments!

```shell
# Run a shortcut with more arguments
> mc someFile extra args
/full/path/to/someFile.txt # output from fileArg
echo contents extra args   # output from printArgs
```
