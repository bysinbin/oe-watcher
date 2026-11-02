package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"medya-filter/pkg/api"
	"medya-filter/pkg/config"
	"medya-filter/pkg/history"
	"medya-filter/pkg/matcher"
	"medya-filter/pkg/media"
	"medya-filter/pkg/models"
	"medya-filter/pkg/notifier"
	"medya-filter/pkg/rules"
	"medya-filter/pkg/tray"
)

type AppStats struct {
	TotalDetections  int            `json:"total_detections"`
	TotalAutoSkips   int            `json:"total_auto_skips"`
	TotalManualSkips int            `json:"total_manual_skips"`
	TopPersons       map[string]int `json:"top_persons"`
}

type App struct {
	mu                  sync.RWMutex
	cfg                 *config.Config
	configPath          string
	client              *api.Client
	rulesMgr            *rules.Manager
	historyMgr          *history.Manager
	matcher             *matcher.Matcher
	currentMedia        models.MediaInfo
	currentMatch        *models.MatchResult
	lastHandledTrackKey string
	autoSkip            bool
	enableDialog        bool
	history             []models.HistoryItem
	pollInterval        time.Duration
	activeCategories    []string
	activeTopics        []string
	syncStatus          string
	lastSyncTime        time.Time
	snoozedPersons      map[string]time.Time
	stats               AppStats
	fastPollChan        chan struct{}
	stateListenersMu    sync.RWMutex
	stateListeners      []func(AppState)
}

func NewAppWithConfig(cfg *config.Config, configPath, dataDir string) *App {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if dataDir == "" {
		dataDir = "data"
	}
	interval := time.Duration(cfg.PollIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 1500 * time.Millisecond
	}

	rulesMgr := rules.NewManager(dataDir)
	historyMgr := history.NewManager(dataDir)

	m := matcher.NewMatcher(nil)
	m.SetRulesManager(rulesMgr)
	m.SetFuzzy(cfg.FuzzyMatching, cfg.FuzzyThreshold)

	cats := []string{"HEPSİ"}
	if cfg.Category != "" && cfg.Category != "HEPSİ" {
		cats = []string{cfg.Category}
	}

	hStats := historyMgr.GetStats()
	appStats := AppStats{
		TotalDetections:  hStats.TotalDetections,
		TotalAutoSkips:   hStats.TotalAutoSkips,
		TotalManualSkips: hStats.TotalManualSkips,
		TopPersons:       hStats.TopPersons,
	}
	if appStats.TopPersons == nil {
		appStats.TopPersons = make(map[string]int)
	}

	return &App{
		cfg:              cfg,
		configPath:       configPath,
		client:           api.NewClient(cfg.ApiURL, dataDir),
		rulesMgr:         rulesMgr,
		historyMgr:       historyMgr,
		matcher:          m,
		autoSkip:         cfg.AutoSkip,
		enableDialog:     cfg.EnableDialog,
		pollInterval:     interval,
		history:          historyMgr.GetAllItems(),
		activeCategories: cats,
		activeTopics:     []string{"TÜM KONULAR"},
		syncStatus:       "Başlatılıyor...",
		snoozedPersons:   make(map[string]time.Time),
		stats:            appStats,
		fastPollChan:     make(chan struct{}, 10),
		stateListeners:   make([]func(AppState), 0),
	}
}


func NewApp(dataDir string, autoSkip, enableDialog bool, pollInterval time.Duration) *App {
	cfg := config.DefaultConfig()
	cfg.AutoSkip = autoSkip
	cfg.EnableDialog = enableDialog
	if pollInterval > 0 {
		cfg.PollIntervalMs = int(pollInterval / time.Millisecond)
	}
	return NewAppWithConfig(cfg, "", dataDir)
}

// TriggerFastPoll immediately wakes up the monitoring loop on Darwin notifications (Spotify/Music).
func (a *App) TriggerFastPoll() {
	select {
	case a.fastPollChan <- struct{}{}:
	default:
	}
}

