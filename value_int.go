package command

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type intValueHandler struct{}

func (*intValueHandler) type_() ValueType {
	return IntType
}

func (*intValueHandler) typeString() string {
	return "Int"
}

func (*intValueHandler) parseAuxValue(av *auxValue) *Value {
	return IntValue(*av.Int)
}

func (ivh *intValueHandler) marshalJSON(v *Value) ([]byte, error) {
	return json.Marshal(&auxInt{ivh.type_(), v.int})
}

func (ivh *intValueHandler) toArgs(v *Value) []string {
	return []string{v.Str()}
}

func (ivh *intValueHandler) str(v *Value) string {
	return fmt.Sprintf(intFmt, v.ToInt())
}

func (ivh *intValueHandler) equal(this, that *Value) bool {
	return (this.int == nil && that.int == nil) || (this.int != nil && that.int != nil && *this.int == *that.int)
}

func (ivh *intValueHandler) transform(sl []*string) (*Value, error) {
	if len(sl) == 0 {
		return IntValue(0), nil
	}
	i, err := strconv.Atoi(*sl[0])
	return IntValue(i), err
}
