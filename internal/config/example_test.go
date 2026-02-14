package config_test

import (
	"testing"
)

func TestLoadConfig_ValidTOML(t *testing.T) {
	t.Skip("not implemented")

	// Given: a valid config.toml with two profiles (work, personal)
	// When: LoadConfig is called with the file path
	// Then: returns Config with correct profiles, default_profile, and global settings
}

func TestLoadConfig_MissingFile(t *testing.T) {
	t.Skip("not implemented")

	// Given: a non-existent config path
	// When: LoadConfig is called
	// Then: returns an appropriate error (exit code 5)
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	t.Skip("not implemented")

	// Given: a config.toml with invalid TOML syntax
	// When: LoadConfig is called
	// Then: returns a parse error
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	t.Skip("not implemented")

	// Given: a config.toml missing required fields (e.g., gh_config_dir)
	// When: LoadConfig is called
	// Then: returns a validation error listing missing fields
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	t.Skip("not implemented")

	// Given: a config.toml without optional fields
	// When: LoadConfig is called
	// Then: default values are applied (prompt_on_ambiguous=true, require_push_guard=true)
}

func TestValidateFilePermissions(t *testing.T) {
	t.Skip("not implemented")

	// Given: a config.toml with various file permissions
	// When: ValidateFilePermissions is called
	// Then: warns if permissions are more permissive than 0600
}

func TestConfigHash(t *testing.T) {
	t.Skip("not implemented")

	// Given: a config with profiles
	// When: ConfigHash is called
	// Then: returns a stable SHA-256 hash of the profiles section
	// And: the hash changes when profiles are modified
}

// --- Profile Store tests (config is the Profile Store per architecture) ---

func TestMatchOwner_SingleMatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: profiles with distinct owners
	//   work: ["company-org", "company-team"]
	//   personal: ["hbjs97", "sutefu23"]
	// When: MatchOwner("company-org") is called
	// Then: returns "work" profile only
}

func TestMatchOwner_NoMatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: profiles with known owners
	// When: MatchOwner("unknown-org") is called
	// Then: returns empty result (no match)
}

func TestMatchOwner_MultipleMatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: profiles where "shared-org" appears in both work and personal owners
	// When: MatchOwner("shared-org") is called
	// Then: returns both profiles (ambiguous)
}

func TestGetProfile_Exists(t *testing.T) {
	t.Skip("not implemented")

	// Given: a config with "work" and "personal" profiles
	// When: GetProfile("work") is called
	// Then: returns the work profile with all fields populated
}

func TestGetProfile_NotExists(t *testing.T) {
	t.Skip("not implemented")

	// Given: a config with "work" and "personal" profiles
	// When: GetProfile("nonexistent") is called
	// Then: returns an error (exit code 5)
}
