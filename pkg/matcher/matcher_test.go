package matcher_test

import (
	"testing"

	"medya-filter/pkg/matcher"
	"medya-filter/pkg/models"
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
