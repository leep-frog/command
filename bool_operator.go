package command

import "strconv"

var (
	boolStringValues = []string{
		"1",
		"t",
		"T",
		"true",
		"TRUE",
		"True",
		"0",
		"f",
		"F",
		"false",
		"FALSE",
		"False",
	}
)

type boolOperator struct{}

func (*boolOperator) toArgs(b bool) []string {
	return []string{strconv.FormatBool(b)}
}

func (*boolOperator) fromArgs(sl []*string) (bool, error) {
	if len(sl) == 0 {
		return false, nil
	}
	return strconv.ParseBool(*sl[0])
}
