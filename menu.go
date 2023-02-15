package command

// MenuFlag returns an `Arg` that is required to be one of the provided choices.
func MenuFlag[T comparable](name string, shortName rune, desc string, choices ...T) FlagWithType[T] {
	var strChoices []string
	op := getOperator[T]()
	for _, c := range choices {
		strChoices = append(strChoices, op.toArgs(c)...)
	}
	return Flag[T](name, shortName, desc, SimpleCompleter[T](strChoices...), InList(choices...))
}

// MenuArg returns an `Arg` that is required to be one of the provided choices.
func MenuArg[T comparable](name, desc string, choices ...T) *Argument[T] {
	var strChoices []string
	op := getOperator[T]()
	for _, c := range choices {
		strChoices = append(strChoices, op.toArgs(c)...)
	}
	return Arg[T](name, desc, SimpleCompleter[T](strChoices...), InList(choices...))
}