// RegisterStateListener registers a callback to receive realtime state changes (e.g. for SSE).
func (a *App) RegisterStateListener(fn func(AppState)) {
	a.stateListenersMu.Lock()
	defer a.stateListenersMu.Unlock()
	a.stateListeners = append(a.stateListeners, fn)
}

func (a *App) notifyStateChanged() {
	a.stateListenersMu.RLock()
	if len(a.stateListeners) == 0 {
		a.stateListenersMu.RUnlock()
		return
	}
	listeners := make([]func(AppState), len(a.stateListeners))
	copy(listeners, a.stateListeners)
	a.stateListenersMu.RUnlock()

	if len(listeners) == 0 {
		return
	}

	state := a.GetState()
	for _, fn := range listeners {
		go fn(state)
	}
}

// Init loads or synchronizes the person database.
func (a *App) Init(forceSync bool) error {
	a.mu.Lock()
	a.syncStatus = "Veri tabanı yükleniyor..."
	a.mu.Unlock()

	persons, err := a.client.GetPersons(forceSync)
	if err != nil {
		a.mu.Lock()
		a.syncStatus = fmt.Sprintf("Hata: %v", err)
		a.mu.Unlock()
		a.notifyStateChanged()
		return err
	}

	a.mu.Lock()
	a.matcher.SetPersons(persons)
	a.matcher.SetFilters(a.activeCategories, a.activeTopics)
	a.syncStatus = fmt.Sprintf("%d kişi yüklendi", len(persons))
	a.lastSyncTime = time.Now()
	a.mu.Unlock()

	a.notifyStateChanged()
	return nil
}

// StartAutoSync runs a periodic background synchronization with the API.
func (a *App) StartAutoSync(ctx context.Context) {
	a.mu.RLock()
	hours := 12
	if a.cfg != nil && a.cfg.AutoSyncHours > 0 {
		hours = a.cfg.AutoSyncHours
	}
	a.mu.RUnlock()

	if hours <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(hours) * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.RLock()
			curHours := 12
			if a.cfg != nil {
				curHours = a.cfg.AutoSyncHours
			}
			a.mu.RUnlock()

			if curHours > 0 {
				fmt.Printf("\n🔄 [Auto-Sync] %d saatlik periyodik API senkronizasyonu başlatılıyor...\n", curHours)
				_ = a.Init(true)
			}
		}
	}
}

// StartMonitoring begins the background polling loop with adaptive polling.
func (a *App) StartMonitoring(ctx context.Context) {
	baseInterval := a.pollInterval
	if baseInterval <= 0 {
		baseInterval = 1500 * time.Millisecond
	}
	idleInterval := baseInterval * 2
	if idleInterval < 3000*time.Millisecond {
		idleInterval = 3000 * time.Millisecond
	}

	ticker := time.NewTicker(baseInterval)
	defer ticker.Stop()

	// Initial check
	a.pollOnce()

	var wasIdle bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollOnce()
			a.mu.RLock()
			isIdle := a.currentMedia.State == "stopped" || a.currentMedia.State == ""
			a.mu.RUnlock()

			if isIdle && !wasIdle {
				ticker.Reset(idleInterval)
				wasIdle = true
			} else if !isIdle && wasIdle {
				ticker.Reset(baseInterval)
				wasIdle = false
			}
		case <-a.fastPollChan:
			time.Sleep(80 * time.Millisecond)
			a.pollOnce()
			ticker.Reset(baseInterval)
			wasIdle = false
		}
	}
}

