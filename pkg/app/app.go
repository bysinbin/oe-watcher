package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"medya-filter/pkg/api"
	"medya-filter/pkg/config"
	"medya-filter/pkg/matcher"
	"medya-filter/pkg/media"
	"medya-filter/pkg/models"
	"medya-filter/pkg/notifier"
	"medya-filter/pkg/rules"
	"medya-filter/pkg/tray"
)

type App struct {
	mu                  sync.RWMutex
	cfg                 *config.Config
	configPath          string
	client              *api.Client
	rulesMgr            *rules.Manager
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
	m := matcher.NewMatcher(nil)
	m.SetRulesManager(rulesMgr)

	cats := []string{"HEPSİ"}
	if cfg.Category != "" && cfg.Category != "HEPSİ" {
		cats = []string{cfg.Category}
	}

	return &App{
		cfg:              cfg,
		configPath:       configPath,
		client:           api.NewClient(cfg.ApiURL, dataDir),
		rulesMgr:         rulesMgr,
		matcher:          m,
		autoSkip:         cfg.AutoSkip,
		enableDialog:     cfg.EnableDialog,
		pollInterval:     interval,
		history:          make([]models.HistoryItem, 0),
		activeCategories: cats,
		activeTopics:     []string{"TÜM KONULAR"},
		syncStatus:       "Başlatılıyor...",
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
		return err
	}

	a.mu.Lock()
	a.matcher.SetPersons(persons)
	a.matcher.SetFilters(a.activeCategories, a.activeTopics)
	a.syncStatus = fmt.Sprintf("%d kişi yüklendi", len(persons))
	a.lastSyncTime = time.Now()
	a.mu.Unlock()

	return nil
}

// StartMonitoring begins the background polling loop.
func (a *App) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollOnce()
		}
	}
}

func (a *App) pollOnce() {
	mediaList, err := media.GetActiveMediaList()
	if err != nil || len(mediaList) == 0 {
		a.mu.Lock()
		a.currentMedia = models.MediaInfo{State: "stopped"}
		a.mu.Unlock()
		return
	}

	// 1. Check if any active media matches the blacklist
	var matchedResult *models.MatchResult
	var matchedTrack models.MediaInfo

	for _, track := range mediaList {
		match := a.matcher.Match(track)
		if match != nil {
			matchedResult = match
			matchedTrack = track
			break
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

	// 3. Skip handling
	if autoSkip {
		fmt.Println("\033[1;32m⏩ Otomatik Geçiş devrede: Şarkı geçiliyor...\033[0m")
		if err := media.SkipTrack(track); err != nil {
			fmt.Printf("\033[1;31m❌ Şarkı geçilemedi: %v\033[0m\n", err)
			notifier.ShowNotification("⚠️ Şarkı Geçilemedi", "Chrome Ayarı Gerekli", "Görünüm > Geliştirici > Apple Events iznini açın")
		} else {
			a.recordHistory(track, match.Person, "auto_skipped")
			notifier.ShowNotification("⏩ Şarkı Otomatik Geçildi", match.Person.DisplayName, track.Title)
		}
		return
	}

	if showDialog {
		// Run dialog in goroutine to not block polling
		go func() {
			fmt.Printf("\033[1;33m❓ Kullanıcıya soruluyor: '%s' geçilsin mi?\033[0m\n", match.Person.DisplayName)
			shouldSkip, err := notifier.PromptSkip(match)
			if err != nil {
				fmt.Printf("Dialog hatası: %v\n", err)
				return
			}

			if shouldSkip {
				fmt.Println("\033[1;32m✅ Kullanıcı 'Şarkıyı Geç' butonuna bastı. Geçiliyor...\033[0m")
				if err := media.SkipTrack(track); err != nil {
					fmt.Printf("\033[1;31m❌ Şarkı geçilemedi: %v\033[0m\n", err)
					notifier.ShowNotification("⚠️ Şarkı Geçilemedi", "Chrome Ayarı Gerekli", "Görünüm > Geliştirici > Apple Events iznini açın")
				} else {
					a.recordHistory(track, match.Person, "skipped")
					notifier.ShowNotification("⏩ Şarkı Geçildi", match.Person.DisplayName, track.Title)
				}
			} else {
				fmt.Println("\033[1;30m⏹️ Kullanıcı şarkıyı dinlemeye devam etmeyi seçti.\033[0m")
				a.recordHistory(track, match.Person, "kept")
			}
		}()
	} else {
		a.recordHistory(track, match.Person, "pending")
	}
}

func (a *App) recordHistory(track models.MediaInfo, person models.Person, action string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	item := models.HistoryItem{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp: time.Now().Format("15:04:05"),
		Track:     track,
		Person:    person,
		Action:    action,
	}

	// Keep max 50 items
	a.history = append([]models.HistoryItem{item}, a.history...)
	if len(a.history) > 50 {
		a.history = a.history[:50]
	}
}

// SkipCurrent forces skip on current media.
func (a *App) SkipCurrent() error {
	a.mu.RLock()
	cur := a.currentMedia
	match := a.currentMatch
	a.mu.RUnlock()

	err := media.SkipTrack(cur)
	if err == nil && match != nil {
		a.recordHistory(cur, match.Person, "manual_skip")
	}
	return err
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
}

func (a *App) GetState() AppState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	lastSyncStr := ""
	if !a.lastSyncTime.IsZero() {
		lastSyncStr = a.lastSyncTime.Format("02.01.2006 15:04:05")
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
		History:          a.history,
		LastSync:         lastSyncStr,
		Rules:            a.rulesMgr.GetSnapshot(),
	}
}

// GetConfig returns current configuration.
func (a *App) GetConfig() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// SetAutoSkip updates auto-skip preference and persists to config.
func (a *App) SetAutoSkip(val bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoSkip = val
	tray.SetAutoSkipState(val)
	if a.cfg != nil && a.configPath != "" {
		a.cfg.AutoSkip = val
		_ = config.Save(a.configPath, a.cfg)
	}
}

// SetFilters updates allowed categories and topics.
func (a *App) SetFilters(categories, topics []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.activeCategories = categories
	a.activeTopics = topics
	a.matcher.SetFilters(categories, topics)
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
	}
	return cp, err
}

// RemoveCustomPerson removes a custom person and rebuilds matcher.
func (a *App) RemoveCustomPerson(id string) bool {
	removed := a.rulesMgr.RemovePerson(id)
	if removed {
		a.matcher.RebuildIndex()
	}
	return removed
}

// SetCustomAlias maps an alias name to a target person name and rebuilds matcher.
func (a *App) SetCustomAlias(alias, target string) error {
	err := a.rulesMgr.SetAlias(alias, target)
	if err == nil {
		a.matcher.RebuildIndex()
	}
	return err
}

// RemoveCustomAlias removes an alias mapping and rebuilds matcher.
func (a *App) RemoveCustomAlias(alias string) bool {
	removed := a.rulesMgr.RemoveAlias(alias)
	if removed {
		a.matcher.RebuildIndex()
	}
	return removed
}

// AddExclude adds a name to the whitelist.
func (a *App) AddExclude(name string) error {
	return a.rulesMgr.AddExclude(name)
}

// RemoveExclude removes a name from the whitelist.
func (a *App) RemoveExclude(name string) bool {
	return a.rulesMgr.RemoveExclude(name)
}
