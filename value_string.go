package command

import (
	"encoding/json"
	"fmt"
)

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
	return []string{v.ToString()}
}

func (svh *stringValueHandler) str(v *Value) string {
	return fmt.Sprintf("%q", v.ToString())
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

func (svh *stringValueHandler) len(v *Value) int {
	return len(*v.string)
}
