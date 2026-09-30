package history

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"medya-filter/pkg/models"
)

const (
	HistoryFileName = "history.json"
	MaxHistoryItems = 1000
)

// Stats tracks aggregate history statistics.
type Stats struct {
	TotalDetections  int            `json:"total_detections"`
	TotalAutoSkips   int            `json:"total_auto_skips"`
	TotalManualSkips int            `json:"total_manual_skips"`
	TotalKept        int            `json:"total_kept"`
	TotalSnoozed     int            `json:"total_snoozed"`
	TopPersons       map[string]int `json:"top_persons"`
	TopApps          map[string]int `json:"top_apps"`
	FirstRecorded    string         `json:"first_recorded,omitempty"`
	LastRecorded     string         `json:"last_recorded,omitempty"`
}

// Storage represents the disk format of history.json.
type Storage struct {
	Version int                  `json:"version"`
	Stats   Stats                `json:"stats"`
	Items   []models.HistoryItem `json:"items"`
}

// Manager manages persistent history logs and reports on disk.
type Manager struct {
	mu       sync.RWMutex
	filePath string
	dataDir  string
	stats    Stats
	items    []models.HistoryItem
}

// NewManager creates or loads a History Manager from dataDir.
func NewManager(dataDir string) *Manager {
	if dataDir == "" {
		dataDir = "data"
	}
	_ = os.MkdirAll(dataDir, 0755)

	m := &Manager{
		filePath: filepath.Join(dataDir, HistoryFileName),
		dataDir:  dataDir,
		items:    make([]models.HistoryItem, 0),
		stats: Stats{
			TopPersons: make(map[string]int),
			TopApps:    make(map[string]int),
		},
	}

	if err := m.Load(); err != nil {
		// If file doesn't exist yet, start fresh
		_ = m.Save()
	}

	return m
}

// Load reads history from history.json.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	var storage Storage
	if err := json.Unmarshal(data, &storage); err != nil {
		return err
	}

	m.items = storage.Items
	if m.items == nil {
		m.items = make([]models.HistoryItem, 0)
	}

	m.stats = storage.Stats
	if m.stats.TopPersons == nil {
		m.stats.TopPersons = make(map[string]int)
	}
	if m.stats.TopApps == nil {
		m.stats.TopApps = make(map[string]int)
	}

	return nil
}

