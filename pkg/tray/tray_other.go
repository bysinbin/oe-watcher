//go:build !darwin && !windows

package tray

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	cbMu      sync.RWMutex
	callbacks MenuCallbacks
	webURL    string
)

// StartTray runs an event loop on non-macOS/non-Windows platforms (e.g. Linux),
// providing console feedback and handling termination signals gracefully.
func StartTray(port int, autoSkip bool, cb MenuCallbacks) {
	cbMu.Lock()
	callbacks = cb
	webURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	cbMu.Unlock()

	fmt.Printf("\n🚀 OE Watcher aktif. Web Kontrol Paneli: %s\n", webURL)
	fmt.Println("💡 Uygulamayı sonlandırmak için Ctrl+C tuşlarına basabilirsiniz.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	cbMu.RLock()
	onQuit := callbacks.OnQuit
	cbMu.RUnlock()

	if onQuit != nil {
		onQuit()
	} else {
		os.Exit(0)
	}
}

// SetTrack updates track display on non-macOS/non-Windows platforms.
func SetTrack(artist, title string, isMatched bool) {}

// SetAutoSkipState updates auto-skip status on non-macOS/non-Windows platforms.
func SetAutoSkipState(enabled bool) {}
