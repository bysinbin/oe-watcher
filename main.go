package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"medya-filter/pkg/app"
	"medya-filter/pkg/config"
	"medya-filter/pkg/models"
	"medya-filter/pkg/permission"
	"medya-filter/pkg/tray"
	"medya-filter/pkg/web"
)

const banner = `
====================================================================
 ███╗   ███╗███████╗██████╗ ██╗   ██╗ █████╗     ███████╗██╗██╗  ████████╗
 ████╗ ████║██╔════╝██╔══██╗╚██╗ ██╔╝██╔══██╗    ██╔════╝██║██║  ╚══██╔══╝
 ██╔████╔██║█████╗  ██║  ██║ ╚████╔╝ ███████║    █████╗  ██║██║     ██║   
 ██║╚██╔╝██║██╔══╝  ██║  ██║  ╚██╔╝  ██╔══██║    ██╔══╝  ██║██║     ██║   
 ██║ ╚═╝ ██║███████╗██████╔╝   ██║   ██║  ██║    ██║     ██║███████╗██║   
 ╚═╝     ╚═╝╚══════╝╚═════╝    ╚═╝   ╚═╝  ╚═╝    ╚═╝     ╚═╝╚══════╝╚═╝   
====================================================================
         OE LIST // macOS Canlı Medya ve Sanatçı Takipçisi
====================================================================
`

func main() {
	runtime.LockOSThread()

	configPath := flag.String("config", "", "config.json dosya yolu (varsayılan: ./config.json)")
	cliMode := flag.Bool("cli", false, "macOS Menü Çubuğu (Tray) olmadan sadece terminal konsol modunda çalıştır")
	autoSkipFlag := flag.Bool("auto-skip", false, "Onay sormadan kara listedeki şarkıları otomatik geç")
	noDialogFlag := flag.Bool("no-dialog", false, "macOS masaüstü onay penceresini devre dışı bırak")
	noWeb := flag.Bool("no-web", false, "Web kontrol panelini devre dışı bırak")
	portFlag := flag.Int("port", 0, "Web kontrol paneli portu (config.json değerini ezer)")
	syncFlag := flag.Bool("sync", false, "Başlangıçta API ile tam senkronizasyon yap")
	testArtist := flag.String("test", "", "Bir sanatçı veya şarkı adını test et ve çık (örn: -test 'Ahmet Kaya')")
	installService := flag.Bool("install-service", false, "Mac açılışında arka planda otomatik başlaması için LaunchAgent servisi kur")
	uninstallService := flag.Bool("uninstall-service", false, "Mac açılışında otomatik başlatan LaunchAgent servisini kaldır")
	flag.Parse()

	if *installService {
		handleInstallService()
		return
	}
	if *uninstallService {
		handleUninstallService()
		return
	}

	fmt.Print(banner)

	// Load configuration from config.json
	resolvedConfig, resolvedDataDir := config.LocateConfigAndData(*configPath)
	cfg, loadedConfigPath, err := config.LoadOrCreate(resolvedConfig)
	if err != nil {
		fmt.Printf("⚠️ Config yüklenirken uyarı: %v (varsayılan ayarlar kullanılacak)\n", err)
	} else {
		fmt.Printf("⚙️ Yapılandırma: \033[1;36m%s\033[0m yüklendi.\n", loadedConfigPath)
	}

	// Change working directory to config directory so relative paths work seamlessly
	if configDir := filepath.Dir(loadedConfigPath); configDir != "" && configDir != "/" {
		_ = os.Chdir(configDir)
	}

	// Override config with CLI flags if explicitly passed
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "auto-skip":
			cfg.AutoSkip = *autoSkipFlag
		case "no-dialog":
			cfg.EnableDialog = !*noDialogFlag
		case "port":
			cfg.WebPort = *portFlag
		}
	})

	application := app.NewAppWithConfig(cfg, loadedConfigPath, resolvedDataDir)

	fmt.Println("⏳ OE List API veritabanı hazırlanıyor...")
	if err := application.Init(*syncFlag); err != nil {
		fmt.Printf("⚠️ Uyarı: Veritabanı başlatılırken hata oluştu: %v\n", err)
	}

	state := application.GetState()
	fmt.Printf("✅ Başarılı: Toplam \033[1;32m%d kişi\033[0m kara listede hazır.\n", state.TotalPersons)

	// Check macOS Accessibility permission
	permission.CheckAndPrompt()

	// Single test mode
	if *testArtist != "" {
		runTest(application, *testArtist)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Web Dashboard server
	if !*noWeb {
		server := web.NewServer(application, cfg.WebPort)
		go func() {
			fmt.Printf("🌐 Web Kontrol Paneli: \033[1;36mhttp://%s:%d\033[0m\n", cfg.BindAddress, cfg.WebPort)
			if cfg.WebPassword != "" {
				fmt.Printf("🔒 Web Paneli Güvenliği: \033[1;33mParola Korumalı\033[0m (Varsayılan şifre config.json 'web_password' alanındadır)\n")
			}
			if cfg.BindAddress == "127.0.0.1" {
				fmt.Printf("🛡️ Ağ Koruması: Yalnızca bu bilgisayardan erişilebilir (127.0.0.1)\n")
			}
			if err := server.Start(); err != nil {
				fmt.Printf("Web sunucu kapandı: %v\n", err)
			}
		}()
	}

	// Print operational modes
	fmt.Println("--------------------------------------------------------------------")
	if cfg.AutoSkip {
		fmt.Println("⚡ Mod: \033[1;31mOTOMATİK GEÇİŞ (Auto-Skip)\033[0m - Eşleşen şarkılar doğrudan geçilecek.")
	} else {
		fmt.Println("🔔 Mod: \033[1;33mONAYLI GEÇİŞ\033[0m - Eşleştiğinde macOS penceresi ile sorulacak.")
	}
	fmt.Printf("🎧 İzlenen Medyalar: Spotify, Apple Music, Google Chrome (YouTube), Safari\n")
	fmt.Println("--------------------------------------------------------------------")

	// Start media polling in background
	go application.StartMonitoring(ctx)

	// Start periodic silent API refresh loop in background
	go application.StartAutoSync(ctx)

	if !*cliMode {
		fmt.Println("🎧 Uygulama macOS Menü Çubuğuna (Bildirim Paneli) yerleşti.")
		fmt.Println("💡 Üst menü çubuğundaki '🎧 OE' simgesinden paneli açabilir, şarkı geçebilir veya çıkış yapabilirsiniz.")

		callbacks := tray.MenuCallbacks{
			OnOpenWebPanel: func() {
				tray.OpenURL(fmt.Sprintf("http://127.0.0.1:%d", cfg.WebPort))
			},
			OnSkipTrack: func() {
				fmt.Println("⏩ Menü Çubuğundan şarkı geçme istendi.")
				_ = application.SkipCurrent()
			},
			OnToggleAutoSkip: func() {
				cur := application.GetState().AutoSkip
				application.SetAutoSkip(!cur)
				fmt.Printf("⚡ Otomatik Geçiş menüden değiştirildi: %v\n", !cur)
			},
			OnSyncAPI: func() {
				fmt.Println("🔄 Menüden API senkronizasyonu başlatıldı...")
				_ = application.Init(true)
			},
			OnQuit: func() {
				fmt.Println("\n🛑 Menüden çıkış yapıldı...")
				cancel()
				os.Exit(0)
			},
			OnFastPoll: func() {
				// Darwin notification received (Spotify/Apple Music track changed)
				application.TriggerFastPoll()
			},
		}

		// Run Cocoa event loop on main OS thread
		tray.StartTray(cfg.WebPort, cfg.AutoSkip, callbacks)
	} else {
		fmt.Println("⏹️ Konsol modunda çalışıyor. Çıkış yapmak için Ctrl+C tuşlarına basabilirsiniz.")
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\n🛑 Uygulama kapatılıyor...")
		cancel()
		time.Sleep(300 * time.Millisecond)
		fmt.Println("👋 Görüşmek üzere!")
	}
}

