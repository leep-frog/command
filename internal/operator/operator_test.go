package operator

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/internal/testutil"
)

type toArgsTest[T any] struct {
	name     string
	operator Operator[T]
	value    T
	want     []string
}

func (tat *toArgsTest[T]) Run(t *testing.T) {
	t.Run(tat.name, func(t *testing.T) {
		prefix := fmt.Sprintf("operator.toArgs(%v)", tat.value)
		testutil.Cmp(t, prefix, tat.want, tat.operator.toArgs(tat.value))
	})
}

type fromArgsTest[T any] struct {
	name     string
	operator Operator[T]
	args     []string
	want     T
	wantErr  error
}

func (fat *fromArgsTest[T]) Run(t *testing.T) {
	t.Run(fat.name, func(t *testing.T) {
		var ptrArgs []*string
		for i := range fat.args {
			ptrArgs = append(ptrArgs, &fat.args[i])
		}
		prefix := fmt.Sprintf("operator.fromArgs(%v)", fat.args)
		got, err := fat.operator.fromArgs(ptrArgs)
		testutil.Cmp(t, prefix, fat.want, got)
		testutil.CmpError(t, prefix, fat.wantErr, err)
	})
}

func TestToArgs(t *testing.T) {
	for _, test := range []testutil.GenericTest{
		// bool operator
		&toArgsTest[bool]{
			name:     "bool false to arg",
			operator: &boolOperator{},
			value:    false,
			want:     []string{"false"},
		},
		&toArgsTest[bool]{
			name:     "bool true to arg",
			operator: &boolOperator{},
			value:    true,
			want:     []string{"true"},
		},
		&fromArgsTest[bool]{
			name:     "bool empty args",
			operator: &boolOperator{},
		},
		&fromArgsTest[bool]{
			name:     "bool arg to false",
			operator: &boolOperator{},
			args:     []string{"false"},
			want:     false,
		},
		&fromArgsTest[bool]{
			name:     "bool arg to true",
			operator: &boolOperator{},
			args:     []string{"true"},
			want:     true,
		},
		&fromArgsTest[bool]{
			name:     "bool arg to true with extra args",
			operator: &boolOperator{},
			args:     []string{"TRUE", "bleh"},
			want:     true,
		},
		&fromArgsTest[bool]{
			name:     "bool arg error",
			operator: &boolOperator{},
			args:     []string{"truth"},
			wantErr:  fmt.Errorf(`strconv.ParseBool: parsing "truth": invalid syntax`),
		},
		// int operator
		&toArgsTest[int]{
			name:     "int negative to arg",
			operator: &intOperator{},
			value:    -123,
			want:     []string{"-123"},
		},
		&toArgsTest[int]{
			name:     "int postive to arg",
			operator: &intOperator{},
			value:    4,
			want:     []string{"4"},
		},
		&fromArgsTest[int]{
			name:     "int empty args",
			operator: &intOperator{},
		},
		&fromArgsTest[int]{
			name:     "int arg to negative",
			operator: &intOperator{},
			args:     []string{"-56"},
			want:     -56,
		},
		&fromArgsTest[int]{
			name:     "int arg to positive",
			operator: &intOperator{},
			args:     []string{"789"},
			want:     789,
		},
		&fromArgsTest[int]{
			name:     "int arg to true with extra args",
			operator: &intOperator{},
			args:     []string{"10", "bleh"},
			want:     10,
		},
		&fromArgsTest[int]{
			name:     "int arg error",
			operator: &intOperator{},
			args:     []string{"eleven"},
			wantErr:  fmt.Errorf(`strconv.Atoi: parsing "eleven": invalid syntax`),
		},
		// int list operator
		&toArgsTest[[]int]{
			name:     "intList values to args",
			operator: &intListOperator{},
			value:    []int{-123, 0, 45},
			want:     []string{"-123", "0", "45"},
		},
		&fromArgsTest[[]int]{
			name:     "intList empty args",
			operator: &intListOperator{},
		},
		&fromArgsTest[[]int]{
			name:     "intList args to values",
			operator: &intListOperator{},
			args:     []string{"-56", "78", "0", "-9"},
			want:     []int{-56, 78, 0, -9},
		},
		&fromArgsTest[[]int]{
			name:     "intList arg error",
			operator: &intListOperator{},
			args:     []string{"12", "thirteen"},
			want:     []int{12, 0},
			wantErr:  fmt.Errorf(`strconv.Atoi: parsing "thirteen": invalid syntax`),
		},
		// float64 operator
		&toArgsTest[float64]{
			name:     "float negative to arg",
			operator: &floatOperator{},
			value:    -12.3,
			want:     []string{"-12.3"},
		},
		&toArgsTest[float64]{
			name:     "float postive to arg",
			operator: &floatOperator{},
			value:    0.4,
			want:     []string{"0.4"},
		},
		&fromArgsTest[float64]{
			name:     "float empty args",
			operator: &floatOperator{},
		},
		&fromArgsTest[float64]{
			name:     "float arg to negative",
			operator: &floatOperator{},
			args:     []string{"-56"},
			want:     -56,
		},
		&fromArgsTest[float64]{
			name:     "float arg to positive",
			operator: &floatOperator{},
			args:     []string{"78.09"},
			want:     78.09,
		},
		&fromArgsTest[float64]{
			name:     "float arg to true with extra args",
			operator: &floatOperator{},
			args:     []string{"1.0", "bleh"},
			want:     1.0,
		},
		&fromArgsTest[float64]{
			name:     "float arg error",
			operator: &floatOperator{},
			args:     []string{"eleven"},
			wantErr:  fmt.Errorf(`strconv.ParseFloat: parsing "eleven": invalid syntax`),
		},
		// float64 list operator
		&toArgsTest[[]float64]{
			name:     "floatList values to args",
			operator: &floatListOperator{},
			value:    []float64{-12.3, 0, 0.45},
			want:     []string{"-12.3", "0", "0.45"},
		},
		&fromArgsTest[[]float64]{
			name:     "floatList empty args",
			operator: &floatListOperator{},
		},
		&fromArgsTest[[]float64]{
			name:     "floatList args to values",
			operator: &floatListOperator{},
			args:     []string{"-5.6", "78", "0", "-0.9"},
			want:     []float64{-5.6, 78, 0, -0.9},
		},
		&fromArgsTest[[]float64]{
			name:     "floatList arg error",
			operator: &floatListOperator{},
			args:     []string{"1.2", "thirteen"},
			want:     []float64{1.2, 0},
			wantErr:  fmt.Errorf(`strconv.ParseFloat: parsing "thirteen": invalid syntax`),
		},
		// string operator
		&toArgsTest[string]{
			name:     "string value to arg",
			operator: &stringOperator{},
			value:    "bloop",
			want:     []string{"bloop"},
		},
		&toArgsTest[string]{
			name:     "string empty to arg",
			operator: &stringOperator{},
			want:     []string{""},
		},
		&fromArgsTest[string]{
			name:     "string empty args",
			operator: &stringOperator{},
		},
		&fromArgsTest[string]{
			name:     "string arg to value",
			operator: &stringOperator{},
			args:     []string{"yo"},
			want:     "yo",
		},
		&fromArgsTest[string]{
			name:     "string arg to value with extra",
			operator: &stringOperator{},
			args:     []string{"yo", "delaeioo"},
			want:     "yo",
		},
		// string list operator
		&toArgsTest[[]string]{
			name:     "stringList values to args",
			operator: &stringListOperator{},
			value:    []string{"un", "2", "iii"},
			want:     []string{"un", "2", "iii"},
		},
		&fromArgsTest[[]string]{
			name:     "stringList empty args",
			operator: &stringListOperator{},
			want:     []string{},
		},
		&fromArgsTest[[]string]{
			name:     "stringList args to values",
			operator: &stringListOperator{},
			args:     []string{"un", "2", "iii"},
			want:     []string{"un", "2", "iii"},
		},
	} {
		test.Run(t)
	}
}