func (a *App) pollOnce() {
	mediaList, err := media.GetActiveMediaList()
	if err != nil || len(mediaList) == 0 {
		a.mu.Lock()
		wasPlaying := a.currentMedia.State == "playing" || a.currentMedia.State == "paused"
		a.currentMedia = models.MediaInfo{State: "stopped"}
		a.currentMatch = nil
		a.lastHandledTrackKey = ""
		a.mu.Unlock()
		if wasPlaying {
			a.notifyStateChanged()
		}
		return
	}

	// 1. Check if any active media matches the blacklist
	var matchedResult *models.MatchResult
	var matchedTrack models.MediaInfo

	for _, track := range mediaList {
		match := a.matcher.Match(track)
		if match != nil {
			if !a.IsSnoozed(match.Person.DisplayName) {
				matchedResult = match
				matchedTrack = track
				break
			}
		}
	}

	primaryTrack := mediaList[0]
	if matchedResult != nil {
		primaryTrack = matchedTrack
	}

	trackKey := fmt.Sprintf("%s|%s|%s", primaryTrack.App, primaryTrack.Artist, primaryTrack.Title)

	a.mu.Lock()
	trackChanged := (trackKey != a.lastHandledTrackKey) && (primaryTrack.State == "playing" || primaryTrack.State == "paused")
	a.currentMedia = primaryTrack
	a.mu.Unlock()

	if !trackChanged {
		return
	}

	a.mu.Lock()
	a.currentMatch = matchedResult
	a.lastHandledTrackKey = trackKey
	autoSkipMode := a.autoSkip
	dialogMode := a.enableDialog
	a.mu.Unlock()

	a.notifyStateChanged()

	if matchedResult != nil {
		tray.SetTrack(primaryTrack.Artist, primaryTrack.Title, true)
		a.handleMatchDetected(matchedResult, matchedTrack, autoSkipMode, dialogMode)
	} else if primaryTrack.Artist != "" || primaryTrack.Title != "" {
		tray.SetTrack(primaryTrack.Artist, primaryTrack.Title, false)
		fmt.Printf("\033[90m[Medya Değişti]\033[0m %s: %s - %s \033[32m(Eşleşme Yok)\033[0m\n", primaryTrack.App, primaryTrack.Artist, primaryTrack.Title)
	} else {
		tray.SetTrack("", "", false)
	}
}

