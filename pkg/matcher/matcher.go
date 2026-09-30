package matcher

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"medya-filter/pkg/models"
	"medya-filter/pkg/rules"
)

var (
	artistSeparators = regexp.MustCompile(`(?i)\s*(?:,|&|feat\.?|ft\.?|vs\.?|with|\/|\bx\b|\bve\b|\+|\bprod\.?\s*(?:by)?\b|\bbeat\s*(?:by)?\b|\bcredits?\b|\bcomposer\b|\blyricist\b)\s*`)
	nonAlphaNumeric  = regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	multipleSpaces   = regexp.MustCompile(`\s+`)
)

type NormalizedCandidate struct {
	Original       string
	TurkishLower   string
	DiacriticFold  string
	TokensFolded   []string
}

type IndexedPerson struct {
	Person     models.Person
	Candidates []NormalizedCandidate
}

type Matcher struct {
	mu                sync.RWMutex
	apiPersons        []models.Person
	persons           []models.Person
	indexedPersons    []IndexedPerson
	AllowedCategories []string
	AllowedTopics     []string
	rulesMgr          *rules.Manager
	fuzzyEnabled      bool
	fuzzyThreshold    float64
}

func NewMatcher(persons []models.Person) *Matcher {
	m := &Matcher{
		fuzzyEnabled:   true,
		fuzzyThreshold: 0.85,
	}
	m.SetPersons(persons)
	return m
}

func (m *Matcher) SetFuzzy(enabled bool, threshold float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fuzzyEnabled = enabled
	if threshold > 0 && threshold <= 1.0 {
		m.fuzzyThreshold = threshold
	}
}

func (m *Matcher) SetRulesManager(rm *rules.Manager) {
	m.mu.Lock()
	m.rulesMgr = rm
	m.mu.Unlock()
	m.RebuildIndex()
}

func (m *Matcher) SetPersons(persons []models.Person) {
	m.mu.Lock()
	m.apiPersons = persons
	m.mu.Unlock()
	m.RebuildIndex()
}

// RebuildIndex updates indexed persons with API persons, custom persons, and aliases.
func (m *Matcher) RebuildIndex() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var allPersons []models.Person
	allPersons = append(allPersons, m.apiPersons...)

	var customAliases map[string]string
	if m.rulesMgr != nil {
		allPersons = append(allPersons, m.rulesMgr.GetCustomPersonsAsModels()...)
		snap := m.rulesMgr.GetSnapshot()
		customAliases = snap.Aliases
	}

	m.persons = allPersons
	m.indexedPersons = make([]IndexedPerson, 0, len(allPersons))

	for _, p := range allPersons {
		names := p.ParsedNames()

		// Add custom aliases mapped to this person
		if customAliases != nil {
			for alias, target := range customAliases {
				targetMatch := strings.EqualFold(target, p.DisplayName)
				if !targetMatch {
					for _, pn := range names {
						if strings.EqualFold(target, pn) {
							targetMatch = true
							break
						}
					}
				}
				if targetMatch {
					names = append(names, alias)
				}
			}
		}

		var candidates []NormalizedCandidate
		seen := make(map[string]bool)

		for _, name := range names {
			clean := strings.TrimSpace(name)
			if clean == "" || seen[strings.ToLower(clean)] {
				continue
			}
			seen[strings.ToLower(clean)] = true

			turkLower := strings.ToLowerSpecial(unicode.TurkishCase, clean)
			folded := FoldDiacritics(turkLower)
			foldedClean := nonAlphaNumeric.ReplaceAllString(folded, " ")
			tokens := strings.Fields(foldedClean)

			candidates = append(candidates, NormalizedCandidate{
				Original:      clean,
				TurkishLower:  turkLower,
				DiacriticFold: strings.TrimSpace(multipleSpaces.ReplaceAllString(foldedClean, " ")),
				TokensFolded:  tokens,
			})
		}

		m.indexedPersons = append(m.indexedPersons, IndexedPerson{
			Person:     p,
			Candidates: candidates,
		})
	}
}

func (m *Matcher) SetFilters(categories, topics []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AllowedCategories = categories
	m.AllowedTopics = topics
}

func (m *Matcher) GetPersons() []models.Person {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.persons
}

