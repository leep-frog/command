package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func rc(pkg string) *command.RunContents {
	return &command.RunContents{
		Name: "git",
		Args: []string{
			"ls-remote",
			fmt.Sprintf("git@github.com:leep-frog/%s.git", pkg),
		},
	}
}

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
				WantRunContents: []*command.RunContents{
					rc("some-package"),
				},
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{
							"1234567890abcdef HEAD",
							"246810 refs/heads/main",
						},
					},
				},
				WantStdout: strings.Join([]string{
					`go get -v "github.com/leep-frog/some-package@246810"`,
					``,
				}, "\n"),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`go get -v "github.com/leep-frog/some-package@246810"`,
					},
				},
			},
		},
		{
			name: "Handles no known main branch",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"some-package",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					packageArg.Name(): []string{
						"some-package",
					},
				}},
				WantRunContents: []*command.RunContents{
					rc("some-package"),
				},
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{
							"1234567890abcdef HEAD",
							"246810abdabdbadb FOOT",
						},
					},
				},
				WantStderr: strings.Join([]string{
					`No main or master branch for package "some-package": [HEAD FOOT]`,
					``,
				}, "\n"),
			},
		},
		{
			name: "Handles shell command error",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"some-package",
				},
				WantData: &command.Data{Values: map[string]interface{}{
					packageArg.Name(): []string{
						"some-package",
					},
				}},
				WantRunContents: []*command.RunContents{
					rc("some-package"),
				},
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{
							"1234567890abcdef HEAD",
							"246810abdabdbadb refs/heads/main",
						},
						Stderr: []string{"rats"},
						Err:    fmt.Errorf("oops"),
					},
				},
				WantStderr: strings.Join([]string{
					"rats",
					`Failed to fetch commit info for package "some-package"`,
				}, "\n"),
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
				WantRunContents: []*command.RunContents{
					rc("ups"),
					rc("fedex"),
					rc("usps"),
				},
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{
							"1a refs/heads/main",
						},
					},
					{
						Stdout: []string{
							"2b refs/heads/master",
						},
					},
					{
						Stdout: []string{
							"3c refs/heads/main",
						},
					},
				},
				WantStdout: strings.Join([]string{
					`go get -v "github.com/leep-frog/ups@1a"`,
					`go get -v "github.com/leep-frog/fedex@2b"`,
					`go get -v "github.com/leep-frog/usps@3c"`,
					``,
				}, "\n"),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`go get -v "github.com/leep-frog/ups@1a"`,
						`go get -v "github.com/leep-frog/fedex@2b"`,
						`go get -v "github.com/leep-frog/usps@3c"`,
					},
				},
			},
		},
		{
			name: "Gets multiple packages with errors",
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
				WantRunContents: []*command.RunContents{
					rc("ups"),
					rc("fedex"),
					rc("usps"),
				},
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{
							"1a refs/heads/main",
						},
					},
					{
						Stdout: []string{
							"2b refs/heads/main",
						},
						Stderr: []string{"who"},
						Err:    fmt.Errorf("what"),
					},
					{
						Stdout: []string{
							"3c refs/heads/master",
						},
					},
				},
				WantStdout: strings.Join([]string{
					`go get -v "github.com/leep-frog/ups@1a"`,
					`go get -v "github.com/leep-frog/usps@3c"`,
					``,
				}, "\n"),
				WantStderr: strings.Join([]string{
					"who",
					`Failed to fetch commit info for package "fedex"`,
				}, "\n"),
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`go get -v "github.com/leep-frog/ups@1a"`,
						`go get -v "github.com/leep-frog/usps@3c"`,
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
