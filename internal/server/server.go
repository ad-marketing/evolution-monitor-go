package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ad-marketing/evolution-monitor-go/internal/config"
	"github.com/ad-marketing/evolution-monitor-go/internal/monitor"
)

// Server é o servidor HTTP para a API do monitor
type Server struct {
	cfg     *config.Config
	monitor *monitor.Monitor
	srv     *http.Server
}

// New cria um novo servidor HTTP
func New(cfg *config.Config, mon *monitor.Monitor) *Server {
	return &Server{
		cfg:     cfg,
		monitor: mon,
	}
}

// Start inicia o servidor HTTP
func (s *Server) Start() {
	mux := http.NewServeMux()

	// Endpoints da API
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/instances", s.handleInstances)
	mux.HandleFunc("/api/stats", s.handleStats)

	// Endpoint raiz (futuro frontend)
	mux.HandleFunc("/", s.handleRoot)

	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.ServerPort),
		Handler:      corsMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("[INFO] Servidor HTTP iniciado na porta %d", s.cfg.ServerPort)

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("[ERROR] Erro no servidor HTTP: %v", err)
	}
}

// Stop encerra o servidor HTTP graciosamente
func (s *Server) Stop() {
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(ctx)
	}
}

// corsMiddleware adiciona headers CORS para o frontend
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":    "Evolution Monitor",
		"version": "1.0.0",
		"status":  "running",
		"endpoints": map[string]string{
			"health":    "/api/health",
			"status":    "/api/status",
			"instances": "/api/instances",
			"stats":     "/api/stats",
		},
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	lastCycle := s.monitor.GetLastCycle()
	if lastCycle == nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "waiting",
			"message": "Nenhum ciclo executado ainda",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "active",
		"last_check":  lastCycle.Timestamp,
		"total":       lastCycle.Total,
		"ok":          lastCycle.Ok,
		"reconnected": lastCycle.Reconnected,
		"failed":      lastCycle.Failed,
		"ignored":     lastCycle.Ignored,
	})
}

func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	lastCycle := s.monitor.GetLastCycle()
	if lastCycle == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	writeJSON(w, http.StatusOK, lastCycle.Instances)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.monitor.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
