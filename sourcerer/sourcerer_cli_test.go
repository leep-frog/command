package sourcerer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestExecute(t *testing.T) {
	type osCheck struct {
		WantExecuteData *command.ExecuteData
	}

	// We loop the OS here (and not in the test), so any underlying test data for
	// each test case is totally recreated (rather than re-using the same data
	// across tests which can be error prone and difficult to debug).
	for _, curOS := range []OS{Linux(), Windows()} {
		for _, test := range []struct {
			name     string
			etc      *command.ExecuteTestCase
			osChecks map[string]*osCheck
		}{
			{
				name: "Sources directory",
				etc: &command.ExecuteTestCase{
					Args: []string{
						filepath.Join("..", "testdata"),
						"ING",
					},
					WantData: &command.Data{Values: map[string]interface{}{
						sourcererDirArg.Name():    command.FilepathAbs(t, "..", "testdata"),
						sourcererSuffixArg.Name(): "ING",
					}},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								"pushd . > /dev/null",
								fmt.Sprintf(`cd %q`, command.FilepathAbs(t, "..", "testdata")),
								`local tmpFile="$(mktemp)"`,
								`go run . source "ING"  > $tmpFile && source $tmpFile `,
								"popd > /dev/null",
							},
						},
					},
					osWindows: {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								"Push-Location",
								fmt.Sprintf(`cd %q`, command.FilepathAbs(t, "..", "testdata")),
								`$Local:tmpFile = New-TemporaryFile`,
								`go run . source "ING"  > $Local:tmpFile`,
								`Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
								`. "$Local:tmpFile.ps1"`,
								"Pop-Location",
							},
						},
					},
				},
			},
			{
				name: "Sources directory with load only",
				etc: &command.ExecuteTestCase{
					Args: []string{
						filepath.Join("..", "testdata"),
						"ING",
						"--load-only",
					},
					WantData: &command.Data{Values: map[string]interface{}{
						sourcererDirArg.Name():    command.FilepathAbs(t, "..", "testdata"),
						sourcererSuffixArg.Name(): "ING",
						loadOnlyFlag.Name():       "--load-only",
					}},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								"pushd . > /dev/null",
								fmt.Sprintf(`cd %q`, command.FilepathAbs(t, "..", "testdata")),
								`local tmpFile="$(mktemp)"`,
								`go run . source "ING" --load-only > $tmpFile && source $tmpFile `,
								"popd > /dev/null",
							},
						},
					},
					osWindows: {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								"Push-Location",
								fmt.Sprintf(`cd %q`, command.FilepathAbs(t, "..", "testdata")),
								`$Local:tmpFile = New-TemporaryFile`,
								`go run . source "ING" --load-only > $Local:tmpFile`,
								`Copy-Item "$Local:tmpFile" "$Local:tmpFile.ps1"`,
								`. "$Local:tmpFile.ps1"`,
								"Pop-Location",
							},
						},
					},
				},
			},
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					t.Skipf("No osCheck set for this OS")
				}

				command.StubValue(t, &CurrentOS, curOS)
				cli := &SourcererCommand{}
				test.etc.Node = cli.Node()
				test.etc.WantExecuteData = oschk.WantExecuteData
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, cli)
			})
		}
	}
}

func TestMetadata(t *testing.T) {
	cli := &SourcererCommand{}
	if diff := cmp.Diff("sourcerer", cli.Name()); diff != "" {
		t.Errorf("Unexpected cli name (-want, +got):\n%s", diff)
	}

	if setup := cli.Setup(); setup != nil {
		t.Errorf("Expected cli.Setup() to be nil; got %v", setup)
	}
}
