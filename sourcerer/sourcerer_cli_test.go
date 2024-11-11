package sourcerer

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
	"github.com/leep-frog/command/internal/testutil"
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
			etc      *commandtest.ExecuteTestCase
			osChecks map[string]*osCheck
		}{
			/*{
				name: "Sources directory",
				etc: &commandtest.ExecuteTestCase{
					Args: []string{
						filepath.Join("..", "commander", "testdata"),
						"ING",
						filepath.Join("cli-output-dir"),
					},
					WantData: &command.Data{Values: map[string]interface{}{
						sourcererSourceDirArg.Name(): testutil.FilepathAbs(t, "..", "commander", "testdata"),
						targetNameArg.Name():         "ING",
					}},
				},
				osChecks: map[string]*osCheck{
					osLinux: {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								"pushd . > /dev/null",
								fmt.Sprintf(`cd %q`, testutil.FilepathAbs(t, "..", "commander", "testdata")),
								fmt.Sprintf(`go run . source %q`, testutil.FilepathAbs(t, "cli-output-dir")),
								fmt.Sprintf("source %q", testutil.FilepathAbs(t, "cli-output-dir", "ING_loader.sh")),
								"popd > /dev/null",
							},
						},
					},
					osWindows: {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								"Push-Location",
								fmt.Sprintf(`cd %q`, testutil.FilepathAbs(t, "..", "commander", "testdata")),
								fmt.Sprintf(`go run . source %q`, testutil.FilepathAbs(t, "cli-output-dir")),
								fmt.Sprintf(". %q", testutil.FilepathAbs(t, "cli-output-dir", "ING_loader.ps1")),
								"Pop-Location",
							},
						},
					},
				},
			},*/
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				oschk, ok := test.osChecks[curOS.Name()]
				if !ok {
					t.Fatalf("No osCheck set for this OS")
				}

				testutil.StubValue(t, &CurrentOS, curOS)
				cli := &SourcererCommand{}
				test.etc.Node = cli.Node()
				test.etc.WantExecuteData = oschk.WantExecuteData
				commandertest.ExecuteTest(t, test.etc)
				commandertest.ChangeTest(t, nil, cli)
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
