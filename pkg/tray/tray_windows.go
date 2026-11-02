//go:build windows

package tray

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procAppendMenuW      = user32.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procSetForeground    = user32.NewProc("SetForegroundWindow")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procPostMessageW     = user32.NewProc("PostMessageW")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procShellNotifyW     = shell32.NewProc("Shell_NotifyIconW")

	trayMu          sync.RWMutex
	trayCallbacks   MenuCallbacks
	trayWebURL      string
	trayAutoSkip    bool
	trayHwnd        uintptr
	trayNID         NOTIFYICONDATAW
	trayClassAtom   uintptr
	trayInitialized bool
	trayWndProcCb   = syscall.NewCallback(trayWndProc)
)

const (
	WM_DESTROY       = 0x0002
	WM_COMMAND       = 0x0111
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_APP           = 0x8000
	WM_TRAY          = WM_APP + 101

	NIM_ADD     = 0x00000000
	NIM_MODIFY  = 0x00000001
	NIM_DELETE  = 0x00000002
	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	MF_STRING       = 0x00000000
	MF_DISABLED     = 0x00000002
	MF_CHECKED      = 0x00000008
	MF_SEPARATOR    = 0x00000800
	TPM_RIGHTBUTTON = 0x0002

	IDI_APPLICATION = 32512

	ID_HEADER          = 1000
	ID_OPEN_WEB        = 1001
	ID_TOGGLE_AUTOSKIP = 1002
	ID_SKIP            = 1003
	ID_SYNC            = 1004
	ID_QUIT            = 1005
)

type WNDCLASSEXW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type POINT struct {
	x, y int32
}

type MSG struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      POINT
}

type NOTIFYICONDATAW struct {
	cbSize            uint32
	hWnd              uintptr
	uID               uint32
	uFlags            uint32
	uCallbackMessage  uint32
	hIcon             uintptr
	szTip             [128]uint16
	dwState           uint32
	dwStateMask       uint32
	szInfo            [256]uint16
	uTimeoutOrVersion uint32
	szInfoTitle       [64]uint16
	dwInfoFlags       uint32
	guidItem          [16]byte
	hBalloonIcon      uintptr
}

func trayWndProc(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr {
	switch msg {
	case WM_TRAY:
		switch lparam {
		case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
			trayMu.RLock()
			onOpen := trayCallbacks.OnOpenWebPanel
			trayMu.RUnlock()
			if onOpen != nil {
				onOpen()
			}
			return 0

		case WM_RBUTTONUP:
			showContextMenu(hwnd)
			return 0
		}

	case WM_COMMAND:
		cmdID := int(wparam & 0xFFFF)
		trayMu.RLock()
		cb := trayCallbacks
		trayMu.RUnlock()

		switch cmdID {
		case ID_OPEN_WEB:
			if cb.OnOpenWebPanel != nil {
				cb.OnOpenWebPanel()
			}
		case ID_TOGGLE_AUTOSKIP:
			if cb.OnToggleAutoSkip != nil {
				cb.OnToggleAutoSkip()
			}
		case ID_SKIP:
			if cb.OnSkipTrack != nil {
				cb.OnSkipTrack()
			}
		case ID_SYNC:
			if cb.OnSyncAPI != nil {
				cb.OnSyncAPI()
			}
		case ID_QUIT:
			removeTrayIcon()
			procPostQuitMessage.Call(0)
			if cb.OnQuit != nil {
				cb.OnQuit()
			} else {
				os.Exit(0)
			}
		}
		return 0

	case WM_DESTROY:
		removeTrayIcon()
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wparam, lparam)
	return ret
}

func showContextMenu(hwnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	trayMu.RLock()
	autoSkip := trayAutoSkip
	trayMu.RUnlock()

	headerStr, _ := syscall.UTF16PtrFromString("🎧 OE Watcher (Aktif)")
	openWebStr, _ := syscall.UTF16PtrFromString("🌐 Web Kontrol Panelini Aç")
	autoSkipLabel := "⚡ Otomatik Geçiş (Auto-Skip)"
	if autoSkip {
		autoSkipLabel = "⚡ Otomatik Geçiş [✓ Aktif]"
	} else {
		autoSkipLabel = "⚡ Otomatik Geçiş [Kapalı]"
	}
	autoSkipStr, _ := syscall.UTF16PtrFromString(autoSkipLabel)
	skipStr, _ := syscall.UTF16PtrFromString("⏩ Şarkıyı Geç")
	syncStr, _ := syscall.UTF16PtrFromString("🔄 API Senkronize Et")
	quitStr, _ := syscall.UTF16PtrFromString("❌ Çıkış")

	autoSkipFlags := uintptr(MF_STRING)
	if autoSkip {
		autoSkipFlags |= uintptr(MF_CHECKED)
	}

	procAppendMenuW.Call(hMenu, uintptr(MF_STRING|MF_DISABLED), uintptr(ID_HEADER), uintptr(unsafe.Pointer(headerStr)))
	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_OPEN_WEB), uintptr(unsafe.Pointer(openWebStr)))
	procAppendMenuW.Call(hMenu, autoSkipFlags, uintptr(ID_TOGGLE_AUTOSKIP), uintptr(unsafe.Pointer(autoSkipStr)))
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_SKIP), uintptr(unsafe.Pointer(skipStr)))
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_SYNC), uintptr(unsafe.Pointer(syncStr)))
	procAppendMenuW.Call(hMenu, uintptr(MF_SEPARATOR), 0, 0)
	procAppendMenuW.Call(hMenu, uintptr(MF_STRING), uintptr(ID_QUIT), uintptr(unsafe.Pointer(quitStr)))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForeground.Call(hwnd)
	procTrackPopupMenu.Call(hMenu, uintptr(TPM_RIGHTBUTTON), uintptr(pt.x), uintptr(pt.y), 0, hwnd, 0)
	procPostMessageW.Call(hwnd, 0, 0, 0)
}

