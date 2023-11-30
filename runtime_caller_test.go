package commander

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRuntimeCaller(t *testing.T) {
	expected, err := filepath.Abs("runtime_caller_test.go")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	d := &Data{}
	o := NewFakeOutput()
	rc := RuntimeCaller()
	if err := rc.Execute(nil, o, d, nil); err != nil {
		t.Fatalf("failed to execute runtime caller: %v", err)
	}

	expected = filepath.ToSlash(expected)
	actual := filepath.ToSlash(rc.Get(d))
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("RuntimeCaller() produced incorrect filepath (-want, +got):\n%s", diff)
	}
}
