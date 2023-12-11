package spycachetest

import (
	"os"
	"testing"
)

type SpyCache interface {
	Put(string, string) error
	PutStruct(string, interface{}) error
}

type SpyCacheBag[C any] struct {
	MakeCache func(dir string)
}

// NewTestCache is a function useful for stubbing out caches in tests.
func NewTestCache[C SpyCache](t *testing.T, makeCache func(dir string) C) C {
	t.Helper()
	return NewTestCacheWithData[C](t, nil, makeCache)
}

// NewTestCacheWithData creates a test cache with the provided key-values.
// String values are set using Cache.Put. All other values are set with Cache.PutStruct.
func NewTestCacheWithData[C SpyCache](t *testing.T, m map[string]interface{}, makeCache func(dir string) C) C {
	t.Helper()
	dir, err := os.MkdirTemp("", "test-leep-frog-command-cache")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("failed to clean up test cache: %v", err)
		}
	})
	c := makeCache(dir)
	for k, v := range m {
		if s, ok := v.(string); ok {
			if err := c.Put(k, s); err != nil {
				t.Fatalf("Cache.Put(%s, %s) returned error: %v", k, s, err)
			}
		} else if err := c.PutStruct(k, v); err != nil {
			t.Fatalf("Cache.PutStruct(%s, %v) returned error: %v", k, v, err)
		}
	}
	return c
}