// Save writes current history and stats to disk.
func (m *Manager) Save() error {
	m.mu.RLock()
	storage := Storage{
		Version: 1,
		Stats:   m.stats,
		Items:   m.items,
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(m.filePath, data, 0644)
}

// Record appends an item to the history log and updates aggregate statistics.
func (m *Manager) Record(item models.HistoryItem) {
	m.mu.Lock()

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	if item.Timestamp == "" {
		item.Timestamp = nowStr
	}
	if item.ID == "" {
		item.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Update statistics
	m.stats.TotalDetections++
	if m.stats.FirstRecorded == "" {
		m.stats.FirstRecorded = nowStr
	}
	m.stats.LastRecorded = nowStr

	if item.Person.DisplayName != "" {
		if m.stats.TopPersons == nil {
			m.stats.TopPersons = make(map[string]int)
		}
		m.stats.TopPersons[item.Person.DisplayName]++
	}

	if item.Track.App != "" {
		if m.stats.TopApps == nil {
			m.stats.TopApps = make(map[string]int)
		}
		m.stats.TopApps[item.Track.App]++
	}

	switch item.Action {
	case "auto_skipped":
		m.stats.TotalAutoSkips++
		if item.ActionLabel == "" {
			item.ActionLabel = "Otomatik Geçildi"
		}
	case "manual_skip", "skipped":
		m.stats.TotalManualSkips++
		if item.ActionLabel == "" {
			item.ActionLabel = "Manuel Geçildi"
		}
	case "kept":
		m.stats.TotalKept++
		if item.ActionLabel == "" {
			item.ActionLabel = "Dinlendi"
		}
	case "snoozed":
		m.stats.TotalSnoozed++
		if item.ActionLabel == "" {
			item.ActionLabel = "1 Saat Ertelendi"
		}
	case "skip_failed":
		if item.ActionLabel == "" {
			item.ActionLabel = "Geçiş Başarısız"
		}
	default:
		if item.ActionLabel == "" {
			item.ActionLabel = item.Action
		}
	}

	// Prepend to list (newest first)
	m.items = append([]models.HistoryItem{item}, m.items...)
	if len(m.items) > MaxHistoryItems {
		m.items = m.items[:MaxHistoryItems]
	}

	m.mu.Unlock()

	// Persist to disk asynchronously
	go func() {
		_ = m.Save()
	}()
}

// GetItems returns paginated and optionally filtered history items.
func (m *Manager) GetItems(limit, offset int, action, search string) ([]models.HistoryItem, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	filtered := make([]models.HistoryItem, 0, len(m.items))
	searchLower := strings.ToLower(strings.TrimSpace(search))
	actionLower := strings.ToLower(strings.TrimSpace(action))

	for _, item := range m.items {
		// Filter by action
		if actionLower != "" && actionLower != "all" && actionLower != "tümü" {
			if !strings.EqualFold(item.Action, actionLower) {
				continue
			}
		}

		// Filter by search query
		if searchLower != "" {
			text := strings.ToLower(fmt.Sprintf("%s %s %s %s %s",
				item.Person.DisplayName,
				item.Track.Artist,
				item.Track.Title,
				item.Track.App,
				item.Category,
			))
			if !strings.Contains(text, searchLower) {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	total := len(filtered)
	if offset >= total {
		return []models.HistoryItem{}, total
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	return filtered[offset:end], total
}

// GetAllItems returns a copy of all items in memory.
func (m *Manager) GetAllItems() []models.HistoryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.HistoryItem, len(m.items))
	copy(res, m.items)
	return res
}

// GetStats returns the current statistics.
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Deep copy maps
	topP := make(map[string]int, len(m.stats.TopPersons))
	for k, v := range m.stats.TopPersons {
		topP[k] = v
	}
	topA := make(map[string]int, len(m.stats.TopApps))
	for k, v := range m.stats.TopApps {
		topA[k] = v
	}

	s := m.stats
	s.TopPersons = topP
	s.TopApps = topA
	return s
}

// Clear removes all history items and resets stats on disk.
func (m *Manager) Clear() error {
	m.mu.Lock()
	m.items = make([]models.HistoryItem, 0)
	m.stats = Stats{
		TopPersons: make(map[string]int),
		TopApps:    make(map[string]int),
	}
	m.mu.Unlock()
	return m.Save()
}

// ExportCSV generates a full CSV export with UTF-8 BOM for Excel/Numbers compatibility.
func (m *Manager) ExportCSV() ([]byte, error) {
	m.mu.RLock()
	items := make([]models.HistoryItem, len(m.items))
	copy(items, m.items)
	m.mu.RUnlock()

	var buf bytes.Buffer
	// UTF-8 BOM for Microsoft Excel / macOS Numbers Turkish character encoding support
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buf)

	// Header row
	header := []string{
		"Kayıt ID",
		"Tarih & Saat",
		"Uygulama",
		"Çalan Şarkı",
		"Sanatçı",
		"Eşleşen Kişi",
		"Kategori",
		"Konular / Etiketler",
		"Uygulanan İşlem",
		"Şarkı URL",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for _, item := range items {
		actionDesc := item.ActionLabel
		if actionDesc == "" {
			actionDesc = item.Action
		}

		topics := strings.Join(item.Topics, "; ")
		if topics == "" && len(item.Person.Topics) > 0 {
			topics = strings.Join(item.Person.Topics, "; ")
		}

		cat := item.Category
		if cat == "" && len(item.Person.Tags) > 0 {
			cat = strings.Join(item.Person.Tags, ", ")
		}

		row := []string{
			item.ID,
			item.Timestamp,
			item.Track.App,
			item.Track.Title,
			item.Track.Artist,
			item.Person.DisplayName,
			cat,
			topics,
			actionDesc,
			item.Track.URL,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ExportJSON generates formatted JSON containing stats and all items.
func (m *Manager) ExportJSON() ([]byte, error) {
	m.mu.RLock()
	storage := Storage{
		Version: 1,
		Stats:   m.stats,
		Items:   m.items,
	}
	m.mu.RUnlock()

	return json.MarshalIndent(storage, "", "  ")
}
