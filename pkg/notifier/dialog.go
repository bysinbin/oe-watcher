package notifier

import (
	"fmt"
	"os/exec"
	"strings"

	"medya-filter/pkg/models"
)

// escapeAppleScript escapes quotes and backslashes for AppleScript strings.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// PlayAlertSound plays a subtle macOS alert sound.
func PlayAlertSound() {
	go func() {
		_ = exec.Command("afplay", "/System/Library/Sounds/Ping.aiff").Run()
	}()
}

// ShowNotification displays a macOS system notification banner.
func ShowNotification(title, subtitle, message string) {
	script := fmt.Sprintf(
		`display notification "%s" with title "%s" subtitle "%s"`,
		escapeAppleScript(message),
		escapeAppleScript(title),
		escapeAppleScript(subtitle),
	)
	_ = exec.Command("osascript", "-e", script).Run()
}

// PromptSkip displays a native macOS dialog asking the user whether to skip the track.
// Returns true if the user clicks "Şarkıyı Geç", false if they choose to keep or if it times out.
func PromptSkip(match *models.MatchResult) (bool, error) {
	PlayAlertSound()

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

	dialogBody := fmt.Sprintf(
		"⚠️ Kara Listedeki Sanatçı Tespit Edildi!\n\n"+
			"• İsim Soyisim : %s\n"+
			"• Kategori     : %s\n"+
			"• Konu         : %s\n"+
			"• Çalan Medya  : %s\n\n"+
			"Bu şarkıyı geçmek istiyor musunuz?",
		match.Person.DisplayName,
		match.Category,
		topicsStr,
		trackDesc,
	)

	script := fmt.Sprintf(
		`try
			set dlg to display dialog "%s" with title "Medya Filtresi" buttons {"Dinlemeye Devam Et", "Şarkıyı Geç"} default button "Şarkıyı Geç" with icon caution giving up after 15
			return button returned of dlg
		on error
			return ""
		end try`,
		escapeAppleScript(dialogBody),
	)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}

	choice := strings.TrimSpace(string(out))
	if choice == "Şarkıyı Geç" {
		return true, nil
	}

	return false, nil
}
