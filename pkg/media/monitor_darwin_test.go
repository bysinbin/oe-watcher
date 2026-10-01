//go:build darwin

package media

import (
	"fmt"
	"strings"
	"testing"

	"medya-filter/pkg/models"
)

func TestActiveMedia(t *testing.T) {
	list, err := GetActiveMediaList()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fmt.Printf("\n=== MEDIA LIST COUNT: %d ===\n", len(list))
	for i, m := range list {
		fmt.Printf("[%d] App: %s | Artist: '%s' | Title: '%s' | URL: %s\n", i, m.App, m.Artist, m.Title, m.URL)
	}
}

func TestApplyOEmbed(t *testing.T) {
	tests := []struct {
		name         string
		emb          YTOEmbed
		initialTrack models.MediaInfo
		wantArtist   string
		wantTitle    string
		wantContrib  string
	}{
		{
			name: "Halil Sezai via record label uploader DokuzSekiz Muzik",
			emb: YTOEmbed{
				Title:      "Halil Sezai - Kafası Kendinden Bile Güzel (İncir Reçeli 2 / Video)",
				AuthorName: "DokuzSekiz Müzik",
			},
			initialTrack: models.MediaInfo{
				App:   "Google Chrome (Music)",
				Title: "Kafası Kendinden Bile Güzel",
			},
			wantArtist:  "Halil Sezai",
			wantTitle:   "Kafası Kendinden Bile Güzel",
			wantContrib: "DokuzSekiz Müzik",
		},
		{
			name: "Uploader is the artist Bayhan",
			emb: YTOEmbed{
				Title:      "Bayhan - Darıldım Darıldım",
				AuthorName: "Bayhan",
			},
			initialTrack: models.MediaInfo{
				App:   "Google Chrome (Music)",
				Title: "Darıldım",
			},
			wantArtist:  "Bayhan",
			wantTitle:   "Darıldım Darıldım",
			wantContrib: "Bayhan",
		},
		{
			name: "Featured track with netd muzik uploader",
			emb: YTOEmbed{
				Title:      "Sezen Aksu - Vazgeçtim (Official Video)",
				AuthorName: "netd müzik",
			},
			initialTrack: models.MediaInfo{
				App: "Google Chrome (YouTube)",
			},
			wantArtist:  "Sezen Aksu",
			wantTitle:   "Vazgeçtim",
			wantContrib: "netd müzik",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := tt.initialTrack
			applyOEmbed(&track, tt.emb)

			if track.Artist != tt.wantArtist {
				t.Errorf("Artist = %q, want %q", track.Artist, tt.wantArtist)
			}
			if !strings.Contains(track.Title, tt.wantTitle) {
				t.Errorf("Title = %q, want containing %q", track.Title, tt.wantTitle)
			}
			if !strings.Contains(track.Contributors, tt.wantContrib) {
				t.Errorf("Contributors = %q, want containing %q", track.Contributors, tt.wantContrib)
			}
		})
	}
}
