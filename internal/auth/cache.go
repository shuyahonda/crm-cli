package auth

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

// fileCache is a file-backed MSAL token cache.
// It persists tokens across CLI invocations so that device_code authentication
// only needs to happen once (until the refresh token expires, typically 90 days).
type fileCache struct {
	mu   sync.Mutex
	path string
}

// newFileCache returns a fileCache writing to the platform-appropriate cache dir.
func newFileCache() *fileCache {
	return &fileCache{path: cacheFilePath()}
}

// cacheFilePath returns the OS-appropriate path for the token cache file.
func cacheFilePath() string {
	var dir string

	switch runtime.GOOS {
	case "windows":
		// %LOCALAPPDATA%\crm-cli\token_cache.json
		dir = os.Getenv("LOCALAPPDATA")
		if dir == "" {
			dir = os.Getenv("APPDATA")
		}
	default:
		// ~/.cache/crm-cli/token_cache.json
		dir = os.Getenv("XDG_CACHE_HOME")
		if dir == "" {
			if home, err := os.UserHomeDir(); err == nil {
				dir = filepath.Join(home, ".cache")
			}
		}
	}

	if dir == "" {
		dir = os.TempDir()
	}

	return filepath.Join(dir, "crm-cli", "token_cache.json")
}

// Export writes the current MSAL token cache to disk.
func (c *fileCache) Export(ctx context.Context, m cache.Marshaler, hints cache.ExportHints) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := m.Marshal()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0700); err != nil {
		return err
	}

	// Write atomically via temp file
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

// Replace loads the token cache from disk into MSAL.
func (c *fileCache) Replace(ctx context.Context, u cache.Unmarshaler, hints cache.ReplaceHints) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no cache yet — not an error
		}
		return err
	}

	return u.Unmarshal(data)
}
