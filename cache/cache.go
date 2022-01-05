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
	keyRegex = `^([a-zA-Z0-9_-]+)$`
)

type Cache struct {
	dir string
}

func (c *Cache) Name() string {
	return "cash"
}

func (c *Cache) Changed() bool     { return false }
func (c *Cache) Setup() []string   { return nil }
func (c *Cache) Load(string) error { return nil }
func (c *Cache) Node() *command.Node {
	arg := command.StringNode("KEY", "Key of the data to get", command.MatchesRegex(keyRegex), &command.Completor{SuggestionFetcher: &fetcher{c}})
	return command.BranchNode(map[string]*command.Node{
		"get": command.SerialNodes(
			arg,
			command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
				s, ok, err := c.Get(d.String(arg.Name()))
				if err != nil {
					return o.Err(err)
				}
				if !ok {
					o.Stderr("key not found")
				} else {
					o.Stdoutln(s)
				}
				return nil
			}),
		),
		// TODO: allow aliases for keys (via separator? "put|p")
		"put": command.SerialNodes(
			arg,
			command.StringListNode("DATA", "Data to store", 1, command.UnboundedList),
			command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
				return o.Err(c.Put(d.String(arg.Name()), strings.Join(d.StringList("DATA"), " ")))
			}),
		),
		"delete": command.SerialNodes(
			arg,
			command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
				return o.Err(c.Delete(d.String(arg.Name())))
			}),
		),
		"list": command.SerialNodes(
			command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
				r, err := c.List()
				if err != nil {
					return o.Err(err)
				}
				sort.Strings(r)
				for _, s := range r {
					o.Stdoutln(s)
				}
				return nil
			}),
		),
	}, nil, true)
}

type fetcher struct {
	c *Cache
}

func (f *fetcher) Fetch(value *command.Value, data *command.Data) (*command.Completion, error) {
	r, err := f.c.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}
	return &command.Completion{
		Suggestions: r,
	}, nil
}

func NewTestCache(t *testing.T) *Cache {
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
	return &Cache{dir}
}

func New(dir string) *Cache {
	return &Cache{dir}
}

func (c *Cache) Put(key, data string) error {
	filename, err := c.fileFromKey(key)
	if err != nil {
		return fmt.Errorf("failed to get file for key: %v", err)
	}
	if err := ioutil.WriteFile(filename, []byte(data), 0666); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

func (c *Cache) List() ([]string, error) {
	fs, err := os.ReadDir(c.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read files in directory: %v", err)
	}
	var r []string
	for _, f := range fs {
		r = append(r, f.Name())
	}
	return r, nil
}

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

// Returns data, whether the file exists, and any error encountered.
func (c *Cache) Get(key string) (string, bool, error) {
	filename, err := c.fileFromKey(key)
	if err != nil {
		return "", false, fmt.Errorf("failed to get file for key: %v", err)
	}

	// Check if the file exists.
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to read file: %v", err)
	}
	return string(data), true, nil
}

func (c *Cache) PutStruct(key string, i interface{}) error {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal struct to json: %v", err)
	}
	return c.Put(key, string(b))
}

func (c *Cache) getCacheDir() (string, error) {
	if c.dir == "" {
		return "", fmt.Errorf("cache directory cannot be empty")
	}

	cacheDir, err := os.Stat(c.dir)
	if err != nil {
		return "", fmt.Errorf("invalid cache path: %v", err)
	}
	if !cacheDir.Mode().IsDir() {
		return "", fmt.Errorf("cache directory must point to a directory")
	}
	return c.dir, nil
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
