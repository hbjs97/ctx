package cache_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hbjs97/ctx/internal/cache"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCache_ValidJSON(t *testing.T) {
	content := `{
		"version": 1,
		"entries": {
			"company-org/api-server": {
				"profile": "work",
				"reason": "owner_rule",
				"resolved_at": "2026-02-14T10:30:00Z",
				"config_hash": "a1b2c3d4"
			}
		}
	}`
	path := testutil.TempCacheFile(t, content)
	c, err := cache.Load(path)

	require.NoError(t, err)
	assert.Equal(t, 1, c.Version)
	assert.Len(t, c.Entries, 1)
	assert.Equal(t, "work", c.Entries["company-org/api-server"].Profile)
}

func TestLoadCache_MissingFile(t *testing.T) {
	c, err := cache.Load("/nonexistent/cache.json")
	require.NoError(t, err) // graceful: empty cache
	assert.Empty(t, c.Entries)
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	path := testutil.TempCacheFile(t, "not json {{{")
	c, err := cache.Load(path)
	require.NoError(t, err) // graceful degradation
	assert.Empty(t, c.Entries)
}

func TestLookup_Hit(t *testing.T) {
	c := &cache.Cache{
		Entries: map[string]cache.Entry{
			"org/repo": {
				Profile:    "work",
				Reason:     "owner_rule",
				ResolvedAt: time.Now().Format(time.RFC3339),
				ConfigHash: "abc123",
			},
		},
	}
	entry, ok := c.Lookup("org/repo", "abc123", 90)
	assert.True(t, ok)
	assert.Equal(t, "work", entry.Profile)
}

func TestLookup_TTLExpired(t *testing.T) {
	c := &cache.Cache{
		Entries: map[string]cache.Entry{
			"org/repo": {
				Profile:    "work",
				ResolvedAt: time.Now().Add(-91 * 24 * time.Hour).Format(time.RFC3339),
				ConfigHash: "abc123",
			},
		},
	}
	_, ok := c.Lookup("org/repo", "abc123", 90)
	assert.False(t, ok)
}

func TestLookup_HashMismatch(t *testing.T) {
	c := &cache.Cache{
		Entries: map[string]cache.Entry{
			"org/repo": {
				Profile:    "work",
				ResolvedAt: time.Now().Format(time.RFC3339),
				ConfigHash: "old_hash",
			},
		},
	}
	_, ok := c.Lookup("org/repo", "new_hash", 90)
	assert.False(t, ok)
}

func TestSave_NewEntry(t *testing.T) {
	c := cache.New()
	c.Set("org/repo", cache.Entry{
		Profile:    "work",
		Reason:     "probe",
		ResolvedAt: time.Now().Format(time.RFC3339),
		ConfigHash: "abc",
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	err := c.Save(path)
	require.NoError(t, err)

	info, _ := os.Stat(path)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	loaded, err := cache.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "work", loaded.Entries["org/repo"].Profile)
}

func TestSave_UpdateEntry(t *testing.T) {
	c := cache.New()
	c.Set("org/repo", cache.Entry{Profile: "work", Reason: "owner_rule"})
	c.Set("org/repo", cache.Entry{Profile: "personal", Reason: "user_select"})
	assert.Equal(t, "personal", c.Entries["org/repo"].Profile)
}

func TestInvalidate_ByProfile(t *testing.T) {
	c := cache.New()
	c.Set("org/repo1", cache.Entry{Profile: "work"})
	c.Set("org/repo2", cache.Entry{Profile: "work"})
	c.Set("user/repo3", cache.Entry{Profile: "personal"})

	c.InvalidateByProfile("work")

	assert.Len(t, c.Entries, 1)
	assert.Contains(t, c.Entries, "user/repo3")
}
