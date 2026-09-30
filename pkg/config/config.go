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

// GetUserConfigDir returns the standard OS user configuration directory:
// macOS:   ~/Library/Application Support/OEWatcher
// Windows: %APPDATA%\OEWatcher (C:\Users\<user>\AppData\Roaming\OEWatcher)
// Linux:   ~/.config/OEWatcher (XDG standard)
func GetUserConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "OEWatcher")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".oewatcher")
	}
	return "data"
}

// LocateConfigAndData reliably discovers the appropriate config.json path and data directory.
// For packaged/bundled applications, user data and config are isolated in the OS user config directory
// so that permissions and settings persist seamlessly across updates.
func LocateConfigAndData(customPath string) (string, string) {
	if customPath != "" {
		abs, err := filepath.Abs(customPath)
		if err == nil {
			return abs, filepath.Join(filepath.Dir(abs), "data")
		}
		return customPath, filepath.Join(filepath.Dir(customPath), "data")
	}

	exe, _ := os.Executable()
	appSupportDir := GetUserConfigDir()

	// If running inside a macOS .app bundle, user data MUST be isolated in Application Support.
	// This ensures macOS never prompts for Desktop/Documents folder access, and settings persist
	// even when the .app is moved to /Applications.
	if exe != "" && strings.Contains(exe, ".app/Contents/MacOS") {
		targetConfig := filepath.Join(appSupportDir, ConfigFileName)
		targetDataDir := filepath.Join(appSupportDir, "data")
		_ = os.MkdirAll(targetDataDir, 0755)

		exeDir := filepath.Dir(exe)
		bundleDir := filepath.Dir(filepath.Dir(exeDir))
		resourcesDir := filepath.Join(bundleDir, "Contents", "Resources")

		// Seed config.json if it doesn't exist in Application Support
		if _, err := os.Stat(targetConfig); os.IsNotExist(err) {
			seedConfigCandidates := []string{
				filepath.Join(resourcesDir, ConfigFileName),
				filepath.Join(filepath.Dir(bundleDir), ConfigFileName),
				filepath.Join(filepath.Dir(filepath.Dir(bundleDir)), ConfigFileName),
			}
			for _, sc := range seedConfigCandidates {
				if data, err := os.ReadFile(sc); err == nil {
					_ = os.WriteFile(targetConfig, data, 0644)
					break
				}
			}
		}

		// Seed data files (custom_rules.json, persons.json, etc.) if not existing in Application Support
		seedDataDirs := []string{
			filepath.Join(resourcesDir, "data"),
			filepath.Join(filepath.Dir(bundleDir), "data"),
			filepath.Join(filepath.Dir(filepath.Dir(bundleDir)), "data"),
		}
		for _, sdd := range seedDataDirs {
			if entries, err := os.ReadDir(sdd); err == nil {
				for _, entry := range entries {
					if !entry.IsDir() {
						destFile := filepath.Join(targetDataDir, entry.Name())
						if _, err := os.Stat(destFile); os.IsNotExist(err) {
							if content, err := os.ReadFile(filepath.Join(sdd, entry.Name())); err == nil {
								_ = os.WriteFile(destFile, content, 0644)
							}
						}
					}
				}
				break
			}
		}

		return targetConfig, targetDataDir
	}

	// CLI / development mode: check current working directory first
	candidates := []string{}
	cwd, err := os.Getwd()
	if err == nil && cwd != "" && cwd != "/" {
		candidates = append(candidates,
			filepath.Join(cwd, ConfigFileName),
			filepath.Join(cwd, "data", ConfigFileName),
		)
	}

	// Binary location
	if exe != "" {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, ConfigFileName),
			filepath.Join(exeDir, "data", ConfigFileName),
		)
	}

	// User Application Support / AppData / .config fallback
	candidates = append(candidates, filepath.Join(appSupportDir, ConfigFileName))

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

	// Fallback to User Config Dir
	fallbackPath := filepath.Join(appSupportDir, ConfigFileName)
	return fallbackPath, filepath.Join(appSupportDir, "data")
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
