package command

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type operator[T any] interface {
	toArgs(T) []string
	fromArgs([]*string) (T, error)
}

var (
	intRegex = regexp.MustCompile("^-?[0-9](_?[0-9])*?$")
)

func parseInt(s string) (int, error) {
	// Replace all underscores *only* if it matches the pattern
	if intRegex.MatchString(s) {
		s = strings.ReplaceAll(s, "_", "")
	}
	return strconv.Atoi(s)
}

type intOperator struct{}

func (*intOperator) toArgs(i int) []string {
	return []string{fmt.Sprintf("%d", i)}
}

func (*intOperator) fromArgs(sl []*string) (int, error) {
	if len(sl) == 0 {
		return 0, nil
	}
	return parseInt(*sl[0])
}

type intListOperator struct{}

func (*intListOperator) toArgs(is []int) []string {
	sl := make([]string, 0, len(is))
	for _, i := range is {
		sl = append(sl, fmt.Sprintf("%d", i))
	}
	return sl
}

func (*intListOperator) fromArgs(sl []*string) ([]int, error) {
	var err error
	var is []int
	for _, s := range sl {
		i, e := parseInt(*s)
		if e != nil {
			err = e
		}
		is = append(is, i)
	}
	return is, err
}
