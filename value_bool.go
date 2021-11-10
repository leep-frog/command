package command

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type boolValueHandler struct{}

func (*boolValueHandler) type_() ValueType {
	return BoolType
}

func (*boolValueHandler) typeString() string {
	return "Bool"
}

func (*boolValueHandler) parseAuxValue(av *auxValue) *Value {
	return BoolValue(*av.Bool)
}

func (fvh *boolValueHandler) marshalJSON(v *Value) ([]byte, error) {
	return json.Marshal(&auxBool{fvh.type_(), v.bool})
}

func (fvh *boolValueHandler) toArgs(v *Value) []string {
	return []string{fvh.str(v)}
}

func (fvh *boolValueHandler) str(v *Value) string {
	return fmt.Sprintf("%v", v.ToBool())
}

func (fvh *boolValueHandler) equal(this, that *Value) bool {
	return (this.bool == nil && that.bool == nil) || (this.bool != nil && that.bool != nil && *this.bool == *that.bool)
}

func (fvh *boolValueHandler) transform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return FalseValue(), nil
	}
	b, err := strconv.ParseBool(*sl[0])
	return BoolValue(b), err
}

func (fvh *boolValueHandler) len(v *Value) int {
	return len(fvh.str(v))
}
