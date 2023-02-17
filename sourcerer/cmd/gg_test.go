package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestGG(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *command.ExecuteTestCase
	}{
		{
			name: "Gets a package",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"some-package",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					packageArg.Name(): []string{
						"some-package",
					},
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`local commitSha="$(git ls-remote git@github.com:leep-frog/some-package.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/some-package@$commitSha"`,
					},
				},
			},
		},
		{
			name: "Gets multiple packages",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"ups",
					"fedex",
					"usps",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					packageArg.Name(): []string{
						"ups",
						"fedex",
						"usps",
					},
				}},
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`local commitSha="$(git ls-remote git@github.com:leep-frog/ups.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/ups@$commitSha"`,
						`local commitSha="$(git ls-remote git@github.com:leep-frog/fedex.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/fedex@$commitSha"`,
						`local commitSha="$(git ls-remote git@github.com:leep-frog/usps.git | grep ma[is][nt] | awk '{print $1}')"`,
						`go get -v "github.com/leep-frog/usps@$commitSha"`,
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run(test.name, func(t *testing.T) {
				cli := &UpdateLeepPackageCommand{}
				test.etc.Node = cli.Node()
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, cli)
			})
		})
	}
}

func TestGGMetadata(t *testing.T) {
	cli := &UpdateLeepPackageCommand{}
	if diff := cmp.Diff("gg", cli.Name()); diff != "" {
		t.Errorf("Unexpected cli name (-want, +got):\n%s", diff)
	}

	if setup := cli.Setup(); setup != nil {
		t.Errorf("Expected cli.Setup() to be nil; got %v", setup)
	}
}
