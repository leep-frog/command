# Complexecute (Complete For Execute)

Complexecute allows you to provide a partial argument when that is the only
argument that would have been completed.  Basically, this features saves you
a tab press (and completion process) and is useful for command executions that
you are running frequently (but not frequently enough to warrant
[shortcuts](./shortcuts.md)).

I find this particularly useful when testing a specific file. While
writing/fixing tests in a file, I will run that test several times, but not
often enough that I'd want to make a shortcut for it. If I know the file prefix
is unique (say it is the only file that starts with `Te`), then I can just write
`myCLI test Te` instead of `myCLI test TerriblyLongFileName.java`.

> Note: this will only work for the last argument provided.

## Example

```go
type myCLI struct {}

// Other sourcerer.CLI functions not included for brevity.

func (mc *myCLI) Name() string {
  return "myCLI"
}

func (mc *myCLI) Node() command.Node {
  fileArg := command.FileArgument("TEST_FILE", "File to test", command.CompleteForExecute())

  return command.AsNode(&command.BranchNode{
    Branches: map[string]command.Node{
      "test": commander.SerialNodes(
        fileArg,
        command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
					return []string{
						fmt.Sprintf("test %q", fileArg.Get(d)),
					}, nil
				}),
      ),
    },
  )
}
```

This allows for the following sequence of command executions:

```shell
# Tests TerriblyLongFileName.java
> myCLI test Te

# Errors because autocompletion doesn't return a single suggestion.
> myCLI test T
```

## Complexecute Options

### ComplexecuteBestEffort

ComplexecuteBestEffort runs Complexecute on a best effort basis. If zero or multiple completions are suggested, then the argument isn't altered.

### ComplexecuteAllowExactMatch
ComplexecuteAllowExactMatch allows exact matches even if multiple completions were returned. For example, if the arg is "Hello", and the resulting completions are ["Hello", "HelloThere", "Hello!"], then we won't error.
