//go:build linux

package media

import (
	"os/exec"
)

// SendHardwareNextTrack triggers next track command on Linux via playerctl, D-Bus MPRIS2, or xdotool.
func SendHardwareNextTrack() {
	// 1. Try playerctl
	if err := exec.Command("playerctl", "next").Run(); err == nil {
		return
	}

	// 2. Try MPRIS2 D-Bus broadcast/direct calls
	players := []string{"spotify", "chromium", "firefox", "vlc", "rhythmbox", "brave"}
	for _, p := range players {
		dest := "org.mpris.MediaPlayer2." + p
		_ = exec.Command("dbus-send", "--type=method_call", "--dest="+dest,
			"/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Next").Run()
	}

	// 3. Fallback to xdotool multimedia key
	_ = exec.Command("xdotool", "key", "XF86AudioNext").Run()
}

// SendHardwarePlayPause triggers play/pause command on Linux via playerctl, D-Bus MPRIS2, or xdotool.
func SendHardwarePlayPause() {
	// 1. Try playerctl
	if err := exec.Command("playerctl", "play-pause").Run(); err == nil {
		return
	}

	// 2. Try MPRIS2 D-Bus calls
	players := []string{"spotify", "chromium", "firefox", "vlc", "rhythmbox", "brave"}
	for _, p := range players {
		dest := "org.mpris.MediaPlayer2." + p
		_ = exec.Command("dbus-send", "--type=method_call", "--dest="+dest,
			"/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.PlayPause").Run()
	}

	// 3. Fallback to xdotool multimedia key
	_ = exec.Command("xdotool", "key", "XF86AudioPlay").Run()
}
