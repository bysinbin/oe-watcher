//go:build darwin

package permission

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework ApplicationServices
#import <Foundation/Foundation.h>
#import <ApplicationServices/ApplicationServices.h>

int CheckAccessibility(int prompt) {
	if (prompt) {
		NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
		return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
	}
	return AXIsProcessTrusted() ? 1 : 0;
}
*/
import "C"

import (
	"fmt"
	"os/exec"

	"medya-filter/pkg/notifier"
)

// IsAccessibilityGranted returns true if the process has macOS Accessibility permission.
func IsAccessibilityGranted() bool {
	return C.CheckAccessibility(0) == 1
}

// RequestAccessibility prompts macOS system permission dialog if not already granted.
func RequestAccessibility() bool {
	return C.CheckAccessibility(1) == 1
}

// OpenAccessibilitySettings opens the Privacy & Security -> Accessibility panel in macOS Settings.
func OpenAccessibilitySettings() error {
	cmd := exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility")
	return cmd.Run()
}

// CheckAndPrompt checks if accessibility permission is granted on app startup.
// If missing, it prints a colored terminal warning, prompts macOS system dialog,
// opens System Settings, and shows a desktop banner notification.
func CheckAndPrompt() bool {
	if IsAccessibilityGranted() {
		fmt.Println("🔒 macOS Erişilebilirlik (Accessibility) İzni: \033[1;32mETKİN (Granted)\033[0m")
		return true
	}

	fmt.Println("\n\033[1;33m⚠️  [DİKKAT] macOS Erişilebilirlik (Accessibility) İzni Eksik!\033[0m")
	fmt.Println("Medya kontrolleri (şarkıyı geçme ve tarayıcı etkileşimi) için bu izin gereklidir.")
	fmt.Println("👉 Sistem Ayarları açılıyor... Lütfen listenizden Terminal veya medya-filter uygulamasını etkinleştirin.")

	// Trigger system prompt
	RequestAccessibility()

	// Open Settings window directly
	_ = OpenAccessibilitySettings()

	// Display desktop notification
	notifier.ShowNotification(
		"Erişilebilirlik İzni Gerekli",
		"OE Watcher",
		"Müzik çalar kontrolleri için lütfen Sistem Ayarları'ndan OE Watcher erişilebilirlik iznini açın.",
	)

	return false
}
