package commander

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/leep-frog/command/commondels"
	"golang.org/x/exp/constraints"
)

type validatable interface {
	Name() string
}

// newValidationErr returns a validationErr. It is private because
// it is only created by the `ValidatorOption` type.
func newValidationErr(arg validatable, err error) error {
	return &validationErr{arg.Name(), err}
}

type validationErr struct {
	argName string
	err     error
}

func (ve *validationErr) Error() string {
	return fmt.Sprintf("validation for %q failed: %v", ve.argName, ve.err)
}

// IsValidationError returns whether or not the provided error
// is a validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*validationErr)
	return ok
}

// ValidatorOption is an `ArgumentOption` and `BashOption` for validating arguments.
type ValidatorOption[T any] struct {
	Validate func(T, *commondels.Data) error
	Usage    string
}

func (vo *ValidatorOption[T]) RunValidation(arg validatable, t T, d *commondels.Data) error {
	if err := vo.Validate(t, d); err != nil {
		return newValidationErr(arg, err)
	}
	return nil
}

func (vo *ValidatorOption[T]) modifyArgumentOption(ao *argumentOption[T]) {
	ao.validators = append(ao.validators, vo)
}

// ListifyValidatorOption changes a single-arg validator (`ValidatorOption[T]`) to a list-arg validator (`ValidatorOption[[]T]`) for the same type.
// Note: we can't do this as a method because it causes an instantiation cycle:
// ValidatorOption[T]   -> defines method `ValidatorOption[T].Listify()   *ValidatorOption[[]T]`
// ValidatorOption[[]T] -> defines method `ValidatorOption[[]T].Listify() *ValidatorOption[[][]T]`
// etc.
func ListifyValidatorOption[T any](vo *ValidatorOption[T]) *ValidatorOption[[]T] {
	return &ValidatorOption[[]T]{
		func(ts []T, d *commondels.Data) error {
			for _, t := range ts {
				if err := vo.Validate(t, d); err != nil {
					return err
				}
			}
			return nil
		},
		vo.Usage,
	}
}

// Not [`ValidatorOption`] inverts the provided validator.
func Not[T any](vo *ValidatorOption[T]) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(t T, d *commondels.Data) error {
			if err := vo.Validate(t, d); err == nil {
				return fmt.Errorf("[Not(%s)] failed", vo.Usage)
			}
			return nil
		},
		fmt.Sprintf("Not(%s)", vo.Usage),
	}
}

// Contains [`ValidatorOption`] validates an argument contains the provided string.
func Contains(s string) *ValidatorOption[string] {
	return &ValidatorOption[string]{
		func(vs string, d *commondels.Data) error {
			if !strings.Contains(vs, s) {
				return fmt.Errorf("[Contains] value doesn't contain substring %q", s)
			}
			return nil
		},
		fmt.Sprintf("Contains(%q)", s),
	}
}

// MatchesRegex [`ValidatorOption`] validates an argument matches the provided regexes.
func MatchesRegex(pattern ...string) *ValidatorOption[string] {
	var rs []*regexp.Regexp
	for _, p := range pattern {
		rs = append(rs, regexp.MustCompile(p))
	}
	return &ValidatorOption[string]{
		func(vs string, d *commondels.Data) error {
			for _, r := range rs {
				if !r.MatchString(vs) {
					return fmt.Errorf("[MatchesRegex] value %q doesn't match regex %q", vs, r.String())
				}
			}
			return nil
		},
		fmt.Sprintf("MatchesRegex(%v)", rs),
	}
}

// IsRegex [`ValidatorOption`] validates an argument is a valid regex.
func IsRegex() *ValidatorOption[string] {
	return &ValidatorOption[string]{
		func(s string, d *commondels.Data) error {
			if _, err := regexp.Compile(s); err != nil {
				return fmt.Errorf("[IsRegex] value %q isn't a valid regex: %v", s, err)
			}
			return nil
		},
		"IsRegex()",
	}
}

// InList [`ValidatorOption`] validates an argument is one of the provided choices.
func InList[T comparable](choices ...T) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(vs T, d *commondels.Data) error {
			for _, c := range choices {
				if vs == c {
					return nil
				}
			}
			return fmt.Errorf("[InList] argument must be one of %v", choices)
		},
		fmt.Sprintf("InList(%v)", choices),
	}
}

type Lengthable[T any] interface {
	string | []T
}

// MinLength [`ValidatorOption`] validates an argument is at least `length` long.
func MinLength[K any, T Lengthable[K]](length int) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(vs T, d *commondels.Data) error {
			if len(vs) < length {
				return fmt.Errorf("[MinLength] length must be at least %d", length)
			}
			return nil
		},
		fmt.Sprintf("MinLength(%d)", length),
	}
}

