package command

import (
	"fmt"
	"strings"
)

type ArgOpt struct {
	Validators  []ArgValidator
	Completor   *Completor
	Transformer ArgTransformer
	Alias       *AliasOpt
}

type AliasOpt struct {
	AliasName string
	AliasCLI  AliasCLI
}

func NewArgOpt(c *Completor, t ArgTransformer, v ...ArgValidator) *ArgOpt {
	return &ArgOpt{
		Completor:   c,
		Validators:  v,
		Transformer: t,
	}
}

func SimpleTransformer(vt ValueType, f func(v *Value) (*Value, error)) ArgTransformer {
	return &simpleTransformer{
		vt: vt,
		t:  f,
	}
}

type simpleTransformer struct {
	vt ValueType
	t  func(v *Value) (*Value, error)
}

func (st *simpleTransformer) ValueType() ValueType {
	return st.vt
}

func (st *simpleTransformer) Transform(v *Value) (*Value, error) {
	return st.t(v)
}

type ArgTransformer interface {
	ValueType() ValueType
	Transform(v *Value) (*Value, error)
}

type ArgValidator interface {
	Validate(*Value) error
}

type validatorOption struct {
	vt       ValueType
	validate func(*Value) error
}

func (vo *validatorOption) Validate(v *Value) error {
	if !v.IsType(vo.vt) {
		return fmt.Errorf("option can only be bound to arguments with type %v", vo.vt)
	}
	return vo.validate(v)
}

// String options
func StringOption(f func(string) bool, err error) ArgValidator {
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

func Contains(s string) ArgValidator {
	return StringOption(
		func(vs string) bool { return strings.Contains(vs, s) },
		fmt.Errorf("[Contains] value doesn't contain substring %q", s),
	)
}

func MinLength(length int) ArgValidator {
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
func IntOption(f func(int) bool, err error) ArgValidator {
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

func IntEQ(i int) ArgValidator {
	return IntOption(
		func(vi int) bool { return vi == i },
		fmt.Errorf("[IntEQ] value isn't equal to %d", i),
	)
}

func IntNE(i int) ArgValidator {
	return IntOption(
		func(vi int) bool { return vi != i },
		fmt.Errorf("[IntNE] value isn't not equal to %d", i),
	)
}

func IntLT(i int) ArgValidator {
	return IntOption(
		func(vi int) bool { return vi < i },
		fmt.Errorf("[IntLT] value isn't less than %d", i),
	)
}

func IntLTE(i int) ArgValidator {
	return IntOption(
		func(vi int) bool { return vi <= i },
		fmt.Errorf("[IntLTE] value isn't less than or equal to %d", i),
	)
}

func IntGT(i int) ArgValidator {
	return IntOption(
		func(vi int) bool { return vi > i },
		fmt.Errorf("[IntGT] value isn't greater than %d", i),
	)
}

func IntGTE(i int) ArgValidator {
	return IntOption(
		func(vi int) bool { return vi >= i },
		fmt.Errorf("[IntGTE] value isn't greater than or equal to %d", i),
	)
}

func IntPositive() ArgValidator {
	return IntOption(
		func(vi int) bool { return vi > 0 },
		fmt.Errorf("[IntPositive] value isn't positive"),
	)
}

func IntNonNegative() ArgValidator {
	return IntOption(
		func(vi int) bool { return vi >= 0 },
		fmt.Errorf("[IntNonNegative] value isn't non-negative"),
	)
}

func IntNegative() ArgValidator {
	return IntOption(
		func(vi int) bool { return vi < 0 },
		fmt.Errorf("[IntNegative] value isn't negative"),
	)
}

// Float options
func FloatOption(f func(float64) bool, err error) ArgValidator {
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

func FloatEQ(f float64) ArgValidator {
	return FloatOption(
		func(vf float64) bool { return vf == f },
		fmt.Errorf("[FloatEQ] value isn't equal to %0.2f", f),
	)
}

func FloatNE(f float64) ArgValidator {
	return FloatOption(
		func(vf float64) bool { return vf != f },
		fmt.Errorf("[FloatNE] value isn't not equal to %0.2f", f),
	)
}

func FloatLT(f float64) ArgValidator {
	return FloatOption(
		func(vf float64) bool { return vf < f },
		fmt.Errorf("[FloatLT] value isn't less than %0.2f", f),
	)
}

func FloatLTE(f float64) ArgValidator {
	return FloatOption(
		func(vf float64) bool { return vf <= f },
		fmt.Errorf("[FloatLTE] value isn't less than or equal to %0.2f", f),
	)
}

func FloatGT(f float64) ArgValidator {
	return FloatOption(
		func(vf float64) bool { return vf > f },
		fmt.Errorf("[FloatGT] value isn't greater than %0.2f", f),
	)
}

func FloatGTE(f float64) ArgValidator {
	return FloatOption(
		func(vf float64) bool { return vf >= f },
		fmt.Errorf("[FloatGTE] value isn't greater than or equal to %0.2f", f),
	)
}

func FloatPositive() ArgValidator {
	return FloatOption(
		func(vi float64) bool { return vi > 0 },
		fmt.Errorf("[FloatPositive] value isn't positive"),
	)
}

func FloatNonNegative() ArgValidator {
	return FloatOption(
		func(vi float64) bool { return vi >= 0 },
		fmt.Errorf("[FloatNonNegative] value isn't non-negative"),
	)
}

func FloatNegative() ArgValidator {
	return FloatOption(
		func(vi float64) bool { return vi < 0 },
		fmt.Errorf("[FloatNegative] value isn't negative"),
	)
}

type fileListTransformer struct{}

func (flt *fileListTransformer) ValueType() ValueType {
	return StringListType
}

func (flt *fileListTransformer) Transform(v *Value) (*Value, error) {
	l := make([]string, 0, len(v.StringList()))
	for i, s := range v.StringList() {
		absStr, err := filepathAbs(s)
		if err != nil {
			return StringListValue(append(l, (v.StringList())[i:]...)...), err
		}
		l = append(l, absStr)
	}
	return StringListValue(l...), nil
}

type fileTransformer struct{}

func (ft *fileTransformer) ValueType() ValueType {
	return StringType
}

func (ft *fileTransformer) Transform(v *Value) (*Value, error) {
	absStr, err := filepathAbs(v.String())
	return StringValue(absStr), err
}

func FileTransformer() ArgTransformer {
	return &fileTransformer{}
}

func FileListTransformer() ArgTransformer {
	return &fileListTransformer{}
}
