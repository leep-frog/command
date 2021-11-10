package command

import (
	"encoding/json"
	"strconv"
)

type floatListValueHandler struct{}

func (*floatListValueHandler) type_() ValueType {
	return FloatListType
}

func (*floatListValueHandler) typeString() string {
	return "FloatList"
}

func (*floatListValueHandler) parseAuxValue(av *auxValue) *Value {
	return FloatListValue(av.FloatList...)
}

func (ivh *floatListValueHandler) marshalJSON(v *Value) ([]byte, error) {
	return json.Marshal(&auxFloatList{ivh.type_(), v.floatList})
}

func (ivh *floatListValueHandler) toArgs(v *Value) []string {
	sl := make([]string, 0, v.Length())
	for _, f := range v.ToFloatList() {
		sl = append(sl, strconv.FormatFloat(f, 'f', -1, 64))
	}
	return sl
}

func (ivh *floatListValueHandler) str(v *Value) string {
	return floatSliceToString(v.ToFloatList())
}

func (ivh *floatListValueHandler) equal(this, that *Value) bool {
	return floatListCmp(this.floatList, that.floatList)
}

func (ivh *floatListValueHandler) transform(sl []*string) (*Value, error) {
	var err error
	var fs []float64
	for _, s := range sl {
		f, e := strconv.ParseFloat(*s, 64)
		if e != nil {
			err = e
		}
		fs = append(fs, f)
	}
	return FloatListValue(fs...), err
}

func (ivh *floatListValueHandler) len(v *Value) int {
	return len(v.floatList)
}
