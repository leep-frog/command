# Caching

Caching allows you to re-run the previous execution of a command. This is useful for commands that you run frequently with other commands in between.

## Example

```go
type myCLI struct {
  MyCache map[string][][]string
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

func (mc *myCLI) Cache() map[string][][]string {
  if mc.MyCache == nil {
    mc.MyCache = map[string][][]string{}
  }
  return mc.MyCache
}

func (mc *myCLI) Node() command.Node {
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

  return command.CacheNode("cache-name", mc, regularNode)
}
```

This makes the following possible:

```shell
# Execute the command normally
> mc someFile.txt echo contents
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs

# Execute other command(s)
> echo another command
another command

# Execute mc using the cached arguments
> mc
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs
```

## Stores Transformed Arguments

It is also important to note that the final, transformed value of the input is stored in the CLI's [persistent data](./persistent_data.md), so this will still work from other directories (provided the `command.FileTransformer()` is used for all of your file args/flags).

```shell
# Execute the command normally
> mc someFile.txt echo contents
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs

# Change directory
> cd ../..

# Execute mc using the cached arguments
> mc
/full/path/to/someFile.txt # output from fileArg
echo contents              # output from printArgs
```

## History

The cache node also includes a `history` command branch that lets you see previous executions of your command:

```shell
> mc history
# ...
# Arguments for 3rd most recent call
# Arguments for 2nd most recent call
/full/path/to/someFile.txt echo contents
```

The default number of calls stored is set by `command.CacheDefaultHistory`. This can be overriden
by including an option to your cache node creation:
```go
return command.CacheNode("cache-name", mc, regularNode, command.CacheHistory(25_000))
```
