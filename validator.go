package command

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TODO: in go 1.18, ValidatorOption can become ValidatorOption[ValueType]
// String options
func StringOption(f func(string) error) *ValidatorOption {
	return &ValidatorOption{
		vt: StringType,
		validate: func(v *Value) error {
			return f(v.ToString())
		},
	}
}

func StringListOption(f func([]string) error) *ValidatorOption {
	return &ValidatorOption{
		vt: StringListType,
		validate: func(v *Value) error {
			return f(v.ToStringList())
		},
	}
}

func StringDoesNotEqual(s string) *ValidatorOption {
	return StringOption(
		func(vs string) error {
			if vs == s {
				return fmt.Errorf("[StringDoesNotEqual] value cannot equal %q", s)
			}
			return nil
		},
	)
}

func Contains(s string) *ValidatorOption {
	return StringOption(
		func(vs string) error {
			if !strings.Contains(vs, s) {
				return fmt.Errorf("[Contains] value doesn't contain substring %q", s)
			}
			return nil
		},
	)
}

func MatchesRegex(pattern ...string) *ValidatorOption {
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

func IsRegex() *ValidatorOption {
	return StringOption(
		func(s string) error {
			if _, err := regexp.Compile(s); err != nil {
				return fmt.Errorf("[IsRegex] value isn't a valid regex: %v", err)
			}
			return nil
		},
	)
}

func ListIsRegex() *ValidatorOption {
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

func ListMatchesRegex(pattern ...string) *ValidatorOption {
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

func InList(choices ...string) *ValidatorOption {
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

func MinLength(length int) *ValidatorOption {
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

func fileExists(vName, s string) (os.FileInfo, error) {
	fi, err := os.Stat(s)
	if os.IsNotExist(err) {
		return fi, fmt.Errorf("[%s] file %q does not exist", vName, s)
	}
	if err != nil {
		return fi, fmt.Errorf("[%s] failed to read file %q: %v", vName, s, err)
	}
	return fi, nil
}

func FileExists() *ValidatorOption {
	return StringOption(
		func(s string) error {
			_, err := fileExists("FileExists", s)
			return err
		},
	)
}

func FilesExist() *ValidatorOption {
	return StringListOption(
		func(ss []string) error {
			for _, s := range ss {
				if _, err := fileExists("FilesExist", s); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func isDir(vName, s string) error {
	fi, err := fileExists(vName, s)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("[%s] argument %q is a file", vName, s)
	}
	return nil
}

func IsDir() *ValidatorOption {
	return StringOption(
		func(s string) error {
			return isDir("IsDir", s)
		},
	)
}

func AreDirs() *ValidatorOption {
	return StringListOption(
		func(ss []string) error {
			for _, s := range ss {
				if err := isDir("AreDirs", s); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func isFile(vName, s string) error {
	fi, err := fileExists(vName, s)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("[%s] argument %q is a directory", vName, s)
	}
	return nil
}

func IsFile() *ValidatorOption {
	return StringOption(
		func(s string) error {
			return isFile("IsFile", s)
		},
	)
}

func AreFiles() *ValidatorOption {
	return StringListOption(
		func(ss []string) error {
			for _, s := range ss {
				if err := isFile("AreFiles", s); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// Int options
func IntOption(f func(int) error) *ValidatorOption {
	return &ValidatorOption{
		vt: IntType,
		validate: func(v *Value) error {
			return f(v.ToInt())
		},
	}
}

func IntEQ(i int) *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi == i {
				return nil
			}
			return fmt.Errorf("[IntEQ] value isn't equal to %d", i)
		},
	)
}

func IntNE(i int) *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi != i {
				return nil
			}
			return fmt.Errorf("[IntNE] value isn't not equal to %d", i)
		},
	)
}

func IntLT(i int) *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi < i {
				return nil
			}
			return fmt.Errorf("[IntLT] value isn't less than %d", i)
		},
	)
}

func IntLTE(i int) *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi <= i {
				return nil
			}
			return fmt.Errorf("[IntLTE] value isn't less than or equal to %d", i)
		},
	)
}

func IntGT(i int) *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi > i {
				return nil
			}
			return fmt.Errorf("[IntGT] value isn't greater than %d", i)
		},
	)
}

func IntGTE(i int) *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi >= i {
				return nil
			}
			return fmt.Errorf("[IntGTE] value isn't greater than or equal to %d", i)
		},
	)
}

func IntPositive() *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi > 0 {
				return nil
			}
			return fmt.Errorf("[IntPositive] value isn't positive")
		},
	)
}

func IntNonNegative() *ValidatorOption {
	return IntOption(
		func(vi int) error {
			if vi >= 0 {
				return nil
			}
			return fmt.Errorf("[IntNonNegative] value isn't non-negative")
		},
	)
}

func IntNegative() *ValidatorOption {
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
func FloatOption(f func(float64) error) *ValidatorOption {
	return &ValidatorOption{
		vt: FloatType,
		validate: func(v *Value) error {
			return f(v.ToFloat())
		},
	}
}

func FloatEQ(f float64) *ValidatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf == f {
				return nil
			}
			return fmt.Errorf("[FloatEQ] value isn't equal to %0.2f", f)
		},
	)
}

func FloatNE(f float64) *ValidatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf != f {
				return nil
			}
			return fmt.Errorf("[FloatNE] value isn't not equal to %0.2f", f)
		},
	)
}

func FloatLT(f float64) *ValidatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf < f {
				return nil
			}
			return fmt.Errorf("[FloatLT] value isn't less than %0.2f", f)
		},
	)
}

func FloatLTE(f float64) *ValidatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf <= f {
				return nil
			}
			return fmt.Errorf("[FloatLTE] value isn't less than or equal to %0.2f", f)
		},
	)
}

func FloatGT(f float64) *ValidatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf > f {
				return nil
			}
			return fmt.Errorf("[FloatGT] value isn't greater than %0.2f", f)
		},
	)
}

func FloatGTE(f float64) *ValidatorOption {
	return FloatOption(
		func(vf float64) error {
			if vf >= f {
				return nil
			}
			return fmt.Errorf("[FloatGTE] value isn't greater than or equal to %0.2f", f)
		},
	)
}

func FloatPositive() *ValidatorOption {
	return FloatOption(
		func(vi float64) error {
			if vi > 0 {
				return nil
			}
			return fmt.Errorf("[FloatPositive] value isn't positive")
		},
	)
}

func FloatNonNegative() *ValidatorOption {
	return FloatOption(
		func(vi float64) error {
			if vi >= 0 {
				return nil
			}
			return fmt.Errorf("[FloatNonNegative] value isn't non-negative")
		},
	)
}

func FloatNegative() *ValidatorOption {
	return FloatOption(
		func(vi float64) error {
			if vi < 0 {
				return nil
			}
			return fmt.Errorf("[FloatNegative] value isn't negative")
		},
	)
}
