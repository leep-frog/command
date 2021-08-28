package command

import (
	"fmt"
)

type ArgOpt interface {
	modifyArgOpt(*argOpt)
}

type argOpt struct {
	validators  []*validatorOption
	completor   *Completor
	transformer *simpleTransformer
	alias       *aliasOpt
	customSet   customSetter
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

type validatorOption struct {
	vt       ValueType
	validate func(*Value) error
}

func (vo *validatorOption) modifyArgOpt(ao *argOpt) {
	ao.validators = append(ao.validators, vo)
}

func (vo *validatorOption) modifyBashNode(bn *bashCommand) {
	bn.validators = append(bn.validators, vo)
}

func (vo *validatorOption) Validate(v *Value) error {
	if !v.IsType(vo.vt) {
		return fmt.Errorf("option can only be bound to arguments with type %v", vo.vt)
	}
	return vo.validate(v)
}
