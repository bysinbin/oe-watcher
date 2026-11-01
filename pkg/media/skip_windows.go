//go:build windows

package media

import (
	"os/exec"
	"syscall"
)

var (
	modUser32      = syscall.NewLazyDLL("user32.dll")
	procKeybdEvent = modUser32.NewProc("keybd_event")
)

const (
	vkMediaNextTrack = 0xB0
	vkMediaPlayPause = 0xB3
	keyeventfKeyUp   = 0x0002
)

const gsmtcControlScript = `param($action)
Add-Type -AssemblyName System.Runtime.WindowsRuntime;
$a = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1' })[0];
function Aw($o, $t) { $m = $a.MakeGenericMethod($t); $k = $m.Invoke($null, @($o)); $k.Wait(); return $k.Result; };
[Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media, ContentType=WindowsRuntime] | Out-Null;
$mgr = Aw ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]);
if ($mgr) {
    $s = $mgr.GetCurrentSession();
    if (-not $s) {
        $all = $mgr.GetSessions();
        if ($all.Count -gt 0) { $s = $all[0] }
    };
    if ($s) {
        if ($action -eq 'skip') {
            $ok = Aw ($s.TrySkipNextAsync()) ([bool]);
            if (-not $ok) {
                # Fallback: if cannot skip next track, pause playback to stop forbidden audio/video
                Aw ($s.TryPauseAsync()) ([bool]) | Out-Null;
            }
        } elseif ($action -eq 'pause') {
            Aw ($s.TryPauseAsync()) ([bool]) | Out-Null;
        } elseif ($action -eq 'toggle') {
            Aw ($s.TryTogglePlayPauseAsync()) ([bool]) | Out-Null;
        }
    }
}`

func sendGSMTCCommand(action string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "& { "+gsmtcControlScript+" }", action)
	return cmd.Run()
}

// SendHardwareNextTrack triggers Windows system-wide media key (VK_MEDIA_NEXT_TRACK).
func SendHardwareNextTrack() {
	procKeybdEvent.Call(uintptr(vkMediaNextTrack), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vkMediaNextTrack), 0, uintptr(keyeventfKeyUp), 0)
}

// SendHardwarePlayPause triggers Windows system-wide media key (VK_MEDIA_PLAY_PAUSE).
func SendHardwarePlayPause() {
	procKeybdEvent.Call(uintptr(vkMediaPlayPause), 0, 0, 0)
	procKeybdEvent.Call(uintptr(vkMediaPlayPause), 0, uintptr(keyeventfKeyUp), 0)
}
