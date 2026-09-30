package tray

import (
	"os/exec"
	"runtime"
)

type MenuCallbacks struct {
	OnOpenWebPanel   func()
	OnSkipTrack      func()
	OnToggleAutoSkip func()
	OnSyncAPI        func()
	OnQuit           func()
	OnFastPoll       func()
}

// OpenURL opens the given URL in the user's default web browser across platforms.
func OpenURL(url string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}
