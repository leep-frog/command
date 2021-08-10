package command

import (
	"fmt"
	"strings"
)

type ArgOpt interface {
	modifyArgOpt(*argOpt)
}

type argOpt struct {
	validators  []ArgValidator
	completor   *Completor
	transformer *simpleTransformer
	alias       *aliasOpt
	customSet   customSetter
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

type ArgValidator interface {
	Validate(*Value) error
	ArgOpt
}

type validatorOption struct {
	vt       ValueType
	validate func(*Value) error
}

func (vo *validatorOption) modifyArgOpt(ao *argOpt) {
	ao.validators = append(ao.validators, vo)
}

func (vo *validatorOption) Validate(v *Value) error {
	if !v.IsType(vo.vt) {
		return fmt.Errorf("option can only be bound to arguments with type %v", vo.vt)
	}
	return vo.validate(v)
}

// String options
func StringOption(f func(string) bool, err error) ArgOpt {
	validator := func(v *Value) error {
		if !f(v.String()) {
			return err
		}
		return nil
	}
	return &validatorOption{
		vt:       StringType,
		validate: validator,
	}
}

func Contains(s string) ArgOpt {
	return StringOption(
		func(vs string) bool { return strings.Contains(vs, s) },
		fmt.Errorf("[Contains] value doesn't contain substring %q", s),
	)
}

func MinLength(length int) ArgOpt {
	var plural string
	if length != 1 {
		plural = "s"
	}
	return StringOption(
		func(vs string) bool { return len(vs) >= length },
		fmt.Errorf("[MinLength] value must be at least %d character%s", length, plural),
	)
}

// Int options
func IntOption(f func(int) bool, err error) ArgOpt {
	validator := func(v *Value) error {
		if !f(v.Int()) {
			return err
		}
		return nil
	}
	return &validatorOption{
		vt:       IntType,
		validate: validator,
	}
}

func IntEQ(i int) ArgOpt {
	return IntOption(
		func(vi int) bool { return vi == i },
		fmt.Errorf("[IntEQ] value isn't equal to %d", i),
	)
}

func IntNE(i int) ArgOpt {
	return IntOption(
		func(vi int) bool { return vi != i },
		fmt.Errorf("[IntNE] value isn't not equal to %d", i),
	)
}

func IntLT(i int) ArgOpt {
	return IntOption(
		func(vi int) bool { return vi < i },
		fmt.Errorf("[IntLT] value isn't less than %d", i),
	)
}

func IntLTE(i int) ArgOpt {
	return IntOption(
		func(vi int) bool { return vi <= i },
		fmt.Errorf("[IntLTE] value isn't less than or equal to %d", i),
	)
}

func IntGT(i int) ArgOpt {
	return IntOption(
		func(vi int) bool { return vi > i },
		fmt.Errorf("[IntGT] value isn't greater than %d", i),
	)
}

func IntGTE(i int) ArgOpt {
	return IntOption(
		func(vi int) bool { return vi >= i },
		fmt.Errorf("[IntGTE] value isn't greater than or equal to %d", i),
	)
}

func IntPositive() ArgOpt {
	return IntOption(
		func(vi int) bool { return vi > 0 },
		fmt.Errorf("[IntPositive] value isn't positive"),
	)
}

func IntNonNegative() ArgOpt {
	return IntOption(
		func(vi int) bool { return vi >= 0 },
		fmt.Errorf("[IntNonNegative] value isn't non-negative"),
	)
}

func IntNegative() ArgOpt {
	return IntOption(
		func(vi int) bool { return vi < 0 },
		fmt.Errorf("[IntNegative] value isn't negative"),
	)
}

// Float options
func FloatOption(f func(float64) bool, err error) ArgOpt {
	validator := func(v *Value) error {
		if !f(v.Float()) {
			return err
		}
		return nil
	}
	return &validatorOption{
		vt:       FloatType,
		validate: validator,
	}
}

func FloatEQ(f float64) ArgOpt {
	return FloatOption(
		func(vf float64) bool { return vf == f },
		fmt.Errorf("[FloatEQ] value isn't equal to %0.2f", f),
	)
}

func FloatNE(f float64) ArgOpt {
	return FloatOption(
		func(vf float64) bool { return vf != f },
		fmt.Errorf("[FloatNE] value isn't not equal to %0.2f", f),
	)
}

func FloatLT(f float64) ArgOpt {
	return FloatOption(
		func(vf float64) bool { return vf < f },
		fmt.Errorf("[FloatLT] value isn't less than %0.2f", f),
	)
}

func FloatLTE(f float64) ArgOpt {
	return FloatOption(
		func(vf float64) bool { return vf <= f },
		fmt.Errorf("[FloatLTE] value isn't less than or equal to %0.2f", f),
	)
}

func FloatGT(f float64) ArgOpt {
	return FloatOption(
		func(vf float64) bool { return vf > f },
		fmt.Errorf("[FloatGT] value isn't greater than %0.2f", f),
	)
}

func FloatGTE(f float64) ArgOpt {
	return FloatOption(
		func(vf float64) bool { return vf >= f },
		fmt.Errorf("[FloatGTE] value isn't greater than or equal to %0.2f", f),
	)
}

func FloatPositive() ArgOpt {
	return FloatOption(
		func(vi float64) bool { return vi > 0 },
		fmt.Errorf("[FloatPositive] value isn't positive"),
	)
}

func FloatNonNegative() ArgOpt {
	return FloatOption(
		func(vi float64) bool { return vi >= 0 },
		fmt.Errorf("[FloatNonNegative] value isn't non-negative"),
	)
}

func FloatNegative() ArgOpt {
	return FloatOption(
		func(vi float64) bool { return vi < 0 },
		fmt.Errorf("[FloatNegative] value isn't negative"),
	)
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

func FileTransformer() ArgOpt {
	return &simpleTransformer{
		vt: StringType,
		t: func(v *Value) (*Value, error) {
			absStr, err := filepathAbs(v.String())
			return StringValue(absStr), err
		},
	}
}
