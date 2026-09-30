package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"medya-filter/pkg/config"
)

func TestLoadOrCreate(t *testing.T) {
	tempDir := t.TempDir()
	customPath := filepath.Join(tempDir, "custom_config.json")

	// 1. Should create default config if not found
	cfg, path, err := config.LoadOrCreate(customPath)
	if err != nil {
		t.Fatalf("LoadOrCreate failed: %v", err)
	}
	if path != customPath {
		t.Errorf("Expected path %s, got %s", customPath, path)
	}
	if cfg.WebPassword != "admin" {
		t.Errorf("Expected default password 'admin', got %s", cfg.WebPassword)
	}
	if cfg.BindAddress != "127.0.0.1" {
		t.Errorf("Expected bind address '127.0.0.1', got %s", cfg.BindAddress)
	}
	if cfg.ApiURL != config.DefaultApiURL {
		t.Errorf("Expected default API URL, got %s", cfg.ApiURL)
	}

	// 2. Modify and save
	cfg.WebPassword = "supersecretpassword123"
	cfg.WebPort = 9090
	cfg.BindAddress = "0.0.0.0"
	err = config.Save(customPath, cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// 3. Reload and verify
	reloaded, _, err := config.LoadOrCreate(customPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if reloaded.WebPassword != "supersecretpassword123" {
		t.Errorf("Expected supersecretpassword123, got %s", reloaded.WebPassword)
	}
	if reloaded.WebPort != 9090 {
		t.Errorf("Expected port 9090, got %d", reloaded.WebPort)
	}
	if reloaded.BindAddress != "0.0.0.0" {
		t.Errorf("Expected 0.0.0.0, got %s", reloaded.BindAddress)
	}

	// Clean up
	_ = os.Remove(customPath)
}
