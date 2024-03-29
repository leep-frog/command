package sourcerer

import (
	"sort"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

type Aliaser struct {
	alias  string
	cli    string
	values []string
}

func (a *Aliaser) modifyCompiledOpts(co *compiledOpts) {
	co.aliasers[a.alias] = a
}

// AliasSourcery outputs all alias source commands to the provided `command.Output`.
func AliasSourcery(goExecutable string, as ...*Aliaser) []string {
	if len(as) == 0 {
		return nil
	}

	sort.Slice(as, func(i, j int) bool {
		this, that := as[i], as[j]
		return this.alias < that.alias
	})

	r := CurrentOS.GlobalAliaserFunc(goExecutable)

	verifiedCLIs := map[string]bool{}
	for _, a := range as {
		// Verify the CLI is a leep-frog CLI (if we haven't already).
		if _, ok := verifiedCLIs[a.cli]; !ok {
			verifiedCLIs[a.cli] = true
			r = append(r, CurrentOS.VerifyAliaser(a)...)
		}

		r = append(r, CurrentOS.RegisterAliaser(goExecutable, a)...)
	}
	return r
}

func NewAliaser(alias string, cli string, values ...string) *Aliaser {
	return &Aliaser{alias, cli, values}
}

func Aliasers(m map[string][]string) Option {
	var opts []Option
	for a, vs := range m {
		opts = append(opts, NewAliaser(a, vs[0], vs[1:]...))
	}
	return multiOpts(opts...)
}

// AliaserCommand creates an alias for another arg
type AliaserCommand struct {
	goExecutableFilePath string
}

var (
	aliasArg    = commander.Arg[string]("ALIAS", "Alias of new command", commander.MinLength[string, string](1))
	aliasCLIArg = commander.Arg[string]("CLI", "CLI of new command")
	aliasPTArg  = commander.ListArg[string]("PASSTHROUGH_ARGS", "Args to passthrough with alias", 0, command.UnboundedList)
)

func (*AliaserCommand) Setup() []string { return nil }
func (*AliaserCommand) Changed() bool   { return false }
func (*AliaserCommand) Name() string    { return "aliaser" }

func (ac *AliaserCommand) Node() command.Node {
	return commander.SerialNodes(
		commander.Description("Alias a command to a cli with some args included"),
		aliasArg,
		aliasCLIArg,
		aliasPTArg,
		commander.ExecutableProcessor(func(_ command.Output, d *command.Data) ([]string, error) {
			aliaser := NewAliaser(aliasArg.Get(d), aliasCLIArg.Get(d), aliasPTArg.Get(d)...)
			return AliasSourcery(ac.goExecutableFilePath, aliaser), nil
		}),
	)
}
