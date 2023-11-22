package commander

import "reflect"

// terminator is a custom type that is passed to panic
// when running `o.Terminate`
type terminator struct {
	terminationError error
}

func Terminate(err error) {
	panic(&terminator{err})
}

type node[I input, O output, D any, E, C, U, N any] interface {
	processor[I, O, D, E, C, U]
	edge[I, O, D, E, C, U, N]
}

type processor[I input, O output, D any, E, C, U any] interface {
	Execute(I, O, D, E) error
	Complete(I, D) (C, error)
	Usage(I, D, U) error
}

type edge[I input, O output, D any, E, C, U, N any] interface {
	Next(I, D) (N, error)
	UsageNext(I, D) (N, error)
}

type input interface {
	FullyProcessed() bool
}

type output interface {
	Stderrln(...interface{}) error
}

type executeFunctionBag[I input, O output, D any, E, C, U, N any] interface {
	ShowUsageAfterError(N, O)
	ExtraArgsErr(I) error
	GetExecutor(E) []func(O, D) error
}

// Separate method for testing purposes.
func Execute[I input, O output, D any, E, C, U any, N node[I, O, D, E, C, U, N]](n N, input I, output O, data D, eData E, fb executeFunctionBag[I, O, D, E, C, U, N]) (retErr error) {
	defer func() {
		r := recover()

		// No panic
		if r == nil {
			return
		}

		// Panicked due to terminate error
		if t, ok := r.(*terminator); ok && t.terminationError != nil {
			retErr = t.terminationError
			return
		}

		// Panicked for other reason
		panic(r)
	}()

	if retErr = ProcessGraphExecution[I, O, D, E, C, U, N](n, input, output, data, eData); retErr != nil {
		return
	}

	if !input.FullyProcessed() {
		retErr = fb.ExtraArgsErr(input)
		output.Stderrln(retErr)
		// TODO: Make this the last node we reached?
		fb.ShowUsageAfterError(n, output)
		return
	}

	for _, ex := range fb.GetExecutor(eData) {
		if retErr = ex(output, data); retErr != nil {
			return
		}
	}

	return
}

// ProcessOrExecute checks if the provided processor is a `Node` or just a `Processor`
// and traverses the subgraph or executes the processor accordingly.
func ProcessOrExecute[I input, O output, D any, E, C, U any, N node[I, O, D, E, C, U, N]](p processor[I, O, D, E, C, U], input I, output O, data D, eData E) error {
	if n, ok := p.(N); ok {
		return ProcessGraphExecution[I, O, D, E, C, U, N](n, input, output, data, eData)
	}
	return p.Execute(input, output, data, eData)
}

// TODO: replace with pointer types
func isNil(o interface{}) bool {
	return o == nil || reflect.ValueOf(o).IsNil()
}

// ProcessGraphExecution processes the provided graph
func ProcessGraphExecution[I input, O output, D any, E, C, U any, N node[I, O, D, E, C, U, N]](root N, input I, output O, data D, eData E) error {
	for n := root; !isNil(n); {
		if err := n.Execute(input, output, data, eData); err != nil {
			return err
		}

		var err error
		if n, err = n.Next(input, data); err != nil {
			return err
		}
	}
	return nil
}
