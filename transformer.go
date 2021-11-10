package command

func FileTransformer() ArgOpt {
	return &simpleTransformer{
		vt: StringType,
		t: func(v *Value) (*Value, error) {
			absStr, err := filepathAbs(v.ToString())
			return StringValue(absStr), err
		},
	}
}

func FileListTransformer() ArgOpt {
	return &simpleTransformer{
		vt: StringListType,
		t: func(v *Value) (*Value, error) {
			// TODO: value len function?
			l := make([]string, 0, len(v.ToStringList()))
			for i, s := range v.ToStringList() {
				absStr, err := filepathAbs(s)
				if err != nil {
					return StringListValue(append(l, (v.ToStringList())[i:]...)...), err
				}
				l = append(l, absStr)
			}
			return StringListValue(l...), nil
		},
	}
}
