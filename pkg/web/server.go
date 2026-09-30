package web

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"medya-filter/pkg/app"
	"medya-filter/pkg/config"
	"medya-filter/pkg/models"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	app         *app.App
	bindAddress string
	port        int
	password    string
	sessionMu   sync.RWMutex
	sessions    map[string]time.Time
}

func NewServer(app *app.App, port int) *Server {
	bindAddr := "127.0.0.1"
	password := ""
	if cfg := app.GetConfig(); cfg != nil {
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

	return &Server{
		app:         app,
		bindAddress: bindAddr,
		port:        port,
		password:    password,
		sessions:    make(map[string]time.Time),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Public Auth endpoints
	mux.HandleFunc("/api/auth-status", s.handleAuthStatus)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/logout", s.handleLogout)

	// Protected API endpoints
	mux.HandleFunc("/api/status", s.authMiddleware(s.handleStatus))
	mux.HandleFunc("/api/skip", s.authMiddleware(s.handleSkip))
	mux.HandleFunc("/api/settings", s.authMiddleware(s.handleSettings))
	mux.HandleFunc("/api/config", s.authMiddleware(s.handleConfig))
	mux.HandleFunc("/api/snooze", s.authMiddleware(s.handleSnooze))
	mux.HandleFunc("/api/player/play-pause", s.authMiddleware(s.handlePlayPause))
	mux.HandleFunc("/api/sync", s.authMiddleware(s.handleSync))
	mux.HandleFunc("/api/persons", s.authMiddleware(s.handlePersons))
	mux.HandleFunc("/api/rules", s.authMiddleware(s.handleRules))
	mux.HandleFunc("/api/rules/export", s.authMiddleware(s.handleRulesExport))
	mux.HandleFunc("/api/rules/import", s.authMiddleware(s.handleRulesImport))
	mux.HandleFunc("/api/rules/person", s.authMiddleware(s.handleRulePerson))
	mux.HandleFunc("/api/rules/alias", s.authMiddleware(s.handleRuleAlias))
	mux.HandleFunc("/api/rules/exclude", s.authMiddleware(s.handleRuleExclude))

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
	AutoSkip         *bool    `json:"auto_skip,omitempty"`
	EnableDialog     *bool    `json:"enable_dialog,omitempty"`
	FuzzyMatching    *bool    `json:"fuzzy_matching,omitempty"`
	FuzzyThreshold   *float64 `json:"fuzzy_threshold,omitempty"`
	ActiveCategories []string `json:"active_categories,omitempty"`
	ActiveTopics     []string `json:"active_topics,omitempty"`
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

		if category != "" && !strings.EqualFold(category, "HEPSİ") {
			hasCat := false
			for _, t := range p.Tags {
				if strings.EqualFold(t, category) {
					hasCat = true
					break
				}
			}
			if !hasCat {
				continue
			}
		}

		if topic != "" && !strings.EqualFold(topic, "TÜM KONULAR") {
			hasTopic := false
			for _, top := range p.Topics {
				if strings.EqualFold(top, topic) {
					hasTopic = true
					break
				}
			}
			if !hasTopic {
				continue
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
