package command

import "strconv"

type floatOperator struct{}

func (*floatOperator) toArgs(f float64) []string {
	return []string{strconv.FormatFloat(f, 'f', -1, 64)}
}

func (*floatOperator) fromArgs(sl []*string) (float64, error) {
	if len(sl) == 0 {
		return 0, nil
	}
	return strconv.ParseFloat(*sl[0], 64)
}

type floatListOperator struct{}

func (*floatListOperator) toArgs(fs []float64) []string {
	sl := make([]string, 0, len(fs))
	for _, f := range fs {
		sl = append(sl, strconv.FormatFloat(f, 'f', -1, 64))
	}
	return sl
}

func (*floatListOperator) fromArgs(sl []*string) ([]float64, error) {
	var err error
	var fs []float64
	for _, s := range sl {
		f, e := strconv.ParseFloat(*s, 64)
		if e != nil {
			err = e
		}
		fs = append(fs, f)
	}
	return fs, err
}
