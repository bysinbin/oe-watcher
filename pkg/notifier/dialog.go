package notifier

import (
	"fmt"
	"strings"

	"medya-filter/pkg/models"
)

// Action constants
const (
	ActionSkip    = "skip"
	ActionSnooze  = "snooze"
	ActionKeep    = "keep"
	ActionTimeout = "timeout"
)

// PromptSkip displays a native system dialog asking the user whether to skip the track.
// Returns true if the user clicks "Şarkıyı Geç", false if they choose to keep or if it times out.
func PromptSkip(match *models.MatchResult) (bool, error) {
	act, err := PromptAction(match)
	if err != nil {
		return false, err
	}
	return act == ActionSkip, nil
}

func formatDialogBody(match *models.MatchResult) string {
	topicsStr := strings.Join(match.Topics, ", ")
	if topicsStr == "" {
		topicsStr = "Belirtilmemiş"
	}

	trackDesc := ""
	if match.Track.Artist != "" && !strings.Contains(match.Track.Title, match.Track.Artist) {
		trackDesc = fmt.Sprintf("%s - %s", match.Track.Artist, match.Track.Title)
	} else {
		trackDesc = match.Track.Title
	}
	if match.Track.App != "" {
		trackDesc += fmt.Sprintf(" (%s)", match.Track.App)
	}

	return fmt.Sprintf(
		"⚠️ Kara Listedeki Sanatçı Tespit Edildi!\n\n"+
			"• İsim Soyisim : %s\n"+
			"• Kategori     : %s\n"+
			"• Konu         : %s\n"+
			"• Çalan Medya  : %s\n\n"+
			"Lütfen bir işlem seçin:",
		match.Person.DisplayName,
		match.Category,
		topicsStr,
		trackDesc,
	)
}
