package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"medya-filter/pkg/app"
	"medya-filter/pkg/models"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	app  *app.App
	port int
}

func NewServer(app *app.App, port int) *Server {
	if port <= 0 {
		port = 8080
	}
	return &Server{
		app:  app,
		port: port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/skip", s.handleSkip)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/persons", s.handlePersons)

	// Embedded Static files
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("static subfs failed: %w", err)
	}
	fileServer := http.FileServer(http.FS(staticSub))
	mux.Handle("/", fileServer)

	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, mux)
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
	AutoSkip         *bool     `json:"auto_skip,omitempty"`
	EnableDialog     *bool     `json:"enable_dialog,omitempty"`
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
