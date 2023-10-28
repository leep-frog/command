package sourcerer

import (
	"fmt"
	"regexp"

	"github.com/leep-frog/command"
)

var (
	RelevantPackages = []string{
		"cd",
		"command",
		"differ",
		"emacs",
		"gocli",
		"grep",
		"labelmaker",
		"notification",
		"pdf",
		"replace",
		"ssh",
		"sourcecontrol",
		"todo",
		"workspace",
	}
)

// UpdateLeepPackageCommand is a CLI for updating github.com/leep-frog packages
type UpdateLeepPackageCommand struct{}

func (*UpdateLeepPackageCommand) Setup() []string { return nil }
func (*UpdateLeepPackageCommand) Changed() bool   { return false }
func (*UpdateLeepPackageCommand) Name() string {
	// gg: "go get"
	return "gg"
}

var (
	packageArg    = command.ListArg[string]("PACKAGE", "Package name", 1, command.UnboundedList, command.SimpleDistinctCompleter[[]string](RelevantPackages...))
	lsRemoteRegex = regexp.MustCompile(`^([0-9a-f]+)\s+([^\s]+)$`)
)

func (*UpdateLeepPackageCommand) Node() command.Node {
	return command.SerialNodes(
		command.Description("gg updates go packages from the github.com/leep-frog repository"),
		packageArg,
		command.ExecutableProcessor(func(o command.Output, d *command.Data) ([]string, error) {
			var r []string
			for _, p := range packageArg.Get(d) {
				sc := &command.ShellCommand[[]string]{
					CommandName: "git",
					Args: []string{
						"ls-remote",
						fmt.Sprintf("git@github.com:leep-frog/%s.git", p),
					},
				}
				result, err := sc.Run(o, d)
				if err != nil {
					o.Stderrf("Failed to fetch commit info for package %q", p)
					continue
				}

				var sha string
				var branches []string
				for _, res := range result {
					m := lsRemoteRegex.FindStringSubmatch(res)
					branches = append(branches, m[2])
					if m != nil && (m[2] == "refs/heads/main" || m[2] == "refs/heads/master") {
						sha = m[1]
						break
					}
				}

				if sha == "" {
					o.Stderrf("No main or master branch for package %q: %v\n", p, branches)
					continue
				}

				r = append(r, fmt.Sprintf(`go get -v "github.com/leep-frog/%s@%s"`, p, sha))
			}
			return r, nil
		}),
		command.EchoExecuteData(),
	)
}
