package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const cacheDir = ".cache"

type Cache struct {
	Data any `json:"data"`
}

func EnsureCacheDir() error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("error creating cache directory: %v", err)
	}
	return nil
}

func GetCachePath(subreddit string) string {
	return filepath.Join(cacheDir, fmt.Sprintf("%s.json", subreddit))
}

func WriteToCache(subreddit string, data interface{}) error {
	if err := EnsureCacheDir(); err != nil {
		return err
	}

	cache := Cache{
		Data: data,
	}

	file, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cache data: %v", err)
	}

	if err := os.WriteFile(GetCachePath(subreddit), file, 0644); err != nil {
		return fmt.Errorf("error writing cache file: %v", err)
	}

	return nil
}

func ReadFromCache(subreddit string) (*Cache, error) {
	file, err := os.ReadFile(GetCachePath(subreddit))
	if err != nil {
		return nil, fmt.Errorf("error reading cache file: %v", err)
	}

	var cache Cache
	if err := json.Unmarshal(file, &cache); err != nil {
		return nil, fmt.Errorf("error unmarshaling cache data: %v", err)
	}

	return &cache, nil
}

func CacheExists(subreddit string) bool {
	_, err := os.Stat(GetCachePath(subreddit))
	return err == nil
}
