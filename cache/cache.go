package cache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

const (
	keyRegex = `^([a-zA-Z0-9_\.-]+)$`
)

var (
	// EnvCacheVar is the environment variable pointing to the path for caching.
	// var so it can be modified for tests
	EnvCacheVar = "LEEP_CACHE"
)

type Cache struct{}

func (c *Cache) Put(key, data string) error {
	return c.writeFile(key, data)
}

func (c *Cache) Get(key string) (string, error) {
	return c.readFile(key)
}

func (c *Cache) PutStruct(key string, i interface{}) error {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal struct to json: %v", err)
	}
	return c.Put(key, string(b))
}

func (c *Cache) writeFile(key, data string) error {
	filename, err := c.fileFromKey(key)
	if err != nil {
		return fmt.Errorf("failed to get file for key: %v", err)
	}
	if err := ioutil.WriteFile(filename, []byte(data), 0666); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

func (c *Cache) readFile(key string) (string, error) {
	filename, err := c.fileFromKey(key)
	if err != nil {
		return "", fmt.Errorf("failed to get file for key: %v", err)
	}

	// Check if the file exists.
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", nil
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	return string(data), nil
}

func (c *Cache) getCacheDir() (string, error) {
	cacheDirStr := os.Getenv(EnvCacheVar)
	if cacheDirStr == "" {
		return "", fmt.Errorf("environment variable %q is not set", EnvCacheVar)
	}

	cacheDir, err := os.Stat(cacheDirStr)
	if err != nil {
		return "", fmt.Errorf("invalid cache path: %v", err)
	}
	if !cacheDir.Mode().IsDir() {
		return "", fmt.Errorf("%q must point to a directory", EnvCacheVar)
	}
	return cacheDirStr, nil
}

func (c *Cache) fileFromKey(key string) (string, error) {
	r, err := regexp.Compile(keyRegex)
	if err != nil {
		return "", fmt.Errorf("invalid key regex: %v", err)
	}
	if !r.MatchString(key) {
		return "", fmt.Errorf("invalid key format: %v", err)
	}

	cacheDir, err := c.getCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache direction: %v", err)
	}
	return filepath.Join(cacheDir, key), nil
}
