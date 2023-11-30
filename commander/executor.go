package commander

import "github.com/leep-frog/command/commondels"

// ExecuteErrNode creates a simple execution node from the provided error-able function.
type ExecutorProcessor struct {
	F func(commondels.Output, *commondels.Data) error
}

func (e *ExecutorProcessor) Execute(_ *commondels.Input, _ commondels.Output, _ *commondels.Data, eData *commondels.ExecuteData) error {
	eData.Executor = append(eData.Executor, e.F)
	return nil
}

func (e *ExecutorProcessor) Complete(*commondels.Input, *commondels.Data) (*commondels.Completion, error) {
	return nil, nil
}

func (e *ExecutorProcessor) Usage(*commondels.Input, *commondels.Data, *commondels.Usage) error { return nil }

type executableAppender struct {
	f func(commondels.Output, *commondels.Data) ([]string, error)
}

func (ea *executableAppender) Execute(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
	sl, err := ea.f(o, d)
	if err != nil {
		return err
	}
	ed.Executable = append(ed.Executable, sl...)
	return nil
}

func (ea *executableAppender) Complete(*commondels.Input, *commondels.Data) (*commondels.Completion, error) {
	return nil, nil
}

func (ea *executableAppender) Usage(*commondels.Input, *commondels.Data, *commondels.Usage) error { return nil }

// SimpleExecutableProcessor returns a `commondels.Processor` that adds to the command's `Executable`.
func SimpleExecutableProcessor(sl ...string) commondels.Processor {
	return ExecutableProcessor(func(_ commondels.Output, d *commondels.Data) ([]string, error) { return sl, nil })
}

// ExecutableProcessor returns a `commondels.Processor` that adds to the command's `Executable`.
// Below are some tips when writing bash outputs for this:
// 1. Be sure to initialize variables with `local` to avoid overriding variables used in
// sourcerer scripts.
// 2. Use `return` rather than `exit` when terminating a session early.
func ExecutableProcessor(f func(commondels.Output, *commondels.Data) ([]string, error)) commondels.Processor {
	return &executableAppender{f}
}

// FunctionWrap sets commondels.ExecuteData.FunctionWrap to true.
func FunctionWrap() commondels.Processor {
	return SimpleProcessor(func(i *commondels.Input, o commondels.Output, d *commondels.Data, ed *commondels.ExecuteData) error {
		ed.FunctionWrap = true
		return nil
	}, nil)
}
