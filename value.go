package command

import (
	"encoding/json"
	"fmt"
	"strings"
)

// valueTypeHandler is the interface that each value type needs to implement.
type valueTypeHandler interface {
	parseAuxValue(*auxValue) *Value
	type_() ValueType
	marshalJSON(*Value) ([]byte, error)
	toArgs(*Value) []string
	str(*Value) string
	equal(this, that *Value) bool
	transform([]*string) (*Value, error)
	typeString() string
}

func (vt ValueType) String() string {
	return vtMap.typeString(vt)
}

type valueHandler map[ValueType]valueTypeHandler

func (vh *valueHandler) parseAuxValue(av *auxValue) *Value {
	if h, ok := (*vh)[av.Type]; ok {
		return h.parseAuxValue(av)
	}
	return nil
}

func (vh *valueHandler) marshalJSON(v *Value) ([]byte, error) {
	if h, ok := (*vh)[v.type_]; ok {
		return h.marshalJSON(v)
	}
	return nil, fmt.Errorf("unknown ValueType: %v", v.type_)
}

func (vh *valueHandler) toArgs(v *Value) []string {
	if h, ok := (*vh)[v.type_]; ok {
		return h.toArgs(v)
	}
	return nil
}

func (vh *valueHandler) str(v *Value) string {
	if h, ok := (*vh)[v.type_]; ok {
		return fmt.Sprintf("%vValue(%v)", v.type_, h.str(v))
	}
	return "UNKNOWN_VALUE_TYPE"
}

func (vh *valueHandler) equal(this, that *Value) bool {
	h, ok := (*vh)[this.type_]
	if !ok {
		return this.type_ == that.type_
	}
	return ok && this.type_ == that.type_ && h.equal(this, that)
}

func (vh *valueHandler) transform(vt ValueType, sl []*string) (*Value, error) {
	if h, ok := (*vh)[vt]; ok {
		return h.transform(sl)
	}
	return nil, fmt.Errorf("unknown value type: %v", vt)
}

func (vh *valueHandler) typeString(vt ValueType) string {
	if h, ok := (*vh)[vt]; ok {
		return h.typeString()
	}
	return "UNKNOWN_VALUE_TYPE"
}

var (
	vtMap = valueHandler{
		StringType:     &stringValueHandler{},
		IntType:        &intValueHandler{},
		FloatType:      &floatValueHandler{},
		BoolType:       &boolValueHandler{},
		StringListType: &stringListValueHandler{},
		IntListType:    &intListValueHandler{},
		FloatListType:  &floatListValueHandler{},
	}
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

func TrueValue() *Value {
	return BoolValue(true)
}

func FalseValue() *Value {
	return BoolValue(false)
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
	s, ok := vtMap[vt]
	if !ok {
		return nil, fmt.Errorf("unknown ValueType: %v", vt)
	}
	return json.Marshal(s.typeString())
}

func (vt *ValueType) UnmarshalJSON(b []byte) error {
	sb := ""
	if err := json.Unmarshal(b, &sb); err != nil {
		return fmt.Errorf("ValueType requires string value: %v", err)
	}
	for t, h := range vtMap {
		if sb == h.typeString() {
			*vt = t
			return nil
		}
	}
	return fmt.Errorf("unknown ValueType: %q", sb)
}

func (v *Value) Type() ValueType {
	return v.type_
}

func (v *Value) ToArgs() []string {
	return vtMap.toArgs(v)
}

func (av *auxValue) toVal() *Value {
	return vtMap.parseAuxValue(av)
}

func (v *Value) MarshalJSON() ([]byte, error) {
	return vtMap.marshalJSON(v)
}

func (v *Value) UnmarshalJSON(b []byte) error {
	av := &auxValue{}
	err := json.Unmarshal(b, av)
	if that := av.toVal(); that != nil {
		*v = *that
	}
	return err
}

var (
	checkFunc = func(actual, want ValueType) {
		if want != actual {
			panic(fmt.Sprintf("Requested value of type %v when actual type is %v", want, actual))
		}
	}
)

func (v *Value) checkType(vt ValueType) {
	checkFunc(v.type_, vt)
}

// Prefix all of these with "To" because the "String()" method is needed for the fmt.Stringer interface
func (v *Value) ToString() string {
	if v == nil || v.string == nil {
		return ""
	}
	v.checkType(StringType)
	return *v.string
}

func (v *Value) ToInt() int {
	if v == nil || v.int == nil {
		return 0
	}
	v.checkType(IntType)
	return *v.int
}

func (v *Value) ToFloat() float64 {
	if v == nil || v.float == nil {
		return 0
	}
	v.checkType(FloatType)
	return *v.float
}

func (v *Value) ToBool() bool {
	if v == nil || v.bool == nil {
		return false
	}
	v.checkType(BoolType)
	return *v.bool
}

func (v *Value) ToStringList() []string {
	if v == nil {
		return nil
	}
	v.checkType(StringListType)
	return v.stringList
}

func (v *Value) ToIntList() []int {
	if v == nil {
		return nil
	}
	v.checkType(IntListType)
	return v.intList
}

func (v *Value) ToFloatList() []float64 {
	if v == nil {
		return nil
	}
	v.checkType(FloatListType)
	return v.floatList
}

type ValueType int

const (
	UnspecifiedValueType ValueType = iota
	StringListType
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

func (v *Value) String() string {
	return vtMap.str(v)
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
	return vtMap.equal(v, that)
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
