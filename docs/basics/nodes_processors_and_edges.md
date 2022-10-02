# Nodes, Processors, and Edges

At a high level, the `command` package simply traverses a graph of `command.Node` objects. A `Node` contains two fields:

1. `Node.Processor` contains the logic that should be executed when a `Node` is reached.

1. `Node.Edge` is the logic executed to determine which `Node` should be visited next.

The graph is traversed until there aren't any more nodes to traverse (i.e. `Node.Edge.Next()` returns `nil`) or until the `Node.Processor` logic or `Node.Edge` logic returns an error.

## Usage

While the graph logic may seem convoluted, this strucutre allows for features like [command execution caching](../features/caching.md) and [command shortcuts](../features/shortcuts.md). Additionally, simple graphs can still be easily constructed with the `SerialNodes` and `BranchNode` helper functions.

## Examples

### Serial Graph

This example constructs a graph that simply works its way through a set of linear nodes:

```go
func SerialGraph() *command.Node {
  firstNameArg := command.Arg[string]("FIRST_NAME", "First name")
  lastNameArg := command.Arg[string]("LAST_NAME", "Last name")
  excArg := command.OptionalArg[int]("EXCITEMENT", "How excited you are", command.Default(1))
  return command.SerialNodes(
    command.Description("A friendly CLI"),
    firstNameArg,
    lastNameArg,
    excArg,
    command.ExecutorNode(func(o command.Output, d *command.Data) {
      o.Stdoutf("Hello, %s %s%s\n", firstNameArg.Get(d), lastNameArg.Get(d), strings.Repeat("!", excArg.Get(d)))
    })
  )
}
```

### Branch Node Graph

This graph does different things depending on the first argument.

```go
func BranchingGraph() *command.Node {
  defaultNode := command.SerialNodes(
    command.ExecutorNode(func(o command.Output, d *command.Data) {
      o.Stdoutln("Why didn't you pick a door?")
    })
  )
  return command.BranchNode(map[string]*command.Node{
    "one": command.SerialNodes(
      command.ExecutorNode(func(o command.Output, d *command.Data) {
        o.Stdoutln("Not quite!")
      }),
    ),
    "two": command.SerialNodes(
      command.ExecutorNode(func(o command.Output, d *command.Data) {
        o.Stdoutln("You won a new car!")
      }),
    ),
    "three": command.SerialNodes(
      command.ExecutorNode(func(o command.Output, d *command.Data) {
        o.Stdoutln("Try again!")
      }),
    ),
  }, defaultNode)
}
```

### And Beyond

As you can see, these two functions alone can be combined to cover a broad range of CLI use cases. By also exposing the `Node`, `Processor`, and `Edge` types, users are empowered to implement even more advanced CLI-specific graph structures.
