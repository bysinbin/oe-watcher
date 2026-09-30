//go:build linux

package notifier

import (
	"os/exec"

	"medya-filter/pkg/models"
)

// PlayAlertSound plays a system bell sound on Linux desktop.
func PlayAlertSound() {
	go func() {
		if err := exec.Command("canberra-gtk-play", "-i", "bell").Run(); err == nil {
			return
		}
		_ = exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/bell.oga").Run()
	}()
}

// ShowNotification displays a desktop notification via notify-send.
func ShowNotification(title, subtitle, message string) {
	fullText := message
	if subtitle != "" {
		fullText = subtitle + "\n" + message
	}
	_ = exec.Command("notify-send", "-a", "OE Watcher", "-i", "dialog-warning", title, fullText).Run()
}

// PromptAction displays a native Linux dialog via zenity or kdialog.
func PromptAction(match *models.MatchResult) (string, error) {
	PlayAlertSound()

	body := formatDialogBody(match)

	// 1. Try zenity
	cmd := exec.Command("zenity", "--question", "--title=OE Watcher // Medya Uyarısı",
		"--text="+body, "--ok-label=Şarkıyı Geç", "--cancel-label=Dinlemeye Devam", "--timeout=15")
	if err := cmd.Run(); err == nil {
		return ActionSkip, nil
	}

	// 2. Try kdialog (KDE)
	kcmd := exec.Command("kdialog", "--yesno", body, "--title", "OE Watcher // Medya Uyarısı")
	if err := kcmd.Run(); err == nil {
		return ActionSkip, nil
	}

	return ActionKeep, nil
}
