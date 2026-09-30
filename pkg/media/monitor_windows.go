//go:build windows

package media

import (
	"encoding/json"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"medya-filter/pkg/models"
)

var (
	procEnumWindows    = modUser32.NewProc("EnumWindows")
	procGetWindowTextW = modUser32.NewProc("GetWindowTextW")
	procIsWindowVisible = modUser32.NewProc("IsWindowVisible")
)

// GetActiveMediaList queries Windows media playback using GSMTC and desktop window titles (Spotify).
func GetActiveMediaList() ([]models.MediaInfo, error) {
	var list []models.MediaInfo

	// 1. Try Windows GSMTC (Global System Media Transport Controls) via PowerShell
	psCmd := `[Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media, ContentType=WindowsRuntime] > $null; ` +
		`$m = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync().GetAwaiter().GetResult(); ` +
		`$s = $m.GetCurrentSession(); ` +
		`if ($s) { ` +
		`  $p = $s.TryGetMediaPropertiesAsync().GetAwaiter().GetResult(); ` +
		`  @{app=$s.SourceAppUserModelId; title=$p.Title; artist=$p.Artist; album=$p.AlbumTitle; state='playing'} | ConvertTo-Json -Compress ` +
		`}`

	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
	if err == nil && len(out) > 0 {
		var item struct {
			App    string `json:"app"`
			Title  string `json:"title"`
			Artist string `json:"artist"`
			Album  string `json:"album"`
			State  string `json:"state"`
		}
		if jsonErr := json.Unmarshal(out, &item); jsonErr == nil && item.Title != "" {
			appName := item.App
			if strings.Contains(strings.ToLower(appName), "spotify") {
				appName = "Spotify"
			} else if strings.Contains(strings.ToLower(appName), "chrome") {
				appName = "Google Chrome"
			} else if strings.Contains(strings.ToLower(appName), "edge") {
				appName = "Microsoft Edge"
			}

			list = append(list, models.MediaInfo{
				App:    appName,
				Title:  item.Title,
				Artist: item.Artist,
				Album:  item.Album,
				State:  item.State,
			})
			return list, nil
		}
	}

	// 2. Fallback: inspect Spotify window title via EnumWindows
	spotifyTrack := findSpotifyWindowTitle()
	if spotifyTrack != "" && strings.Contains(spotifyTrack, " - ") {
		parts := strings.SplitN(spotifyTrack, " - ", 2)
		list = append(list, models.MediaInfo{
			App:    "Spotify",
			Artist: strings.TrimSpace(parts[0]),
			Title:  strings.TrimSpace(parts[1]),
			State:  "playing",
		})
	}

	return list, nil
}

// findSpotifyWindowTitle enumerates top-level visible windows to find Spotify's "Artist - Track" title.
func findSpotifyWindowTitle() string {
	var foundTitle string

	callback := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1 // continue
		}

		buf := make([]uint16, 512)
		len, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
		if len > 0 {
			title := syscall.UTF16ToString(buf)
			low := strings.ToLower(title)
			// Spotify window title is "Artist - Track" when playing; "Spotify Free/Premium" when paused
			if strings.Contains(title, " - ") && !strings.Contains(low, "chrome") && !strings.Contains(low, "edge") && !strings.Contains(low, "firefox") {
				foundTitle = title
				return 0 // stop enumeration
			}
		}
		return 1
	})

	procEnumWindows.Call(callback, 0)
	return foundTitle
}

// SkipTrack commands Windows media playback to advance to next track.
func SkipTrack(info models.MediaInfo) error {
	SendHardwareNextTrack()
	return nil
}

// PauseTrack commands Windows media playback to pause.
func PauseTrack(info models.MediaInfo) error {
	SendHardwarePlayPause()
	return nil
}

// PlayPauseTrack toggles Windows media playback.
func PlayPauseTrack(info models.MediaInfo) error {
	SendHardwarePlayPause()
	return nil
}
