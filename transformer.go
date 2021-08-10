package command

func FileTransformer() ArgOpt {
	return &simpleTransformer{
		vt: StringType,
		t: func(v *Value) (*Value, error) {
			absStr, err := filepathAbs(v.String())
			return StringValue(absStr), err
		},
	}
}

func FileListTransformer() ArgOpt {
	return &simpleTransformer{
		vt: StringListType,
		t: func(v *Value) (*Value, error) {
			l := make([]string, 0, len(v.StringList()))
			for i, s := range v.StringList() {
				absStr, err := filepathAbs(s)
				if err != nil {
					return StringListValue(append(l, (v.StringList())[i:]...)...), err
				}
				l = append(l, absStr)
			}
			return StringListValue(l...), nil
		},
	}
}
