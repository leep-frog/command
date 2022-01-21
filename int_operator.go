package command

import (
	"fmt"
	"strconv"
)

type operator[T any] interface {
	toArgs(T) []string
	fromArgs([]*string) (T, error)
}

type intOperator struct {}

func (*intOperator) toArgs(i int) []string {
	return []string{strconv.Itoa(i)}
}

func (*intOperator) fromArgs(sl []*string) (int, error) {
	if len(sl) == 0 {
		return 0, nil
	}
	return strconv.Atoi(*sl[0])
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
		i, e := strconv.Atoi(*s)
		if e != nil {
			err = e
		}
		is = append(is, i)
	}
	return is, err
}
