package command

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type intListValueHandler struct{}

func (*intListValueHandler) type_() ValueType {
	return IntListType
}

func (*intListValueHandler) typeString() string {
	return "IntList"
}

func (*intListValueHandler) parseAuxValue(av *auxValue) *Value {
	return IntListValue(av.IntList...)
}

func (ivh *intListValueHandler) marshalJSON(v *Value) ([]byte, error) {
	return json.Marshal(&auxIntList{ivh.type_(), v.intList})
}

func (ivh *intListValueHandler) toArgs(v *Value) []string {
	sl := make([]string, 0, len(v.IntList()))
	for _, i := range v.IntList() {
		sl = append(sl, fmt.Sprintf("%d", i))
	}
	return sl
}

func (ivh *intListValueHandler) str(v *Value) string {
	return intSliceToString(v.IntList())
}

func (ivh *intListValueHandler) equal(this, that *Value) bool {
	return intListCmp(this.intList, that.intList)
}

func (ivh *intListValueHandler) transform(sl []*string) (*Value, error) {
	var err error
	var is []int
	for _, s := range sl {
		i, e := strconv.Atoi(*s)
		if e != nil {
			// TODO: add failed to load field to values.
			// These can be used in autocomplete if necessary.
			err = e
		}
		is = append(is, i)
	}
	return IntListValue(is...), err
}