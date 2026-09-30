package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"medya-filter/pkg/models"
)

const RulesFileName = "custom_rules.json"

type CustomPerson struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Tags        []string `json:"tags"`
	Topics      []string `json:"topics"`
	Aliases     []string `json:"aliases"`
	CreatedAt   string   `json:"created_at"`
}

func (cp CustomPerson) ToPerson() models.Person {
	// Format with primary alias if present
	displayName := cp.DisplayName
	if len(cp.Aliases) > 0 {
		displayName = fmt.Sprintf("%s (%s)", cp.DisplayName, strings.Join(cp.Aliases, ", "))
	}

	return models.Person{
		ID:          cp.ID,
		DisplayName: displayName,
		Tags:        cp.Tags,
		Topics:      cp.Topics,
		CreatedAt:   cp.CreatedAt,
		UpdatedAt:   cp.CreatedAt,
		Summary:     map[string]interface{}{"custom": true},
	}
}

type Manager struct {
	mu          sync.RWMutex
	filePath    string
	Persons     []CustomPerson    `json:"custom_persons"`
	Aliases     map[string]string `json:"aliases"`      // alias (lowercase folded) -> Target Name
	ExcludeList []string          `json:"exclude_list"` // Names or artists to never match
	Keywords    []string          `json:"keywords"`     // Custom keywords or hashtags to block/shield
}

func NewManager(dataDir string) *Manager {
	if dataDir == "" {
		dataDir = "data"
	}
	_ = os.MkdirAll(dataDir, 0755)

	m := &Manager{
		filePath:    filepath.Join(dataDir, RulesFileName),
		Persons:     make([]CustomPerson, 0),
		Aliases:     make(map[string]string),
		ExcludeList: make([]string, 0),
		Keywords:    make([]string, 0),
	}

	_ = m.Load()
	return m
}

func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return m.saveLocked()
		}
		return err
	}

	type fileFormat struct {
		CustomPersons []CustomPerson    `json:"custom_persons"`
		Aliases       map[string]string `json:"aliases"`
		ExcludeList   []string          `json:"exclude_list"`
		Keywords      []string          `json:"keywords"`
	}

	var ff fileFormat
	if err := json.Unmarshal(data, &ff); err != nil {
		return err
	}

	m.Persons = ff.CustomPersons
	if m.Persons == nil {
		m.Persons = make([]CustomPerson, 0)
	}
	m.Aliases = ff.Aliases
	if m.Aliases == nil {
		m.Aliases = make(map[string]string)
	}
	m.ExcludeList = ff.ExcludeList
	if m.ExcludeList == nil {
		m.ExcludeList = make([]string, 0)
	}
	m.Keywords = ff.Keywords
	if m.Keywords == nil {
		m.Keywords = make([]string, 0)
	}

	return nil
}

func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	type fileFormat struct {
		CustomPersons []CustomPerson    `json:"custom_persons"`
		Aliases       map[string]string `json:"aliases"`
		ExcludeList   []string          `json:"exclude_list"`
		Keywords      []string          `json:"keywords"`
	}

	ff := fileFormat{
		CustomPersons: m.Persons,
		Aliases:       m.Aliases,
		ExcludeList:   m.ExcludeList,
		Keywords:      m.Keywords,
	}

	data, err := json.MarshalIndent(ff, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0644)
}

// AddPerson adds a custom person to the watchlist.
func (m *Manager) AddPerson(name string, tags, topics, aliases []string) (CustomPerson, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CustomPerson{}, fmt.Errorf("isim boş olamaz")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	for _, p := range m.Persons {
		if strings.EqualFold(p.DisplayName, name) {
			return CustomPerson{}, fmt.Errorf("bu isimde bir kişi zaten özel listede var")
		}
	}

	if len(tags) == 0 {
		tags = []string{"Diğer"}
	}
	if len(topics) == 0 {
		topics = []string{"Özel Tanımlı"}
	}

	cp := CustomPerson{
		ID:          fmt.Sprintf("custom-%d", time.Now().UnixNano()),
		DisplayName: name,
		Tags:        tags,
		Topics:      topics,
		Aliases:     aliases,
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
	}

	m.Persons = append(m.Persons, cp)
	_ = m.saveLocked()
	return cp, nil
}

// RemovePerson removes a custom person by ID or DisplayName.
func (m *Manager) RemovePerson(identifier string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.Persons {
		if p.ID == identifier || strings.EqualFold(p.DisplayName, identifier) {
			m.Persons = append(m.Persons[:i], m.Persons[i+1:]...)
			_ = m.saveLocked()
			return true
		}
	}
	return false
}

// SetAlias sets an alias mapping (e.g. "Süperstar" -> "Ajda Pekkan").
func (m *Manager) SetAlias(alias, targetPerson string) error {
	alias = strings.TrimSpace(alias)
	targetPerson = strings.TrimSpace(targetPerson)
	if alias == "" || targetPerson == "" {
		return fmt.Errorf("alias ve hedef kişi adı boş olamaz")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Aliases[alias] = targetPerson
	return m.saveLocked()
}

// RemoveAlias removes an alias mapping.
func (m *Manager) RemoveAlias(alias string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for k := range m.Aliases {
		if strings.EqualFold(k, alias) {
			delete(m.Aliases, k)
			_ = m.saveLocked()
			return true
		}
	}
	return false
}

// NormalizeURL normalizes a URL string for matching by stripping scheme (http/https), www., and trailing slashes.
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "www.")
	raw = strings.TrimRight(raw, "/")
	return raw
}

