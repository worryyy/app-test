package config

import (
	"path/filepath"
	"testing"
)

func TestLoadKeepsJWAPIKeyWhenProfileOverridesBaseURL(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "configs", "ecampus")

	t.Setenv("APP_PROFILE", "prod")
	cfg := Load(configDir)

	if cfg.JW.BaseURL != "https://www.fangfangfang.top/sztu_jw" {
		t.Fatalf("unexpected jw base url: %q", cfg.JW.BaseURL)
	}
	if cfg.JW.APIKey != "U2FsdGVkX18BNFq4BRJwIzXUPmKQ2Ngj" {
		t.Fatalf("jw api key should be merged from base config, got %q", cfg.JW.APIKey)
	}
}

func TestLoadDefaultProfileKeepsJWConfigFromBase(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "configs", "ecampus")

	t.Setenv("APP_PROFILE", "")
	cfg := Load(configDir)

	if cfg.JW.BaseURL != "https://www.fangfangfang.top/sztu_jw" {
		t.Fatalf("unexpected jw base url: %q", cfg.JW.BaseURL)
	}
	if cfg.JW.APIKey != "U2FsdGVkX18BNFq4BRJwIzXUPmKQ2Ngj" {
		t.Fatalf("unexpected jw api key: %q", cfg.JW.APIKey)
	}
}
