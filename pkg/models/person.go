package models

import (
	"regexp"
	"strings"
)

var aliasRegex = regexp.MustCompile(`^(.*?)\s*\((.*?)\)\s*$`)

// Person represents a person/entity in the oelist API.
type Person struct {
	ID           string                 `json:"id"`
	DisplayName  string                 `json:"display_name"`
	Notes        *string                `json:"notes"`
	Tags         []string               `json:"tags"`
	Topics       []string               `json:"topics"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	Summary      map[string]interface{} `json:"summary"`
	AccountCount int                    `json:"account_count"`
}

// ParsedNames extracts the primary display name and any aliases in parentheses.
// e.g., "Ezhel (Ömer Sercan İpekçioğlu)" -> ["Ezhel", "Ömer Sercan İpekçioğlu", "Ezhel (Ömer Sercan İpekçioğlu)"]
func (p Person) ParsedNames() []string {
	var names []string
	cleanDisplay := strings.TrimSpace(p.DisplayName)
	if cleanDisplay == "" {
		return names
	}

	matches := aliasRegex.FindStringSubmatch(cleanDisplay)
	if len(matches) == 3 {
		primary := strings.TrimSpace(matches[1])
		alias := strings.TrimSpace(matches[2])
		if primary != "" {
			names = append(names, primary)
		}
		if alias != "" {
			names = append(names, alias)
		}
	}

	names = append(names, cleanDisplay)
	return names
}

// MediaInfo represents currently playing audio or video track.
type MediaInfo struct {
	App          string `json:"app"`
	Artist       string `json:"artist"`
	Title        string `json:"title"`
	Album        string `json:"album"`
	State        string `json:"state"` // "playing", "paused", "stopped"
	URL          string `json:"url,omitempty"`
	Contributors string `json:"contributors,omitempty"`
}

// MatchResult represents a matched person against currently playing media.
type MatchResult struct {
	Matched     bool      `json:"matched"`
	Person      Person    `json:"person"`
	MatchedName string    `json:"matched_name"`
	Category    string    `json:"category"`
	Topics      []string  `json:"topics"`
	Track       MediaInfo `json:"track"`
}

// HistoryItem represents an event of track detection and user action.
type HistoryItem struct {
	ID        string      `json:"id"`
	Timestamp string      `json:"timestamp"`
	Track     MediaInfo   `json:"track"`
	Person    Person      `json:"person"`
	Action    string      `json:"action"` // "skipped", "kept", "auto_skipped", "pending"
}
