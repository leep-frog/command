// Package command provides classes and functions for defining custom CLIs.
package commondels

// Node defines a cohesive node in the command graph. It is simply a combination
// of a `Processor` and an `Edge`.
type Node interface {
	Processor
	Edge
}

// Processor defines the logic that should be executed at a `Node`.
type Processor interface {
	// Execute is the function called when a command graph is
	// being executed.
	Execute(*Input, Output, *Data, *ExecuteData) error
	// Complete is the function called when a command graph is
	// being autocompleted. If it returns a non-nil `Completion` object,
	// then the graph traversal stops and uses the returned object
	// to construct the command completion suggestions.
	Complete(*Input, *Data) (*Completion, error)
	// Usage is the function called when the usage data for a command
	// graph is being constructed. A
	// The input `Usage` object should be
	// updated for each `Node`.
	Usage(*Input, *Data, *Usage) error
}

// Edge determines which `Node` to execute next.
type Edge interface {
	// Next fetches the next node in the command graph based on
	// the provided `Input` and `Data`.
	Next(*Input, *Data) (Node, error)
	// UsageNext fetches the next node in the command graph when
	// command graph usage is being constructed. This is separate from
	// the `Next` function because `Next` is input-dependent whereas `UsageNext`
	// receives no input arguments.
	UsageNext(*Input, *Data) (Node, error)
}

// ExecuteData contains operations to resolve after all nodes have been processed.
// This separation is needed for caching and shortcuts nodes.
type ExecuteData struct {
	// Executable is a list of bash commands to run after all nodes have been processed.
	Executable []string
	// Executor is a set of functions to run after all nodes have been processed.
	Executor []func(Output, *Data) error
	// FunctionWrap is whether or not to wrap the Executable contents
	// in a function. This allows Executable to use things like "return" and "local".
	FunctionWrap bool
}
