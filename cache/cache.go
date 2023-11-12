package cache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/leep-frog/command"
)

const (
	keyRegex = `^([a-zA-Z0-9_\.-]+)$`
)

var (
	osMkdirAll = os.MkdirAll
)

// Cache is a type for caching data in JSON files. It implements the `sourcerer.CLI` interface.
type Cache struct {
	// Dir is the location for storing the cache data.
	Dir     string
	changed bool
}

// Name returns the name of the cache CLI.
func (c *Cache) Name() string {
	return "cash"
}

// Changed returns whether or not the `Cache` object (not cache data) has changed.
func (c *Cache) Changed() bool {
	return c.changed
}

// Setup fulfills the `sourcerer.CLI` interface.
func (c *Cache) Setup() []string { return nil }

// Node returns the `command.Node` for the cache CLI.
func (c *Cache) Node() command.Node {
	arg := command.Arg[string]("KEY", "Key of the data to get", command.MatchesRegex(keyRegex), completer(c))
	return &command.BranchNode{
		Branches: map[string]command.Node{
			"setdir": command.SerialNodes(
				command.FileArgument("DIR", "Directory in which to store data", command.IsDir()),
				&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
					c.Dir = d.String("DIR")
					c.changed = true
					return nil
				}},
			),
			"get": command.SerialNodes(
				arg,
				&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
					s, ok, err := c.Get(d.String(arg.Name()))
					if err != nil {
						return o.Err(err)
					}
					if !ok {
						o.Stderrln("key not found")
					} else {
						o.Stdoutln(s)
					}
					return nil
				}},
			),
			"put": command.SerialNodes(
				arg,
				command.ListArg[string]("DATA", "Data to store", 1, command.UnboundedList),
				&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
					return o.Err(c.Put(d.String(arg.Name()), strings.Join(d.StringList("DATA"), " ")))
				}},
			),
			"delete": command.SerialNodes(
				arg,
				&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
					return o.Err(c.Delete(d.String(arg.Name())))
				}},
			),
			"list": command.SerialNodes(
				&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
					r, err := c.List()
					if err != nil {
						return o.Err(err)
					}
					sort.Strings(r)
					for _, s := range r {
						o.Stdoutln(s)
					}
					return nil
				}},
			),
		},
	}
}

func completer(c *Cache) command.Completer[string] {
	return command.CompleterFromFunc(func(s string, d *command.Data) (*command.Completion, error) {
		r, err := c.List()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %v", err)
		}
		return &command.Completion{
			Suggestions: r,
		}, nil
	})
}

// NewTestCache is a function useful for stubbing out caches in tests.
func NewTestCache(t *testing.T) *Cache {
	t.Helper()
	return NewTestCacheWithData(t, nil)
}

// NewTestCacheWithData creates a test cache with the provided key-values.
// String values are set using Cache.Put. All other values are set with Cache.PutStruct.
func NewTestCacheWithData(t *testing.T, m map[string]interface{}) *Cache {
	t.Helper()
	dir, err := ioutil.TempDir("", "test-leep-frog-command-cache")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("failed to clean up test cache: %v", err)
		}
	})
	c := &Cache{
		Dir: dir,
	}
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

// ForDir returns a cache pointing to the provided directory.
func ForDir(dir string) (*Cache, error) {
	c := &Cache{
		Dir: dir,
	}
	if _, err := c.getCacheDir(); err != nil {
		return nil, fmt.Errorf("invalid directory (%s) for cache: %v", dir, err)
	}
	return c, nil
}

// FromEnvVar creates a new cache pointing to the directory specified
// by the provided environment variable.
func FromEnvVar(e string) (*Cache, error) {
	v, ok := command.OSLookupEnv(e)
	if !ok || v == "" {
		return nil, fmt.Errorf("environment variable %q is not set", e)
	}
	return ForDir(v)
}

// FromEnvVar creates a new cache pointing to the directory specified
// by the provided environment variable.
func FromEnvVarOrDir(e, dir string) (*Cache, error) {
	v, ok := command.OSLookupEnv(e)
	if !ok || v == "" {
		return ForDir(dir)
	}
	return ForDir(v)
}

// Put puts data in the cache.
func (c *Cache) Put(key, data string) error {
	filename, err := c.fileFromKey(key)
	if err != nil {
		return fmt.Errorf("failed to get file for key: %v", err)
	}
	if err := ioutil.WriteFile(filename, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

// List lists all cache keys.
func (c *Cache) List() ([]string, error) {
	dir, err := c.getCacheDir()
	if err != nil {
		return nil, err
	}
	fs, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read files in directory: %v", err)
	}
	var r []string
	for _, f := range fs {
		r = append(r, f.Name())
	}
	return r, nil
}

// Delete deletes data from the cache.
func (c *Cache) Delete(key string) error {
	filename, err := c.fileFromKey(key)
	if err != nil {
		return fmt.Errorf("failed to get file for key: %v", err)
	}
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}

// GetBytes returns data, whether the file exists, and any error encountered.
func (c *Cache) GetBytes(key string) ([]byte, bool, error) {
	filename, err := c.fileFromKey(key)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get file for key: %v", err)
	}

	// Check if the file exists.
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read file: %v", err)
	}
	return data, true, nil
}

// Get retrieves data from the cache and returns the data (as a string), whether the file exists, and any error encountered.
func (c *Cache) Get(key string) (string, bool, error) {
	s, b, e := c.GetBytes(key)
	return string(s), b, e
}

// GetStruct retrieves data from the cache and stores it in the provided object.
// This function returns whether the cache exists and any error encountered.
func (c *Cache) GetStruct(key string, obj interface{}) (bool, error) {
	bytes, ok, err := c.GetBytes(key)
	if !ok || err != nil || bytes == nil {
		return ok, err
	}
	if err := json.Unmarshal(bytes, obj); err != nil {
		return ok, fmt.Errorf("failed to unmarshal cache data: %v", err)
	}
	return ok, nil
}

// PutStruct json-deserializes the provided struct and stores the data in the cache.
func (c *Cache) PutStruct(key string, i interface{}) error {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal struct to json: %v", err)
	}
	return c.Put(key, string(b))
}

func (c *Cache) getCacheDir() (string, error) {
	if c.Dir == "" {
		return "", fmt.Errorf("cache directory cannot be empty")
	}

	cacheDir, err := os.Stat(c.Dir)
	if os.IsNotExist(err) {
		if err := osMkdirAll(c.Dir, 0644); err != nil {
			return "", fmt.Errorf("cache directory does not exist and could not be created: %v", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to get info for cache: %v", err)
	} else if !cacheDir.Mode().IsDir() {
		return "", fmt.Errorf("cache directory must point to a directory, not a file")
	}
	return c.Dir, nil
}

func (c *Cache) fileFromKey(key string) (string, error) {
	r, err := regexp.Compile(keyRegex)
	if err != nil {
		return "", fmt.Errorf("invalid key regex: %v", err)
	}
	if !r.MatchString(key) {
		return "", fmt.Errorf("invalid key format")
	}

	cacheDir, err := c.getCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache directory: %v", err)
	}
	return filepath.Join(cacheDir, key), nil
}
