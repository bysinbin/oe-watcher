//go:build !darwin

package permission

// IsAccessibilityGranted returns true on non-macOS platforms as special accessibility permissions
// like macOS TCC are not required to send media keys or interact with media players.
func IsAccessibilityGranted() bool {
	return true
}

// RequestAccessibility is a no-op on non-macOS platforms.
func RequestAccessibility() bool {
	return true
}

// OpenAccessibilitySettings is a no-op on non-macOS platforms.
func OpenAccessibilitySettings() error {
	return nil
}

// CheckAndPrompt returns true on non-macOS platforms.
func CheckAndPrompt() bool {
	return true
}
