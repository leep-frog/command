package operator

import (
	"strconv"
)

type boolOperator struct{}

func (*boolOperator) ToArgs(b bool) []string {
	return []string{strconv.FormatBool(b)}
}

func (*boolOperator) FromArgs(sl []*string) (bool, error) {
	if len(sl) == 0 {
		return false, nil
	}
	return strconv.ParseBool(*sl[0])
}