func runTest(app *app.App, query string) {
	fmt.Printf("\n🔍 Test Sorgusu: '%s'\n", query)

	if app.GetRulesManager() != nil && app.GetRulesManager().IsExcluded(query) {
		fmt.Printf("\n\033[1;33m[BEYAZ LİSTE // EXCLUDED]\033[0m\n")
		fmt.Printf("• '%s' beyaz listede (exclude list) yer aldığı için eşleşse bile filtrelenmeyecek/atlanmayacak.\n", query)
		return
	}

	info := models.MediaInfo{
		Title:  query,
		Artist: query,
	}

	match := app.GetMatcher().Match(info)
	if match != nil {
		topicsStr := strings.Join(match.Topics, ", ")
		if topicsStr == "" {
			topicsStr = "-"
		}
		fmt.Println("\n\033[1;31m[EŞLEŞME BULUNDU]\033[0m")
		fmt.Printf("• İsim Soyisim : \033[1;33m%s\033[0m\n", match.Person.DisplayName)
		fmt.Printf("• Kategori     : \033[1;36m%s\033[0m\n", match.Category)
		fmt.Printf("• Konu(lar)    : \033[1;35m%s\033[0m\n", topicsStr)
		if match.MatchedName != "" {
			fmt.Printf("• Eşleşen İsim / Alias: \033[1;32m%s\033[0m\n", match.MatchedName)
		}
		return
	}

	fmt.Println("❌ Eşleşen kişi bulunamadı.")
}

