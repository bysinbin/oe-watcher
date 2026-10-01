package web

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"medya-filter/pkg/app"
	"medya-filter/pkg/config"
	"medya-filter/pkg/models"
	"medya-filter/pkg/notifier"
)

//go:embed static/*
var staticFS embed.FS

type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]bool
}

func newSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan []byte]bool),
	}
}

func (h *SSEHub) Subscribe() chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan []byte, 16)
	h.clients[ch] = true
	return ch
}

func (h *SSEHub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

func (h *SSEHub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
		}
	}
}

type Server struct {
	app         *app.App
	bindAddress string
	port        int
	password    string
	sessionMu   sync.RWMutex
	sessions    map[string]time.Time
	hub         *SSEHub
}

func NewServer(application *app.App, port int) *Server {
	bindAddr := "127.0.0.1"
	password := ""
	if cfg := application.GetConfig(); cfg != nil {
		if cfg.BindAddress != "" {
			bindAddr = cfg.BindAddress
		}
		if port <= 0 && cfg.WebPort > 0 {
			port = cfg.WebPort
		}
		password = cfg.WebPassword
	}
	if port <= 0 {
		port = 8080
	}

	hub := newSSEHub()
	srv := &Server{
		app:         application,
		bindAddress: bindAddr,
		port:        port,
		password:    password,
		sessions:    make(map[string]time.Time),
		hub:         hub,
	}

	application.RegisterStateListener(func(st app.AppState) {
		if data, err := json.Marshal(st); err == nil {
			hub.Broadcast(data)
		}
	})

	return srv
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Public Auth & Extension endpoints
	mux.HandleFunc("/api/auth-status", s.handleAuthStatus)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/logout", s.handleLogout)
	mux.HandleFunc("/api/extension/sync", s.handleExtensionSync)
	mux.HandleFunc("/api/extension/exclude", s.handleRuleExclude)
	mux.HandleFunc("/api/extension/keyword", s.handleRuleKeyword)
	mux.HandleFunc("/api/extension/person", s.handleExtensionPerson)
	mux.HandleFunc("/api/extension/auto-skip", s.handleExtensionAutoSkip)
	mux.HandleFunc("/api/extension/skip", s.handleExtensionSkip)

	// Protected API endpoints
	mux.HandleFunc("/api/events", s.authMiddleware(s.handleEvents))
	mux.HandleFunc("/api/status", s.authMiddleware(s.handleStatus))
	mux.HandleFunc("/api/skip", s.authMiddleware(s.handleSkip))
	mux.HandleFunc("/api/settings", s.authMiddleware(s.handleSettings))
	mux.HandleFunc("/api/settings/test-webhook", s.authMiddleware(s.handleTestWebhook))
	mux.HandleFunc("/api/config", s.authMiddleware(s.handleConfig))
	mux.HandleFunc("/api/snooze", s.authMiddleware(s.handleSnooze))
	mux.HandleFunc("/api/player/play-pause", s.authMiddleware(s.handlePlayPause))
	mux.HandleFunc("/api/sync", s.authMiddleware(s.handleSync))
	mux.HandleFunc("/api/persons", s.authMiddleware(s.handlePersons))
	mux.HandleFunc("/api/history", s.authMiddleware(s.handleHistory))
	mux.HandleFunc("/api/history/export", s.authMiddleware(s.handleHistoryExport))
	mux.HandleFunc("/api/history/clear", s.authMiddleware(s.handleHistoryClear))
	mux.HandleFunc("/api/rules", s.authMiddleware(s.handleRules))
	mux.HandleFunc("/api/rules/export", s.authMiddleware(s.handleRulesExport))
	mux.HandleFunc("/api/rules/import", s.authMiddleware(s.handleRulesImport))
	mux.HandleFunc("/api/rules/person", s.authMiddleware(s.handleRulePerson))
	mux.HandleFunc("/api/rules/alias", s.authMiddleware(s.handleRuleAlias))
	mux.HandleFunc("/api/rules/exclude", s.authMiddleware(s.handleRuleExclude))
	mux.HandleFunc("/api/rules/keyword", s.authMiddleware(s.handleRuleKeyword))

	// Embedded Static files
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("static subfs failed: %w", err)
	}
	fileServer := http.FileServer(http.FS(staticSub))
	mux.Handle("/", fileServer)

	addr := fmt.Sprintf("%s:%d", s.bindAddress, s.port)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) generateSessionToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) isAuthenticated(r *http.Request) bool {
	if s.password == "" {
		return true // Parola tanımlanmamışsa herkes girebilir
	}

	// 1. Bearer Token kontrolü
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if s.validateSession(token) {
			return true
		}
	}

	// 2. Cookie kontrolü
	cookie, err := r.Cookie("oe_session")
	if err == nil && cookie != nil {
		if s.validateSession(cookie.Value) {
			return true
		}
	}

	// 3. Query Param kontrolü (EventSource/SSE için)
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		if s.validateSession(qToken) {
			return true
		}
	}

	return false
}

