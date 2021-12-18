package command

import (
	"fmt"
)

type ArgOpt interface {
	modifyArgOpt(*argOpt)
}

type argOpt struct {
	validators  []*ValidatorOption
	completor   *Completor
	transformer *simpleTransformer
	alias       *aliasOpt
	customSet   customSetter
	_default    *Value
	breaker     *ListBreaker

	hiddenUsage bool
}

func newArgOpt(opts ...ArgOpt) *argOpt {
	ao := &argOpt{}
	for _, opt := range opts {
		opt.modifyArgOpt(ao)
	}
	return ao
}

func AliasOpt(name string, ac AliasCLI) ArgOpt {
	return &aliasOpt{
		AliasName: name,
		AliasCLI:  ac,
	}
}

type aliasOpt struct {
	AliasName string
	AliasCLI  AliasCLI
}

func (ao *aliasOpt) modifyArgOpt(argO *argOpt) {
	argO.alias = ao
}

func CustomSetter(f func(*Value, *Data)) ArgOpt {
	cs := customSetter(f)
	return &cs
}

type customSetter func(*Value, *Data)

func (cs *customSetter) modifyArgOpt(ao *argOpt) {
	ao.customSet = *cs
}

type simpleTransformer struct {
	vt ValueType
	t  func(v *Value) (*Value, error)
	// forComplete is whether or not the value
	// should be transformed during completions.
	forComplete bool
}

func (st *simpleTransformer) modifyArgOpt(ao *argOpt) {
	ao.transformer = st
}

func Transformer(vt ValueType, f func(*Value) (*Value, error), forComplete bool) ArgOpt {
	return &simpleTransformer{
		vt:          vt,
		t:           f,
		forComplete: forComplete,
	}
}

type ValidatorOption struct {
	vt       ValueType
	validate func(*Value) error
}

func (vo *ValidatorOption) modifyArgOpt(ao *argOpt) {
	ao.validators = append(ao.validators, vo)
}

func (vo *ValidatorOption) modifyBashNode(bn *bashCommand) {
	bn.validators = append(bn.validators, vo)
}

func (vo *ValidatorOption) Validate(v *Value) error {
	if !v.IsType(vo.vt) {
		return fmt.Errorf("option can only be bound to arguments with type %v", vo.vt)
	}
	return vo.validate(v)
}

// Default arg option
type defaultArgOpt struct {
	v *Value
}

func (dao *defaultArgOpt) modifyArgOpt(ao *argOpt) {
	ao._default = dao.v
}

func StringDefault(s string) ArgOpt {
	return &defaultArgOpt{StringValue(s)}
}

func IntDefault(i int) ArgOpt {
	return &defaultArgOpt{IntValue(i)}
}

func FloatDefault(f float64) ArgOpt {
	return &defaultArgOpt{FloatValue(f)}
}

func StringListDefault(ss ...string) ArgOpt {
	return &defaultArgOpt{StringListValue(ss...)}
}

func IntListDefault(is ...int) ArgOpt {
	return &defaultArgOpt{IntListValue(is...)}
}

func FloatListDefault(fs ...float64) ArgOpt {
	return &defaultArgOpt{FloatListValue(fs...)}
}
