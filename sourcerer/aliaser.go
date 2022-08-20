package sourcerer

import (
	"fmt"
	"strings"

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

func AliasSourcery(o command.Output, as ...*Aliaser) {
	if len(as) == 0 {
		return
	}

	slices.SortFunc(as, func(this, that *Aliaser) bool {
		return this.alias < that.alias
	})

	o.Stdoutln(globalAutocompleteForAliasFunction)

	verifiedCLIs := map[string]bool{}
	for _, a := range as {
		// Verify the CLI is a leep-frog CLI (if we haven't already).
		if _, ok := verifiedCLIs[a.cli]; !ok {
			verifiedCLIs[a.cli] = true
			o.Stdoutf(strings.Join([]string{
				FileStringFromCLI(a.cli),
				`if [ -z "$file" ]; then`,
				fmt.Sprintf(`  echo Provided CLI %q is not a CLI generated with github.com/leep-frog/command`, a.cli),
				`  return 1`,
				`fi`,
				``,
				``,
			}, "\n"))
		}

		// Output the bash alias and completion commands
		var qas []string
		for _, v := range a.values {
			qas = append(qas, fmt.Sprintf("%q", v))
		}
		quotedArgs := strings.Join(qas, " ")
		aliasTo := fmt.Sprintf("%s %s", a.cli, quotedArgs)
		o.Stdoutf(strings.Join([]string{
			fmt.Sprintf("alias -- %s=%q", a.alias, aliasTo),
			fmt.Sprintf(autocompleteForAliasFunction, a.alias, a.cli, quotedArgs),
			fmt.Sprintf("complete -F _custom_autocomplete_for_alias_%s %s %s", a.alias, NosortString(), a.alias),
			``,
			``,
		}, "\n"))
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

// FileStringFromCLI returns a bash command that retrieves the binary file that
// is actually executed for a leep-frog-generated CLI.
func FileStringFromCLI(cli string) string {
	return fmt.Sprintf(`local file="$(type %s | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`, cli)
}