func (s *Server) validateSession(token string) bool {
	if token == "" {
		return false
	}
	s.sessionMu.RLock()
	expiry, exists := s.sessions[token]
	s.sessionMu.RUnlock()

	if !exists || time.Now().After(expiry) {
		return false
	}
	return true
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthenticated(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Bu işlem için şifre ile giriş yapmanız gerekmektedir.",
			})
			return
		}
		next(w, r)
	}
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"required":      s.password != "",
		"authenticated": s.isAuthenticated(r),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz istek", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if s.password != "" && req.Password != s.password {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Hatalı parola! Lütfen config.json dosyasındaki şifreyi girin.",
		})
		return
	}

	token := s.generateSessionToken()
	s.sessionMu.Lock()
	s.sessions[token] = time.Now().Add(7 * 24 * time.Hour) // 7 gün geçerli
	s.sessionMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "oe_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("oe_session"); err == nil {
		s.sessionMu.Lock()
		delete(s.sessions, cookie.Value)
		s.sessionMu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oe_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial state immediately upon SSE connection
	state := s.app.GetState()
	if data, err := json.Marshal(state); err == nil {
		fmt.Fprintf(w, "event: state\ndata: %s\n\n", data)
		flusher.Flush()
	}

	ch := s.hub.Subscribe()
	defer s.hub.Unsubscribe(ch)

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: state\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state := s.app.GetState()
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) handleSkip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := s.app.SkipCurrent()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

type SettingsRequest struct {
	AutoSkip          *bool    `json:"auto_skip,omitempty"`
	EnableDialog      *bool    `json:"enable_dialog,omitempty"`
	FuzzyMatching     *bool    `json:"fuzzy_matching,omitempty"`
	FuzzyThreshold    *float64 `json:"fuzzy_threshold,omitempty"`
	ActiveCategories  []string `json:"active_categories,omitempty"`
	ActiveTopics      []string `json:"active_topics,omitempty"`
	DiscordWebhookURL *string  `json:"discord_webhook_url,omitempty"`
	TelegramBotToken  *string  `json:"telegram_bot_token,omitempty"`
	TelegramChatID    *string  `json:"telegram_chat_id,omitempty"`
	EnableSkipToast   *bool    `json:"enable_skip_toast,omitempty"`
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.AutoSkip != nil {
		s.app.SetAutoSkip(*req.AutoSkip)
	}
	if req.EnableDialog != nil {
		s.app.SetEnableDialog(*req.EnableDialog)
	}
	if req.FuzzyMatching != nil {
		th := 0.85
		if req.FuzzyThreshold != nil {
			th = *req.FuzzyThreshold
		} else if cfg := s.app.GetConfig(); cfg != nil {
			th = cfg.FuzzyThreshold
		}
		s.app.SetFuzzy(*req.FuzzyMatching, th)
	} else if req.FuzzyThreshold != nil {
		fm := true
		if cfg := s.app.GetConfig(); cfg != nil {
			fm = cfg.FuzzyMatching
		}
		s.app.SetFuzzy(fm, *req.FuzzyThreshold)
	}

	if req.ActiveCategories != nil || req.ActiveTopics != nil {
		state := s.app.GetState()
		cats := state.ActiveCategories
		topics := state.ActiveTopics
		if req.ActiveCategories != nil {
			cats = req.ActiveCategories
		}
		if req.ActiveTopics != nil {
			topics = req.ActiveTopics
		}
		s.app.SetFilters(cats, topics)
	}

	if req.DiscordWebhookURL != nil || req.TelegramBotToken != nil || req.TelegramChatID != nil || req.EnableSkipToast != nil {
		cfg := s.app.GetConfig()
		dURL := ""
		tgToken := ""
		tgChat := ""
		skipToast := true
		if cfg != nil {
			dURL = cfg.DiscordWebhookURL
			tgToken = cfg.TelegramBotToken
			tgChat = cfg.TelegramChatID
			skipToast = cfg.EnableSkipToast
		}
		if req.DiscordWebhookURL != nil {
			dURL = *req.DiscordWebhookURL
		}
		if req.TelegramBotToken != nil {
			tgToken = *req.TelegramBotToken
		}
		if req.TelegramChatID != nil {
			tgChat = *req.TelegramChatID
		}
		if req.EnableSkipToast != nil {
			skipToast = *req.EnableSkipToast
		}
		s.app.SetWebhookSettings(dURL, tgToken, tgChat, skipToast)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.app.GetState())
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(s.app.GetConfig())
		return
	}

	if r.Method == http.MethodPost {
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.app.UpdateConfig(&newCfg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Update in-memory server password if changed
		s.password = newCfg.WebPassword

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"config":  s.app.GetConfig(),
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

type SnoozeRequest struct {
	Person          string `json:"person"`
	DurationMinutes int    `json:"duration_minutes"`
}

func (s *Server) handleSnooze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SnoozeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mins := req.DurationMinutes
	if mins <= 0 {
		mins = 60
	}

	s.app.SnoozePerson(req.Person, time.Duration(mins)*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"person":  req.Person,
		"expires": time.Now().Add(time.Duration(mins) * time.Minute).Format("15:04:05"),
	})
}

