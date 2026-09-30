package matcher

import (
	"regexp"
	"strings"
	"sync"
	"unicode"

	"medya-filter/pkg/models"
)

var (
	artistSeparators = regexp.MustCompile(`(?i)\s*(?:,|&|feat\.?|ft\.?|vs\.?|with|\/|\bx\b|\bve\b|\+)\s*`)
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
	persons           []models.Person
	indexedPersons    []IndexedPerson
	AllowedCategories []string
	AllowedTopics     []string
}

func NewMatcher(persons []models.Person) *Matcher {
	m := &Matcher{}
	m.SetPersons(persons)
	return m
}

func (m *Matcher) SetPersons(persons []models.Person) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.persons = persons
	m.indexedPersons = make([]IndexedPerson, 0, len(persons))

	for _, p := range persons {
		names := p.ParsedNames()
		var candidates []NormalizedCandidate
		for _, name := range names {
			clean := strings.TrimSpace(name)
			if clean == "" {
				continue
			}

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

	// Split artists if multiple (e.g. "Ceza, Sezen Aksu" -> ["ceza", "sezen aksu"])
	artistPartsRaw := artistSeparators.Split(track.Artist, -1)
	var artistPartsFolded []string
	for _, part := range artistPartsRaw {
		pClean := CleanAndFold(part)
		if pClean != "" {
			artistPartsFolded = append(artistPartsFolded, pClean)
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
		}
	}

	return nil
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
