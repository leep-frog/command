package main

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func TestMancli(t *testing.T) {
	type osCheck struct {
		WantExecuteData *command.ExecuteData
	}
	for _, curOS := range []sourcerer.OS{sourcerer.Linux(), sourcerer.Windows()} {
		for _, test := range []struct {
			name     string
			etc      *command.ExecuteTestCase
			osChecks map[string]*osCheck
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
				},
				osChecks: map[string]*osCheck{
					"linux": {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								sourcerer.FileStringFromCLI("someCLI"),
								`if [ -z "$file" ]; then`,
								`  echo someCLI is not a CLI generated via github.com/leep-frog/command`,
								`  return 1`,
								`fi`,
								`  "$GOPATH/bin/_${file}_runner" usage someCLI `,
							},
						},
					},
					"windows": {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								`if (!(Test-Path alias:someCLI) -or !(Get-Alias someCLI | where {$_.DEFINITION -match "_custom_execute"}).NAME) {`,
								`  throw "The CLI provided (someCLI) is not a sourcerer-generated command"`,
								`}`,
								`$Local:targetName = (Get-Alias someCLI).DEFINITION.split("_")[3]`,
								`Invoke-Expression "$env:GOPATH\bin\_${Local:targetName}_runner.exe usage someCLI "`,
							},
						},
					},
				},
			},
			{
				name: "Gets usage with args",
				etc: &command.ExecuteTestCase{
					Args: []string{
						"someCLI",
						"arg1",
						"arg 2",
						`arg " 3`,
						`arg'4 `,
					},
					WantData: &command.Data{Values: map[string]interface{}{
						usageCLIArg.Name(): "someCLI",
						extraMancliArgs.Name(): []string{
							"arg1",
							"arg 2",
							`arg " 3`,
							`arg'4 `,
						},
					}},
				},
				osChecks: map[string]*osCheck{
					"linux": {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								sourcerer.FileStringFromCLI("someCLI"),
								`if [ -z "$file" ]; then`,
								`  echo someCLI is not a CLI generated via github.com/leep-frog/command`,
								`  return 1`,
								`fi`,
								`  "$GOPATH/bin/_${file}_runner" usage someCLI "arg1" "arg 2" "arg \" 3" "arg'4 "`,
							},
						},
					},
					"windows": {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								`if (!(Test-Path alias:someCLI) -or !(Get-Alias someCLI | where {$_.DEFINITION -match "_custom_execute"}).NAME) {`,
								`  throw "The CLI provided (someCLI) is not a sourcerer-generated command"`,
								`}`,
								`$Local:targetName = (Get-Alias someCLI).DEFINITION.split("_")[3]`,
								`Invoke-Expression "$env:GOPATH\bin\_${Local:targetName}_runner.exe usage someCLI arg1 arg_2 arg___3 arg_4_"`,
							},
						},
					},
				},
			},
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					t.Fatalf("No osCheck set for this OS")
				}
				command.StubValue(t, &sourcerer.CurrentOS, curOS)

				cli := &UsageCommand{}
				test.etc.Node = cli.Node()
				test.etc.WantExecuteData = oschk.WantExecuteData
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, cli)
			})
		}
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
