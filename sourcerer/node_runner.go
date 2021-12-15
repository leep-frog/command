package sourcerer

import "github.com/leep-frog/command"

type GoLeep struct{}

func (gl *GoLeep) Name() string {
	return "goleep"
}

func (gl *GoLeep) Load(json string) error { return nil }
func (gl *GoLeep) Changed() bool          { return false }
func (gl *GoLeep) Setup() []string        { return nil }
func (gl *GoLeep) Node() *command.Node {
	return command.SerialNodes(nil) //command.ListBreaker(),
}
