package cache_test

import (
	"testing"
)

func TestLoadCache_ValidJSON(t *testing.T) {
	t.Skip("not implemented")

	// Given: a valid cache.json with entries
	// When: LoadCache is called
	// Then: returns Cache with correct entries
}

func TestLoadCache_MissingFile(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache.json does not exist
	// When: LoadCache is called
	// Then: returns empty cache (not an error)
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache.json with invalid JSON
	// When: LoadCache is called
	// Then: returns empty cache with warning (graceful degradation)
}

func TestLookup_Hit(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache has "company-org/api-server" -> "work", TTL valid, hash matches
	// When: Lookup("company-org/api-server", currentHash) is called
	// Then: returns "work" entry
}

func TestLookup_TTLExpired(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache entry with resolved_at older than 90 days
	// When: Lookup is called
	// Then: returns nil (cache miss)
}

func TestLookup_HashMismatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache entry with config_hash "old" but current hash is "new"
	// When: Lookup is called
	// Then: returns nil (cache invalidated)
}

func TestSave_NewEntry(t *testing.T) {
	t.Skip("not implemented")

	// Given: empty cache
	// When: Save("owner/repo", "work", "owner_rule", hash) is called
	// Then: cache.json is written with the new entry and file permissions 0600
}

func TestSave_UpdateEntry(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache with existing entry for "owner/repo"
	// When: Save is called with different profile
	// Then: entry is updated with new profile, reason, and timestamp
}

func TestInvalidate_ByProfile(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache with multiple entries referencing "work" profile
	// When: InvalidateByProfile("work") is called
	// Then: all entries with profile "work" are removed
}
