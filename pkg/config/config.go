package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigFileName = "config.json"
	DefaultApiURL  = "https://oelist-api.frax.tr/v1"
)

type Config struct {
	ApiURL         string  `json:"api_url"`
	WebPort        int     `json:"web_port"`
	BindAddress    string  `json:"bind_address"`
	WebPassword    string  `json:"web_password"`
	AutoSkip       bool    `json:"auto_skip"`
	EnableDialog   bool    `json:"enable_dialog"`
	PollIntervalMs int     `json:"poll_interval_ms"`
	Category       string  `json:"category"`
	FuzzyMatching  bool    `json:"fuzzy_matching"`
	FuzzyThreshold float64 `json:"fuzzy_threshold"`
	AutoSyncHours  int     `json:"auto_sync_hours"`
}

func DefaultConfig() *Config {
	return &Config{
		ApiURL:         DefaultApiURL,
		WebPort:        8080,
		BindAddress:    "127.0.0.1", // Sadece yerel makineden erişim (Ağdaki başkaları IP yazsa bile bağlanamaz)
		WebPassword:    "admin",     // Web panelini korumak için parola
		AutoSkip:       false,
		EnableDialog:   true,
		PollIntervalMs: 1500,
		Category:       "HEPSİ",
		FuzzyMatching:  true,
		FuzzyThreshold: 0.85,
		AutoSyncHours:  12,
	}
}

// LocateConfigAndData reliably discovers the appropriate config.json path and data directory,
// even when launched from macOS Finder / Cocoa .app bundle where working directory is "/".
func LocateConfigAndData(customPath string) (string, string) {
	if customPath != "" {
		abs, err := filepath.Abs(customPath)
		if err == nil {
			return abs, filepath.Join(filepath.Dir(abs), "data")
		}
		return customPath, filepath.Join(filepath.Dir(customPath), "data")
	}

	candidates := []string{}

	// 1. Current working directory (if not root "/")
	cwd, err := os.Getwd()
	if err == nil && cwd != "" && cwd != "/" {
		candidates = append(candidates,
			filepath.Join(cwd, ConfigFileName),
			filepath.Join(cwd, "data", ConfigFileName),
		)
	}

	// 2. Binary location / macOS App Bundle location
	exe, err := os.Executable()
	if err == nil && exe != "" {
		exeDir := filepath.Dir(exe)
		if strings.Contains(exe, ".app/Contents/MacOS") {
			bundleDir := filepath.Dir(filepath.Dir(exeDir)) // e.g. .../OE Watcher.app
			parentDir := filepath.Dir(bundleDir)            // e.g. .../dist
			grandParent := filepath.Dir(parentDir)         // e.g. .../medya-filter (workspace root)
			candidates = append(candidates,
				filepath.Join(grandParent, ConfigFileName),
				filepath.Join(parentDir, ConfigFileName),
				filepath.Join(bundleDir, "Contents", "Resources", ConfigFileName),
			)
		} else {
			candidates = append(candidates,
				filepath.Join(exeDir, ConfigFileName),
				filepath.Join(exeDir, "data", ConfigFileName),
			)
		}
	}

	// 3. User standard Application Support directory
	home, _ := os.UserHomeDir()
	if home != "" {
		appSupportDir := filepath.Join(home, "Library", "Application Support", "OEWatcher")
		candidates = append(candidates, filepath.Join(appSupportDir, ConfigFileName))
	}

	// Check if any candidate already exists on disk
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs, filepath.Join(filepath.Dir(abs), "data")
			}
			return c, filepath.Join(filepath.Dir(c), "data")
		}
	}

	// If none exist yet, pick the first valid non-root candidate
	for _, c := range candidates {
		dir := filepath.Dir(c)
		if dir != "/" && dir != "" {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs, filepath.Join(filepath.Dir(abs), "data")
			}
			return c, filepath.Join(filepath.Dir(c), "data")
		}
	}

	// Fallback to Application Support
	fallbackDir := filepath.Join(home, "Library", "Application Support", "OEWatcher")
	fallbackPath := filepath.Join(fallbackDir, ConfigFileName)
	return fallbackPath, filepath.Join(fallbackDir, "data")
}

// LoadOrCreate loads config from config.json (checks root or data directory).
// If not found, creates a default config.json.
func LoadOrCreate(customPath string) (*Config, string, error) {
	configPath, _ := LocateConfigAndData(customPath)

	data, err := os.ReadFile(configPath)
	if err == nil {
		cfg := DefaultConfig()
		if err := json.Unmarshal(data, cfg); err == nil {
			// Ensure defaults for empty fields
			if cfg.ApiURL == "" {
				cfg.ApiURL = DefaultApiURL
			}
			if cfg.WebPort <= 0 {
				cfg.WebPort = 8080
			}
			if cfg.BindAddress == "" {
				cfg.BindAddress = "127.0.0.1"
			}
			if cfg.PollIntervalMs <= 0 {
				cfg.PollIntervalMs = 1500
			}
			if cfg.Category == "" {
				cfg.Category = "HEPSİ"
			}
			if cfg.FuzzyThreshold <= 0 || cfg.FuzzyThreshold > 1.0 {
				cfg.FuzzyThreshold = 0.85
			}
			return cfg, configPath, nil
		}
	}

	// Create new config file at configPath
	cfg := DefaultConfig()
	if err := Save(configPath, cfg); err != nil {
		return cfg, configPath, fmt.Errorf("config oluşturulamadı (%s): %w", configPath, err)
	}

	return cfg, configPath, nil
}

// Save writes config to target path with nice indentation.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
