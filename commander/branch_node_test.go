package commander

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/testutil"
)

func TestIsBranch(t *testing.T) {
	for _, test := range []struct {
		name string
		b    *BranchNode
		s    string
		want bool
	}{
		{
			name: "false for empty branch node",
			b:    &BranchNode{},
		},
		{
			name: "false for empty string with populated branch node",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un":   nil,
					"deux": nil,
				},
			},
		},
		{
			name: "true for present branch when no synonyms",
			s:    "un",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un": nil,
				},
			},
			want: true,
		},
		{
			name: "false when not branch hit",
			s:    "deux",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un": nil,
				},
			},
		},
		{
			name: "true for present branch when synonyms",
			s:    "un",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un u": nil,
				},
			},
			want: true,
		},
		{
			name: "true for present branch when synonyms",
			s:    "u",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un u": nil,
				},
			},
			want: true,
		},
		{
			name: "true for present branch when multiple branches",
			s:    "deux",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un u":   nil,
					"deux d": nil,
					"trois":  nil,
				},
			},
			want: true,
		},
		{
			name: "true when Synonyms is set",
			s:    "other",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un u":   nil,
					"deux d": nil,
					"trois":  nil,
				},
				Synonyms: map[string]string{
					"other": "trois",
				},
			},
			want: true,
		},
		{
			name: "false when all are set but no match",
			s:    "another",
			b: &BranchNode{
				Branches: map[string]command.Node{
					"un u":   nil,
					"deux d": nil,
					"trois":  nil,
				},
				Synonyms: map[string]string{
					"other": "trois",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testutil.Cmp(t, fmt.Sprintf("IsBranchNode(%s) returned incorrect value", test.s), test.want, test.b.IsBranch(test.s))
		})
	}
}