// FoldDiacritics converts Turkish and accented letters to simple lowercase ASCII.
func FoldDiacritics(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'ç', 'Ç':
			b.WriteRune('c')
		case 'ğ', 'Ğ':
			b.WriteRune('g')
		case 'ı', 'I':
			b.WriteRune('i')
		case 'i', 'İ':
			b.WriteRune('i')
		case 'ö', 'Ö':
			b.WriteRune('o')
		case 'ş', 'Ş':
			b.WriteRune('s')
		case 'ü', 'Ü':
			b.WriteRune('u')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// CleanAndFold normalizes text for matching.
func CleanAndFold(s string) string {
	low := strings.ToLowerSpecial(unicode.TurkishCase, s)
	folded := FoldDiacritics(low)
	cleaned := nonAlphaNumeric.ReplaceAllString(folded, " ")
	return strings.TrimSpace(multipleSpaces.ReplaceAllString(cleaned, " "))
}

// Match checks whether the given media track matches any indexed person.
func (m *Matcher) Match(track models.MediaInfo) *models.MatchResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if strings.TrimSpace(track.Artist) == "" && strings.TrimSpace(track.Title) == "" {
		return nil
	}

	artistFolded := CleanAndFold(track.Artist)
	titleFolded := CleanAndFold(track.Title)
	contribFolded := CleanAndFold(track.Contributors)

	// Split artists if multiple (e.g. "Ceza, Sezen Aksu" -> ["ceza", "sezen aksu"])
	artistPartsRaw := artistSeparators.Split(track.Artist, -1)
	var artistPartsFolded []string
	for _, part := range artistPartsRaw {
		pClean := CleanAndFold(part)
		if pClean != "" {
			artistPartsFolded = append(artistPartsFolded, pClean)
		}
	}

	// Whitelist / Exclude check
	if m.rulesMgr != nil {
		if track.URL != "" && m.rulesMgr.IsExcluded(track.URL) {
			return nil
		}
		if m.rulesMgr.IsExcluded(track.Artist) || m.rulesMgr.IsExcluded(track.Title) {
			return nil
		}
		for _, part := range artistPartsRaw {
			if m.rulesMgr.IsExcluded(part) {
				return nil
			}
		}

		// Custom Keywords / Hashtags check
		fullMediaText := fmt.Sprintf("%s %s %s", track.Artist, track.Title, track.Contributors)
		if matchedKw, ok := m.rulesMgr.MatchesKeyword(fullMediaText); ok {
			kwPerson := models.Person{
				DisplayName: matchedKw,
				Tags:        []string{"Özel Kelime"},
				Topics:      []string{"Özel Engellenen Kelime"},
			}
			return m.buildResult(kwPerson, matchedKw, track)
		}
	}

	for _, item := range m.indexedPersons {
		// Category filter check
		if !m.isCategoryAllowed(item.Person.Tags) {
			continue
		}

		// Topic filter check
		if !m.isTopicAllowed(item.Person.Topics) {
			continue
		}

		// Check each candidate name
		for _, cand := range item.Candidates {
			candFolded := cand.DiacriticFold
			if len(candFolded) < 2 {
				continue
			}

			// 1. Direct equality with track artist or one of the split artist parts
			if artistFolded != "" && artistFolded == candFolded {
				return m.buildResult(item.Person, cand.Original, track)
			}
			for _, part := range artistPartsFolded {
				if part == candFolded {
					return m.buildResult(item.Person, cand.Original, track)
				}
			}

			// 2. Check if artist contains candidate name as a word boundary
			if artistFolded != "" && containsWord(artistFolded, candFolded) {
				return m.buildResult(item.Person, cand.Original, track)
			}

			// 3. For video/stream titles (like YouTube), check if title contains the candidate name
			if titleFolded != "" && containsWord(titleFolded, candFolded) {
				return m.buildResult(item.Person, cand.Original, track)
			}

			// 4. For contributors / credits (YouTube description, Söz, Müzik, Beat, Prod)
			if contribFolded != "" && containsWord(contribFolded, candFolded) {
				return m.buildResult(item.Person, cand.Original+" (Katkıda Bulunan: Söz/Müzik/Prod)", track)
			}
		}
	}

	// 4. Fuzzy Matching Pass (if enabled and no exact match found)
	if m.fuzzyEnabled && m.fuzzyThreshold > 0.5 {
		titleWords := strings.Fields(titleFolded)

		for _, item := range m.indexedPersons {
			if !m.isCategoryAllowed(item.Person.Tags) || !m.isTopicAllowed(item.Person.Topics) {
				continue
			}

			for _, cand := range item.Candidates {
				candFolded := cand.DiacriticFold
				if len(candFolded) < 4 {
					continue
				}

				// Check similarity against artist parts
				for _, part := range artistPartsFolded {
					if len(part) >= 4 {
						sim := Similarity(part, candFolded)
						if sim >= m.fuzzyThreshold {
							label := fmt.Sprintf("%s (Bulanık: %%%.0f)", cand.Original, sim*100)
							return m.buildResult(item.Person, label, track)
						}
					}
				}

				// Check similarity against artist full string
				if len(artistFolded) >= 4 {
					sim := Similarity(artistFolded, candFolded)
					if sim >= m.fuzzyThreshold {
						label := fmt.Sprintf("%s (Bulanık: %%%.0f)", cand.Original, sim*100)
						return m.buildResult(item.Person, label, track)
					}
				}

				// Check similarity against words or word-windows in title
				candTokenCount := len(cand.TokensFolded)
				if candTokenCount > 0 && len(titleWords) >= candTokenCount {
					for i := 0; i <= len(titleWords)-candTokenCount; i++ {
						window := strings.Join(titleWords[i:i+candTokenCount], " ")
						if len(window) >= 4 {
							sim := Similarity(window, candFolded)
							if sim >= m.fuzzyThreshold {
								label := fmt.Sprintf("%s (Bulanık: %%%.0f)", cand.Original, sim*100)
								return m.buildResult(item.Person, label, track)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// Levenshtein calculates minimum edit distance between two rune slices.
func Levenshtein(a, b []rune) int {
	la := len(a)
	lb := len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	dp := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		dp[j] = j
	}

	for i := 1; i <= la; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= lb; j++ {
			temp := dp[j]
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			val := dp[j] + 1
			if dp[j-1]+1 < val {
				val = dp[j-1] + 1
			}
			if prev+cost < val {
				val = prev + cost
			}
			dp[j] = val
			prev = temp
		}
	}
	return dp[lb]
}

// Similarity returns a score between 0.0 and 1.0.
func Similarity(s1, s2 string) float64 {
	r1 := []rune(s1)
	r2 := []rune(s2)
	maxLen := len(r1)
	if len(r2) > maxLen {
		maxLen = len(r2)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := Levenshtein(r1, r2)
	return 1.0 - float64(dist)/float64(maxLen)
}

func containsWord(haystack, needle string) bool {
	if haystack == needle {
		return true
	}
	// Check as substring with boundary spaces
	paddedHaystack := " " + haystack + " "
	paddedNeedle := " " + needle + " "
	if strings.Contains(paddedHaystack, paddedNeedle) {
		return true
	}
	return false
}

func (m *Matcher) isCategoryAllowed(tags []string) bool {
	if len(m.AllowedCategories) == 0 {
		return true
	}
	for _, allowed := range m.AllowedCategories {
		normAllowed := strings.ToUpper(strings.TrimSpace(allowed))
		if normAllowed == "HEPSİ" || normAllowed == "TÜMÜ" || normAllowed == "ALL" || normAllowed == "" {
			return true
		}
		for _, tag := range tags {
			if strings.EqualFold(strings.TrimSpace(tag), strings.TrimSpace(allowed)) {
				return true
			}
		}
	}
	return false
}

func (m *Matcher) isTopicAllowed(topics []string) bool {
	if len(m.AllowedTopics) == 0 {
		return true
	}
	for _, allowed := range m.AllowedTopics {
		normAllowed := strings.ToUpper(strings.TrimSpace(allowed))
		if normAllowed == "TÜM KONULAR" || normAllowed == "HEPSİ" || normAllowed == "ALL" || normAllowed == "" {
			return true
		}
		for _, topic := range topics {
			if strings.EqualFold(strings.TrimSpace(topic), strings.TrimSpace(allowed)) {
				return true
			}
		}
	}
	return false
}

func (m *Matcher) buildResult(person models.Person, matchedName string, track models.MediaInfo) *models.MatchResult {
	catStr := strings.Join(person.Tags, ", ")
	if catStr == "" {
		catStr = "Belirtilmemiş"
	}

	return &models.MatchResult{
		Matched:     true,
		Person:      person,
		MatchedName: matchedName,
		Category:    catStr,
		Topics:      person.Topics,
		Track:       track,
	}
}
