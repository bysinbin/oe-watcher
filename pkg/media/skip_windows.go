//go:build windows

package media

import (
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
