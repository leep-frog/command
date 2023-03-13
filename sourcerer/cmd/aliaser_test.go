package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestAliaser(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *command.ExecuteTestCase
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
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`function _leep_frog_autocompleter {`,
						`  local file="$(type "$1" | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`  local tFile=$(mktemp)`,
						`  $GOPATH/bin/_${file}_runner autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`,
						`  local IFS='`,
						`';`,
						`  COMPREPLY=( $(cat $tFile) )`,
						`  rm $tFile`,
						`}`,
						``,
						`local file="$(type someCLI | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`if [ -z "$file" ]; then`,
						`  echo Provided CLI "someCLI" is not a CLI generated with github.com/leep-frog/command`,
						`  return 1`,
						`fi`,
						``,
						`alias -- some-alias="someCLI"`,
						`function _custom_autocomplete_for_alias_some-alias {`,
						`  _leep_frog_autocompleter "someCLI" `,
						`}`,
						``,
						`complete -F _custom_autocomplete_for_alias_some-alias -o nosort some-alias`,
						``,
						``,
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
				WantExecuteData: &command.ExecuteData{
					Executable: []string{
						`function _leep_frog_autocompleter {`,
						`  local file="$(type "$1" | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`  local tFile=$(mktemp)`,
						`  $GOPATH/bin/_${file}_runner autocomplete "$1" "$COMP_TYPE" $COMP_POINT "$COMP_LINE" "${@:2}" > $tFile`,
						`  local IFS='`,
						`';`,
						`  COMPREPLY=( $(cat $tFile) )`,
						`  rm $tFile`,
						`}`,
						``,
						`local file="$(type someCLI | head -n 1 | grep "is aliased to.*_custom_execute_" | grep "_custom_execute_[^[:space:]]*" -o | sed s/_custom_execute_//g)"`,
						`if [ -z "$file" ]; then`,
						`  echo Provided CLI "someCLI" is not a CLI generated with github.com/leep-frog/command`,
						`  return 1`,
						`fi`,
						``,
						`alias -- some-alias="someCLI \"un\" \"2\" \"trois\""`,
						`function _custom_autocomplete_for_alias_some-alias {`,
						`  _leep_frog_autocompleter "someCLI" "un" "2" "trois"`,
						`}`,
						``,
						`complete -F _custom_autocomplete_for_alias_some-alias -o nosort some-alias`,
						``,
						``,
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run(test.name, func(t *testing.T) {
				cli := &AliaserCommand{}
				test.etc.Node = cli.Node()
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, cli)
			})
		})
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
