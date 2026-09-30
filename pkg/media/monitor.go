package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"medya-filter/pkg/models"
)

const queryScript = `
function run() {
	let list = [];

	// 1. Check Spotify
	try {
		if (Application("Spotify").running()) {
			let sp = Application("Spotify");
			let state = sp.playerState();
			if (state === "playing" || state === "paused") {
				let track = sp.currentTrack;
				list.push({
					app: "Spotify",
					artist: track.artist() || "",
					title: track.name() || "",
					album: track.album() || "",
					state: state,
					priority: 0
				});
			}
		}
	} catch(e) {}

	// 2. Check Apple Music
	try {
		if (Application("Music").running()) {
			let m = Application("Music");
			let state = m.playerState();
			if (state === "playing" || state === "paused") {
				let track = m.currentTrack;
				list.push({
					app: "Apple Music",
					artist: track.artist() || "",
					title: track.name() || "",
					album: track.album() || "",
					state: state,
					priority: 0
				});
			}
		}
	} catch(e) {}

	// 3. Check Google Chrome (YouTube Music, Spotify Web, SoundCloud, YouTube)
	try {
		if (Application("Google Chrome").running()) {
			let chrome = Application("Google Chrome");
			for (let win of chrome.windows()) {
				for (let tab of win.tabs()) {
					let url = tab.url() || "";
					let rawTitle = tab.title() || "";
					let isMusicSite = url.includes("music.youtube.com") || url.includes("open.spotify.com") || url.includes("soundcloud.com") || url.includes("deezer.com");
					let isVideoSite = url.includes("youtube.com/watch");

					if (isMusicSite || isVideoSite) {
						let cleanTitle = rawTitle
							.replace(/\s*\|\s*YouTube Music$/i, "")
							.replace(/\s*-\s*YouTube Music$/i, "")
							.replace(/\s*-\s*YouTube$/i, "")
							.replace(/\s*\|\s*YouTube$/i, "")
							.replace(/\s*-\s*Spotify$/i, "")
							.replace(/\s*\|\s*SoundCloud$/i, "")
							.trim();

						let artist = "";
						let trackTitle = cleanTitle;
						if (cleanTitle.includes(" - ")) {
							let parts = cleanTitle.split(" - ");
							artist = parts[0].trim();
							trackTitle = parts.slice(1).join(" - ").trim();
						}

						list.push({
							app: isMusicSite ? "YouTube Music" : "Google Chrome",
							artist: artist,
							title: cleanTitle,
							album: "",
							state: "playing",
							url: url,
							priority: isMusicSite ? 1 : 2
						});
					}
				}
			}
		}
	} catch(e) {}

	// 4. Check Safari (YouTube, Music)
	try {
		if (Application("Safari").running()) {
			let safari = Application("Safari");
			for (let win of safari.windows()) {
				for (let tab of win.tabs()) {
					let url = tab.url() || "";
					let rawTitle = tab.name() || "";
					let isMusicSite = url.includes("music.youtube.com") || url.includes("open.spotify.com") || url.includes("soundcloud.com");
					let isVideoSite = url.includes("youtube.com/watch");

					if (isMusicSite || isVideoSite) {
						let cleanTitle = rawTitle
							.replace(/\s*\|\s*YouTube Music$/i, "")
							.replace(/\s*-\s*YouTube Music$/i, "")
							.replace(/\s*-\s*YouTube$/i, "")
							.replace(/\s*\|\s*YouTube$/i, "")
							.trim();

						let artist = "";
						if (cleanTitle.includes(" - ")) {
							let parts = cleanTitle.split(" - ");
							artist = parts[0].trim();
						}

						list.push({
							app: isMusicSite ? "Safari (Music)" : "Safari",
							artist: artist,
							title: cleanTitle,
							album: "",
							state: "playing",
							url: url,
							priority: isMusicSite ? 1 : 2
						});
					}
				}
			}
		}
	} catch(e) {}

	// Sort by priority (0 = desktop apps, 1 = dedicated web music, 2 = generic videos)
	list.sort((a, b) => a.priority - b.priority);

	return JSON.stringify(list);
}
`

