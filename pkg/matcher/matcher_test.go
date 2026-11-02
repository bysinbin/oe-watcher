package matcher_test

import (
	"strings"
	"testing"

	"medya-filter/pkg/matcher"
	"medya-filter/pkg/models"
	"medya-filter/pkg/rules"
)

func samplePersons() []models.Person {
	return []models.Person{
		{
			ID:          "1",
			DisplayName: "Ceza (Bilgin Özçalkan)",
			Tags:        []string{"Şarkıcı"},
			Topics:      []string{"Ahmet Kaya Paylaşımı"},
		},
		{
			ID:          "2",
			DisplayName: "Cem Adrian",
			Tags:        []string{"Şarkıcı"},
			Topics:      []string{"Ahmet Kaya Paylaşımı", "Sırrı Süreyya Önder Paylaşımı"},
		},
		{
			ID:          "3",
			DisplayName: "Abdurrahman Dilipak",
			Tags:        []string{"Gazeteci"},
			Topics:      []string{"Akil İnsanlar Heyeti Üyesi"},
		},
		{
			ID:          "4",
			DisplayName: "Blok3 (Hakan Aydın)",
			Tags:        []string{"Şarkıcı"},
			Topics:      []string{"Sırrı Süreyya Önder Paylaşımı"},
		},
	}
}

func TestMatcherDirectMatch(t *testing.T) {
	m := matcher.NewMatcher(samplePersons())

	track := models.MediaInfo{
		App:    "Spotify",
		Artist: "Ceza",
		Title:  "Holokost",
	}

	result := m.Match(track)
	if result == nil {
		t.Fatalf("Expected match for Ceza, got nil")
	}
	if result.Person.DisplayName != "Ceza (Bilgin Özçalkan)" {
		t.Errorf("Expected Ceza (Bilgin Özçalkan), got %s", result.Person.DisplayName)
	}
}

func TestMatcherCollaborativeTrack(t *testing.T) {
	m := matcher.NewMatcher(samplePersons())

	// Collaborative track on Spotify
	track := models.MediaInfo{
		App:    "Spotify",
		Artist: "Killa Hakan, Ceza, Ezhel, Ben Fero",
		Title:  "Fight Kulüp",
	}

	result := m.Match(track)
	if result == nil {
		t.Fatalf("Expected match for collaborative track containing Ceza, got nil")
	}
	if result.Person.DisplayName != "Ceza (Bilgin Özçalkan)" {
		t.Errorf("Expected Ceza, got %s", result.Person.DisplayName)
	}
}

func TestMatcherDiacriticAndCasing(t *testing.T) {
	m := matcher.NewMatcher(samplePersons())

	// Uppercase and diacritic variations
	track := models.MediaInfo{
		App:    "Apple Music",
		Artist: "BLOK3",
		Title:  "Vur",
	}

	result := m.Match(track)
	if result == nil {
		t.Fatalf("Expected match for BLOK3, got nil")
	}
	if result.Person.DisplayName != "Blok3 (Hakan Aydın)" {
		t.Errorf("Expected Blok3, got %s", result.Person.DisplayName)
	}
}

func TestMatcherCategoryFilter(t *testing.T) {
	m := matcher.NewMatcher(samplePersons())
	m.SetFilters([]string{"Gazeteci"}, nil)

	// Ceza is a Şarkıcı, should not match when filter is Gazeteci
	track := models.MediaInfo{
		App:    "Spotify",
		Artist: "Ceza",
		Title:  "Neyim Var Ki",
	}

	result := m.Match(track)
	if result != nil {
		t.Fatalf("Expected no match when Gazeteci filter is set, got %v", result)
	}

	// Abdurrahman Dilipak is a Gazeteci, should match
	track2 := models.MediaInfo{
		App:    "Google Chrome",
		Artist: "Abdurrahman Dilipak",
		Title:  "Gündem Değerlendirmesi",
	}
	result2 := m.Match(track2)
	if result2 == nil {
		t.Fatalf("Expected match for Abdurrahman Dilipak with Gazeteci filter, got nil")
	}
}

func TestMatcherCustomPersonAndAlias(t *testing.T) {
	tempDir := t.TempDir()
	rm := rules.NewManager(tempDir)

	// Add custom person: Ersin Koç
	_, err := rm.AddPerson("Ersin Koç", []string{"Şarkıcı"}, []string{"Özel"}, []string{"Ersin Koç"})
	if err != nil {
		t.Fatalf("Failed to add custom person: %v", err)
	}

	// Add custom alias: "Üstad" -> "Cem Adrian"
	err = rm.SetAlias("Üstad", "Cem Adrian")
	if err != nil {
		t.Fatalf("Failed to add alias: %v", err)
	}

	m := matcher.NewMatcher(samplePersons())
	m.SetRulesManager(rm)

	// 1. Test custom person match
	trackErsin := models.MediaInfo{
		App:    "YouTube Music",
		Artist: "Ersin Koç",
		Title:  "Şarkı",
	}
	res1 := m.Match(trackErsin)
	if res1 == nil {
		t.Fatalf("Expected match for custom person Ersin Koç, got nil")
	}
	if !strings.Contains(res1.Person.DisplayName, "Ersin Koç") {
		t.Errorf("Expected Ersin Koç, got %s", res1.Person.DisplayName)
	}

	// 2. Test alias match
	trackAlias := models.MediaInfo{
		App:    "Spotify",
		Artist: "Üstad",
		Title:  "Kül",
	}
	res2 := m.Match(trackAlias)
	if res2 == nil {
		t.Fatalf("Expected match for alias Üstad -> Cem Adrian, got nil")
	}
	if res2.Person.DisplayName != "Cem Adrian" {
		t.Errorf("Expected Cem Adrian, got %s", res2.Person.DisplayName)
	}

	// 3. Test alias targeting parsed primary name: "Rapstar" -> "Ceza" (target is "Ceza (Bilgin Özçalkan)")
	err = rm.SetAlias("Rapstar", "Ceza")
	if err != nil {
		t.Fatalf("Failed to add alias Rapstar -> Ceza: %v", err)
	}
	m.RebuildIndex()

	trackRapstar := models.MediaInfo{
		App:    "Spotify",
		Artist: "Rapstar",
		Title:  "Neyim Var Ki",
	}
	res3 := m.Match(trackRapstar)
	if res3 == nil {
		t.Fatalf("Expected match for alias Rapstar -> Ceza, got nil")
	}
	if res3.Person.DisplayName != "Ceza (Bilgin Özçalkan)" {
		t.Errorf("Expected Ceza (Bilgin Özçalkan), got %s", res3.Person.DisplayName)
	}
}