func removeTrayIcon() {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayInitialized {
		procShellNotifyW.Call(uintptr(NIM_DELETE), uintptr(unsafe.Pointer(&trayNID)))
		trayInitialized = false
	}
}

// StartTray runs the Windows event loop, displays a native system tray icon with context menu,
// and gracefully responds to termination signals.
func StartTray(port int, autoSkip bool, cb MenuCallbacks) {
	runtime.LockOSThread()

	trayMu.Lock()
	trayCallbacks = cb
	trayWebURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	trayAutoSkip = autoSkip
	trayMu.Unlock()

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := syscall.UTF16PtrFromString("OEWatcherTrayWndClass")

	var wc WNDCLASSEXW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = trayWndProcCb
	wc.hInstance = hInstance
	wc.lpszClassName = className
	atom, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	trayClassAtom = atom

	wndName, _ := syscall.UTF16PtrFromString("OEWatcherTrayWindow")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(wndName)),
		0,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	trayHwnd = hwnd

	hIcon, _, _ := procLoadIconW.Call(0, uintptr(IDI_APPLICATION))

	trayMu.Lock()
	trayNID.cbSize = uint32(unsafe.Sizeof(trayNID))
	trayNID.hWnd = hwnd
	trayNID.uID = 100
	trayNID.uFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	trayNID.uCallbackMessage = WM_TRAY
	trayNID.hIcon = hIcon
	copy(trayNID.szTip[:], syscall.StringToUTF16("OE Watcher // Medya ve İçerik Filtresi"))
	procShellNotifyW.Call(uintptr(NIM_ADD), uintptr(unsafe.Pointer(&trayNID)))
	trayInitialized = true
	trayMu.Unlock()

	fmt.Printf("\n🚀 OE Watcher aktif. Web Kontrol Paneli: %s\n", trayWebURL)
	fmt.Println("🎧 Windows Sistem Tepsisine (System Tray) simge yerleşti.")
	fmt.Println("💡 Sağ tık menüsünden kontrol edebilir veya Ctrl+C tuşlarıyla çıkabilirsiniz.")

	// Handle Ctrl+C signal
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		removeTrayIcon()
		procPostMessageW.Call(trayHwnd, WM_DESTROY, 0, 0)
	}()

	// Windows Message Loop
	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	trayMu.RLock()
	onQuit := trayCallbacks.OnQuit
	trayMu.RUnlock()

	if onQuit != nil {
		onQuit()
	} else {
		os.Exit(0)
	}
}

// SetTrack updates track display on Windows system tray tooltip.
func SetTrack(artist, title string, isMatched bool) {
	trayMu.Lock()
	defer trayMu.Unlock()

	if !trayInitialized {
		return
	}

	var tip string
	if artist != "" || title != "" {
		trackStr := title
		if artist != "" && !stringsContains(title, artist) {
			trackStr = fmt.Sprintf("%s - %s", artist, title)
		}
		if isMatched {
			tip = fmt.Sprintf("⚠️ OE Watcher [KARA LİSTE]\n%s", trackStr)
		} else {
			tip = fmt.Sprintf("OE Watcher\nÇalan: %s", trackStr)
		}
	} else {
		tip = "OE Watcher // Medya ve İçerik Filtresi"
	}

	if len(tip) > 120 {
		tip = tip[:117] + "..."
	}

	var zeroTip [128]uint16
	trayNID.szTip = zeroTip
	copy(trayNID.szTip[:], syscall.StringToUTF16(tip))
	trayNID.uFlags = NIF_TIP
	procShellNotifyW.Call(uintptr(NIM_MODIFY), uintptr(unsafe.Pointer(&trayNID)))
}

// SetAutoSkipState updates auto-skip status on Windows system tray menu.
func SetAutoSkipState(enabled bool) {
	trayMu.Lock()
	defer trayMu.Unlock()
	trayAutoSkip = enabled
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && len(s) > 0 && searchSubstr(s, substr))
}

func searchSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
