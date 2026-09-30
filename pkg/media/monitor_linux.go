//go:build linux

package media

import (
	"os/exec"
	"strings"

	"medya-filter/pkg/models"
)

// GetActiveMediaList queries active media players on Linux via playerctl or MPRIS2.
func GetActiveMediaList() ([]models.MediaInfo, error) {
	var list []models.MediaInfo

	// 1. Try playerctl
	out, err := exec.Command("playerctl", "-a", "metadata", "--format", "{{playerName}};;{{status}};;{{artist}};;{{title}};;{{album}}").Output()
	if err == nil && len(out) > 0 {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			parts := strings.Split(line, ";;")
			if len(parts) >= 4 {
				app := strings.TrimSpace(parts[0])
				status := strings.ToLower(strings.TrimSpace(parts[1]))
				artist := strings.TrimSpace(parts[2])
				title := strings.TrimSpace(parts[3])
				album := ""
				if len(parts) >= 5 {
					album = strings.TrimSpace(parts[4])
				}

				if title != "" && (status == "playing" || status == "paused") {
					list = append(list, models.MediaInfo{
						App:    app,
						Title:  title,
						Artist: artist,
						Album:  album,
						State:  status,
					})
				}
			}
		}
		if len(list) > 0 {
			return list, nil
		}
	}

	return list, nil
}

// SkipTrack commands Linux media player to advance to next track.
func SkipTrack(info models.MediaInfo) error {
	SendHardwareNextTrack()
	return nil
}

// PauseTrack commands Linux media player to pause.
func PauseTrack(info models.MediaInfo) error {
	SendHardwarePlayPause()
	return nil
}

// PlayPauseTrack toggles Linux media playback.
func PlayPauseTrack(info models.MediaInfo) error {
	SendHardwarePlayPause()
	return nil
}