func (a *App) handleMatchDetected(match *models.MatchResult, track models.MediaInfo, autoSkip, showDialog bool) {
	a.mu.Lock()
	a.stats.TotalDetections++
	if a.stats.TopPersons == nil {
		a.stats.TopPersons = make(map[string]int)
	}
	a.stats.TopPersons[match.Person.DisplayName]++
	a.mu.Unlock()

	// 1. Terminal alert output
	topicsStr := strings.Join(match.Topics, ", ")
	if topicsStr == "" {
		topicsStr = "-"
	}

	trackDesc := track.Title
	if track.Artist != "" && !strings.Contains(track.Title, track.Artist) {
		trackDesc = fmt.Sprintf("%s - %s", track.Artist, track.Title)
	}

	fmt.Println("\n\033[1;31m====================================================================\033[0m")
	fmt.Printf("\033[1;31m🚨 [UYARI] KARA LİSTEDEKİ SANATÇI / KİŞİ TESPİT EDİLDİ!\033[0m\n")
	fmt.Printf("\033[1;37m• İsim Soyisim : \033[1;33m%s\033[0m\n", match.Person.DisplayName)
	fmt.Printf("\033[1;37m• Kategori     : \033[1;36m%s\033[0m\n", match.Category)
	fmt.Printf("\033[1;37m• Konu(lar)    : \033[1;35m%s\033[0m\n", topicsStr)
	fmt.Printf("\033[1;37m• Çalan Parça  : \033[1;37m%s\033[0m (\033[34m%s\033[0m)\n", trackDesc, track.App)
	fmt.Println("\033[1;31m====================================================================\033[0m")

	// 2. Desktop notification
	notifier.ShowNotification(
		"⚠️ Medya Filtresi: Eşleşme Bulundu!",
		fmt.Sprintf("%s (%s)", match.Person.DisplayName, match.Category),
		fmt.Sprintf("Çalan: %s", trackDesc),
	)

	// Send Discord / Telegram Webhook alert if configured
	if a.cfg != nil {
		notifier.SendWebhookAlert(
			a.cfg.DiscordWebhookURL,
			a.cfg.TelegramBotToken,
			a.cfg.TelegramChatID,
			"🚨 Kara Listedeki Kişi / İçerik Tespit Edildi!",
			fmt.Sprintf("%s adlı kişi çalınan medyada tespit edildi.", match.Person.DisplayName),
			map[string]string{
				"Kişi / Başlık": match.Person.DisplayName,
				"Kategori":      match.Category,
				"Konular":       topicsStr,
				"Parça / Video": trackDesc,
				"Uygulama":      track.App,
			},
		)
	}

	// 3. Skip handling
	if autoSkip {
		fmt.Println("\033[1;32m⏩ Otomatik Geçiş devrede: Şarkı geçiliyor...\033[0m")
		if err := media.SkipTrack(track); err != nil {
			fmt.Printf("\033[1;31m❌ Şarkı geçilemedi: %v\033[0m\n", err)
			notifier.ShowNotification("⚠️ Şarkı Geçilemedi", "Chrome Ayarı Gerekli", "Görünüm > Geliştirici > Apple Events iznini açın")
			a.recordHistoryWithDetails(track, match.Person, "skip_failed", false, match.Category, match.Topics, "Geçiş Başarısız")
		} else {
			a.recordHistoryWithDetails(track, match.Person, "auto_skipped", true, match.Category, match.Topics, "Otomatik Geçildi")
			if a.cfg == nil || a.cfg.EnableSkipToast {
				notifier.ShowNotification("⏩ Şarkı Otomatik Geçildi", match.Person.DisplayName, track.Title)
			}
		}
		return
	}

	if showDialog {
		// Run interactive dialog in goroutine to not block polling loop
		go func() {
			fmt.Printf("\033[1;33m❓ Kullanıcıya soruluyor: '%s' için işlem seçiniz...\033[0m\n", match.Person.DisplayName)
			action, err := notifier.PromptAction(match)
			if err != nil {
				fmt.Printf("Dialog hatası: %v\n", err)
				return
			}

			switch action {
			case notifier.ActionSkip:
				fmt.Println("\033[1;32m✅ Kullanıcı 'Şarkıyı Geç' butonuna bastı. Geçiliyor...\033[0m")
				if err := media.SkipTrack(track); err != nil {
					fmt.Printf("\033[1;31m❌ Şarkı geçilemedi: %v\033[0m\n", err)
					notifier.ShowNotification("⚠️ Şarkı Geçilemedi", "Chrome Ayarı Gerekli", "Görünüm > Geliştirici > Apple Events iznini açın")
					a.recordHistoryWithDetails(track, match.Person, "skip_failed", false, match.Category, match.Topics, "Geçiş Başarısız")
				} else {
					a.recordHistoryWithDetails(track, match.Person, "skipped", true, match.Category, match.Topics, "Onay ile Geçildi")
					notifier.ShowNotification("⏩ Şarkı Geçildi", match.Person.DisplayName, track.Title)
				}
			case notifier.ActionSnooze:
				fmt.Printf("\033[1;33m⏳ Kullanıcı '1 Saat Ertele' seçti: '%s' 1 saat boyunca muaf tutulacak.\033[0m\n", match.Person.DisplayName)
				a.SnoozePerson(match.Person.DisplayName, 1*time.Hour)
				a.recordHistoryWithDetails(track, match.Person, "snoozed", true, match.Category, match.Topics, "1 Saat Ertelendi")
				notifier.ShowNotification("⏳ 1 Saat Ertelendi", match.Person.DisplayName, "Bu kişi 1 saat boyunca uyarılmayacak.")
			case notifier.ActionKeep:
				fmt.Println("\033[1;30m⏹️ Kullanıcı şarkıyı dinlemeye devam etmeyi seçti.\033[0m")
				a.recordHistoryWithDetails(track, match.Person, "kept", true, match.Category, match.Topics, "Dinlendi")
			default:
				a.recordHistoryWithDetails(track, match.Person, "timeout", true, match.Category, match.Topics, "Zaman Aşımı")
			}
		}()
	} else {
		a.recordHistoryWithDetails(track, match.Person, "pending", true, match.Category, match.Topics, "Eşleşti (Beklemede)")
	}
}

