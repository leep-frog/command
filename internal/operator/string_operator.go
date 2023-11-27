package operator

type stringOperator struct{}

func (*stringOperator) toArgs(s string) []string {
	return []string{s}
}

func (*stringOperator) fromArgs(sl []*string) (string, error) {
	if len(sl) == 0 {
		return "", nil
	}
	return *sl[0], nil
}

type stringListOperator struct{}

func (*stringListOperator) toArgs(sl []string) []string {
	return sl
}

func (*stringListOperator) fromArgs(sl []*string) ([]string, error) {
	r := make([]string, 0, len(sl))
	for _, s := range sl {
		r = append(r, *s)
	}
	return r, nil
}
