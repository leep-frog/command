package command

// StaticCLIs returns a set of static CLIs.
func StaticCLIs(m map[string][]string) []*staticCLI {
	r := make([]*staticCLI, 0, len(m))
	for k, v := range m {
		r = append(r, StaticCLI(k, v...))
	}
	return r
}

// StaticCLI returns a static CLI.
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
func (sc *staticCLI) Changed() bool   { return false }
func (sc *staticCLI) Setup() []string { return nil }
func (sc *staticCLI) Node() *Node {
	return SerialNodes(SimpleExecutableNode(sc.commands...))
}
