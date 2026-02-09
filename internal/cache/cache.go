package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
)

type Cache struct {
	dir     string
	enabled bool
	ttl     int
}

type CacheEntry struct {
	Hash       string          `json:"hash"`
	AnalyzedAt time.Time       `json:"analyzed_at"`
	Result     *ai.ReceiptInfo `json:"result"`
}

func New(cfg *config.CacheConfig) (*Cache, error) {
	dir := filepath.Join(config.DefaultCachePath(), "analysis")

	if cfg.Enabled {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	return &Cache{
		dir:     dir,
		enabled: cfg.Enabled,
		ttl:     cfg.TTL,
	}, nil
}

func (c *Cache) Get(pdfPath string) (*ai.ReceiptInfo, bool) {
	if !c.enabled {
		return nil, false
	}

	hash, err := c.hashFile(pdfPath)
	if err != nil {
		return nil, false
	}

	cachePath := filepath.Join(c.dir, hash+".json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	if c.ttl > 0 {
		expiry := entry.AnalyzedAt.AddDate(0, 0, c.ttl)
		if time.Now().After(expiry) {
			os.Remove(cachePath)
			return nil, false
		}
	}

	return entry.Result, true
}

func (c *Cache) Set(pdfPath string, info *ai.ReceiptInfo) error {
	if !c.enabled {
		return nil
	}

	hash, err := c.hashFile(pdfPath)
	if err != nil {
		return err
	}

	entry := CacheEntry{
		Hash:       hash,
		AnalyzedAt: time.Now(),
		Result:     info,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	cachePath := filepath.Join(c.dir, hash+".json")
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			path := filepath.Join(c.dir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove cache file: %w", err)
			}
		}
	}

	return nil
}

func (c *Cache) Count() (int, error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}

	return count, nil
}

func (c *Cache) hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file for hashing: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
