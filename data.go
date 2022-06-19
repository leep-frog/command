package command

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

// Data contains argument data.
type Data struct {
	// Values is a map from argument name to the data for that argument.
	Values map[string]interface{}
}

// Set sets the provided key-value pair in the `Data` object.
func (d *Data) Set(k string, i interface{}) {
	if d.Values == nil {
		d.Values = map[string]interface{}{}
	}
	d.Values[k] = i
}

// SetupOutputFile returns the name of the setup file for the command.
func (d *Data) SetupOutputFile() string {
	return d.String(SetupArg.Name())
}

// SetupOutputString returns the file contents, as a string, of the setup file for the command.
func (d *Data) SetupOutputString() (string, error) {
	b, err := ioutil.ReadFile(d.SetupOutputFile())
	if err != nil {
		return "", fmt.Errorf("failed to read setup file (%s): %v", d.SetupOutputFile(), err)
	}
	return strings.TrimSpace(string(b)), nil
}

// SetupOutputString returns the file contents, as a string slice, of the setup file for the command.
func (d *Data) SetupOutputContents() ([]string, error) {
	s, err := d.SetupOutputString()
	return strings.Split(s, "\n"), err
}

// GetData fetches the value for a given key.
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

// Has returns whether or not key has been set in the `Data` object.
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

// String returns the string data for an argument.
func (d *Data) String(k string) string {
	return GetData[string](d, k)
}

// Get returns the interface data for an argument.
func (d *Data) Get(k string) interface{} {
	return GetData[string](d, k)
}

// StringList returns the string slice data for an argument.
func (d *Data) StringList(k string) []string {
	return GetData[[]string](d, k)
}

// Int returns the int data for an argument.
func (d *Data) Int(k string) int {
	return GetData[int](d, k)
}

// IntList returns the int slice data for an argument.
func (d *Data) IntList(k string) []int {
	return GetData[[]int](d, k)
}

// Float returns the float data for an argument.
func (d *Data) Float(k string) float64 {
	return GetData[float64](d, k)
}

// FloatList returns the float slice data for an argument.
func (d *Data) FloatList(k string) []float64 {
	return GetData[[]float64](d, k)
}

// Bool returns the bool data for an argument.
func (d *Data) Bool(k string) bool {
	return GetData[bool](d, k)
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
