package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type GenericTest interface {
	Run(t *testing.T)
}

func CmpError(t *testing.T, funcString string, wantErr, err error, opts ...cmp.Option) {
	t.Helper()

	if wantErr == nil && err != nil {
		t.Errorf("%s returned error (%v) when shouldn't have", funcString, err)
	}
	if wantErr != nil {
		if err == nil {
			t.Errorf("%s returned no error when should have returned %v", funcString, wantErr)
		} else if diff := cmp.Diff(wantErr.Error(), err.Error(), opts...); diff != "" {
			t.Errorf("%s returned unexpected error (-want, +got):\n%s", funcString, diff)
		}
	}
}

func Cmp[T any](t *testing.T, prefix string, want, got T, opts ...cmp.Option) {
	t.Helper()

	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Errorf("%s (-want, +got):\n%s", prefix, diff)
	}
}

func CmpPanic[T any](t *testing.T, funcString string, f func() T, want interface{}, opts ...cmp.Option) T {
	t.Helper()

	defer func() {
		Cmp(t, fmt.Sprintf("%s panicked with incorrect value", funcString), want, recover(), opts...)
	}()

	return f()
}

func TempFile(t *testing.T, pattern string) *os.File {
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { tmp.Close() })
	return tmp
}

func StubValue[T any](t *testing.T, originalValue *T, newValue T) {
	oldValue := *originalValue
	*originalValue = newValue
	t.Cleanup(func() {
		*originalValue = oldValue
	})
}

func FilepathAbs(t *testing.T, s ...string) string {
	t.Helper()
	r, err := filepath.Abs(filepath.Join(s...))
	if err != nil {
		t.Fatalf("Failed to get absolute path for file: %v", err)
	}
	return r
}
