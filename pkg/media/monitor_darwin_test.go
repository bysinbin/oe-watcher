//go:build darwin

package media

import (
	"fmt"
	"testing"
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
