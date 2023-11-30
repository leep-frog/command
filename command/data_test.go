package command

import (
	"fmt"
	"testing"

	"github.com/leep-frog/command/internal/testutil"
)

func TestHasArg(t *testing.T) {
	d := &Data{}
	d.Set("yes", "hello")

	if !d.Has("yes") {
		t.Errorf("data.Has('yes') returned false; want true")
	}

	if d.Has("no") {
		t.Errorf("data.Has('no') returned true; want false")
	}
}

type getDataTest[T any] struct {
	d    *Data
	key  string
	want T
}

func (gdt *getDataTest[T]) Run(t *testing.T) {
	got := GetData[T](gdt.d, gdt.key)
	testutil.Cmp(t, fmt.Sprintf("GetData(%v, %s)", gdt.d, gdt.key), gdt.want, got)
}

func toPtr[T any](t T) *T {
	return &t
}

func TestGetData(t *testing.T) {
	for _, test := range []struct {
		name string
		gdt  testutil.GenericTest
	}{
		// string tests
		{
			"nil data returns nil-ish value for string",
			&getDataTest[string]{
				want: "",
			},
		},
		{
			"nil data.values returns nil-ish value for string",
			&getDataTest[string]{
				d:    &Data{},
				want: "",
			},
		},
		{
			"missing key returns nil-ish value for string",
			&getDataTest[string]{
				d:    &Data{Values: map[string]interface{}{}},
				want: "",
			},
		},
		{
			"valid key returns value for string",
			&getDataTest[string]{
				key: "some-key",
				d: &Data{Values: map[string]interface{}{
					"some-key": "some-value",
				}},
				want: "some-value",
			},
		},
		// int tests
		{
			"nil data returns nil-ish value for int",
			&getDataTest[int]{
				want: 0,
			},
		},
		{
			"nil data.values returns nil-ish value for int",
			&getDataTest[int]{
				d:    &Data{},
				want: 0,
			},
		},
		{
			"missing key returns nil-ish value for int",
			&getDataTest[int]{
				d:    &Data{Values: map[string]interface{}{}},
				want: 0,
			},
		},
		{
			"valid key returns value for int",
			&getDataTest[int]{
				key: "some-key",
				d: &Data{Values: map[string]interface{}{
					"some-key": 2468,
				}},
				want: 2468,
			},
		},
		// pointer tests
		{
			"nil data returns nil value for pointer",
			&getDataTest[*int]{
				want: nil,
			},
		},
		{
			"nil data.values returns nil value for pointer",
			&getDataTest[*int]{
				d:    &Data{},
				want: nil,
			},
		},
		{
			"missing key returns nil value for pointer",
			&getDataTest[*int]{
				d:    &Data{Values: map[string]interface{}{}},
				want: nil,
			},
		},
		{
			"valid key returns value for pointer",
			&getDataTest[*int]{
				key: "some-key",
				d: &Data{Values: map[string]interface{}{
					"some-key": toPtr(1234),
				}},
				want: toPtr(1234),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.gdt.Run(t)
		})
	}
}

type dataGetTest[T any] struct {
	d         *Data
	f         func(*Data) T
	want      T
	wantPanic interface{}
}

func (d *dataGetTest[T]) Run(t *testing.T) {
	got := testutil.CmpPanic(t, "func(*Data)", func() T { return d.f(d.d) }, d.wantPanic)
	testutil.Cmp(t, "func(*Data) returned incorrect data", d.want, got)
}

func TestDataGet(t *testing.T) {

	type someType struct {
		S string
		I *int
	}

	for _, test := range []struct {
		name string
		gt   testutil.GenericTest
	}{
		// Note: the logic of GetData is tested above, so this just needs to verify typing info
		{
			name: "Data.String(...)",
			gt: &dataGetTest[string]{
				d: &Data{Values: map[string]interface{}{
					"some-key": "some-value",
				}},
				f:    func(d *Data) string { return d.String("some-key") },
				want: "some-value",
			},
		},
		{
			name: "Data.StringList(...)",
			gt: &dataGetTest[[]string]{
				d: &Data{Values: map[string]interface{}{
					"some-key": []string{"some", "value"},
				}},
				f:    func(d *Data) []string { return d.StringList("some-key") },
				want: []string{"some", "value"},
			},
		},
		{
			name: "Data.Int(...)",
			gt: &dataGetTest[int]{
				d: &Data{Values: map[string]interface{}{
					"some-key": 123,
				}},
				f:    func(d *Data) int { return d.Int("some-key") },
				want: 123,
			},
		},
		{
			name: "Data.IntList(...)",
			gt: &dataGetTest[[]int]{
				d: &Data{Values: map[string]interface{}{
					"some-key": []int{1, 23, -4},
				}},
				f:    func(d *Data) []int { return d.IntList("some-key") },
				want: []int{1, 23, -4},
			},
		},
		{
			name: "Data.Float(...)",
			gt: &dataGetTest[float64]{
				d: &Data{Values: map[string]interface{}{
					"some-key": 12.3,
				}},
				f:    func(d *Data) float64 { return d.Float("some-key") },
				want: 12.3,
			},
		},
		{
			name: "Data.FloatList(...)",
			gt: &dataGetTest[[]float64]{
				d: &Data{Values: map[string]interface{}{
					"some-key": []float64{1, 2.3, -0.4},
				}},
				f:    func(d *Data) []float64 { return d.FloatList("some-key") },
				want: []float64{1, 2.3, -0.4},
			},
		},
		{
			name: "Data.Bool(...)",
			gt: &dataGetTest[bool]{
				d: &Data{Values: map[string]interface{}{
					"some-key": true,
				}},
				f:    func(d *Data) bool { return d.Bool("some-key") },
				want: true,
			},
		},
		{
			name: "Data.Get(...) returns interface",
			gt: &dataGetTest[interface{}]{
				d: &Data{Values: map[string]interface{}{
					"some-key": &someType{"abc", toPtr(123)},
				}},
				f:    func(d *Data) interface{} { return d.Get("some-key") },
				want: &someType{"abc", toPtr(123)},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.gt.Run(t)
		})
	}
}
