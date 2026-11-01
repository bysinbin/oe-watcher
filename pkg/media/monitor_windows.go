//go:build windows

package media

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"medya-filter/pkg/models"
)

var (
	procEnumWindows     = modUser32.NewProc("EnumWindows")
	procGetWindowTextW  = modUser32.NewProc("GetWindowTextW")
	procIsWindowVisible = modUser32.NewProc("IsWindowVisible")

	spotifyEnumMu       sync.Mutex
	spotifyFoundTitle   string
	spotifyEnumCallback = syscall.NewCallback(spotifyEnumProc)

	creditsRegex = regexp.MustCompile(`(?i)(?:feat\.?|ft\.?|prod\.?|beat|söz|müzik|düet|with)[\s:]+([^(\[\-\|]+)`)
	ytSuffixes   = []string{" - youtube music", " - youtube", " - soundcloud", " - spotify"}
)

const gsmtcScript = `Add-Type -AssemblyName System.Runtime.WindowsRuntime;
$a = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1' })[0];
function Aw($o, $t) { $m = $a.MakeGenericMethod($t); $k = $m.Invoke($null, @($o)); $k.Wait(); return $k.Result; };
[Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media, ContentType=WindowsRuntime] | Out-Null;
$mgr = Aw ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]);
if ($mgr) {
    $res = @();
    $curr = $mgr.GetCurrentSession();
    $all = $mgr.GetSessions();
    $ordered = @();
    if ($curr) { $ordered += $curr };
    foreach ($s in $all) {
        if (-not $curr -or $s.SourceAppUserModelId -ne $curr.SourceAppUserModelId) { $ordered += $s }
    };
    foreach ($s in $ordered) {
        try {
            $p = Aw ($s.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties]);
            if ($p -and ($p.Title -or $p.Artist)) {
                $info = $s.GetPlaybackInfo();
                $st = if ($info) { $info.PlaybackStatus.ToString().ToLower() } else { 'playing' };
                $res += @{
                    app = $s.SourceAppUserModelId;
                    title = $p.Title;
                    artist = $p.Artist;
                    album = $p.AlbumTitle;
                    state = $st;
                }
            }
        } catch {}
    };
    if ($res.Count -gt 0) { $res | ConvertTo-Json -Compress }
}`

type rawMediaItem struct {
	App    string `json:"app"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	State  string `json:"state"`
}

func cleanMediaTitleAndArtist(rawTitle, rawArtist string) (string, string, string) {
	t := strings.TrimSpace(rawTitle)
	for _, suffix := range ytSuffixes {
		if strings.HasSuffix(strings.ToLower(t), suffix) {
			t = strings.TrimSpace(t[:len(t)-len(suffix)])
		}
	}

	artist := strings.TrimSpace(rawArtist)
	title := t
	var credits string

	if m := creditsRegex.FindAllString(t, -1); len(m) > 0 {
		credits = strings.Join(m, " | ")
	}

	lowArtist := strings.ToLower(artist)
	if artist == "" || lowArtist == "youtube" || lowArtist == "chrome" || lowArtist == "google chrome" || lowArtist == "edge" || lowArtist == "brave" {
		for _, sep := range []string{" - ", " – ", " — ", " | "} {
			if strings.Contains(t, sep) {
				parts := strings.SplitN(t, sep, 2)
				artist = strings.TrimSpace(parts[0])
				title = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	return artist, title, credits
}

func normalizeAppName(rawApp string) string {
	low := strings.ToLower(rawApp)
	if strings.Contains(low, "spotify") {
		return "Spotify"
	}
	if strings.Contains(low, "chrome") {
		return "Google Chrome"
	}
	if strings.Contains(low, "edge") || strings.Contains(low, "msedge") {
		return "Microsoft Edge"
	}
	if strings.Contains(low, "brave") {
		return "Brave"
	}
	if strings.Contains(low, "firefox") {
		return "Mozilla Firefox"
	}
	if strings.Contains(low, "vlc") {
		return "VLC"
	}
	return rawApp
}

func spotifyEnumProc(hwnd uintptr, lparam uintptr) uintptr {
	vis, _, _ := procIsWindowVisible.Call(hwnd)
	if vis == 0 {
		return 1 // continue
	}

	buf := make([]uint16, 512)
	textLen, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
	if textLen > 0 {
		title := syscall.UTF16ToString(buf)
		low := strings.ToLower(title)
		// Spotify window title is "Artist - Track" when playing; "Spotify Free/Premium" when paused
		if strings.Contains(title, " - ") && !strings.Contains(low, "chrome") && !strings.Contains(low, "edge") && !strings.Contains(low, "firefox") {
			spotifyFoundTitle = title
			return 0 // stop enumeration
		}
	}
	return 1
}

// GetActiveMediaList queries Windows media playback using GSMTC and desktop window titles (Spotify).
func GetActiveMediaList() ([]models.MediaInfo, error) {
	var list []models.MediaInfo

	// 1. Try Windows GSMTC (Global System Media Transport Controls) via PowerShell
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", gsmtcScript)
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		var rawItems []rawMediaItem
		if jErr := json.Unmarshal(out, &rawItems); jErr != nil {
			var single rawMediaItem
			if jErr2 := json.Unmarshal(out, &single); jErr2 == nil {
				rawItems = append(rawItems, single)
			}
		}

		for _, item := range rawItems {
			if item.Title == "" && item.Artist == "" {
				continue
			}

			artist, title, credits := cleanMediaTitleAndArtist(item.Title, item.Artist)
			appName := normalizeAppName(item.App)
			state := strings.ToLower(item.State)
			if state == "" {
				state = "playing"
			}

			list = append(list, models.MediaInfo{
				App:          appName,
				Title:        title,
				Artist:       artist,
				Album:        item.Album,
				State:        state,
				Contributors: credits,
			})
		}

		if len(list) > 0 {
			return list, nil
		}
	}

	// 2. Fallback: inspect Spotify window title via EnumWindows
	spotifyTrack := findSpotifyWindowTitle()
	if spotifyTrack != "" && strings.Contains(spotifyTrack, " - ") {
		parts := strings.SplitN(spotifyTrack, " - ", 2)
		artist, title, credits := cleanMediaTitleAndArtist(parts[1], parts[0])
		list = append(list, models.MediaInfo{
			App:          "Spotify",
			Artist:       artist,
			Title:        title,
			State:        "playing",
			Contributors: credits,
		})
	}

	return list, nil
}

// findSpotifyWindowTitle enumerates top-level visible windows to find Spotify's "Artist - Track" title.
func findSpotifyWindowTitle() string {
	spotifyEnumMu.Lock()
	defer spotifyEnumMu.Unlock()

	spotifyFoundTitle = ""
	procEnumWindows.Call(spotifyEnumCallback, 0)
	return spotifyFoundTitle
}

// SkipTrack commands Windows media playback to advance to next track.
func SkipTrack(info models.MediaInfo) error {
	_ = sendGSMTCCommand("skip")
	SendHardwareNextTrack()
	return nil
}

// PauseTrack commands Windows media playback to pause.
func PauseTrack(info models.MediaInfo) error {
	_ = sendGSMTCCommand("pause")
	SendHardwarePlayPause()
	return nil
}

// PlayPauseTrack toggles Windows media playback.
func PlayPauseTrack(info models.MediaInfo) error {
	_ = sendGSMTCCommand("toggle")
	SendHardwarePlayPause()
	return nil
}
