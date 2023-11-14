package sourcerer

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestAliaser(t *testing.T) {
	type osCheck struct {
		WantExecuteData *command.ExecuteData
	}
	fakeGoExecutableFilePath := command.TempFile(t, "leepFrogSourcerer-test")
	for _, curOS := range []OS{Linux(), Windows()} {
		for _, test := range []struct {
			name     string
			etc      *command.ExecuteTestCase
			osChecks map[string]*osCheck
		}{
			{
				name: "Creates aliaser with no passthrough args",
				etc: &command.ExecuteTestCase{
					Args: []string{
						"some-alias",
						"someCLI",
					},
					WantData: &command.Data{Values: map[string]interface{}{
						aliasArg.Name():    "some-alias",
						aliasCLIArg.Name(): "someCLI",
					}},
				},
				osChecks: map[string]*osCheck{
					"linux": {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								`function _leep_frog_autocompleter {`,
								`  local tFile=$(mktemp)`,
								fmt.Sprintf(`  %s autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`, fakeGoExecutableFilePath.Name()),
								`  local IFS='`,
								`';`,
								`  COMPREPLY=( $(cat $tFile) )`,
								`  rm $tFile`,
								`}`,
								``,
								`local file="$(type someCLI | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
								`if [ -z "$file" ]; then`,
								`  local file="$(type someCLI | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
								`  if [ -z "$file" ]; then`,
								`    echo Provided CLI "someCLI" is not a CLI generated with github.com/leep-frog/command`,
								`    return 1`,
								`  fi`,
								`fi`,
								``,
								``,
								`alias -- some-alias="someCLI"`,
								`function _custom_autocomplete_for_alias_some-alias {`,
								`  _leep_frog_autocompleter "someCLI" `,
								`}`,
								``,
								`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_some-alias -o nosort some-alias`,
								``,
							},
						},
					},
					"windows": {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								`if (!(Test-Path alias:someCLI) -or !(Get-Alias someCLI | where {$_.DEFINITION -match "_custom_execute"}).NAME) {`,
								`  throw "The CLI provided (someCLI) is not a sourcerer-generated command"`,
								"}",
								"function _sourcerer_alias_execute_some-alias {",
								`  $Local:functionName = "$((Get-Alias "someCLI").DEFINITION)"`,
								"  Invoke-Expression ($Local:functionName $args)",
								"}",
								"$_sourcerer_alias_autocomplete_some-alias = {",
								"  param($wordToComplete, $commandAst, $compPoint)",
								fmt.Sprintf(`  (Invoke-Expression '& %s autocomplete "someCLI" "0" $compPoint "$commandAst" ') | ForEach-Object {`, fakeGoExecutableFilePath.Name()),
								`    "$_"`,
								"  }",
								"}",
								"(Get-Alias) | Where { $_.NAME -match '^some-alias$'} | ForEach-Object { del alias:${_} -Force }",
								"Set-Alias some-alias _sourcerer_alias_execute_some-alias",
								"Register-ArgumentCompleter -CommandName some-alias -ScriptBlock $_sourcerer_alias_autocomplete_some-alias",
							},
						},
					},
				},
			},
			{
				name: "Creates aliaser with passthrough args",
				etc: &command.ExecuteTestCase{
					Args: []string{
						"some-alias",
						"someCLI",
						"un",
						"2",
						"trois",
					},
					WantData: &command.Data{Values: map[string]interface{}{
						aliasArg.Name():    "some-alias",
						aliasCLIArg.Name(): "someCLI",
						aliasPTArg.Name(): []string{
							"un",
							"2",
							"trois",
						},
					}},
				},
				osChecks: map[string]*osCheck{
					"linux": {
						WantExecuteData: &command.ExecuteData{
							Executable: []string{
								`function _leep_frog_autocompleter {`,
								`  local tFile=$(mktemp)`,
								fmt.Sprintf(`  %s autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`, fakeGoExecutableFilePath.Name()),
								`  local IFS='`,
								`';`,
								`  COMPREPLY=( $(cat $tFile) )`,
								`  rm $tFile`,
								`}`,
								``,
								`local file="$(type someCLI | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
								`if [ -z "$file" ]; then`,
								`  local file="$(type someCLI | head -n 1 | grep "is an alias for.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
								`  if [ -z "$file" ]; then`,
								`    echo Provided CLI "someCLI" is not a CLI generated with github.com/leep-frog/command`,
								`    return 1`,
								`  fi`,
								`fi`,
								``,
								``,
								`alias -- some-alias="someCLI \"un\" \"2\" \"trois\""`,
								`function _custom_autocomplete_for_alias_some-alias {`,
								`  _leep_frog_autocompleter "someCLI" "un" "2" "trois"`,
								`}`,
								``,
								`(type complete > /dev/null 2>&1) && complete -F _custom_autocomplete_for_alias_some-alias -o nosort some-alias`,
								``,
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

				cli := &AliaserCommand{fakeGoExecutableFilePath.Name()}
				test.etc.Node = cli.Node()
				test.etc.WantExecuteData = oschk.WantExecuteData
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, cli)
			})
		}
	}
}

func TestAliaserMetadata(t *testing.T) {
	cli := &AliaserCommand{}
	if diff := cmp.Diff("aliaser", cli.Name()); diff != "" {
		t.Errorf("Unexpected cli name (-want, +got):\n%s", diff)
	}

	if setup := cli.Setup(); setup != nil {
		t.Errorf("Expected cli.Setup() to be nil; got %v", setup)
	}
}
