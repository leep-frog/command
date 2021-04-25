package command

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func StringListValue(s ...string) *Value {
	return &Value{
		type_:      StringListType,
		stringList: s,
	}
}

func IntListValue(l ...int) *Value {
	return &Value{
		type_:   IntListType,
		intList: l,
	}
}

func FloatListValue(l ...float64) *Value {
	return &Value{
		type_:     FloatListType,
		floatList: l,
	}
}

func BoolValue(b bool) *Value {
	return &Value{
		type_: BoolType,
		bool:  &b,
	}
}

func StringValue(s string) *Value {
	return &Value{
		type_:  StringType,
		string: &s,
	}
}

func IntValue(i int) *Value {
	return &Value{
		type_: IntType,
		int:   &i,
	}
}

func FloatValue(f float64) *Value {
	return &Value{
		type_: FloatType,
		float: &f,
	}
}

type Value struct {
	type_ ValueType

	string     *string
	int        *int
	float      *float64
	bool       *bool
	stringList []string
	intList    []int
	floatList  []float64
}

type auxString struct {
	Type   ValueType
	String *string
}
type auxInt struct {
	Type ValueType
	Int  *int
}
type auxFloat struct {
	Type  ValueType
	Float *float64
}
type auxBool struct {
	Type ValueType
	Bool *bool
}
type auxStringList struct {
	Type       ValueType
	StringList []string
}
type auxIntList struct {
	Type    ValueType
	IntList []int
}
type auxFloatList struct {
	Type      ValueType
	FloatList []float64
}

type auxValue struct {
	Type       ValueType
	String     *string
	Int        *int
	Float      *float64
	Bool       *bool
	StringList []string
	IntList    []int
	FloatList  []float64
}

func (vt ValueType) MarshalJSON() ([]byte, error) {
	s, ok := typeToString[vt]
	if !ok {
		return nil, fmt.Errorf("unknown ValueType: %v", vt)
	}
	return json.Marshal(s)
}

func (vt *ValueType) UnmarshalJSON(b []byte) error {
	sb := ""
	if err := json.Unmarshal(b, &sb); err != nil {
		return fmt.Errorf("ValueType requires string value: %v", err)
	}
	for t, s := range typeToString {
		if sb == s {
			*vt = t
			return nil
		}
	}
	return fmt.Errorf("unknown ValueType: %q", sb)
}

var (
	typeToString = map[ValueType]string{
		StringType:     "String",
		IntType:        "Int",
		FloatType:      "Float",
		BoolType:       "Bool",
		StringListType: "StringList",
		IntListType:    "IntList",
		FloatListType:  "FloatList",
	}
)

func (v *Value) Type() ValueType {
	return v.type_
}

func (v *Value) ToArgs() []string {
	switch v.type_ {
	case StringType, IntType, BoolType:
		return []string{v.Str()}
	case FloatType:
		return []string{strconv.FormatFloat(v.Float(), 'f', -1, 64)}
	case StringListType:
		return v.StringList()
	case IntListType:
		sl := make([]string, 0, len(v.IntList()))
		for _, i := range v.IntList() {
			sl = append(sl, fmt.Sprintf("%d", i))
		}
		return sl
	case FloatListType:
		sl := make([]string, 0, len(v.FloatList()))
		for _, f := range v.FloatList() {
			sl = append(sl, strconv.FormatFloat(f, 'f', -1, 64))
		}
		return sl
	}
	return nil
}

func (av *auxValue) toVal() *Value {
	switch av.Type {
	case StringType:
		return StringValue(*av.String)
	case IntType:
		return IntValue(*av.Int)
	case FloatType:
		return FloatValue(*av.Float)
	case BoolType:
		return BoolValue(*av.Bool)
	case StringListType:
		return StringListValue(av.StringList...)
	case IntListType:
		return IntListValue(av.IntList...)
	case FloatListType:
		return FloatListValue(av.FloatList...)
	}
	return nil
}

func (v *Value) MarshalJSON() ([]byte, error) {
	t := v.type_
	switch v.type_ {
	case StringType:
		return json.Marshal(&auxString{t, v.string})
	case IntType:
		return json.Marshal(&auxInt{t, v.int})
	case FloatType:
		return json.Marshal(&auxFloat{t, v.float})
	case BoolType:
		return json.Marshal(&auxBool{t, v.bool})
	case StringListType:
		return json.Marshal(&auxStringList{t, v.stringList})
	case IntListType:
		return json.Marshal(&auxIntList{t, v.intList})
	case FloatListType:
		return json.Marshal(&auxFloatList{t, v.floatList})
	}
	return nil, fmt.Errorf("unknown ValueType: %v", v.type_)
}

