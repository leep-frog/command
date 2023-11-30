package sourcerer

import "github.com/leep-frog/command/commondels"

// ToCLI converts a node to a CLI.
func ToCLI(name string, root commondels.Node) CLI {
	return &simpleCLI{name, root}
}

type simpleCLI struct {
	name string
	root commondels.Node
}

func (sc *simpleCLI) Name() string       { return sc.name }
func (sc *simpleCLI) Setup() []string    { return nil }
func (sc *simpleCLI) Changed() bool      { return false }
func (sc *simpleCLI) Node() commondels.Node { return sc.root }
