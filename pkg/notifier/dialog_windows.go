//go:build windows

package notifier

import (
	"fmt"
	"os/exec"
	"strings"

	"medya-filter/pkg/models"
)

// PlayAlertSound plays a default alert beep on Windows.
func PlayAlertSound() {
	go func() {
		_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "[System.Media.SystemSounds]::Asterisk.Play()").Run()
	}()
}

// ShowNotification displays a Windows system tray notification balloon or toast.
func ShowNotification(title, subtitle, message string) {
	fullText := message
	if subtitle != "" {
		fullText = subtitle + "\n" + message
	}
	// Use PowerShell notify icon balloon tip
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = [System.Drawing.SystemIcons]::Information
$n.Visible = $true
$n.ShowBalloonTip(5000, "%s", "%s", [System.Windows.Forms.ToolTipIcon]::Info)
Start-Sleep -Seconds 1
$n.Dispose()
`, escapeQuotes(title), escapeQuotes(fullText))

	_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
}

// PromptAction displays a native Windows confirmation dialog.
func PromptAction(match *models.MatchResult) (string, error) {
	PlayAlertSound()

	body := formatDialogBody(match)
	ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$result = [System.Windows.Forms.MessageBox]::Show(
    "%s",
    "OE Watcher // Medya Uyarısı",
    [System.Windows.Forms.MessageBoxButtons]::YesNoCancel,
    [System.Windows.Forms.MessageBoxIcon]::Warning
)
switch ($result) {
    "Yes" { "skip" }
    "No"  { "snooze" }
    default { "keep" }
}
`, escapeQuotes(body+"\n\n[Evet: Şarkıyı Geç] [Hayır: 1 Saat Ertele] [İptal: Dinlemeye Devam]"))

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		return ActionKeep, err
	}

	choice := strings.TrimSpace(string(out))
	switch choice {
	case "skip":
		return ActionSkip, nil
	case "snooze":
		return ActionSnooze, nil
	default:
		return ActionKeep, nil
	}
}

func escapeQuotes(s string) string {
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, `"`, "`\"")
	return s
}