func (a *App) recordHistoryWithDetails(track models.MediaInfo, person models.Person, action string, success bool, category string, topics []string, actionLabel string) {
	now := time.Now()
	item := models.HistoryItem{
		ID:          fmt.Sprintf("%d", now.UnixNano()),
		Timestamp:   now.Format("2006-01-02 15:04:05"),
		Track:       track,
		Person:      person,
		Category:    category,
		Topics:      topics,
		Action:      action,
		ActionLabel: actionLabel,
		Success:     success,
	}

	if a.historyMgr != nil {
		a.historyMgr.Record(item)
		hStats := a.historyMgr.GetStats()
		a.mu.Lock()
		a.history = a.historyMgr.GetAllItems()
		a.stats = AppStats{
			TotalDetections:  hStats.TotalDetections,
			TotalAutoSkips:   hStats.TotalAutoSkips,
			TotalManualSkips: hStats.TotalManualSkips,
			TopPersons:       hStats.TopPersons,
		}
		a.mu.Unlock()
	} else {
		a.mu.Lock()
		a.history = append([]models.HistoryItem{item}, a.history...)
		if len(a.history) > 100 {
			a.history = a.history[:100]
		}
		a.mu.Unlock()
	}

	a.notifyStateChanged()
}

func (a *App) recordHistory(track models.MediaInfo, person models.Person, action string) {
	a.recordHistoryWithDetails(track, person, action, true, "", nil, "")
}

// SkipCurrent forces skip on current media.
func (a *App) SkipCurrent() error {
	a.mu.RLock()
	cur := a.currentMedia
	match := a.currentMatch
	autoSkip := a.autoSkip
	a.mu.RUnlock()

	err := media.SkipTrack(cur)
	if err == nil {
		if match != nil {
			action := "manual_skip"
			label := "Manuel Geçildi"
			if autoSkip {
				action = "auto_skipped"
				label = "Otomatik Geçildi"
			}
			a.recordHistoryWithDetails(cur, match.Person, action, true, match.Category, match.Topics, label)
		}
	} else if match != nil {
		a.recordHistoryWithDetails(cur, match.Person, "skip_failed", false, match.Category, match.Topics, "Geçiş Başarısız")
	}
	return err
}


// PlayPauseCurrent toggles playback of current media.
func (a *App) PlayPauseCurrent() error {
	a.mu.RLock()
	cur := a.currentMedia
	a.mu.RUnlock()
	return media.PlayPauseTrack(cur)
}

// SnoozePerson temporarily ignores a person for a given duration.
func (a *App) SnoozePerson(displayName string, duration time.Duration) {
	a.mu.Lock()
	if a.snoozedPersons == nil {
		a.snoozedPersons = make(map[string]time.Time)
	}
	clean := strings.ToLower(strings.TrimSpace(displayName))
	a.snoozedPersons[clean] = time.Now().Add(duration)
	a.mu.Unlock()
	fmt.Printf("⏳ [Snooze] '%s' kişisi %v süresince muaf tutuldu.\n", displayName, duration)
	go a.notifyStateChanged()
}

// IsSnoozed checks if a person is currently snoozed.
func (a *App) IsSnoozed(displayName string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.snoozedPersons == nil {
		return false
	}
	clean := strings.ToLower(strings.TrimSpace(displayName))
	expiry, exists := a.snoozedPersons[clean]
	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

// GetSnoozedList returns currently active snoozed names.
func (a *App) GetSnoozedList() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var list []string
	now := time.Now()
	for name, expiry := range a.snoozedPersons {
		if now.Before(expiry) {
			list = append(list, name)
		}
	}
	return list
}

