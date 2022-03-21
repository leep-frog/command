package command

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/exp/constraints"
)

func Option[T any](f func(T) error) *ValidatorOption[T] {
	return &ValidatorOption[T]{f}
}

func Contains(s string) *ValidatorOption[string] {
	return Option(
		func(vs string) error {
			if !strings.Contains(vs, s) {
				return fmt.Errorf("[Contains] value doesn't contain substring %q", s)
			}
			return nil
		},
	)
}

func MatchesRegex(pattern ...string) *ValidatorOption[string] {
	var rs []*regexp.Regexp
	for _, p := range pattern {
		rs = append(rs, regexp.MustCompile(p))
	}
	return Option(
		func(vs string) error {
			for _, r := range rs {
				if !r.MatchString(vs) {
					return fmt.Errorf("[MatchesRegex] value %q doesn't match regex %q", vs, r.String())
				}
			}
			return nil
		},
	)
}

func IsRegex() *ValidatorOption[string] {
	return Option(
		func(s string) error {
			if _, err := regexp.Compile(s); err != nil {
				return fmt.Errorf("[IsRegex] value %q isn't a valid regex: %v", s, err)
			}
			return nil
		},
	)
}

func InList[T comparable](choices ...T) *ValidatorOption[T] {
	return Option(
		func(vs T) error {
			for _, c := range choices {
				if vs == c {
					return nil
				}
			}
			return fmt.Errorf("[InList] argument must be one of %v", choices)
		},
	)
}

// TODO: can this be generic? Is there an interface for len?
func MinLength(length int) *ValidatorOption[string] {
	var plural string
	if length != 1 {
		plural = "s"
	}
	return Option(
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

func FileExists() *ValidatorOption[string] {
	return Option(
		func(s string) error {
			_, err := fileExists("FileExists", s)
			return err
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

func IsDir() *ValidatorOption[string] {
	return Option(
		func(s string) error {
			return isDir("IsDir", s)
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

func IsFile() *ValidatorOption[string] {
	return Option(
		func(s string) error {
			return isFile("IsFile", s)
		},
	)
}

// Ordered options
func EQ[T constraints.Ordered](n T) *ValidatorOption[T] {
	return Option(
		func(v T) error {
			if v == n {
				return nil
			}
			return fmt.Errorf("[EQ] value isn't equal to %v", n)
		},
	)
}

func NEQ[T constraints.Ordered](n T) *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			if v != n {
				return nil
			}
			return fmt.Errorf("[NEQ] value cannot equal %v", n)
		},
	)
}

func LT[T constraints.Ordered](n T) *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			if v < n {
				return nil
			}
			return fmt.Errorf("[LT] value isn't less than %v", n)
		},
	)
}

func LTE[T constraints.Ordered](n T) *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			if v <= n {
				return nil
			}
			return fmt.Errorf("[LTE] value isn't less than or equal to %v", n)
		},
	)
}

func GT[T constraints.Ordered](n T) *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			if v > n {
				return nil
			}
			return fmt.Errorf("[GT] value isn't greater than %v", n)
		},
	)
}

func GTE[T constraints.Ordered](n T) *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			if v >= n {
				return nil
			}
			return fmt.Errorf("[GTE] value isn't greater than or equal to %v", n)
		},
	)
}

func Positive[T constraints.Ordered]() *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			var t T
			if v > t {
				return nil
			}
			return fmt.Errorf("[Positive] value isn't positive")
		},
	)
}

func NonNegative[T constraints.Ordered]() *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			var t T
			if v >= t {
				return nil
			}
			return fmt.Errorf("[NonNegative] value isn't non-negative")
		},
	)
}

func Negative[T constraints.Ordered]() *ValidatorOption[T] {
	return Option[T](
		func(v T) error {
			var t T
			if v < t {
				return nil
			}
			return fmt.Errorf("[Negative] value isn't negative")
		},
	)
}
