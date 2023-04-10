package sourcerer

import (
	"github.com/leep-frog/command"
	"golang.org/x/exp/slices"
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
func AliasSourcery(o command.Output, as ...*Aliaser) {
	if len(as) == 0 {
		return
	}

	slices.SortFunc(as, func(this, that *Aliaser) bool {
		return this.alias < that.alias
	})

	CurrentOS.GlobalAliaserFunc(o)

	verifiedCLIs := map[string]bool{}
	for _, a := range as {
		// Verify the CLI is a leep-frog CLI (if we haven't already).
		if _, ok := verifiedCLIs[a.cli]; !ok {
			verifiedCLIs[a.cli] = true
			CurrentOS.VerifyAliaser(o, a)
		}

		CurrentOS.RegisterAliaser(o, a)
	}
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
