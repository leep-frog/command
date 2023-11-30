package operator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// IntRegex is the regex checked for int `Args`. Underscores will be removed
	// if they are in a valid position (not the first or last character).
	IntRegex = regexp.MustCompile("^-?[0-9](_?[0-9])*?$")
)

func ParseInt(s string) (int, error) {
	// Replace all underscores *only* if it matches the pattern
	if IntRegex.MatchString(s) {
		s = strings.ReplaceAll(s, "_", "")
	}
	return strconv.Atoi(s)
}

type intOperator struct{}

func (*intOperator) ToArgs(i int) []string {
	return []string{fmt.Sprintf("%d", i)}
}

func (*intOperator) FromArgs(sl []*string) (int, error) {
	if len(sl) == 0 {
		return 0, nil
	}
	return ParseInt(*sl[0])
}

type intListOperator struct{}

func (*intListOperator) ToArgs(is []int) []string {
	sl := make([]string, 0, len(is))
	for _, i := range is {
		sl = append(sl, fmt.Sprintf("%d", i))
	}
	return sl
}

func (*intListOperator) FromArgs(sl []*string) ([]int, error) {
	var err error
	var is []int
	for _, s := range sl {
		i, e := ParseInt(*s)
		if e != nil {
			err = e
		}
		is = append(is, i)
	}
	return is, err
}
