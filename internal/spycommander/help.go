package spycommander

import "github.com/leep-frog/command/command"

func HelpBehavior(n command.Node, i *command.Input, o command.Output, isUsageError func(error) bool) error {
	u, err := Use(n, i)
	if err != nil {
		o.Err(err)
		if isUsageError(err) {
			ShowUsageAfterError(n, o)
		}
		return err
	}
	o.Stdoutln(u.String())
	return nil
}
