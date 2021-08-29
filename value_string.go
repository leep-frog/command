package command

import "encoding/json"

type stringValueHandler struct{}

func (*stringValueHandler) type_() ValueType {
	return StringType
}

func (*stringValueHandler) typeString() string {
	return "String"
}

func (*stringValueHandler) parseAuxValue(av *auxValue) *Value {
	return StringValue(*av.String)
}

func (svh *stringValueHandler) marshalJSON(v *Value) ([]byte, error) {
	return json.Marshal(&auxString{svh.type_(), v.string})
}

func (svh *stringValueHandler) toArgs(v *Value) []string {
	return []string{v.Str()}
}

func (svh *stringValueHandler) str(v *Value) string {
	return v.String()
}

func (svh *stringValueHandler) equal(this, that *Value) bool {
	return (this.string == nil && that.string == nil) || (this.string != nil && that.string != nil && *this.string == *that.string)
}

func (svh *stringValueHandler) transform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return StringValue(""), nil
	}
	return StringValue(*sl[0]), nil
}
