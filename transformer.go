package command

func FileTransformer() *Transformer[string] {
	return &Transformer[string]{
		t: func(s string) (string, error) {
			return filepathAbs(s)
		},
	}
}
