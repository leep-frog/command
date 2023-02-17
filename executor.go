package command

// ExecuteErrNode creates a simple execution node from the provided error-able function.
type ExecutorProcessor struct {
	F func(Output, *Data) error
}

func (e *ExecutorProcessor) Execute(_ *Input, _ Output, _ *Data, eData *ExecuteData) error {
	eData.Executor = append(eData.Executor, e.F)
	return nil
}

func (e *ExecutorProcessor) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

func (e *ExecutorProcessor) Usage(*Input, *Data, *Usage) error { return nil }

type executableAppender struct {
	f func(Output, *Data) ([]string, error)
}

func (ea *executableAppender) Execute(i *Input, o Output, d *Data, ed *ExecuteData) error {
	sl, err := ea.f(o, d)
	if err != nil {
		return err
	}
	ed.Executable = append(ed.Executable, sl...)
	return nil
}

func (ea *executableAppender) Complete(*Input, *Data) (*Completion, error) {
	return nil, nil
}

func (ea *executableAppender) Usage(*Input, *Data, *Usage) error { return nil }

// SimpleExecutableProcessor returns a `Processor` that adds to the command's `Executable`.
func SimpleExecutableProcessor(sl ...string) Processor {
	return ExecutableProcessor(func(_ Output, d *Data) ([]string, error) { return sl, nil })
}

// ExecutableProcessor returns a `Processor` that adds to the command's `Executable`.
// Below are some tips when writing bash outputs for this:
// 1. Be sure to initialize variables with `local` to avoid overriding variables used in
// sourcerer scripts.
// 2. Use `return` rather than `exit` when terminating a session early.
func ExecutableProcessor(f func(Output, *Data) ([]string, error)) Processor {
	return &executableAppender{f}
}

// FunctionWrap sets ExecuteData.FunctionWrap to true.
func FunctionWrap() Processor {
	return SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		ed.FunctionWrap = true
		return nil
	}, nil)
}