func TestMatcherExcludeList(t *testing.T) {
	tempDir := t.TempDir()
	rm := rules.NewManager(tempDir)

	// Exclude Cem Adrian from matching
	_ = rm.AddExclude("Cem Adrian")

	m := matcher.NewMatcher(samplePersons())
	m.SetRulesManager(rm)

	track := models.MediaInfo{
		App:    "Spotify",
		Artist: "Cem Adrian",
		Title:  "Kül",
	}

	// Even though Cem Adrian is in samplePersons, it should be ignored because it is in ExcludeList
	result := m.Match(track)
	if result != nil {
		t.Fatalf("Expected nil for excluded artist Cem Adrian, got match: %v", result)
	}
}

func TestMatcherURLExclude(t *testing.T) {
	tempDir := t.TempDir()
	rm := rules.NewManager(tempDir)

	// Exclude a specific URL / repo
	_ = rm.AddExclude("https://github.com/bysinbin/oe-watcher")

	m := matcher.NewMatcher(samplePersons())
	m.SetRulesManager(rm)

	// A track matching a watched person (Cem Adrian) but played from the excluded URL
	track := models.MediaInfo{
		App:    "Chrome",
		Artist: "Cem Adrian",
		Title:  "Kül",
		URL:    "https://github.com/bysinbin/oe-watcher/blob/main/README.md",
	}

	result := m.Match(track)
	if result != nil {
		t.Fatalf("Expected nil for track played from excluded URL, got match: %v", result)
	}
}

func TestMatcherFuzzyMatch(t *testing.T) {
	m := matcher.NewMatcher(samplePersons())
	m.SetFuzzy(true, 0.85)

	// "Cem Adriann" has 1 typo letter compared to "Cem Adrian" (len 10 vs 11, dist 1, sim ~91%)
	track := models.MediaInfo{
		App:    "Spotify",
		Artist: "Cem Adriann",
		Title:  "Kül",
	}

	result := m.Match(track)
	if result == nil {
		t.Fatalf("Expected fuzzy match for 'Cem Adriann', got nil")
	}
	if !strings.Contains(result.MatchedName, "Bulanık") {
		t.Errorf("Expected matched name to indicate fuzzy match, got: %s", result.MatchedName)
	}
}

func TestMatcherContributors(t *testing.T) {
	m := matcher.NewMatcher(samplePersons())

	// An unknown singer is performing, but the lyricist / beat maker in description is Ceza
	track := models.MediaInfo{
		App:          "Google Chrome",
		Artist:       "Bilinmeyen Sanatçı",
		Title:        "Yeni Şarkı 2026",
		Contributors: "Söz: Ceza | Müzik: Anonymous | Beat: Studio",
	}

	result := m.Match(track)
	if result == nil {
		t.Fatalf("Expected match for contributor 'Ceza' in description credits, got nil")
	}
	if !strings.Contains(result.MatchedName, "Katkıda Bulunan") {
		t.Errorf("Expected matched name to indicate contributor, got: %s", result.MatchedName)
	}
	if result.Person.DisplayName != "Ceza (Bilgin Özçalkan)" {
		t.Errorf("Expected Ceza (Bilgin Özçalkan), got %s", result.Person.DisplayName)
	}
}



func TestMatcherTurkishSuffixes(t *testing.T) {
	m := matcher.NewMatcher(samplePersons())

	// Test "Cezadan yeni klip" without apostrophe
	track := models.MediaInfo{
		App:    "YouTube",
		Artist: "Netd Müzik",
		Title:  "Cezadan Özel Açıklama",
	}

	result := m.Match(track)
	if result == nil {
		t.Fatalf("Expected match for 'Cezadan', got nil")
	}
	if result.Person.DisplayName != "Ceza (Bilgin Özçalkan)" {
		t.Errorf("Expected Ceza, got %s", result.Person.DisplayName)
	}

	// Test "Cem Adriannın" (suffix "nın" folded to "nin")
	track2 := models.MediaInfo{
		App:    "YouTube",
		Artist: "Cem Adriannin En Iyi Sarkilari",
		Title:  "Konser Kaydı",
	}
	result2 := m.Match(track2)
	if result2 == nil {
		t.Fatalf("Expected match for 'Cem Adriannin', got nil")
	}
}
