package rules

import (
	"testing"
)

func TestRulesManagerURLExclude(t *testing.T) {
	tempDir := t.TempDir()
	rm := NewManager(tempDir)

	// Add a URL
	err := rm.AddExclude("https://github.com/bysinbin/oe-watcher")
	if err != nil {
		t.Fatalf("AddExclude failed: %v", err)
	}

	// Add an artist
	_ = rm.AddExclude("Ahmet Aslan")

	// Add a domain
	_ = rm.AddExclude("youtube.com")

	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		{"exact URL match", "https://github.com/bysinbin/oe-watcher", true},
		{"URL without scheme", "github.com/bysinbin/oe-watcher", true},
		{"URL with trailing slash", "https://github.com/bysinbin/oe-watcher/", true},
		{"URL subpath file", "https://github.com/bysinbin/oe-watcher/blob/main/README.md", true},
		{"URL query param", "https://github.com/bysinbin/oe-watcher?tab=readme-ov-file", true},
		{"different repo on github", "https://github.com/other/repo", false},
		{"artist match", "Ahmet Aslan", true},
		{"artist in song title", "Ahmet Aslan - Tanımadığım Ten", true},
		{"domain match exact", "youtube.com", true},
		{"domain match with scheme", "https://www.youtube.com/watch?v=123", true},
		{"unrelated site", "https://google.com/search", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.IsExcluded(tt.target)
			if got != tt.expected {
				t.Errorf("IsExcluded(%q) = %v; want %v", tt.target, got, tt.expected)
			}
		})
	}
}