func (s *Server) handlePlayPause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := s.app.PlayPauseCurrent()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleRulesExport(w http.ResponseWriter, r *http.Request) {
	data, err := s.app.ExportRules()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"oe_custom_rules.json\"")
	_, _ = w.Write(data)
}

func (s *Server) handleRulesImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "Geçersiz JSON", http.StatusBadRequest)
		return
	}

	if err := s.app.ImportRules(raw); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"rules":   s.app.GetRulesManager().GetSnapshot(),
	})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := s.app.Init(true)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"state":   s.app.GetState(),
	})
}

func (s *Server) handlePersons(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	topic := strings.TrimSpace(r.URL.Query().Get("topic"))

	all := s.app.GetAllPersons()
	filtered := make([]models.Person, 0)

	for _, p := range all {
		if query != "" {
			match := strings.Contains(strings.ToLower(p.DisplayName), query)
			if !match {
				for _, t := range p.Tags {
					if strings.Contains(strings.ToLower(t), query) {
						match = true
						break
					}
				}
			}
			if !match {
				for _, top := range p.Topics {
					if strings.Contains(strings.ToLower(top), query) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}

		if category != "" {
			foldCat := foldTurkishStr(category)
			if foldCat != "" && foldCat != "hepsi" && foldCat != "tumu" && foldCat != "all" {
				hasCat := false
				for _, t := range p.Tags {
					if foldTurkishStr(t) == foldCat {
						hasCat = true
						break
					}
				}
				if !hasCat {
					continue
				}
			}
		}

		if topic != "" {
			foldTop := foldTurkishStr(topic)
			if foldTop != "" && foldTop != "tum konular" && foldTop != "hepsi" && foldTop != "tumu" && foldTop != "all" {
				hasTopic := false
				for _, top := range p.Topics {
					if foldTurkishStr(top) == foldTop {
						hasTopic = true
						break
					}
				}
				if !hasTopic {
					continue
				}
			}
		}

		filtered = append(filtered, p)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(filtered)
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	snap := s.app.GetRulesManager().GetSnapshot()
	_ = json.NewEncoder(w).Encode(snap)
}

type AddPersonReq struct {
	Name    string   `json:"name"`
	Tags    []string `json:"tags"`
	Topics  []string `json:"topics"`
	Aliases []string `json:"aliases"`
}

type DelPersonReq struct {
	ID string `json:"id"`
}

func (s *Server) handleRulePerson(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		var req AddPersonReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cp, err := s.app.AddCustomPerson(req.Name, req.Tags, req.Topics, req.Aliases)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(cp)
		return
	} else if r.Method == http.MethodDelete {
		var req DelPersonReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok := s.app.RemoveCustomPerson(req.ID)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": ok})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

type AliasReq struct {
	Alias  string `json:"alias"`
	Target string `json:"target"`
}

func (s *Server) handleRuleAlias(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		var req AliasReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := s.app.SetCustomAlias(req.Alias, req.Target)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	} else if r.Method == http.MethodDelete {
		var req AliasReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok := s.app.RemoveCustomAlias(req.Alias)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": ok})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

type ExcludeReq struct {
	Name string `json:"name"`
}

func (s *Server) handleRuleExclude(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		var req ExcludeReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := s.app.AddExclude(req.Name)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	} else if r.Method == http.MethodDelete {
		var req ExcludeReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok := s.app.RemoveExclude(req.Name)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": ok})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	limit := 50
	if l := q.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if limit > 500 {
		limit = 500
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	action := q.Get("action")
	search := q.Get("search")

	items, total := s.app.GetHistory(limit, offset, action, search)
	hStats := s.app.GetHistoryStats()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"stats":  hStats,
	})
}

func (s *Server) handleHistoryExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	nowDate := time.Now().Format("2006-01-02")

	if format == "json" {
		data, err := s.app.ExportHistoryJSON()
		if err != nil {
			http.Error(w, "JSON dışa aktarılamadı: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"oe_watcher_gecmis_raporu_%s.json\"", nowDate))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	// Default: CSV export with UTF-8 BOM
	data, err := s.app.ExportHistoryCSV()
	if err != nil {
		http.Error(w, "CSV dışa aktarılamadı: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"oe_watcher_gecmis_raporu_%s.csv\"", nowDate))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleHistoryClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.app.ClearHistory(); err != nil {
		http.Error(w, "Geçmiş temizlenemedi: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Geçmiş raporu ve istatistikler başarıyla temizlendi.",
	})
}

func (s *Server) handleExtensionSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	all := s.app.GetAllPersons()
	snap := s.app.GetRulesManager().GetSnapshot()

	type ExtensionPerson struct {
		Name   string   `json:"name"`
		Tags   []string `json:"tags"`
		Topics []string `json:"topics"`
	}

	list := make([]ExtensionPerson, 0, len(all))
	for _, p := range all {
		list = append(list, ExtensionPerson{
			Name:   p.DisplayName,
			Tags:   p.Tags,
			Topics: p.Topics,
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"count":        len(list),
		"persons":      list,
		"aliases":      snap.Aliases,
		"exclude_list": snap.ExcludeList,
		"keywords":     snap.Keywords,
		"auto_skip":    s.app.GetState().AutoSkip,
	})
}

func (s *Server) handleExtensionAutoSkip(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"auto_skip": s.app.GetState().AutoSkip,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			AutoSkip bool `json:"auto_skip"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.app.SetAutoSkip(req.AutoSkip)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"auto_skip": s.app.GetState().AutoSkip,
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleExtensionSkip(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := s.app.SkipCurrent()
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

type KeywordReq struct {
	Keyword string `json:"keyword"`
}

func (s *Server) handleRuleKeyword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		var req KeywordReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := s.app.AddKeyword(req.Keyword)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	} else if r.Method == http.MethodDelete {
		var req KeywordReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ok := s.app.RemoveKeyword(req.Keyword)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": ok})
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

type ExtensionPersonReq struct {
	Name string `json:"name"`
}

func (s *Server) handleExtensionPerson(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ExtensionPersonReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	clean := strings.TrimSpace(req.Name)
	if clean == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "İsim boş olamaz"})
		return
	}

	cp, err := s.app.AddCustomPerson(clean, []string{"Eklenti"}, []string{"Tarayıcıdan Eklendi"}, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "person": cp})
}

type TestWebhookReq struct {
	Type       string `json:"type"` // "discord" or "telegram"
	WebhookURL string `json:"webhook_url"`
	BotToken   string `json:"bot_token"`
	ChatID     string `json:"chat_id"`
}

func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestWebhookReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if req.Type == "discord" {
		url := strings.TrimSpace(req.WebhookURL)
		if url == "" && s.app.GetConfig() != nil {
			url = s.app.GetConfig().DiscordWebhookURL
		}
		if url == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Discord Webhook URL belirtilmedi"})
			return
		}
		err := notifier.SendDiscordNotification(
			url,
			"🔔 Test Bildirimi: OE Watcher",
			"Discord webhook entegrasyonu başarıyla doğrulandı!",
			map[string]string{
				"Durum": "Aktif",
				"Zaman": time.Now().Format("15:04:05"),
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Discord test bildirimi başarıyla gönderildi!"})
		return
	} else if req.Type == "telegram" {
		token := strings.TrimSpace(req.BotToken)
		chatID := strings.TrimSpace(req.ChatID)
		if (token == "" || chatID == "") && s.app.GetConfig() != nil {
			token = s.app.GetConfig().TelegramBotToken
			chatID = s.app.GetConfig().TelegramChatID
		}
		if token == "" || chatID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Telegram Bot Token veya Chat ID belirtilmedi"})
			return
		}
		msg := "<b>🔔 Test Bildirimi: OE Watcher</b>\nTelegram bildirim entegrasyonu başarıyla doğrulandı!\n\n• <b>Durum:</b> Aktif\n• <b>Zaman:</b> " + time.Now().Format("15:04:05")
		err := notifier.SendTelegramNotification(token, chatID, msg)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Telegram test bildirimi başarıyla gönderildi!"})
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Geçersiz webhook türü ('discord' veya 'telegram' olmalı)"})
}

func foldTurkishStr(s string) string {
	s = strings.ReplaceAll(s, "İ", "i")
	s = strings.ReplaceAll(s, "I", "ı")
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ı", "i")
	s = strings.ReplaceAll(s, "ç", "c")
	s = strings.ReplaceAll(s, "ğ", "g")
	s = strings.ReplaceAll(s, "ö", "o")
	s = strings.ReplaceAll(s, "ş", "s")
	s = strings.ReplaceAll(s, "ü", "u")
	return strings.TrimSpace(s)
}



