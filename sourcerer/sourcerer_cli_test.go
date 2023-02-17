package sourcerer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *command.ExecuteTestCase
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
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"pushd . > /dev/null",
						fmt.Sprintf(`cd %q`, command.FilepathAbs(t, "..", "testdata")),
						`local tmpFile="$(mktemp)"`,
						`go run . "ING"  > $tmpFile && source $tmpFile `,
						"popd > /dev/null",
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
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						"pushd . > /dev/null",
						fmt.Sprintf(`cd %q`, command.FilepathAbs(t, "..", "testdata")),
						`local tmpFile="$(mktemp)"`,
						`go run . "ING" --load-only > $tmpFile && source $tmpFile `,
						"popd > /dev/null",
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cli := &SourcererCommand{}
			test.etc.Node = cli.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, nil, cli)
		})
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