func TestParseInt(t *testing.T) {
	errFunc := func(s string) error {
		return fmt.Errorf(`strconv.Atoi: parsing %q: invalid syntax`, s)
	}
	for _, test := range []struct {
		name    string
		arg     string
		wantErr error
		want    int
	}{
		{
			name: "No underscore",
			arg:  "123",
			want: 123,
		},
		{
			name:    "Decimal",
			arg:     "12.3",
			wantErr: errFunc("12.3"),
		},
		{
			name: "Underscore in the middle",
			arg:  "1_2_3",
			want: 123,
		},
		{
			name:    "Fails if underscore at the beginning",
			arg:     "_123",
			wantErr: errFunc("_123"),
		},
		{
			name:    "Fails if underscore at the end",
			arg:     "123_",
			wantErr: errFunc("123_"),
		},
		{
			name:    "Fails if multiple underscores in a row",
			arg:     "1__23",
			wantErr: errFunc("1__23"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseInt(test.arg)
			testutil.CmpError(t, "ParseInt(%s)", test.wantErr, err)
			testutil.Cmp(t, fmt.Sprintf("ParseInt(%s) returned incorrect value", test.arg), test.want, got)
		})
	}
}

func TestParseFloat(t *testing.T) {
	errFunc := func(s string) error {
		return fmt.Errorf(`strconv.ParseFloat: parsing %q: invalid syntax`, s)
	}
	for _, test := range []struct {
		name    string
		arg     string
		wantErr error
		want    float64
	}{
		{
			name: "No underscore",
			arg:  "123",
			want: 123,
		},
		{
			name: "No numbers after decimal",
			arg:  "123.",
			want: 123,
		},
		{
			name: "No underscore with decimals",
			arg:  "123.45",
			want: 123.45,
		},
		{
			name: "No underscore with decimal zero",
			arg:  "123.0",
			want: 123,
		},
		{
			name: "No underscore with multiple decimal zeros",
			arg:  "123.000",
			want: 123,
		},
		{
			name: "No underscore with multiple trailing decimal zeros",
			arg:  "123.0300",
			want: 123.03,
		},
		{
			name: "Underscore in the middle",
			arg:  "1_2_3",
			want: 123,
		},
		{
			name: "Underscore in the middle of decimal",
			arg:  "1_2_3.4_5_6",
			want: 123.456,
		},
		{
			name:    "Fails if underscore at the beginning",
			arg:     "_123",
			wantErr: errFunc("_123"),
		},
		{
			name:    "Fails if underscore at the beginning of decimal",
			arg:     "123._4",
			wantErr: errFunc("123._4"),
		},
		{
			name:    "Fails if underscore at the end",
			arg:     "123_",
			wantErr: errFunc("123_"),
		},
		{
			name:    "Fails if underscore at the end of decimal",
			arg:     "123.456_",
			wantErr: errFunc("123.456_"),
		},
		{
			name:    "Fails if multiple underscores in a row",
			arg:     "1__23",
			wantErr: errFunc("1__23"),
		},
		{
			name:    "Fails if multiple underscores in a row in decimal",
			arg:     "123.45__6",
			wantErr: errFunc("123.45__6"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseFloat(test.arg)
			testutil.CmpError(t, "parseFloat(%s)", test.wantErr, err)
			testutil.Cmp(t, fmt.Sprintf("ParseFloat(%s) returned incorrect value", test.arg), test.want, got)
		})
	}
}