// IsURLPattern checks whether the string resembles a URL or web domain.
func IsURLPattern(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return true
	}
	if strings.Contains(s, "/") && strings.Contains(s, ".") {
		return true
	}
	return false
}

// AddExclude adds an artist, song name, or URL/domain to the whitelist.
func (m *Manager) AddExclude(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("değer boş olamaz")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ex := range m.ExcludeList {
		if strings.EqualFold(ex, name) {
			return nil // already exists
		}
	}

	m.ExcludeList = append(m.ExcludeList, name)
	return m.saveLocked()
}

// RemoveExclude removes an entry from the exclude list.
func (m *Manager) RemoveExclude(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, ex := range m.ExcludeList {
		if strings.EqualFold(ex, name) {
			m.ExcludeList = append(m.ExcludeList[:i], m.ExcludeList[i+1:]...)
			_ = m.saveLocked()
			return true
		}
	}
	return false
}

// IsExcluded checks whether a given name, artist, song title, or URL is in the exclude list.
func (m *Manager) IsExcluded(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(name))
	if clean == "" {
		return false
	}

	normName := NormalizeURL(clean)

	for _, ex := range m.ExcludeList {
		exClean := strings.ToLower(strings.TrimSpace(ex))
		if exClean == "" {
			continue
		}

		// Exact match or substring for regular text
		if clean == exClean || strings.Contains(clean, exClean) {
			return true
		}

		// URL and domain matching
		if IsURLPattern(exClean) || IsURLPattern(clean) {
			normEx := NormalizeURL(exClean)
			if normEx != "" {
				if normName == normEx {
					return true
				}
				// Subpath match (e.g. repo URL whitelists all files and issues under it)
				if strings.HasPrefix(normName, normEx+"/") || strings.HasPrefix(normName, normEx+"?") || strings.HasPrefix(normName, normEx+"#") {
					return true
				}
				// Domain or subdomain match (e.g. github.com whitelists everything on github.com)
				if !strings.Contains(normEx, "/") {
					host := normName
					if idx := strings.Index(host, "/"); idx != -1 {
						host = host[:idx]
					}
					if host == normEx || strings.HasSuffix(host, "."+normEx) {
						return true
					}
				}
			}
		}
	}
	return false
}

// AddKeyword adds a keyword or hashtag to the custom blocked words list.
func (m *Manager) AddKeyword(keyword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clean := strings.TrimSpace(keyword)
	if clean == "" {
		return fmt.Errorf("anahtar kelime boş olamaz")
	}

	for _, k := range m.Keywords {
		if strings.EqualFold(k, clean) {
			return nil // already exists
		}
	}

	m.Keywords = append(m.Keywords, clean)
	return m.saveLocked()
}

// RemoveKeyword removes a keyword from the custom list.
func (m *Manager) RemoveKeyword(keyword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clean := strings.TrimSpace(keyword)
	filtered := make([]string, 0, len(m.Keywords))
	for _, k := range m.Keywords {
		if !strings.EqualFold(k, clean) {
			filtered = append(filtered, k)
		}
	}
	m.Keywords = filtered
	return m.saveLocked()
}

// GetKeywords returns the list of custom blocked keywords.
func (m *Manager) GetKeywords() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]string, len(m.Keywords))
	copy(res, m.Keywords)
	return res
}

// MatchesKeyword checks if given text contains any custom blocked keyword.
func (m *Manager) MatchesKeyword(text string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lowText := strings.ToLower(text)
	for _, kw := range m.Keywords {
		cleanKw := strings.ToLower(strings.TrimSpace(kw))
		if cleanKw != "" && strings.Contains(lowText, cleanKw) {
			return kw, true
		}
	}
	return "", false
}

// GetSnapshot returns current custom rules snapshot.
type RulesSnapshot struct {
	CustomPersons []CustomPerson    `json:"custom_persons"`
	Aliases       map[string]string `json:"aliases"`
	ExcludeList   []string          `json:"exclude_list"`
	Keywords      []string          `json:"keywords"`
}

func (m *Manager) GetSnapshot() RulesSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return RulesSnapshot{
		CustomPersons: m.Persons,
		Aliases:       m.Aliases,
		ExcludeList:   m.ExcludeList,
		Keywords:      m.Keywords,
	}
}

// GetCustomPersonsAsModels converts custom persons into models.Person.
func (m *Manager) GetCustomPersonsAsModels() []models.Person {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]models.Person, len(m.Persons))
	for i, cp := range m.Persons {
		res[i] = cp.ToPerson()
	}
	return res
}

// Export returns formatted JSON bytes representing the custom rules.
func (m *Manager) Export() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := RulesSnapshot{
		CustomPersons: m.Persons,
		Aliases:       m.Aliases,
		ExcludeList:   m.ExcludeList,
		Keywords:      m.Keywords,
	}
	return json.MarshalIndent(snap, "", "  ")
}

// Import unmarshals JSON bytes and replaces the rules, then saves to file.
func (m *Manager) Import(data []byte) error {
	var snap RulesSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("geçersiz kurallar JSON formatı: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if snap.CustomPersons != nil {
		m.Persons = snap.CustomPersons
	}
	if snap.Aliases != nil {
		m.Aliases = snap.Aliases
	}
	if snap.ExcludeList != nil {
		m.ExcludeList = snap.ExcludeList
	}
	if snap.Keywords != nil {
		m.Keywords = snap.Keywords
	}

	return m.saveLocked()
}
