package cachetest

import (
	"testing"

	"github.com/leep-frog/command/cache"
	"github.com/leep-frog/command/internal/spycachetest"
)

func makeCache(dir string) *cache.Cache {
	return &cache.Cache{Dir: dir}
}

// NewTestCache is a function useful for stubbing out caches in tests.
func NewTestCache(t *testing.T) *cache.Cache {
	t.Helper()
	return spycachetest.NewTestCache[*cache.Cache](t, makeCache)
}

// NewTestCacheWithData creates a test cache with the provided key-values.
// String values are set using Cache.Put. All other values are set with Cache.PutStruct.
func NewTestCacheWithData(t *testing.T, m map[string]interface{}) *cache.Cache {
	t.Helper()
	return spycachetest.NewTestCacheWithData[*cache.Cache](t, m, makeCache)
}
