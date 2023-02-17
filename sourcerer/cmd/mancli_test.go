package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func TestMancli(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *command.ExecuteTestCase
	}{
		{
			name: "Gets usage",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"someCLI",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					usageCLIArg.Name(): "someCLI",
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						sourcerer.FileStringFromCLI("someCLI"),
						`if [ -z "$file" ]; then`,
						`  echo someCLI is not a CLI generated via github.com/leep-frog/command`,
						`  return 1`,
						`fi`,
						`  "$GOPATH/bin/_${file}_runner" usage someCLI`,
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run(test.name, func(t *testing.T) {
				cli := &UsageCommand{}
				test.etc.Node = cli.Node()
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, cli)
			})
		})
	}
}

func TestMancliMetadata(t *testing.T) {
	cli := &UsageCommand{}
	if diff := cmp.Diff("mancli", cli.Name()); diff != "" {
		t.Errorf("Unexpected cli name (-want, +got):\n%s", diff)
	}

	if setup := cli.Setup(); setup != nil {
		t.Errorf("Expected cli.Setup() to be nil; got %v", setup)
	}
}
