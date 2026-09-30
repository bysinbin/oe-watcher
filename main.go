package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"medya-filter/pkg/app"
	"medya-filter/pkg/permission"
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
	autoSkip := flag.Bool("auto-skip", false, "Onay sormadan kara listedeki şarkıları otomatik geç")
	noDialog := flag.Bool("no-dialog", false, "macOS masaüstü onay penceresini devre dışı bırak")
	noWeb := flag.Bool("no-web", false, "Web kontrol panelini devre dışı bırak")
	port := flag.Int("port", 8080, "Web kontrol paneli portu")
	interval := flag.Duration("interval", 1500*time.Millisecond, "Medya kontrol sıklığı")
	category := flag.String("category", "HEPSİ", "Filtrelenecek kategori (örn: Şarkıcı, Oyuncu)")
	forceSync := flag.Bool("sync", false, "Başlangıçta API ile tam senkronizasyon yap")
	testArtist := flag.String("test", "", "Bir sanatçı veya şarkı adını test et ve çık (örn: -test 'Ahmet Kaya')")
	flag.Parse()

	fmt.Print(banner)

	application := app.NewApp("data", *autoSkip, !*noDialog, *interval)

	if *category != "" && *category != "HEPSİ" {
		application.SetFilters([]string{*category}, []string{"TÜM KONULAR"})
	}

	fmt.Println("⏳ OE List API veritabanı hazırlanıyor...")
	if err := application.Init(*forceSync); err != nil {
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
		server := web.NewServer(application, *port)
		go func() {
			fmt.Printf("🌐 Web Kontrol Paneli: \033[1;36mhttp://localhost:%d\033[0m\n", *port)
			if err := server.Start(); err != nil {
				fmt.Printf("Web sunucu kapandı: %v\n", err)
			}
		}()
	}

	// Print operational modes
	fmt.Println("--------------------------------------------------------------------")
	if *autoSkip {
		fmt.Println("⚡ Mod: \033[1;31mOTOMATİK GEÇİŞ (Auto-Skip)\033[0m - Eşleşen şarkılar doğrudan geçilecek.")
	} else {
		fmt.Println("🔔 Mod: \033[1;33mONAYLI GEÇİŞ\033[0m - Eşleştiğinde macOS penceresi ile sorulacak.")
	}
	fmt.Printf("🎧 İzlenen Medyalar: Spotify, Apple Music, Google Chrome (YouTube), Safari\n")
	fmt.Println("⏹️ Çıkış yapmak için Ctrl+C tuşlarına basabilirsiniz.")
	fmt.Println("--------------------------------------------------------------------")

	// Start media polling in background
	go application.StartMonitoring(ctx)

	// Wait for OS shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Uygulama kapatılıyor...")
	cancel()
	time.Sleep(300 * time.Millisecond)
	fmt.Println("👋 Görüşmek üzere!")
}

func runTest(app *app.App, query string) {
	fmt.Printf("\n🔍 Test Sorgusu: '%s'\n", query)
	persons := app.GetAllPersons()

	matchFound := false
	for _, p := range persons {
		cleanQ := strings.ToLower(strings.TrimSpace(query))
		cleanName := strings.ToLower(p.DisplayName)
		if strings.Contains(cleanName, cleanQ) || strings.Contains(cleanQ, cleanName) {
			matchFound = true
			fmt.Println("\n\033[1;31m[EŞLEŞME BULUNDU]\033[0m")
			fmt.Printf("• İsim Soyisim : \033[1;33m%s\033[0m\n", p.DisplayName)
			fmt.Printf("• Kategori     : \033[1;36m%s\033[0m\n", strings.Join(p.Tags, ", "))
			fmt.Printf("• Konu(lar)    : \033[1;35m%s\033[0m\n", strings.Join(p.Topics, ", "))
		}
	}

	if !matchFound {
		fmt.Println("❌ Eşleşen kişi bulunamadı.")
	}
}
