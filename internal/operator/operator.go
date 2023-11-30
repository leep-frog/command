package operator

import (
	"fmt"
	"strings"
)

// Operator is an interface for mapping strings to and from specific types
type Operator[T any] interface {
	// ToArgs converts a type to its command line equivalent
	ToArgs(T) []string
	// FromArgs converts a list of command line arguments to types
	FromArgs([]*string) (T, error)
}

// FromArgs converts variadic basic string args to the respective type
func FromArgs[T any](op Operator[T], ss ...string) (T, error) {
	var sps []*string
	for _, s := range ss {
		sc := strings.Clone(s)
		sps = append(sps, &sc)
	}
	return op.FromArgs(sps)
}

// GetOperator returns an operator for type T
func GetOperator[T any]() Operator[T] {
	var t T
	var f interface{}
	switch any(t).(type) {
	case string:
		f = &stringOperator{}
	case []string:
		f = &stringListOperator{}
	case int:
		f = &intOperator{}
	case []int:
		f = &intListOperator{}
	case float64:
		f = &floatOperator{}
	case []float64:
		f = &floatListOperator{}
	case bool:
		f = &boolOperator{}
	default:
		panic(fmt.Sprintf("no operator defined for type %T", t))
	}
	return f.(Operator[T])
}
