//go:build !darwin && !windows && !linux

package media

// SendHardwareNextTrack is a no-op fallback for unsupported OS platforms.
func SendHardwareNextTrack() {}

// SendHardwarePlayPause is a no-op fallback for unsupported OS platforms.
func SendHardwarePlayPause() {}
