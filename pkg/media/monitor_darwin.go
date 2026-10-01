//go:build darwin

package media

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"medya-filter/pkg/models"
)

var (
	ytOEmbedCache sync.Map
	ytHTTPClient  = &http.Client{Timeout: 2 * time.Second}
)

type YTOEmbed struct {
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
}

const queryScript = `
function cleanTitleAndArtist(raw) {
	if (!raw) return { artist: "", title: "", cleanFull: "", skip: true };
	let t = raw
		.replace(/\s*[-|•–—]\s*YouTube Music$/i, "")
		.replace(/\s*[-|•–—]\s*YouTube$/i, "")
		.replace(/\s*[-|•–—]\s*Spotify$/i, "")
		.replace(/\s*[-|•–—]\s*SoundCloud$/i, "")
		.trim();

	let low = t.toLowerCase();
	if (t === "" || low === "youtube music" || low === "youtube" || low === "spotify" || low === "soundcloud") {
		return { artist: "", title: "", cleanFull: "", skip: true };
	}

	let artist = "";
	let title = t;
	let seps = [" - ", " – ", " — ", " | "];
	for (let sep of seps) {
		if (t.includes(sep)) {
			let parts = t.split(sep);
			artist = parts[0].trim();
			title = parts.slice(1).join(sep).trim();
			break;
		}
	}

	return { artist: artist, title: title, cleanFull: t, skip: false };
}

function extractCredits(rawTitle) {
	let credits = [];
	try {
		let m = rawTitle.match(/[\(\[](?:feat\.?|ft\.?|prod\.?|beat|söz|müzik|düet|with)[\s:]+([^\)\]]+)[\)\]]/gi);
		if (m) {
			credits.push(m.join(" "));
		}
	} catch(e) {}
	return credits.join(" | ");
}

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

	// 3. Check Google Chrome
	try {
		if (Application("Google Chrome").running()) {
			let chrome = Application("Google Chrome");
			for (let win of chrome.windows()) {
				for (let tab of win.tabs()) {
					let url = tab.url() || "";
					let rawTitle = tab.title() || "";
					let isMusicSite = url.includes("music.youtube.com") || url.includes("open.spotify.com") || url.includes("soundcloud.com");
					let isVideoSite = url.includes("youtube.com/watch") || url.includes("youtube.com/shorts");

					if (isMusicSite || isVideoSite) {
						let res = cleanTitleAndArtist(rawTitle);
						if (res.skip) continue;

						let contribs = extractCredits(rawTitle);

						list.push({
							app: isMusicSite ? "Google Chrome (Music)" : "Google Chrome (YouTube)",
							artist: res.artist,
							title: res.title || res.cleanFull,
							album: "",
							state: "playing",
							url: url,
							contributors: contribs,
							priority: isMusicSite ? 1 : 2
						});
					}
				}
			}
		}
	} catch(e) {}

	// 4. Check Brave Browser
	try {
		if (Application("Brave Browser").running()) {
			let brave = Application("Brave Browser");
			for (let win of brave.windows()) {
				for (let tab of win.tabs()) {
					let url = tab.url() || "";
					let rawTitle = tab.title() || "";
					let isMusicSite = url.includes("music.youtube.com") || url.includes("open.spotify.com") || url.includes("soundcloud.com");
					let isVideoSite = url.includes("youtube.com/watch") || url.includes("youtube.com/shorts");

					if (isMusicSite || isVideoSite) {
						let res = cleanTitleAndArtist(rawTitle);
						if (res.skip) continue;

						let contribs = extractCredits(rawTitle);

						list.push({
							app: isMusicSite ? "Brave (Music)" : "Brave (YouTube)",
							artist: res.artist,
							title: res.title || res.cleanFull,
							album: "",
							state: "playing",
							url: url,
							contributors: contribs,
							priority: isMusicSite ? 1 : 2
						});
					}
				}
			}
		}
	} catch(e) {}

	// 5. Check Safari
	try {
		if (Application("Safari").running()) {
			let safari = Application("Safari");
			for (let win of safari.windows()) {
				for (let tab of win.tabs()) {
					let url = tab.url() || "";
					let rawTitle = tab.name() || "";
					let isMusicSite = url.includes("music.youtube.com") || url.includes("open.spotify.com") || url.includes("soundcloud.com");
					let isVideoSite = url.includes("youtube.com/watch") || url.includes("youtube.com/shorts");

					if (isMusicSite || isVideoSite) {
						let res = cleanTitleAndArtist(rawTitle);
						if (res.skip) continue;

						let contribs = extractCredits(rawTitle);

						list.push({
							app: isMusicSite ? "Safari (Music)" : "Safari (YouTube)",
							artist: res.artist,
							title: res.title || res.cleanFull,
							album: "",
							state: "playing",
							url: url,
							contributors: contribs,
							priority: isMusicSite ? 1 : 2
						});
					}
				}
			}
		}
	} catch(e) {}

	// 6. Check Arc
	try {
		if (Application("Arc").running()) {
			let arc = Application("Arc");
			for (let win of arc.windows()) {
				for (let tab of win.tabs()) {
					let url = tab.url() || "";
					let rawTitle = tab.title() || "";
					let isMusicSite = url.includes("music.youtube.com") || url.includes("open.spotify.com") || url.includes("soundcloud.com");
					let isVideoSite = url.includes("youtube.com/watch") || url.includes("youtube.com/shorts");

					if (isMusicSite || isVideoSite) {
						let res = cleanTitleAndArtist(rawTitle);
						if (res.skip) continue;

						let contribs = extractCredits(rawTitle);

						list.push({
							app: isMusicSite ? "Arc (Music)" : "Arc",
							artist: res.artist,
							title: res.title || res.cleanFull,
							album: "",
							state: "playing",
							url: url,
							contributors: contribs,
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

func extractYouTubeVideoID(rawURL string) string {
	if strings.Contains(rawURL, "youtu.be/") {
		parts := strings.Split(rawURL, "youtu.be/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "?")[0]
			return strings.Split(id, "&")[0]
		}
	}
	if strings.Contains(rawURL, "watch?") || strings.Contains(rawURL, "watch/") {
		delim := "watch?"
		if !strings.Contains(rawURL, "watch?") {
			delim = "watch/"
		}
		parts := strings.Split(rawURL, delim)
		if len(parts) > 1 {
			params := strings.Split(parts[1], "&")
			for _, p := range params {
				if strings.HasPrefix(p, "v=") {
					val := strings.TrimPrefix(p, "v=")
					return strings.Split(val, "?")[0]
				}
			}
			val := strings.Split(parts[1], "?")[0]
			val = strings.Split(val, "&")[0]
			if len(val) >= 6 && len(val) <= 16 && !strings.Contains(val, "=") {
				return val
			}
		}
	}
	for _, pathPart := range []string{"/shorts/", "/embed/", "/live/"} {
		if strings.Contains(rawURL, pathPart) {
			parts := strings.Split(rawURL, pathPart)
			if len(parts) > 1 {
				id := strings.Split(parts[1], "?")[0]
				return strings.Split(id, "&")[0]
			}
		}
	}
	return ""
}

func enrichYouTubeTrack(track *models.MediaInfo) {
	if track.URL == "" || (!strings.Contains(track.URL, "youtube.com") && !strings.Contains(track.URL, "youtu.be")) {
		return
	}
	videoID := extractYouTubeVideoID(track.URL)
	if videoID == "" {
		return
	}

	if val, ok := ytOEmbedCache.Load(videoID); ok {
		if emb, ok := val.(YTOEmbed); ok {
			applyOEmbed(track, emb)
			return
		}
	}

	oembedURL := fmt.Sprintf("https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=%s&format=json", videoID)
	req, err := http.NewRequest("GET", oembedURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := ytHTTPClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var emb YTOEmbed
		if err := json.NewDecoder(resp.Body).Decode(&emb); err == nil {
			ytOEmbedCache.Store(videoID, emb)
			applyOEmbed(track, emb)
		}
	}
}

func cleanVideoDecorators(raw string) string {
	t := strings.TrimSpace(raw)
	suffixes := []string{
		" (Official Video)",
		" (Official Music Video)",
		" (Official Audio)",
		" (Official Lyric Video)",
		" (Lyric Video)",
		" (Visualizer)",
		" [Official Video]",
		" [Official Music Video]",
		" [Official Audio]",
		" [Official Lyric Video]",
		" (Video)",
		" (Klip)",
		" (Audio)",
		" [Video]",
		" [Audio]",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(strings.ToLower(t), strings.ToLower(s)) {
			t = t[:len(t)-len(s)]
			t = strings.TrimSpace(t)
		}
	}
	return t
}

func applyOEmbed(track *models.MediaInfo, emb YTOEmbed) {
	author := strings.TrimSpace(emb.AuthorName)
	author = strings.TrimSuffix(author, " - Topic")
	author = strings.TrimSuffix(author, "VEVO")
	author = strings.TrimSpace(author)

	rawTitle := strings.TrimSpace(emb.Title)
	cleanedTitle := cleanVideoDecorators(rawTitle)

	var titleArtist, titleSong string
	// Split by standard musical artist-title separators: " - ", " – ", " — "
	for _, sep := range []string{" - ", " – ", " — "} {
		if strings.Contains(cleanedTitle, sep) {
			parts := strings.SplitN(cleanedTitle, sep, 2)
			titleArtist = strings.TrimSpace(parts[0])
			titleSong = cleanVideoDecorators(strings.TrimSpace(parts[1]))
			break
		}
	}

	var contribList []string
	if author != "" {
		contribList = append(contribList, author)
	}
	if rawTitle != "" {
		contribList = append(contribList, rawTitle)
	}
	if track.Contributors != "" && !strings.Contains(track.Contributors, rawTitle) {
		contribList = append(contribList, track.Contributors)
	}

	if titleArtist != "" {
		// Video title has "Artist - Song" format (e.g. "Halil Sezai - Kafası Kendinden Bile Güzel")
		track.Artist = titleArtist
		if titleSong != "" {
			track.Title = titleSong
		}
	} else {
		// Video title has no separator; fallback to channel uploader as artist if missing
		if cleanedTitle != "" {
			track.Title = cleanedTitle
		}
		if track.Artist == "" || strings.EqualFold(track.Artist, "YouTube") || strings.EqualFold(track.Artist, "YouTube Music") {
			if author != "" {
				track.Artist = author
			}
		}
	}

	if len(contribList) > 0 {
		track.Contributors = strings.Join(contribList, " | ")
	}
}

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

	for i := range list {
		enrichYouTubeTrack(&list[i])
	}

	return list, nil
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

// PlayPauseTrack toggles playback state on active media player.
func PlayPauseTrack(info models.MediaInfo) error {
	appLower := strings.ToLower(info.App)
	if strings.Contains(appLower, "spotify") {
		return exec.Command("osascript", "-l", "JavaScript", "-e", `Application("Spotify").playpause();`).Run()
	}
	if strings.Contains(appLower, "music") && !strings.Contains(appLower, "youtube") {
		return exec.Command("osascript", "-l", "JavaScript", "-e", `Application("Music").playpause();`).Run()
	}
	SendHardwarePlayPause()
	return nil
}
