package command

import (
	"fmt"
	"regexp"
)

type Data struct {
	Values map[string]interface{}
}

func (d *Data) Set(k string, i interface{}) {
	if d.Values == nil {
		d.Values = map[string]interface{}{}
	}
	d.Values[k] = i
}

func GetData[T any](d *Data, k string) T {
	var ret T
	if d.Values == nil {
		return ret
	}
	i, ok := d.Values[k]
	if !ok {
		return ret
	}
	return i.(T)
}

func (d *Data) Has(k string) bool {
	_, ok := d.Values[k]
	return ok
}

// Regexp returns a regexp.Regexp object that is created from the corresponding string node.
// This function should only be used with string nodes that use the IsRegex valiator.
func (d *Data) Regexp(k string) *regexp.Regexp {
	return regexp.MustCompile(GetData[string](d, k))
}

// RegexpList returns a slice of regexp.Regexp objects that is created from the corresponding string list node.
// This function should only be used with string list nodes that use the ListIsRegex valiator.
func (d *Data) RegexpList(k string) []*regexp.Regexp {
	var rs []*regexp.Regexp
	for _, s := range GetData[[]string](d, k) {
		rs = append(rs, regexp.MustCompile(s))
	}
	return rs
}

func (d *Data) String(k string) string {
	return GetData[string](d, k)
}

func (d *Data) Get(k string) interface{} {
	return GetData[string](d, k)
}

func (d *Data) StringList(k string) []string {
	return GetData[[]string](d, k)
}

func (d *Data) Int(k string) int {
	return GetData[int](d, k)
}

func (d *Data) IntList(k string) []int {
	return GetData[[]int](d, k)
}

func (d *Data) Float(k string) float64 {
	return GetData[float64](d, k)
}

func (d *Data) FloatList(k string) []float64 {
	return GetData[[]float64](d, k)
}

func (d *Data) Bool(k string) bool {
	return GetData[bool](d, k)
}

type ExecuteData struct {
	// Executable is a list of commands to run after execution in the commands package.
	Executable []string
	Executor   []func(Output, *Data) error
}

func getOperator[T any]() operator[T] {
	var t T
	var f interface{}
	switch any(t).(type) {
	case string:
		// TODO: cache these
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
	return f.(operator[T])
}
