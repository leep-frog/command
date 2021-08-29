package command

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type floatValueHandler struct{}

func (*floatValueHandler) type_() ValueType {
	return FloatType
}

func (*floatValueHandler) typeString() string {
	return "Float"
}

func (*floatValueHandler) parseAuxValue(av *auxValue) *Value {
	return FloatValue(*av.Float)
}

func (fvh *floatValueHandler) marshalJSON(v *Value) ([]byte, error) {
	return json.Marshal(&auxFloat{fvh.type_(), v.float})
}

func (fvh *floatValueHandler) toArgs(v *Value) []string {
	return []string{strconv.FormatFloat(v.Float(), 'f', -1, 64)}
}

func (fvh *floatValueHandler) str(v *Value) string {
	return fmt.Sprintf(floatFmt, v.Float())
}

func (fvh *floatValueHandler) equal(this, that *Value) bool {
	return (this.float == nil && that.float == nil) || (this.float != nil && that.float != nil && *this.float == *that.float)
}

func (fvh *floatValueHandler) transform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return FloatValue(0), nil
	}
	f, err := strconv.ParseFloat(*sl[0], 64)
	return FloatValue(f), err
}