// GetState returns snapshot of current application status.
type AppState struct {
	CurrentMedia     models.MediaInfo     `json:"current_media"`
	CurrentMatch     *models.MatchResult  `json:"current_match"`
	AutoSkip         bool                 `json:"auto_skip"`
	EnableDialog     bool                 `json:"enable_dialog"`
	ActiveCategories []string             `json:"active_categories"`
	ActiveTopics     []string             `json:"active_topics"`
	SyncStatus       string               `json:"sync_status"`
	TotalPersons     int                  `json:"total_persons"`
	History          []models.HistoryItem `json:"history"`
	LastSync         string               `json:"last_sync"`
	Rules            rules.RulesSnapshot  `json:"rules"`
	Config           *config.Config       `json:"config"`
	Stats            AppStats             `json:"stats"`
	SnoozedList      []string             `json:"snoozed_list"`
}

func (a *App) GetState() AppState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	lastSyncStr := ""
	if !a.lastSyncTime.IsZero() {
		lastSyncStr = a.lastSyncTime.Format("02.01.2006 15:04:05")
	}

	var snoozed []string
	now := time.Now()
	for name, expiry := range a.snoozedPersons {
		if now.Before(expiry) {
			snoozed = append(snoozed, name)
		}
	}

	latestHistory := a.history
	if len(latestHistory) > 50 {
		latestHistory = latestHistory[:50]
	}

	return AppState{
		CurrentMedia:     a.currentMedia,
		CurrentMatch:     a.currentMatch,
		AutoSkip:         a.autoSkip,
		EnableDialog:     a.enableDialog,
		ActiveCategories: a.activeCategories,
		ActiveTopics:     a.activeTopics,
		SyncStatus:       a.syncStatus,
		TotalPersons:     len(a.matcher.GetPersons()),
		History:          latestHistory,
		LastSync:         lastSyncStr,
		Rules:            a.rulesMgr.GetSnapshot(),
		Config:           a.cfg,
		Stats:            a.stats,
		SnoozedList:      snoozed,
	}
}

// GetHistory returns paginated and filtered history items.
func (a *App) GetHistory(limit, offset int, action, search string) ([]models.HistoryItem, int) {
	if a.historyMgr != nil {
		return a.historyMgr.GetItems(limit, offset, action, search)
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.history, len(a.history)
}

// GetHistoryStats returns aggregated statistics from history manager.
func (a *App) GetHistoryStats() history.Stats {
	if a.historyMgr != nil {
		return a.historyMgr.GetStats()
	}
	return history.Stats{}
}

// ClearHistory wipes all history records and resets statistics.
func (a *App) ClearHistory() error {
	a.mu.Lock()
	a.history = make([]models.HistoryItem, 0)
	a.stats = AppStats{
		TopPersons: make(map[string]int),
	}
	a.mu.Unlock()

	if a.historyMgr != nil {
		if err := a.historyMgr.Clear(); err != nil {
			return err
		}
	}
	a.notifyStateChanged()
	return nil
}

// ExportHistoryCSV returns CSV bytes of all history records with UTF-8 BOM.
func (a *App) ExportHistoryCSV() ([]byte, error) {
	if a.historyMgr != nil {
		return a.historyMgr.ExportCSV()
	}
	return nil, fmt.Errorf("geçmiş yöneticisi aktif değil")
}

// ExportHistoryJSON returns JSON bytes of all history records and stats.
func (a *App) ExportHistoryJSON() ([]byte, error) {
	if a.historyMgr != nil {
		return a.historyMgr.ExportJSON()
	}
	return nil, fmt.Errorf("geçmiş yöneticisi aktif değil")
}

// GetConfig returns current configuration.
func (a *App) GetConfig() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *App) saveConfigLocked() {

	if a.cfg != nil && a.configPath != "" {
		if err := config.Save(a.configPath, a.cfg); err != nil {
			fmt.Printf("⚠️ config.json kaydedilemedi (%s): %v\n", a.configPath, err)
		} else {
			fmt.Printf("💾 [Config] Ayarlar başarıyla kaydedildi: %s (AutoSkip: %v, Dialog: %v, Kategori: %s, Fuzzy: %v)\n",
				a.configPath, a.cfg.AutoSkip, a.cfg.EnableDialog, a.cfg.Category, a.cfg.FuzzyMatching)
		}
	}
}

