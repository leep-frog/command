package commander

import "github.com/leep-frog/command/command"

// ExecuteErrNode creates a simple execution node from the provided error-able function.
type ExecutorProcessor struct {
	F func(command.Output, *command.Data) error
}

func (e *ExecutorProcessor) Execute(_ *command.Input, _ command.Output, _ *command.Data, eData *command.ExecuteData) error {
	eData.Executor = append(eData.Executor, e.F)
	return nil
}

func (e *ExecutorProcessor) Complete(*command.Input, *command.Data) (*command.Completion, error) {
	return nil, nil
}

func (e *ExecutorProcessor) Usage(*command.Input, *command.Data, *command.Usage) error { return nil }

type executableAppender struct {
	f func(command.Output, *command.Data) ([]string, error)
}

func (ea *executableAppender) Execute(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
	sl, err := ea.f(o, d)
	if err != nil {
		return err
	}
	ed.Executable = append(ed.Executable, sl...)
	return nil
}

func (ea *executableAppender) Complete(*command.Input, *command.Data) (*command.Completion, error) {
	return nil, nil
}

func (ea *executableAppender) Usage(*command.Input, *command.Data, *command.Usage) error { return nil }

// SimpleExecutableProcessor returns a `command.Processor` that adds to the command's `Executable`.
func SimpleExecutableProcessor(sl ...string) command.Processor {
	return ExecutableProcessor(func(_ command.Output, d *command.Data) ([]string, error) { return sl, nil })
}

// ExecutableProcessor returns a `command.Processor` that adds to the command's `Executable`.
// Below are some tips when writing bash outputs for this:
// 1. Be sure to initialize variables with `local` to avoid overriding variables used in
// sourcerer scripts.
// 2. Use `return` rather than `exit` when terminating a session early.
func ExecutableProcessor(f func(command.Output, *command.Data) ([]string, error)) command.Processor {
	return &executableAppender{f}
}

// FunctionWrap sets command.ExecuteData.FunctionWrap to true.
func FunctionWrap() command.Processor {
	return SimpleProcessor(func(i *command.Input, o command.Output, d *command.Data, ed *command.ExecuteData) error {
		ed.FunctionWrap = true
		return nil
	}, nil)
}
