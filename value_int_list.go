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
	sl := make([]string, 0, v.Length())
	for _, i := range v.ToIntList() {
		sl = append(sl, fmt.Sprintf("%d", i))
	}
	return sl
}

func (ivh *intListValueHandler) str(v *Value) string {
	return intSliceToString(v.ToIntList())
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
			err = e
		}
		is = append(is, i)
	}
	return IntListValue(is...), err
}

func (ivh *intListValueHandler) len(v *Value) int {
	return len(v.intList)
}