// UpdateConfig updates entire configuration and persists to config.json.
func (a *App) UpdateConfig(newCfg *config.Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if newCfg == nil {
		return fmt.Errorf("boş config")
	}

	a.cfg = newCfg
	a.autoSkip = newCfg.AutoSkip
	a.enableDialog = newCfg.EnableDialog
	if newCfg.PollIntervalMs > 0 {
		a.pollInterval = time.Duration(newCfg.PollIntervalMs) * time.Millisecond
	}
	if newCfg.Category != "" {
		if newCfg.Category == "HEPSİ" {
			a.activeCategories = []string{"HEPSİ"}
		} else {
			a.activeCategories = []string{newCfg.Category}
		}
		a.matcher.SetFilters(a.activeCategories, a.activeTopics)
	}
	a.matcher.SetFuzzy(newCfg.FuzzyMatching, newCfg.FuzzyThreshold)
	tray.SetAutoSkipState(newCfg.AutoSkip)

	a.saveConfigLocked()
	go a.notifyStateChanged()

	if newCfg.AutoSkip && a.currentMatch != nil && (a.currentMedia.State == "playing" || a.currentMedia.State == "paused") {
		curMatch := a.currentMatch
		curTrack := a.currentMedia
		if !a.IsSnoozed(curMatch.Person.DisplayName) {
			go func() {
				fmt.Printf("⏩ [Auto-Skip Aktif Edildi] Kara listedeki '%s' hemen geçiliyor...\n", curMatch.Person.DisplayName)
				_ = a.SkipCurrent()
				notifier.ShowNotification("⏩ Şarkı Otomatik Geçildi", curMatch.Person.DisplayName, curTrack.Title)
			}()
		}
	}

	return nil
}

// SetAutoSkip updates auto-skip preference and persists to config.
func (a *App) SetAutoSkip(val bool) {
	a.mu.Lock()
	a.autoSkip = val
	tray.SetAutoSkipState(val)
	if a.cfg != nil {
		a.cfg.AutoSkip = val
		a.saveConfigLocked()
	}
	curMatch := a.currentMatch
	curTrack := a.currentMedia
	a.mu.Unlock()

	go a.notifyStateChanged()

	// If auto_skip was turned ON and there is currently a matching track playing, skip it immediately!
	if val && curMatch != nil && (curTrack.State == "playing" || curTrack.State == "paused") {
		if !a.IsSnoozed(curMatch.Person.DisplayName) {
			go func() {
				fmt.Printf("⏩ [Auto-Skip Aktif Edildi] Kara listedeki '%s' hemen geçiliyor...\n", curMatch.Person.DisplayName)
				_ = a.SkipCurrent()
				notifier.ShowNotification("⏩ Şarkı Otomatik Geçildi", curMatch.Person.DisplayName, curTrack.Title)
			}()
		}
	}
}

// SetEnableDialog updates dialog preference and persists to config.
func (a *App) SetEnableDialog(val bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enableDialog = val
	if a.cfg != nil {
		a.cfg.EnableDialog = val
		a.saveConfigLocked()
	}
	go a.notifyStateChanged()
}

// SetFuzzy updates fuzzy matching preference and persists to config.
func (a *App) SetFuzzy(enabled bool, threshold float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.matcher.SetFuzzy(enabled, threshold)
	if a.cfg != nil {
		a.cfg.FuzzyMatching = enabled
		if threshold > 0 && threshold <= 1.0 {
			a.cfg.FuzzyThreshold = threshold
		}
		a.saveConfigLocked()
	}
	go a.notifyStateChanged()
}

// SetFilters updates allowed categories and topics and persists default category.
func (a *App) SetFilters(categories, topics []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.activeCategories = categories
	a.activeTopics = topics
	a.matcher.SetFilters(categories, topics)

	if len(categories) > 0 && a.cfg != nil {
		a.cfg.Category = categories[0]
		a.saveConfigLocked()
	}
	go a.notifyStateChanged()
}