func (v *Value) UnmarshalJSON(b []byte) error {
	av := &auxValue{}
	err := json.Unmarshal(b, av)
	if that := av.toVal(); that != nil {
		*v = *that
	}
	return err
}

// TODO: test this.
func (v *Value) Provided() bool {
	return v != nil
}

func (v *Value) String() string {
	if v == nil || v.string == nil {
		return ""
	}
	return *v.string
}

func (v *Value) Int() int {
	if v == nil || v.int == nil {
		return 0
	}
	return *v.int
}

func (v *Value) Float() float64 {
	if v == nil || v.float == nil {
		return 0
	}
	return *v.float
}

func (v *Value) Bool() bool {
	if v == nil || v.bool == nil {
		return false
	}
	return *v.bool
}

func (v *Value) StringList() []string {
	if v == nil {
		return nil
	}
	return v.stringList
}

func (v *Value) IntList() []int {
	if v == nil {
		return nil
	}
	return v.intList
}

func (v *Value) FloatList() []float64 {
	if v == nil {
		return nil
	}
	return v.floatList
}

type ValueType int

const (
	UnspecifiedValueType ValueType = iota
	StringListType                 // ValueType = iota
	StringType
	IntType
	IntListType
	FloatType
	FloatListType
	BoolType

	floatFmt = "%.2f"
	intFmt   = "%d"
)

var (
	boolStringMap = map[string]bool{
		"1":     true,
		"t":     true,
		"T":     true,
		"true":  true,
		"TRUE":  true,
		"True":  true,
		"0":     false,
		"f":     false,
		"F":     false,
		"false": false,
		"FALSE": false,
		"False": false,
	}
)

func (v *Value) IsType(vt ValueType) bool {
	return v.type_ == vt
}

func (v *Value) Str() string {
	switch v.type_ {
	case StringType:
		return v.String()
	case IntType:
		return fmt.Sprintf(intFmt, v.Int())
	case FloatType:
		return fmt.Sprintf(floatFmt, v.Float())
	case BoolType:
		return fmt.Sprintf("%v", v.Bool())
	case StringListType:
		return strings.Join(v.StringList(), ", ")
	case IntListType:
		return intSliceToString(v.IntList())
	case FloatListType:
		return floatSliceToString(v.FloatList())
	}
	// Unreachable
	return "UNKNOWN_VALUE_TYPE"
}

func intSliceToString(is []int) string {
	ss := make([]string, 0, len(is))
	for _, i := range is {
		ss = append(ss, fmt.Sprintf("%d", i))
	}
	return strings.Join(ss, ", ")
}

func floatSliceToString(fs []float64) string {
	ss := make([]string, 0, len(fs))
	for _, f := range fs {
		ss = append(ss, fmt.Sprintf(floatFmt, f))
	}
	return strings.Join(ss, ", ")
}

func (v *Value) Equal(that *Value) bool {
	if v == nil && that == nil {
		return true
	}
	if v == nil || that == nil {
		return false
	}
	if v.type_ != that.type_ {
		return false
	}
	// TODO: check provided
	switch v.type_ {
	case StringType:
		return (v.string == nil && that.string == nil) || (v.string != nil && that.string != nil && *v.string == *that.string)
	case IntType:
		return (v.int == nil && that.int == nil) || (v.int != nil && that.int != nil && *v.int == *that.int)
	case FloatType:
		return (v.float == nil && that.float == nil) || (v.float != nil && that.float != nil && *v.float == *that.float)
	case BoolType:
		return (v.bool == nil && that.bool == nil) || (v.bool != nil && that.bool != nil && *v.bool == *that.bool)
	case StringListType:
		return strListCmp(v.stringList, that.stringList)
	case IntListType:
		return intListCmp(v.intList, that.intList)
	case FloatListType:
		return floatListCmp(v.floatList, that.floatList)
	}
	// Unreachable
	return true
}

func intListCmp(this, that []int) bool {
	if len(this) != len(that) {
		return false
	}
	for i := range this {
		if this[i] != that[i] {
			return false
		}
	}
	return true
}

func floatListCmp(this, that []float64) bool {
	if len(this) != len(that) {
		return false
	}
	for i := range this {
		if this[i] != that[i] {
			return false
		}
	}
	return true
}

func strListCmp(this, that []string) bool {
	if len(this) != len(that) {
		return false
	}
	for i := range this {
		if this[i] != that[i] {
			return false
		}
	}
	return true
}