// GetActiveMediaList queries running macOS media players and active browser tabs.
func GetActiveMediaList() ([]models.MediaInfo, error) {
	cmd := exec.Command("osascript", "-l", "JavaScript", "-e", queryScript)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("media query failed: %w", err)
	}

	var list []models.MediaInfo
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, fmt.Errorf("failed to parse media JSON: %w", err)
	}

	return list, nil
}

// GetCurrentMedia returns the primary active media track.
func GetCurrentMedia() (*models.MediaInfo, error) {
	list, err := GetActiveMediaList()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &models.MediaInfo{State: "stopped"}, nil
	}
	return &list[0], nil
}

// SkipTrack commands the media player to skip to the next track.
func SkipTrack(info models.MediaInfo) error {
	appLower := strings.ToLower(info.App)

	// 1. Desktop Spotify
	if strings.Contains(appLower, "spotify") && !strings.Contains(appLower, "chrome") && !strings.Contains(appLower, "safari") {
		script := `Application("Spotify").nextTrack();`
		return exec.Command("osascript", "-l", "JavaScript", "-e", script).Run()
	}

	// 2. Desktop Apple Music
	if strings.Contains(appLower, "apple music") || (strings.Contains(appLower, "music") && !strings.Contains(appLower, "youtube") && !strings.Contains(appLower, "chrome") && !strings.Contains(appLower, "safari")) {
		script := `Application("Music").nextTrack();`
		return exec.Command("osascript", "-l", "JavaScript", "-e", script).Run()
	}

	// 3. Google Chrome (YouTube Music, YouTube, Spotify Web)
	if strings.Contains(appLower, "chrome") || strings.Contains(appLower, "youtube") {
		// First try direct DOM interaction via AppleScript
		script := `
		(function() {
			let chrome = Application("Google Chrome");
			for (let win of chrome.windows()) {
				for (let tab of win.tabs()) {
					let url = tab.url() || "";
					if (url.includes("music.youtube.com") || url.includes("youtube.com/watch") || url.includes("open.spotify.com") || url.includes("soundcloud.com")) {
						try {
							tab.execute({javascript: "let b = document.querySelector('.next-button, .ytp-next-button'); if(b) { b.click(); } let v = document.querySelector('video'); if(v) { v.currentTime = v.duration || 99999; }"});
							return "ok";
						} catch(e) {
							return "need_media_key";
						}
					}
				}
			}
			return "no_tab";
		})();
		`
		out, _ := exec.Command("osascript", "-l", "JavaScript", "-e", script).Output()
		res := strings.TrimSpace(string(out))

		// Post native macOS hardware media key (Next Track) & MediaRemote command
		SendHardwareNextTrack()

		if res == "ok" {
			return nil
		}
		return nil
	}

	// 4. Safari (YouTube, Music)
	if strings.Contains(appLower, "safari") {
		SendHardwareNextTrack()
		return nil
	}

	// 5. Global Fallback
	SendHardwareNextTrack()
	return nil
}

// PauseTrack commands the active media player to pause.
func PauseTrack(info models.MediaInfo) error {
	appLower := strings.ToLower(info.App)
	if strings.Contains(appLower, "spotify") {
		return exec.Command("osascript", "-l", "JavaScript", "-e", `Application("Spotify").pause();`).Run()
	}
	if strings.Contains(appLower, "music") && !strings.Contains(appLower, "youtube") {
		return exec.Command("osascript", "-l", "JavaScript", "-e", `Application("Music").pause();`).Run()
	}
	if strings.Contains(appLower, "chrome") || strings.Contains(appLower, "youtube") {
		script := `
		(function() {
			let chrome = Application("Google Chrome");
			for (let win of chrome.windows()) {
				for (let tab of win.tabs()) {
					if (tab.url().includes("youtube.com") || tab.url().includes("spotify.com")) {
						try { tab.execute({javascript: "document.querySelector('video')?.pause();"}); } catch(e){}
					}
				}
			}
		})();
		`
		return exec.Command("osascript", "-l", "JavaScript", "-e", script).Run()
	}
	return nil
}