// ExportRules exports rules as formatted JSON.
func (a *App) ExportRules() ([]byte, error) {
	return a.rulesMgr.Export()
}

// ImportRules imports rules from JSON and rebuilds index.
func (a *App) ImportRules(data []byte) error {
	err := a.rulesMgr.Import(data)
	if err == nil {
		a.matcher.RebuildIndex()
		go a.notifyStateChanged()
	}
	return err
}

// GetAllPersons returns all indexed persons.
func (a *App) GetAllPersons() []models.Person {
	return a.matcher.GetPersons()
}

// GetRulesManager returns the rules manager.
func (a *App) GetRulesManager() *rules.Manager {
	return a.rulesMgr
}

// GetMatcher returns the matcher engine.
func (a *App) GetMatcher() *matcher.Matcher {
	return a.matcher
}

// AddCustomPerson adds a custom person to the rules and rebuilds matcher.
func (a *App) AddCustomPerson(name string, tags, topics, aliases []string) (rules.CustomPerson, error) {
	cp, err := a.rulesMgr.AddPerson(name, tags, topics, aliases)
	if err == nil {
		a.matcher.RebuildIndex()
		go a.notifyStateChanged()
	}
	return cp, err
}

// RemoveCustomPerson removes a custom person and rebuilds matcher.
func (a *App) RemoveCustomPerson(id string) bool {
	removed := a.rulesMgr.RemovePerson(id)
	if removed {
		a.matcher.RebuildIndex()
		go a.notifyStateChanged()
	}
	return removed
}

// SetCustomAlias maps an alias name to a target person name and rebuilds matcher.
func (a *App) SetCustomAlias(alias, target string) error {
	err := a.rulesMgr.SetAlias(alias, target)
	if err == nil {
		a.matcher.RebuildIndex()
		go a.notifyStateChanged()
	}
	return err
}

// RemoveCustomAlias removes an alias mapping and rebuilds matcher.
func (a *App) RemoveCustomAlias(alias string) bool {
	removed := a.rulesMgr.RemoveAlias(alias)
	if removed {
		a.matcher.RebuildIndex()
		go a.notifyStateChanged()
	}
	return removed
}

// AddExclude adds a name to the whitelist.
func (a *App) AddExclude(name string) error {
	err := a.rulesMgr.AddExclude(name)
	if err == nil {
		go a.notifyStateChanged()
	}
	return err
}

// RemoveExclude removes a name from the whitelist.
func (a *App) RemoveExclude(name string) bool {
	removed := a.rulesMgr.RemoveExclude(name)
	if removed {
		go a.notifyStateChanged()
	}
	return removed
}

// AddKeyword adds a custom keyword or hashtag to the block list.
func (a *App) AddKeyword(keyword string) error {
	err := a.rulesMgr.AddKeyword(keyword)
	if err == nil {
		go a.notifyStateChanged()
	}
	return err
}

// RemoveKeyword removes a custom keyword from the block list.
func (a *App) RemoveKeyword(keyword string) bool {
	err := a.rulesMgr.RemoveKeyword(keyword)
	if err == nil {
		go a.notifyStateChanged()
		return true
	}
	return false
}

// GetKeywords returns the custom blocked keywords.
func (a *App) GetKeywords() []string {
	return a.rulesMgr.GetKeywords()
}

// SetWebhookSettings updates Discord/Telegram and SkipToast settings.
func (a *App) SetWebhookSettings(discordURL, tgToken, tgChatID string, enableSkipToast bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cfg != nil {
		a.cfg.DiscordWebhookURL = strings.TrimSpace(discordURL)
		a.cfg.TelegramBotToken = strings.TrimSpace(tgToken)
		a.cfg.TelegramChatID = strings.TrimSpace(tgChatID)
		a.cfg.EnableSkipToast = enableSkipToast
		a.saveConfigLocked()
	}
	go a.notifyStateChanged()
}

