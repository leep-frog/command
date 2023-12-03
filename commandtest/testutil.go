package commandtest

import (
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command/internal/testutil"
)

// CmpError compares the provided errors
func CmpError(t *testing.T, funcString string, wantErr, err error, opts ...cmp.Option) {
	t.Helper()
	testutil.CmpError(t, funcString, wantErr, err, opts...)
}

// Cmp compares the two provided fields.
func Cmp[T any](t *testing.T, prefix string, want, got T, opts ...cmp.Option) {
	t.Helper()
	testutil.Cmp(t, prefix, want, got, opts...)
}

// CmpPanic runs the provided function, `f`, and verifies the proper panic value is recoverd.
func CmpPanic[T any](t *testing.T, funcString string, f func() T, want interface{}, opts ...cmp.Option) T {
	t.Helper()
	return testutil.CmpPanic(t, funcString, f, want, opts...)
}

// TempFile creates a temporary file and fails the test if not successful.
func TempFile(t *testing.T, pattern string) *os.File {
	t.Helper()
	return testutil.TempFile(t, pattern)
}

// StubValue stubs the originalValue with newValue for the duration of the test run.
func StubValue[T any](t *testing.T, originalValue *T, newValue T) {
	t.Helper()
	testutil.StubValue(t, originalValue, newValue)
}

// FilepathAbs returns the absolute, joined filepath for the provided strings. Fails the test if unsuccessful.
func FilepathAbs(t *testing.T, s ...string) string {
	t.Helper()
	return testutil.FilepathAbs(t, s...)
}

// Write writes the provided contents (joined by newline characters) to iow.
func Write(t *testing.T, iow io.Writer, contents []string) {
	t.Helper()
	testutil.Write(t, iow, contents)
}
