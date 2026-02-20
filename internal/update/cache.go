package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

// defaultCachePath returns ~/.cache/ssherpa/update.json (XDG cache dir).
func defaultCachePath() string {
	return filepath.Join(xdg.CacheHome, "ssherpa", "update.json")
}

// readCache loads the cache file. Returns zero-value CacheData if missing.
func readCache(path string) (CacheData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return CacheData{}, nil
		}
		return CacheData{}, err
	}

	var cache CacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		// Corrupt cache — treat as empty
		return CacheData{}, nil
	}
	return cache, nil
}

// writeCache atomically writes cache data to disk, creating directories as needed.
func writeCache(path string, data CacheData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o644)
}

// cacheExpired returns true if the cache's last check is older than checkInterval.
func cacheExpired(cache CacheData) bool {
	if cache.LastCheck.IsZero() {
		return true
	}
	return time.Since(cache.LastCheck) > checkInterval
}
