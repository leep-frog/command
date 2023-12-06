package spycommandertest

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/internal/spycommandtest"
	"github.com/leep-frog/command/internal/spyinput"
	"github.com/leep-frog/command/internal/testutil"
)

type inputTester struct {
	skip bool
	want *spycommandtest.SpyInput
}

func (*inputTester) setup(*testing.T, *testContext) {}
func (it *inputTester) check(t *testing.T, tc *testContext) {
	t.Helper()
	if it.skip {
		if it.want != nil {
			t.Fatalf("SkipInputCheck set to true, but WantInput was provided")
		}
		return
	}

	if it.want == nil {
		it.want = &spycommandtest.SpyInput{}
	}

	gotPtr := getUnexportedField(tc.input, "si").(*spyinput.SpyInput[command.InputBreaker])
	got := &spycommandtest.SpyInput{}
	if gotPtr != nil {
		got = (*spycommandtest.SpyInput)(gotPtr)
	}
	testutil.Cmp(t, fmt.Sprintf("%s incorrectly modified input", tc.prefix), it.want, got, cmpopts.EquateEmpty())
}

// From https://stackoverflow.com/questions/42664837/how-to-access-unexported-struct-fields
func getUnexportedField(obj any, fieldName string) interface{} {
	field := reflect.ValueOf(obj).Elem().FieldByName(fieldName)
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}