// MaxLength [`ValidatorOption`] validates an argument is at most `length` long.
func MaxLength[K any, T Lengthable[K]](length int) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(vs T, d *commondels.Data) error {
			if len(vs) > length {
				return fmt.Errorf("[MaxLength] length must be at most %d", length)
			}
			return nil
		},
		fmt.Sprintf("MaxLength(%d)", length),
	}
}

// Length [`ValidatorOption`] validates an argument is exactly length.
func Length[K any, T Lengthable[K]](length int) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(vs T, d *commondels.Data) error {
			if len(vs) != length {
				return fmt.Errorf("[Length] length must be exactly %d", length)
			}
			return nil
		},
		fmt.Sprintf("Length(%d)", length),
	}
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

// FileExists [`ValidatorOption`] validates the file or directory exists.
func FileExists() *ValidatorOption[string] {
	return &ValidatorOption[string]{
		func(s string, d *commondels.Data) error {
			_, err := fileExists("FileExists", s)
			return err
		},
		"FileExists()",
	}
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

// IsDir [`ValidatorOption`] validates an argument is a directory.
func IsDir() *ValidatorOption[string] {
	return &ValidatorOption[string]{
		func(s string, d *commondels.Data) error {
			return isDir("IsDir", s)
		},
		"IsDir()",
	}
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

// IsFile [`ValidatorOption`] validates an argument is a file.
func IsFile() *ValidatorOption[string] {
	return &ValidatorOption[string]{
		func(s string, d *commondels.Data) error {
			return isFile("IsFile", s)
		},
		"IsFile()",
	}
}

// Ordered options

// EQ [`ValidatorOption`] validates an argument equals `n`.
func EQ[T comparable](n T) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			if v == n {
				return nil
			}
			return fmt.Errorf("[EQ] value isn't equal to %v", n)
		},
		fmt.Sprintf("EQ(%v)", n),
	}
}

// NEQ [`ValidatorOption`] validates an argument does not equal `n`.
func NEQ[T constraints.Ordered](n T) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			if v != n {
				return nil
			}
			return fmt.Errorf("[NEQ] value cannot equal %v", n)
		},
		fmt.Sprintf("NEQ(%v)", n),
	}
}

// LT [`ValidatorOption`] validates an argument is less than `n`.
func LT[T constraints.Ordered](n T) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			if v < n {
				return nil
			}
			return fmt.Errorf("[LT] value isn't less than %v", n)
		},
		fmt.Sprintf("LT(%v)", n),
	}
}

// LTE [`ValidatorOption`] validates an argument is less than or equal to `n`.
func LTE[T constraints.Ordered](n T) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			if v <= n {
				return nil
			}
			return fmt.Errorf("[LTE] value isn't less than or equal to %v", n)
		},
		fmt.Sprintf("LTE(%v)", n),
	}
}

// GT [`ValidatorOption`] validates an argument is greater than `n`.
func GT[T constraints.Ordered](n T) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			if v > n {
				return nil
			}
			return fmt.Errorf("[GT] value isn't greater than %v", n)
		},
		fmt.Sprintf("GT(%v)", n),
	}
}

// GTE [`ValidatorOption`] validates an argument is greater than or equal to `n`.
func GTE[T constraints.Ordered](n T) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			if v >= n {
				return nil
			}
			return fmt.Errorf("[GTE] value isn't greater than or equal to %v", n)
		},
		fmt.Sprintf("GTE(%v)", n),
	}
}

// Positive [`ValidatorOption`] validates an argument is positive.
func Positive[T constraints.Ordered]() *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			var t T
			if v > t {
				return nil
			}
			return fmt.Errorf("[Positive] value isn't positive")
		},
		"Positive()",
	}
}

// NonNegative [`ValidatorOption`] validates an argument is non-negative.
func NonNegative[T constraints.Ordered]() *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			var t T
			if v >= t {
				return nil
			}
			return fmt.Errorf("[NonNegative] value isn't non-negative")
		},
		"NonNegative()",
	}
}

// Negative [`ValidatorOption`] validates an argument is negative.
func Negative[T constraints.Ordered]() *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			var t T
			if v < t {
				return nil
			}
			return fmt.Errorf("[Negative] value isn't negative")
		},
		"Negative()",
	}
}

// Between [`ValidatorOption`] validates an argument is between two numbers.
func Between[T constraints.Ordered](start, end T, inclusive bool) *ValidatorOption[T] {
	return &ValidatorOption[T]{
		func(v T, d *commondels.Data) error {
			if v < start {
				return fmt.Errorf("[Between] value is less than lower bound (%v)", start)
			}
			if v > end {
				return fmt.Errorf("[Between] value is greater than upper bound (%v)", end)
			}

			if !inclusive {
				if v == start {
					return fmt.Errorf("[Between] value equals exclusive lower bound (%v)", start)
				}
				if v == end {
					return fmt.Errorf("[Between] value equals exclusive upper bound (%v)", end)
				}
			}

			return nil
		},
		"Negative()",
	}
}
