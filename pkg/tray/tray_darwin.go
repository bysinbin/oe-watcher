package tray

/*
#cgo CFLAGS: -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework AppKit
#include "tray_darwin.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

type MenuCallbacks struct {
	OnOpenWebPanel   func()
	OnSkipTrack      func()
	OnToggleAutoSkip func()
	OnSyncAPI        func()
	OnQuit           func()
	OnFastPoll       func()
}

var (
	cbMu      sync.RWMutex
	callbacks MenuCallbacks
	webURL    string
)

//export goOpenWebPanel
func goOpenWebPanel() {
	cbMu.RLock()
	cb := callbacks.OnOpenWebPanel
	url := webURL
	cbMu.RUnlock()

	if cb != nil {
		cb()
	} else if url != "" {
		OpenURL(url)
	}
}

//export goSkipTrack
func goSkipTrack() {
	cbMu.RLock()
	cb := callbacks.OnSkipTrack
	cbMu.RUnlock()

	if cb != nil {
		go cb()
	}
}

//export goToggleAutoSkip
func goToggleAutoSkip() {
	cbMu.RLock()
	cb := callbacks.OnToggleAutoSkip
	cbMu.RUnlock()

	if cb != nil {
		go cb()
	}
}

//export goSyncAPI
func goSyncAPI() {
	cbMu.RLock()
	cb := callbacks.OnSyncAPI
	cbMu.RUnlock()

	if cb != nil {
		go cb()
	}
}

//export goQuitApp
func goQuitApp() {
	cbMu.RLock()
	cb := callbacks.OnQuit
	cbMu.RUnlock()

	if cb != nil {
		cb()
	}
}

//export goTriggerFastPoll
func goTriggerFastPoll() {
	cbMu.RLock()
	cb := callbacks.OnFastPoll
	cbMu.RUnlock()

	if cb != nil {
		go cb()
	}
}

// StartTray initializes Cocoa status bar item and starts the main event loop.
// MUST be called on the main OS thread.
func StartTray(port int, autoSkip bool, cb MenuCallbacks) {
	runtime.LockOSThread()

	cbMu.Lock()
	callbacks = cb
	webURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	cbMu.Unlock()

	autoSkipVal := 0
	if autoSkip {
		autoSkipVal = 1
	}

	cTitle := C.CString("🎧 OE")
	defer C.free(unsafe.Pointer(cTitle))

	C.InitTray(cTitle, C.int(port), C.int(autoSkipVal))

	// Blocks and processes OS menu bar events
	C.RunCocoaApp()
}

// SetTrack updates the menu bar display with currently playing track.
func SetTrack(artist, title string, isMatched bool) {
	trackStr := "-"
	if artist != "" || title != "" {
		if artist != "" && title != "" {
			trackStr = fmt.Sprintf("%s - %s", artist, title)
		} else if title != "" {
			trackStr = title
		} else {
			trackStr = artist
		}
	}

	// Limit length for menu bar display
	if len([]rune(trackStr)) > 40 {
		trackStr = string([]rune(trackStr)[:37]) + "..."
	}

	matchedInt := 0
	if isMatched {
		matchedInt = 1
	}

	cStr := C.CString(trackStr)
	defer C.free(unsafe.Pointer(cStr))

	C.UpdateTrayTrack(cStr, C.int(matchedInt))
}

// SetAutoSkipState updates the menu toggle title.
func SetAutoSkipState(enabled bool) {
	val := 0
	if enabled {
		val = 1
	}
	C.UpdateTrayAutoSkip(C.int(val))
}

// OpenURL opens the given URL in the user's default web browser.
func OpenURL(url string) {
	cStr := C.CString(url)
	defer C.free(unsafe.Pointer(cStr))

	C.OpenBrowserURL(cStr)
}
