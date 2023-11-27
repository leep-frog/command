package operator

import "fmt"

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
