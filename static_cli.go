package command

func StaticCLIs(m map[string][]string) []*staticCLI {
	r := make([]*staticCLI, 0, len(m))
	for k, v := range m {
		r = append(r, StaticCLI(k, v...))
	}
	return r
}

func StaticCLI(name string, commands ...string) *staticCLI {
	return &staticCLI{
		name:     name,
		commands: commands,
	}
}

type staticCLI struct {
	name     string
	commands []string
}

func (sc *staticCLI) Name() string {
	return sc.name
}
func (sc *staticCLI) Load(string) error {
	return nil
}
func (sc *staticCLI) Changed() bool   { return false }
func (sc *staticCLI) Setup() []string { return nil }
func (sc *staticCLI) Node() *Node {
	return SerialNodes(SimpleProcessor(func(i *Input, o Output, d *Data, ed *ExecuteData) error {
		ed.Executable = append(ed.Executable, sc.commands...)
		return nil
	}, nil))
}