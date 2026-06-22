package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	// Endpoints de configuração
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/settings/test-notification", s.handleTestNotification)
	mux.HandleFunc("/api/chatwoot/resync", s.handleChatwootResync)

	// Endpoint raiz
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
		"version": "2.1.0",
		"status":  "running",
		"endpoints": map[string]string{
			"health":            "/api/health",
			"status":            "/api/status",
			"instances":         "/api/instances",
			"stats":             "/api/stats",
			"settings":          "/api/settings",
			"test-notification": "/api/settings/test-notification",
			"chatwoot-resync":   "/api/chatwoot/resync",
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

// handleSettings GET: retorna configurações, POST: salva configurações
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getSettings(w, r)
	case http.MethodPost:
		s.saveSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	telegramCfg := s.cfg.GetTelegram()
	templateCfg := s.cfg.GetMessageTemplate()
	evolutionCfg := s.cfg.GetEvolution()
	chatwootCfg := s.cfg.GetChatwoot()

	// Mascarar o token para segurança (mostrar apenas últimos 8 chars)
	maskedToken := telegramCfg.BotToken
	if len(maskedToken) > 8 {
		maskedToken = "***" + maskedToken[len(maskedToken)-8:]
	}

	// Mascarar a API Key (mostrar apenas últimos 6 chars)
	maskedAPIKey := evolutionCfg.APIKey
	if len(maskedAPIKey) > 6 {
		maskedAPIKey = "***" + maskedAPIKey[len(maskedAPIKey)-6:]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"evolution": map[string]interface{}{
			"api_url":        evolutionCfg.APIURL,
			"api_key":        maskedAPIKey,
			"api_key_set":    evolutionCfg.APIKey != "",
			"check_interval": evolutionCfg.CheckInterval,
		},
		"telegram": map[string]interface{}{
			"bot_token":     maskedToken,
			"bot_token_set": telegramCfg.BotToken != "",
			"chat_id":       telegramCfg.ChatID,
			"enabled":       telegramCfg.Enabled,
		},
		"message_template": templateCfg,
		"chatwoot": map[string]interface{}{
			"enabled":          chatwootCfg.Enabled,
			"interval_minutes": chatwootCfg.IntervalMinutes,
			"on_reconnect":     chatwootCfg.OnReconnect,
		},
	})
}

type settingsPayload struct {
	Evolution *struct {
		APIURL        string `json:"api_url"`
		APIKey        string `json:"api_key"`
		CheckInterval int    `json:"check_interval"`
	} `json:"evolution,omitempty"`
	Telegram *struct {
		BotToken string `json:"bot_token"`
		ChatID   string `json:"chat_id"`
		Enabled  bool   `json:"enabled"`
	} `json:"telegram,omitempty"`
	MessageTemplate *struct {
		Template string `json:"template"`
	} `json:"message_template,omitempty"`
	Chatwoot *struct {
		Enabled         bool `json:"enabled"`
		IntervalMinutes int  `json:"interval_minutes"`
		OnReconnect     bool `json:"on_reconnect"`
	} `json:"chatwoot,omitempty"`
}

func (s *Server) saveSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Erro ao ler body"})
		return
	}
	defer r.Body.Close()

	var payload settingsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	// Processar Evolution config
	var evolutionCfg *config.EvolutionConfig
	if payload.Evolution != nil {
		evolutionCfg = &config.EvolutionConfig{
			APIURL:        payload.Evolution.APIURL,
			APIKey:        payload.Evolution.APIKey,
			CheckInterval: payload.Evolution.CheckInterval,
		}
		// Se a API Key vier mascarada (***...), manter a atual
		if len(evolutionCfg.APIKey) > 3 && evolutionCfg.APIKey[:3] == "***" {
			evolutionCfg.APIKey = s.cfg.GetAPIKey()
		}
	}

	// Processar Telegram config
	var telegramCfg *config.TelegramConfig
	if payload.Telegram != nil {
		telegramCfg = &config.TelegramConfig{
			BotToken: payload.Telegram.BotToken,
			ChatID:   payload.Telegram.ChatID,
			Enabled:  payload.Telegram.Enabled,
		}
		// Se o token vier mascarado (***...), manter o atual
		if len(telegramCfg.BotToken) > 3 && telegramCfg.BotToken[:3] == "***" {
			currentTelegram := s.cfg.GetTelegram()
			telegramCfg.BotToken = currentTelegram.BotToken
		}
	}

	// Processar Template config
	var templateCfg *config.MessageTemplateConfig
	if payload.MessageTemplate != nil {
		templateCfg = &config.MessageTemplateConfig{
			Template: payload.MessageTemplate.Template,
		}
		// Se template vazio, usar padrão
		if templateCfg.Template == "" {
			templateCfg.Template = config.DefaultTemplate
		}
	}

	// Processar Chatwoot config
	var chatwootCfg *config.ChatwootReconnectConfig
	if payload.Chatwoot != nil {
		chatwootCfg = &config.ChatwootReconnectConfig{
			Enabled:         payload.Chatwoot.Enabled,
			IntervalMinutes: payload.Chatwoot.IntervalMinutes,
			OnReconnect:     payload.Chatwoot.OnReconnect,
		}
	}

	// Salvar
	if err := s.cfg.UpdateSettings(evolutionCfg, telegramCfg, templateCfg, chatwootCfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao salvar configurações"})
		return
	}

	log.Println("[INFO] Configurações atualizadas via API")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Configurações salvas com sucesso"})
}

// handleTestNotification envia uma notificação de teste
func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Erro ao ler body"})
		return
	}
	defer r.Body.Close()

	var payload settingsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
		return
	}

	var telegramCfg config.TelegramConfig
	if payload.Telegram != nil {
		telegramCfg = config.TelegramConfig{
			BotToken: payload.Telegram.BotToken,
			ChatID:   payload.Telegram.ChatID,
			Enabled:  payload.Telegram.Enabled,
		}
	} else {
		telegramCfg = s.cfg.GetTelegram()
	}

	// Se token mascarado, usar o salvo
	if len(telegramCfg.BotToken) > 3 && telegramCfg.BotToken[:3] == "***" {
		currentTelegram := s.cfg.GetTelegram()
		telegramCfg.BotToken = currentTelegram.BotToken
	}

	var templateCfg config.MessageTemplateConfig
	if payload.MessageTemplate != nil {
		templateCfg = config.MessageTemplateConfig{
			Template: payload.MessageTemplate.Template,
		}
	} else {
		templateCfg = s.cfg.GetMessageTemplate()
	}
	if templateCfg.Template == "" {
		templateCfg.Template = config.DefaultTemplate
	}

	// Enviar teste
	err = s.monitor.SendTestNotification(telegramCfg, templateCfg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Falha ao enviar: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Notificação de teste enviada!"})
}

// handleChatwootResync dispara uma re-sincronização manual da integração Chatwoot
func (s *Server) handleChatwootResync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chatwootCfg := s.cfg.GetChatwoot()
	if !chatwootCfg.Enabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Chatwoot Reconnector está desabilitado. Habilite e salve antes de re-sincronizar.",
		})
		return
	}

	log.Println("[INFO] Re-sincronização Chatwoot disparada manualmente via API")
	go s.monitor.RunChatwootResyncCycle()

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Re-sincronização do Chatwoot iniciada para todas as instâncias conectadas.",
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
