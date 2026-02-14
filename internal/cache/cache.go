package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache는 리포-프로필 매핑 캐시다.
type Cache struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
}

// Entry는 하나의 캐시 항목이다.
type Entry struct {
	Profile    string `json:"profile"`
	Reason     string `json:"reason"`
	ResolvedAt string `json:"resolved_at"`
	ConfigHash string `json:"config_hash"`
}

// New는 빈 캐시를 생성한다.
func New() *Cache {
	return &Cache{Version: 1, Entries: make(map[string]Entry)}
}

// Load는 캐시 파일을 파싱한다. 파일 없음/파싱 실패 시 빈 캐시 반환 (graceful).
func Load(path string) (*Cache, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache.Load: %w", err)
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return New(), nil
	}
	if c.Entries == nil {
		c.Entries = make(map[string]Entry)
	}
	return &c, nil
}

// Lookup은 키로 캐시를 조회한다. TTL과 config_hash가 유효해야 hit.
func (c *Cache) Lookup(key, configHash string, ttlDays int) (*Entry, bool) {
	e, ok := c.Entries[key]
	if !ok {
		return nil, false
	}
	if e.ConfigHash != configHash {
		return nil, false
	}
	resolved, err := time.Parse(time.RFC3339, e.ResolvedAt)
	if err != nil {
		return nil, false
	}
	if time.Since(resolved) > time.Duration(ttlDays)*24*time.Hour {
		return nil, false
	}
	return &e, true
}

// Set은 캐시 항목을 추가하거나 갱신한다.
func (c *Cache) Set(key string, entry Entry) {
	c.Entries[key] = entry
}

// Save는 캐시를 JSON 파일로 저장한다 (0600 권한).
func (c *Cache) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cache.Save: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("cache.Save: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// InvalidateByProfile은 특정 프로필의 모든 캐시 항목을 제거한다.
func (c *Cache) InvalidateByProfile(profile string) {
	for key, entry := range c.Entries {
		if entry.Profile == profile {
			delete(c.Entries, key)
		}
	}
}
