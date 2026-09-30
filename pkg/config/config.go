package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// LoadOrCreate loads config from config.json (checks root or data directory).
// If not found, creates a default config.json in root.
func LoadOrCreate(customPath string) (*Config, string, error) {
	candidates := []string{}
	targetPath := ConfigFileName

	if customPath != "" {
		candidates = append(candidates, customPath)
		targetPath = customPath
	} else {
		candidates = append(candidates,
			ConfigFileName,
			filepath.Join("data", ConfigFileName),
		)
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
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
				absPath, err := filepath.Abs(path)
				if err == nil {
					return cfg, absPath, nil
				}
				return cfg, path, nil
			}
		}
	}

	// Create new config file at targetPath
	cfg := DefaultConfig()
	absTarget, err := filepath.Abs(targetPath)
	if err == nil {
		targetPath = absTarget
	}
	if err := Save(targetPath, cfg); err != nil {
		return cfg, targetPath, fmt.Errorf("config oluşturulamadı: %w", err)
	}

	return cfg, targetPath, nil
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
