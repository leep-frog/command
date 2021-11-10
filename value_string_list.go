package command

import (
	"encoding/json"
	"fmt"
	"strings"
)

type stringListValueHandler struct{}

func (*stringListValueHandler) type_() ValueType {
	return StringListType
}

func (*stringListValueHandler) typeString() string {
	return "StringList"
}

func (*stringListValueHandler) parseAuxValue(av *auxValue) *Value {
	return StringListValue(av.StringList...)
}

func (ivh *stringListValueHandler) marshalJSON(v *Value) ([]byte, error) {
	return json.Marshal(&auxStringList{ivh.type_(), v.stringList})
}

func (ivh *stringListValueHandler) toArgs(v *Value) []string {
	return v.ToStringList()
}

func (ivh *stringListValueHandler) str(v *Value) string {
	var r []string
	for _, s := range v.ToStringList() {
		r = append(r, fmt.Sprintf("%q", s))
	}
	return strings.Join(r, ", ")
}

func (ivh *stringListValueHandler) equal(this, that *Value) bool {
	return strListCmp(this.stringList, that.stringList)
}

func (ivh *stringListValueHandler) transform(sl []*string) (*Value, error) {
	r := make([]string, 0, len(sl))
	for _, s := range sl {
		r = append(r, *s)
	}
	return StringListValue(r...), nil
}