type getOperatorTest[T any] struct {
	want      Operator[T]
	wantPanic interface{}
}

func (g *getOperatorTest[T]) Run(t *testing.T) {
	got := testutil.CmpPanic(t, "GetOperator()", func() Operator[T] { return GetOperator[T]() }, g.wantPanic)
	testutil.Cmp(t, "GetOperator() returned incorrect operator", g.want, got)
}

func TestGetOperator(t *testing.T) {
	for _, test := range []struct {
		name string
		gt   testutil.GenericTest
	}{
		{
			name: "string operator",
			gt: &getOperatorTest[string]{
				want: &stringOperator{},
			},
		},
		{
			name: "string list operator",
			gt: &getOperatorTest[[]string]{
				want: &stringListOperator{},
			},
		},
		{
			name: "int operator",
			gt: &getOperatorTest[int]{
				want: &intOperator{},
			},
		},
		{
			name: "int list operator",
			gt: &getOperatorTest[[]int]{
				want: &intListOperator{},
			},
		},
		{
			name: "float operator",
			gt: &getOperatorTest[float64]{
				want: &floatOperator{},
			},
		},
		{
			name: "float list operator",
			gt: &getOperatorTest[[]float64]{
				want: &floatListOperator{},
			},
		},
		{
			name: "bool operator",
			gt: &getOperatorTest[bool]{
				want: &boolOperator{},
			},
		},
		{
			name: "panic on unknown operator",
			gt: &getOperatorTest[*string]{
				wantPanic: `no operator defined for type *string`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.gt.Run(t)
		})
	}
}
