package command

import "strconv"

var (
	boolStringMap = map[string]bool{
		"1":     true,
		"t":     true,
		"T":     true,
		"true":  true,
		"TRUE":  true,
		"True":  true,
		"0":     false,
		"f":     false,
		"F":     false,
		"false": false,
		"FALSE": false,
		"False": false,
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
