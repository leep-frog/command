package operator

type stringOperator struct{}

func (*stringOperator) ToArgs(s string) []string {
	return []string{s}
}

func (*stringOperator) FromArgs(sl []*string) (string, error) {
	if len(sl) == 0 {
		return "", nil
	}
	return *sl[0], nil
}

type stringListOperator struct{}

func (*stringListOperator) ToArgs(sl []string) []string {
	return sl
}

func (*stringListOperator) FromArgs(sl []*string) ([]string, error) {
	r := make([]string, 0, len(sl))
	for _, s := range sl {
		r = append(r, *s)
	}
	return r, nil
}
