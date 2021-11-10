package command

import (
	"fmt"
	"regexp"
	"strings"
)

// String options
func StringOption(f func(string) error) *validatorOption {
	return &validatorOption{
		vt: StringType,
		validate: func(v *Value) error {
			return f(v.ToString())
		},
	}
}

func StringListOption(f func([]string) error) *validatorOption {
	return &validatorOption{
		vt: StringListType,
		validate: func(v *Value) error {
			return f(v.ToStringList())
		},
	}
}

func Contains(s string) *validatorOption {
	return StringOption(
		func(vs string) error {
			if !strings.Contains(vs, s) {
				return fmt.Errorf("[Contains] value doesn't contain substring %q", s)
			}
			return nil
		},
	)
}

func MatchesRegex(pattern ...string) *validatorOption {
	var rs []*regexp.Regexp
	for _, p := range pattern {
		rs = append(rs, regexp.MustCompile(p))
	}
	return StringOption(
		func(vs string) error {
			for _, r := range rs {
				if !r.MatchString(vs) {
					return fmt.Errorf("[MatchesRegex] value doesn't match regex %q", r.String())
				}
			}
			return nil
		},
	)
}

func IsRegex() *validatorOption {
	return StringOption(
		func(s string) error {
			if _, err := regexp.Compile(s); err != nil {
				return fmt.Errorf("[IsRegex] value isn't a valid regex: %v", err)
			}
			return nil
		},
	)
}

func ListIsRegex() *validatorOption {
	return StringListOption(
		func(ss []string) error {
			for _, s := range ss {
				if _, err := regexp.Compile(s); err != nil {
					return fmt.Errorf("[ListIsRegex] value %q isn't a valid regex: %v", s, err)
				}
			}
			return nil
		},
	)
}

func ListMatchesRegex(pattern ...string) *validatorOption {
	var rs []*regexp.Regexp
	for _, p := range pattern {
		rs = append(rs, regexp.MustCompile(p))
	}
	return StringListOption(
		func(vs []string) error {
			for _, v := range vs {
				for _, r := range rs {
					if !r.MatchString(v) {
						return fmt.Errorf("[ListMatchesRegex] value %q doesn't match regex %q", v, r.String())
					}
				}
			}
			return nil
		},
	)
}

func InList(choices ...string) *validatorOption {
	return StringOption(
		func(vs string) error {
			for _, c := range choices {
				if vs == c {
					return nil
				}
			}
			return fmt.Errorf("[InList] argument must be one of %v", choices)
		},
	)
}

func MinLength(length int) *validatorOption {
	var plural string
	if length != 1 {
		plural = "s"
	}
	return StringOption(
		func(vs string) error {
			if len(vs) < length {
				return fmt.Errorf("[MinLength] value must be at least %d character%s", length, plural)
			}
			return nil
		},
	)
}

// Int options
func IntOption(f func(int) error) *validatorOption {
	return &validatorOption{
		vt: IntType,
		validate: func(v *Value) error {
			return f(v.ToInt())
		},
	}
}

func IntEQ(i int) *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi == i {
				return nil
			}
			return fmt.Errorf("[IntEQ] value isn't equal to %d", i)
		},
	)
}

func IntNE(i int) *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi != i {
				return nil
			}
			return fmt.Errorf("[IntNE] value isn't not equal to %d", i)
		},
	)
}

func IntLT(i int) *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi < i {
				return nil
			}
			return fmt.Errorf("[IntLT] value isn't less than %d", i)
		},
	)
}

func IntLTE(i int) *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi <= i {
				return nil
			}
			return fmt.Errorf("[IntLTE] value isn't less than or equal to %d", i)
		},
	)
}

func IntGT(i int) *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi > i {
				return nil
			}
			return fmt.Errorf("[IntGT] value isn't greater than %d", i)
		},
	)
}

func IntGTE(i int) *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi >= i {
				return nil
			}
			return fmt.Errorf("[IntGTE] value isn't greater than or equal to %d", i)
		},
	)
}

func IntPositive() *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi > 0 {
				return nil
			}
			return fmt.Errorf("[IntPositive] value isn't positive")
		},
	)
}

func IntNonNegative() *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi >= 0 {
				return nil
			}
			return fmt.Errorf("[IntNonNegative] value isn't non-negative")
		},
	)
}

func IntNegative() *validatorOption {
	return IntOption(
		func(vi int) error {
			if vi < 0 {
				return nil
			}
			return fmt.Errorf("[IntNegative] value isn't negative")
		},
	)
}

// Float options
func FloatOption(f func(float64) error) *validatorOption {
	return &validatorOption{
		vt: FloatType,
		validate: func(v *Value) error {
			return f(v.ToFloat())
		},
	}
}

func FloatEQ(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf == f {
				return nil
			}
			return fmt.Errorf("[FloatEQ] value isn't equal to %0.2f", f)
		},
	)
}

func FloatNE(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf != f {
				return nil
			}
			return fmt.Errorf("[FloatNE] value isn't not equal to %0.2f", f)
		},
	)
}

func FloatLT(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf < f {
				return nil
			}
			return fmt.Errorf("[FloatLT] value isn't less than %0.2f", f)
		},
	)
}

func FloatLTE(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf <= f {
				return nil
			}
			return fmt.Errorf("[FloatLTE] value isn't less than or equal to %0.2f", f)
		},
	)
}

func FloatGT(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf > f {
				return nil
			}
			return fmt.Errorf("[FloatGT] value isn't greater than %0.2f", f)
		},
	)
}

func FloatGTE(f float64) *validatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf >= f {
				return nil
			}
			return fmt.Errorf("[FloatGTE] value isn't greater than or equal to %0.2f", f)
		},
	)
}

func FloatPositive() *validatorOption {
	return FloatOption(
		func(vi float64) error {
			if vi > 0 {
				return nil
			}
			return fmt.Errorf("[FloatPositive] value isn't positive")
		},
	)
}

func FloatNonNegative() *validatorOption {
	return FloatOption(
		func(vi float64) error {
			if vi >= 0 {
				return nil
			}
			return fmt.Errorf("[FloatNonNegative] value isn't non-negative")
		},
	)
}

func FloatNegative() *validatorOption {
	return FloatOption(
		func(vi float64) error {
			if vi < 0 {
				return nil
			}
			return fmt.Errorf("[FloatNegative] value isn't negative")
		},
	)
}
