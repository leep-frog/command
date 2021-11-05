package command

import (
	"fmt"
	"regexp"
	"strings"
)

// String options
func StringOption(f func(string) bool, err error) *validatorOption {
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

func Contains(s string) *validatorOption {
	return StringOption(
		func(vs string) bool { return strings.Contains(vs, s) },
		fmt.Errorf("[Contains] value doesn't contain substring %q", s),
	)
}

func MatchesRegex(pattern string) *validatorOption {
	r := regexp.MustCompile(pattern)
	return StringOption(
		func(vs string) bool {
			return r.MatchString(vs)
		},
		fmt.Errorf("[MatchesRegex] value doesn't match regex %q", pattern),
	)
}

func InList(choices ...string) *validatorOption {
	return StringOption(
		func(vs string) bool {
			for _, c := range choices {
				if vs == c {
					return true
				}
			}
			return false
		},
		fmt.Errorf("[InList] argument must be one of %v", choices),
	)
}

func MinLength(length int) *validatorOption {
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
func IntOption(f func(int) bool, err error) *validatorOption {
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

func IntEQ(i int) *validatorOption {
	return IntOption(
		func(vi int) bool { return vi == i },
		fmt.Errorf("[IntEQ] value isn't equal to %d", i),
	)
}

func IntNE(i int) *validatorOption {
	return IntOption(
		func(vi int) bool { return vi != i },
		fmt.Errorf("[IntNE] value isn't not equal to %d", i),
	)
}

func IntLT(i int) *validatorOption {
	return IntOption(
		func(vi int) bool { return vi < i },
		fmt.Errorf("[IntLT] value isn't less than %d", i),
	)
}

func IntLTE(i int) *validatorOption {
	return IntOption(
		func(vi int) bool { return vi <= i },
		fmt.Errorf("[IntLTE] value isn't less than or equal to %d", i),
	)
}

func IntGT(i int) *validatorOption {
	return IntOption(
		func(vi int) bool { return vi > i },
		fmt.Errorf("[IntGT] value isn't greater than %d", i),
	)
}

func IntGTE(i int) *validatorOption {
	return IntOption(
		func(vi int) bool { return vi >= i },
		fmt.Errorf("[IntGTE] value isn't greater than or equal to %d", i),
	)
}

func IntPositive() *validatorOption {
	return IntOption(
		func(vi int) bool { return vi > 0 },
		fmt.Errorf("[IntPositive] value isn't positive"),
	)
}

func IntNonNegative() *validatorOption {
	return IntOption(
		func(vi int) bool { return vi >= 0 },
		fmt.Errorf("[IntNonNegative] value isn't non-negative"),
	)
}

func IntNegative() *validatorOption {
	return IntOption(
		func(vi int) bool { return vi < 0 },
		fmt.Errorf("[IntNegative] value isn't negative"),
	)
}

// Float options
func FloatOption(f func(float64) bool, err error) *validatorOption {
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

func FloatEQ(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) bool { return vf == f },
		fmt.Errorf("[FloatEQ] value isn't equal to %0.2f", f),
	)
}

func FloatNE(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) bool { return vf != f },
		fmt.Errorf("[FloatNE] value isn't not equal to %0.2f", f),
	)
}

func FloatLT(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) bool { return vf < f },
		fmt.Errorf("[FloatLT] value isn't less than %0.2f", f),
	)
}

func FloatLTE(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) bool { return vf <= f },
		fmt.Errorf("[FloatLTE] value isn't less than or equal to %0.2f", f),
	)
}

func FloatGT(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) bool { return vf > f },
		fmt.Errorf("[FloatGT] value isn't greater than %0.2f", f),
	)
}

func FloatGTE(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) bool { return vf >= f },
		fmt.Errorf("[FloatGTE] value isn't greater than or equal to %0.2f", f),
	)
}

func FloatPositive() *validatorOption {
	return FloatOption(
		func(vi float64) bool { return vi > 0 },
		fmt.Errorf("[FloatPositive] value isn't positive"),
	)
}

func FloatNonNegative() *validatorOption {
	return FloatOption(
		func(vi float64) bool { return vi >= 0 },
		fmt.Errorf("[FloatNonNegative] value isn't non-negative"),
	)
}

func FloatNegative() *validatorOption {
	return FloatOption(
		func(vi float64) bool { return vi < 0 },
		fmt.Errorf("[FloatNegative] value isn't negative"),
	)
}
