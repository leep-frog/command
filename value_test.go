package command

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestValueCommands(t *testing.T) {
	for _, test := range []struct {
		name           string
		etc            *ExecuteTestCase
		wantString     string
		wantStringList []string
		wantInt        int
		wantIntList    []int
		wantFloat      float64
		wantFloatList  []float64
		wantBool       bool
		wantProvided   bool
	}{
		{
			name: "empty value",
		},
		{
			name: "string is populated",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringNode("argName", nil)),
				Args: []string{"string-val"},
				WantData: &Data{
					"argName": StringValue("string-val"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "string-val"},
					},
				},
			},
			wantString:   "string-val",
			wantProvided: true,
		},
		{
			name: "string list is populated",
			etc: &ExecuteTestCase{
				Node: SerialNodes(StringListNode("argName", 2, 3, nil)),
				Args: []string{"string", "list", "val"},
				WantData: &Data{
					"argName": StringListValue("string", "list", "val"),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "string"},
						{value: "list"},
						{value: "val"},
					},
				},
			},
			wantStringList: []string{"string", "list", "val"},
			wantProvided:   true,
		},
		{
			name: "int is populated",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntNode("argName", nil)),
				Args: []string{"123"},
				WantData: &Data{
					"argName": IntValue(123),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "123"},
					},
				},
			},
			wantInt:      123,
			wantProvided: true,
		},
		{
			name: "int list is populated",
			etc: &ExecuteTestCase{
				Node: SerialNodes(IntListNode("argName", 2, 3, nil)),
				Args: []string{"12", "345", "6"},
				WantData: &Data{
					"argName": IntListValue(12, 345, 6),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "12"},
						{value: "345"},
						{value: "6"},
					},
				},
			},
			wantProvided: true,
			wantIntList:  []int{12, 345, 6},
		},
		{
			name: "flaot is populated",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatNode("argName", nil)),
				Args: []string{"12.3"},
				WantData: &Data{
					"argName": FloatValue(12.3),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "12.3"},
					},
				},
			},
			wantFloat:    12.3,
			wantProvided: true,
		},
		{
			name: "float list is populated",
			etc: &ExecuteTestCase{
				Node: SerialNodes(FloatListNode("argName", 2, 3, nil)),
				Args: []string{"1.2", "-345", ".6"},
				WantData: &Data{
					"argName": FloatListValue(1.2, -345, 0.6),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "1.2"},
						{value: "-345"},
						{value: "0.6"},
					},
				},
			},
			wantFloatList: []float64{1.2, -345, .6},
			wantProvided:  true,
		},
		{
			name: "bool is populated",
			etc: &ExecuteTestCase{
				Node: SerialNodes(BoolNode("argName")),
				Args: []string{"true"},
				WantData: &Data{
					"argName": BoolValue(true),
				},
				wantInput: &Input{
					args: []*inputArg{
						{value: "true"},
					},
				},
			},
			wantBool:     true,
			wantProvided: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.etc == nil {
				test.etc = &ExecuteTestCase{}
			}
			test.etc.Node = SerialNodesTo(test.etc.Node, ExecutorNode(func(output Output, data *Data) error {
				name := "argName"
				v := data.get(name)

				if v.Provided() != test.wantProvided {
					t.Errorf("Provided() returned incorrect value, got %v, want %v", v.Provided(), test.wantProvided)
				}

				// strings
				if diff := cmp.Diff(test.wantString, v.String()); diff != "" {
					t.Errorf("String() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantString, data.String(name)); diff != "" {
					t.Errorf("data.String() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantStringList, v.StringList()); diff != "" {
					t.Errorf("StringList() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantStringList, data.StringList(name)); diff != "" {
					t.Errorf("data.StringList() produced diff (-want, +got):\n%s", diff)
				}

				// ints
				if diff := cmp.Diff(test.wantInt, v.Int()); diff != "" {
					t.Errorf("Int() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantInt, data.Int(name)); diff != "" {
					t.Errorf("data.Int() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantIntList, v.IntList()); diff != "" {
					t.Errorf("IntList() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantIntList, data.IntList(name)); diff != "" {
					t.Errorf("data.IntList() produced diff (-want, +got):\n%s", diff)
				}

				// floats
				if diff := cmp.Diff(test.wantFloat, v.Float()); diff != "" {
					t.Errorf("Float() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantFloat, data.Float(name)); diff != "" {
					t.Errorf("data.Float() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantFloatList, v.FloatList()); diff != "" {
					t.Errorf("FloatList() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantFloatList, data.FloatList(name)); diff != "" {
					t.Errorf("data.FloatList() produced diff (-want, +got):\n%s", diff)
				}

				// bool
				if diff := cmp.Diff(test.wantBool, v.Bool()); diff != "" {
					t.Errorf("Bool() produced diff (-want, +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantBool, data.Bool(name)); diff != "" {
					t.Errorf("data.Bool() produced diff (-want, +got):\n%s", diff)
				}

				return nil
			}))

			ExecuteTest(t, test.etc, &ExecuteTestOptions{testInput: true})
		})
	}
}

func TestValueStrAndListAndJson(t *testing.T) {
	for _, test := range []struct {
		name        string
		v           *Value
		wantStr     string
		wantStrList []string
	}{
		{
			name:        "string value",
			v:           StringValue("hello there"),
			wantStr:     "hello there",
			wantStrList: []string{"hello there"},
		},
		{
			name:        "int value",
			v:           IntValue(12),
			wantStr:     "12",
			wantStrList: []string{"12"},
		},
		{
			name:        "float value with extra decimal points",
			v:           FloatValue(123.4567),
			wantStr:     "123.46",
			wantStrList: []string{"123.4567"},
		},
		{
			name:        "float value with no decimal points",
			v:           FloatValue(123),
			wantStr:     "123.00",
			wantStrList: []string{"123"},
		},
		{
			name:        "bool true value",
			v:           BoolValue(true),
			wantStr:     "true",
			wantStrList: []string{"true"},
		},
		{
			name:        "bool false value",
			v:           BoolValue(false),
			wantStr:     "false",
			wantStrList: []string{"false"},
		},
		{
			name:        "string list",
			v:           StringListValue("hello", "there"),
			wantStr:     "hello, there",
			wantStrList: []string{"hello", "there"},
		},
		{
			name:        "int list",
			v:           IntListValue(12, -34, 5678),
			wantStr:     "12, -34, 5678",
			wantStrList: []string{"12", "-34", "5678"},
		},
		{
			name:        "float list",
			v:           FloatListValue(0.12, -3.4, 567.8910),
			wantStr:     "0.12, -3.40, 567.89",
			wantStrList: []string{"0.12", "-3.4", "567.891"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if diff := cmp.Diff(test.wantStr, test.v.Str()); diff != "" {
				t.Errorf("Value.Str() returned incorrect string (-want, +got):\n%s", diff)
			}

			if diff := cmp.Diff(test.wantStrList, test.v.ToArgs()); diff != "" {
				t.Errorf("Value.ToArgs() returned incorrect string list (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestValueEqualAndJSONMarshaling(t *testing.T) {
	for _, test := range []struct {
		name         string
		this         *Value
		that         *Value
		wantThisJSON string
		wantThatJSON string
		want         bool
	}{
		{
			name:         "nil values are equal",
			want:         true,
			wantThisJSON: "null",
			wantThatJSON: "null",
		},
		{
			name:         "nil vs not nil aren't equal",
			this:         StringValue(""),
			wantThisJSON: `{"Type":"String","String":""}`,
			wantThatJSON: "null",
		},
		{
			name:         "values of different types are not equal",
			this:         IntValue(0),
			that:         FloatValue(0),
			wantThisJSON: `{"Type":"Int","Int":0}`,
			wantThatJSON: `{"Type":"Float","Float":0}`,
		},
		{
			name:         "values of different list types are not equal",
			this:         IntListValue(),
			that:         FloatListValue(),
			wantThisJSON: `{"Type":"IntList","IntList":null}`,
			wantThatJSON: `{"Type":"FloatList","FloatList":null}`,
		},
		{
			name:         "equal empty string values",
			this:         StringValue(""),
			that:         StringValue(""),
			want:         true,
			wantThisJSON: `{"Type":"String","String":""}`,
			wantThatJSON: `{"Type":"String","String":""}`,
		},
		{
			name:         "equal string values",
			this:         StringValue("this"),
			that:         StringValue("this"),
			wantThisJSON: `{"Type":"String","String":"this"}`,
			wantThatJSON: `{"Type":"String","String":"this"}`,
			want:         true,
		},
		{
			name:         "unequal string values",
			this:         StringValue("this"),
			that:         StringValue("that"),
			wantThisJSON: `{"Type":"String","String":"this"}`,
			wantThatJSON: `{"Type":"String","String":"that"}`,
		},
		{
			name:         "empty equal int values",
			this:         IntValue(0),
			that:         IntValue(0),
			want:         true,
			wantThisJSON: `{"Type":"Int","Int":0}`,
			wantThatJSON: `{"Type":"Int","Int":0}`,
		},
		{
			name:         "equal int values",
			this:         IntValue(1),
			that:         IntValue(1),
			want:         true,
			wantThisJSON: `{"Type":"Int","Int":1}`,
			wantThatJSON: `{"Type":"Int","Int":1}`,
		},
		{
			name:         "unequal int values",
			this:         IntValue(0),
			that:         IntValue(1),
			wantThisJSON: `{"Type":"Int","Int":0}`,
			wantThatJSON: `{"Type":"Int","Int":1}`,
		},
		{
			name:         "empty equal float values",
			this:         FloatValue(0),
			that:         FloatValue(0),
			want:         true,
			wantThisJSON: `{"Type":"Float","Float":0}`,
			wantThatJSON: `{"Type":"Float","Float":0}`,
		},
		{
			name:         "equal float values",
			this:         FloatValue(2.4),
			that:         FloatValue(2.4),
			want:         true,
			wantThisJSON: `{"Type":"Float","Float":2.4}`,
			wantThatJSON: `{"Type":"Float","Float":2.4}`,
		},
		{
			name:         "unequal float values",
			this:         FloatValue(1.1),
			that:         FloatValue(2.2),
			wantThisJSON: `{"Type":"Float","Float":1.1}`,
			wantThatJSON: `{"Type":"Float","Float":2.2}`,
		},
		{
			name:         "equal bool values",
			this:         BoolValue(true),
			that:         BoolValue(true),
			want:         true,
			wantThisJSON: `{"Type":"Bool","Bool":true}`,
			wantThatJSON: `{"Type":"Bool","Bool":true}`,
		},
		{
			name:         "unequal bool values",
			this:         BoolValue(true),
			that:         BoolValue(false),
			wantThisJSON: `{"Type":"Bool","Bool":true}`,
			wantThatJSON: `{"Type":"Bool","Bool":false}`,
		},
		{
			name:         "empty string list",
			this:         StringListValue(),
			that:         StringListValue(),
			want:         true,
			wantThisJSON: `{"Type":"StringList","StringList":null}`,
			wantThatJSON: `{"Type":"StringList","StringList":null}`,
		},
		{
			name:         "unequal empty string list",
			this:         StringListValue("a"),
			that:         StringListValue(),
			wantThisJSON: `{"Type":"StringList","StringList":["a"]}`,
			wantThatJSON: `{"Type":"StringList","StringList":null}`,
		},
		{
			name:         "populated string list",
			this:         StringListValue("a", "bc", "d"),
			that:         StringListValue("a", "bc", "d"),
			want:         true,
			wantThisJSON: `{"Type":"StringList","StringList":["a","bc","d"]}`,
			wantThatJSON: `{"Type":"StringList","StringList":["a","bc","d"]}`,
		},
		{
			name:         "different string list",
			this:         StringListValue("a", "bc", "def"),
			that:         StringListValue("a", "bc", "d"),
			wantThisJSON: `{"Type":"StringList","StringList":["a","bc","def"]}`,
			wantThatJSON: `{"Type":"StringList","StringList":["a","bc","d"]}`,
		},
		{
			name:         "unequal populated string list",
			this:         StringListValue("a", "bc", "d"),
			that:         StringListValue("a", "bc"),
			wantThisJSON: `{"Type":"StringList","StringList":["a","bc","d"]}`,
			wantThatJSON: `{"Type":"StringList","StringList":["a","bc"]}`,
		},
		{
			name:         "empty int list",
			this:         IntListValue(),
			that:         IntListValue(),
			want:         true,
			wantThisJSON: `{"Type":"IntList","IntList":null}`,
			wantThatJSON: `{"Type":"IntList","IntList":null}`,
		},
		{
			name:         "unequal empty int list",
			this:         IntListValue(0),
			that:         IntListValue(),
			wantThisJSON: `{"Type":"IntList","IntList":[0]}`,
			wantThatJSON: `{"Type":"IntList","IntList":null}`,
		},
		{
			name:         "populated int list",
			this:         IntListValue(1, -23, 456),
			that:         IntListValue(1, -23, 456),
			want:         true,
			wantThisJSON: `{"Type":"IntList","IntList":[1,-23,456]}`,
			wantThatJSON: `{"Type":"IntList","IntList":[1,-23,456]}`,
		},
		{
			name:         "different int list",
			this:         IntListValue(1, -23, 789),
			that:         IntListValue(1, -23, 456),
			wantThisJSON: `{"Type":"IntList","IntList":[1,-23,789]}`,
			wantThatJSON: `{"Type":"IntList","IntList":[1,-23,456]}`,
		},
		{
			name:         "unequal populated int list",
			this:         IntListValue(1, -23, 456),
			that:         IntListValue(1, -23),
			wantThisJSON: `{"Type":"IntList","IntList":[1,-23,456]}`,
			wantThatJSON: `{"Type":"IntList","IntList":[1,-23]}`,
		},
		{
			name:         "empty float list",
			this:         FloatListValue(),
			that:         FloatListValue(),
			want:         true,
			wantThisJSON: `{"Type":"FloatList","FloatList":null}`,
			wantThatJSON: `{"Type":"FloatList","FloatList":null}`,
		},
		{
			name:         "unequal empty float list",
			this:         FloatListValue(0),
			that:         FloatListValue(),
			wantThisJSON: `{"Type":"FloatList","FloatList":[0]}`,
			wantThatJSON: `{"Type":"FloatList","FloatList":null}`,
		},
		{
			name:         "populated float list",
			this:         FloatListValue(1, -2.3, 0.456),
			that:         FloatListValue(1, -2.3, 0.456),
			want:         true,
			wantThisJSON: `{"Type":"FloatList","FloatList":[1,-2.3,0.456]}`,
			wantThatJSON: `{"Type":"FloatList","FloatList":[1,-2.3,0.456]}`,
		},
		{
			name:         "different float list",
			this:         FloatListValue(1, -2.3, 45.6),
			that:         FloatListValue(1, -2.3, 0.456),
			wantThisJSON: `{"Type":"FloatList","FloatList":[1,-2.3,45.6]}`,
			wantThatJSON: `{"Type":"FloatList","FloatList":[1,-2.3,0.456]}`,
		},
		{
			name:         "unequal populated float list",
			this:         FloatListValue(1, -2.3, 0.456),
			that:         FloatListValue(-2.3, 0.456),
			wantThisJSON: `{"Type":"FloatList","FloatList":[1,-2.3,0.456]}`,
			wantThatJSON: `{"Type":"FloatList","FloatList":[-2.3,0.456]}`,
		},
		/* Usefor for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.this.Equal(test.that); got != test.want {
				t.Errorf("Value(%v).Equal(Value(%v)) returned %v; want %v", test.this, test.that, got, test.want)
			}

			if got := test.that.Equal(test.this); got != test.want {
				t.Errorf("Value(%v).Equal(Value(%v)) returned %v; want %v", test.that, test.this, got, test.want)
			}

			gotThisJSON, err := json.Marshal(test.this)
			if err != nil {
				t.Fatalf("json.Marshal(%v) [this] returned error: %v", test.this, err)
			}
			if diff := cmp.Diff(test.wantThisJSON, string(gotThisJSON)); diff != "" {
				t.Errorf("json.Marshal(%v) [this] produced diff (-want, +got):\n%s", test.this, diff)
			}

			gotThatJSON, err := json.Marshal(test.that)
			if err != nil {
				t.Fatalf("json.Marshal(%v) [that] returned error: %v", test.that, err)
			}
			if diff := cmp.Diff(test.wantThatJSON, string(gotThatJSON)); diff != "" {
				t.Errorf("json.Marshal(%v) [that] produced diff (-want, +got):\n%s", test.that, diff)
			}

			// Unmarshal and verify still equal.
			unmarshalledThis := &Value{}
			if err := json.Unmarshal(gotThisJSON, unmarshalledThis); err != nil {
				t.Fatalf("json.Unmarshal(%s) [this] returned an error: %v", gotThisJSON, err)
			}
			wantThis := test.this
			if test.this == nil {
				wantThis = &Value{}
			}
			if diff := cmp.Diff(wantThis, unmarshalledThis); diff != "" {
				t.Errorf("json marshal + unmarshal [this] produced diff (-want, +got):\n%s", diff)
			}

			unmarshalledThat := &Value{}
			if err := json.Unmarshal(gotThatJSON, unmarshalledThat); err != nil {
				t.Fatalf("json.Unmarshal(%s) [that] returned an error: %v", gotThatJSON, err)
			}
			wantThat := test.that
			if test.that == nil {
				wantThat = &Value{}
			}
			if diff := cmp.Diff(wantThat, unmarshalledThat); diff != "" {
				t.Errorf("json marshal + unmarshal [that] produced diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestValueTypeErrors(t *testing.T) {
	for _, val := range []int{0, 8, -3, 15} {
		t.Run(fmt.Sprintf("marshaling ValueType(%d)", val), func(t *testing.T) {
			vt := ValueType(val)
			wantErr := fmt.Sprintf("json: error calling MarshalJSON for type command.ValueType: unknown ValueType: %d", val)
			_, err := json.Marshal(vt)
			if err == nil {
				t.Fatalf("json.Marshal(%v) returned nil error; want %q", vt, wantErr)
			}
			if diff := cmp.Diff(err.Error(), wantErr); diff != "" {
				t.Errorf("json.Marshal(%v) returned error diff:\n%s", vt, diff)
			}
		})
	}

	for _, test := range []struct {
		name    string
		val     string
		wantErr string
	}{
		{
			name:    "empty string",
			val:     "",
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "empty JSON object",
			val:     "{}",
			wantErr: "ValueType requires string value: json: cannot unmarshal object into Go value of type string",
		},
		{
			name:    "number",
			val:     "123",
			wantErr: "ValueType requires string value: json: cannot unmarshal number into Go value of type string",
		},
		{
			name:    "float",
			val:     "12.3",
			wantErr: "ValueType requires string value: json: cannot unmarshal number into Go value of type string",
		},
		{
			name:    "null",
			val:     "null",
			wantErr: `unknown ValueType: ""`,
		},
		{
			name:    "empty string",
			val:     `""`,
			wantErr: `unknown ValueType: ""`,
		},
		{
			name:    "random string",
			val:     `"hello"`,
			wantErr: `unknown ValueType: "hello"`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var vt ValueType
			err := json.Unmarshal([]byte(test.val), &vt)
			if err == nil {
				t.Fatalf("json.Unmarshal(%v) returned nil error; want %q", vt, test.wantErr)
			}
			if diff := cmp.Diff(err.Error(), test.wantErr); diff != "" {
				t.Errorf("json.Unmarshal(%v) returned error diff:\n%s", vt, diff)
			}
		})
	}

	for _, test := range []struct {
		name    string
		val     *Value
		wantErr string
		wantStr string
	}{
		{
			name:    "empty value",
			val:     &Value{},
			wantErr: "json: error calling MarshalJSON for type *command.Value: unknown ValueType: 0",
			wantStr: "UNKNOWN_VALUE_TYPE",
		},
		{
			name:    "value with invalid type",
			val:     &Value{type_: 8},
			wantErr: "json: error calling MarshalJSON for type *command.Value: unknown ValueType: 8",
			wantStr: "UNKNOWN_VALUE_TYPE",
		},
		{
			name:    "value with other invalid type",
			val:     &Value{type_: -1},
			wantErr: "json: error calling MarshalJSON for type *command.Value: unknown ValueType: -1",
			wantStr: "UNKNOWN_VALUE_TYPE",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := json.Marshal(test.val)
			if err == nil {
				t.Fatalf("json.Mmarshal(%v) returned nil error; want %q", test.val, test.wantErr)
			}
			if diff := cmp.Diff(err.Error(), test.wantErr); diff != "" {
				t.Errorf("json.Marshal(%v) returned error diff:\n%s", test.val, diff)
			}
			if diff := cmp.Diff(test.wantStr, test.val.Str()); diff != "" {
				t.Errorf("Value(%v).Str() produced diff: %v", test.val, diff)
			}
		})
	}
}

func TestNilValueReturnsAllNil(t *testing.T) {
	var v *Value
	if v.String() != "" {
		t.Errorf(`Value(nil).String() returned %s; want ""`, v.String())
	}
	if v.Int() != 0 {
		t.Errorf(`Value(nil).Int() returned %d; want 0`, v.Int())
	}
	if v.Float() != 0 {
		t.Errorf(`Value(nil).Float() returned %0.2f; want 0.0`, v.Float())
	}
	if v.Bool() != false {
		t.Errorf(`Value(nil).Bool() returned %v; want false`, v.Bool())
	}
	if v.StringList() != nil {
		t.Errorf(`Value(nil).StringList() returned %v; want false`, v.StringList())
	}
	if v.IntList() != nil {
		t.Errorf(`Value(nil).IntList() returned %v; want false`, v.IntList())
	}
	if v.FloatList() != nil {
		t.Errorf(`Value(nil).FloatList() returned %v; want false`, v.FloatList())
	}
}

func TestHasArg(t *testing.T) {
	d := &Data{}
	d.Set("yes", StringValue("hello"))

	if !d.HasArg("yes") {
		t.Errorf("data.HasArg('yes') returned false; want true")
	}

	if d.HasArg("no") {
		t.Errorf("data.HasArg('no') returned true; want false")
	}
}