func handleInstallService() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ Program dosya yolu bulunamadı: %v\n", err)
		return
	}
	appSupportDir := config.GetUserConfigDir()
	_ = os.MkdirAll(appSupportDir, 0755)

	switch runtime.GOOS {
	case "windows":
		// Windows Registry HKCU Startup Run key
		cmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "OEWatcher", "/t", "REG_SZ", "/d", exePath, "/f")
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Windows başlangıç kaydı eklenemedi: %v\n", err)
			return
		}
		fmt.Println("✅ Başarılı! OE Watcher Windows başlangıcına (Startup Run) eklendi.")
		fmt.Printf("📁 Yürütülebilir Dosya: %s\n", exePath)

	case "linux":
		// Install systemd user service and autostart desktop entry
		homeDir, _ := os.UserHomeDir()
		systemdDir := filepath.Join(homeDir, ".config", "systemd", "user")
		_ = os.MkdirAll(systemdDir, 0755)
		servicePath := filepath.Join(systemdDir, "oe-watcher.service")

		serviceContent := fmt.Sprintf(`[Unit]
Description=OE Watcher Background Service
After=network.target sound.target

[Service]
ExecStart=%s
WorkingDirectory=%s
Restart=always
RestartSec=5
StandardOutput=append:%s/oe-watcher.log
StandardError=append:%s/oe-watcher.err

[Install]
WantedBy=default.target
`, exePath, appSupportDir, appSupportDir, appSupportDir)

		_ = os.WriteFile(servicePath, []byte(serviceContent), 0644)
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		_ = exec.Command("systemctl", "--user", "enable", "--now", "oe-watcher.service").Run()

		autostartDir := filepath.Join(homeDir, ".config", "autostart")
		_ = os.MkdirAll(autostartDir, 0755)
		desktopPath := filepath.Join(autostartDir, "oe-watcher.desktop")
		desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=OE Watcher
Comment=Canlı Medya ve Sanatçı Takipçisi
Exec=%s
Terminal=false
Categories=AudioVideo;Utility;
X-GNOME-Autostart-enabled=true
`, exePath)
		_ = os.WriteFile(desktopPath, []byte(desktopContent), 0644)

		fmt.Println("✅ Başarılı! OE Watcher Linux systemd ve masaüstü başlangıcına eklendi.")
		fmt.Printf("📁 Servis dosyası: %s\n", servicePath)

	default: // darwin
		homeDir, _ := os.UserHomeDir()
		launchDir := filepath.Join(homeDir, "Library", "LaunchAgents")
		_ = os.MkdirAll(launchDir, 0755)
		plistPath := filepath.Join(launchDir, "com.bysinbin.oe-watcher.plist")

		plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.bysinbin.oe-watcher</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/oe-watcher.log</string>
    <key>StandardErrorPath</key>
    <string>%s/oe-watcher.err</string>
</dict>
</plist>`, exePath, appSupportDir, appSupportDir, appSupportDir)

		if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
			fmt.Printf("❌ Plist dosyası oluşturulamadı: %v\n", err)
			return
		}

		_ = exec.Command("launchctl", "unload", plistPath).Run()
		if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
			fmt.Printf("⚠️ Servis dosyası oluşturuldu fakat launchctl ile yüklenemedi: %v\n", err)
		} else {
			fmt.Println("✅ Başarılı! OE Watcher LaunchAgent servisi kuruldu.")
			fmt.Println("💡 Bilgisayarınızı her açtığınızda uygulama arka planda menü çubuğunda sessizce başlayacak.")
			fmt.Printf("📁 Servis dosyası: %s\n", plistPath)
		}
	}
}

func handleUninstallService() {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "OEWatcher", "/f").Run()
		fmt.Println("✅ OE Watcher Windows başlangıç kaydı silindi.")

	case "linux":
		homeDir, _ := os.UserHomeDir()
		_ = exec.Command("systemctl", "--user", "stop", "oe-watcher.service").Run()
		_ = exec.Command("systemctl", "--user", "disable", "oe-watcher.service").Run()
		_ = os.Remove(filepath.Join(homeDir, ".config", "systemd", "user", "oe-watcher.service"))
		_ = os.Remove(filepath.Join(homeDir, ".config", "autostart", "oe-watcher.desktop"))
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		fmt.Println("✅ OE Watcher Linux servisi ve otomatik başlatma kaydı kaldırıldı.")

	default: // darwin
		homeDir, _ := os.UserHomeDir()
		plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.bysinbin.oe-watcher.plist")
		_ = exec.Command("launchctl", "unload", plistPath).Run()
		_ = os.Remove(plistPath)
		fmt.Println("✅ OE Watcher otomatik başlatma servisi başarıyla kaldırıldı.")
	}
}
