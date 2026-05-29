package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ad-marketing/evolution-monitor-go/internal/api"
	"github.com/ad-marketing/evolution-monitor-go/internal/config"
	"github.com/ad-marketing/evolution-monitor-go/internal/monitor"
	"github.com/ad-marketing/evolution-monitor-go/internal/server"
)

const version = "1.0.0"

func main() {
	// Carregar configurações
	cfg := config.Load()

	// Banner
	printBanner(cfg)

	// Criar cliente da API monitorada
	monitoredClient := api.NewClient(cfg.EvolutionAPIURL, cfg.EvolutionAPIKey)

	// Criar cliente da API de notificação (pode ser externo)
	var notificationClient *api.Client
	if cfg.NotificationAPIURL != "" && cfg.NotificationAPIURL != cfg.EvolutionAPIURL {
		notificationClient = api.NewClient(cfg.NotificationAPIURL, cfg.NotificationAPIKey)
	} else {
		notificationClient = monitoredClient
	}

	// Criar o monitor
	mon := monitor.New(cfg, monitoredClient, notificationClient)

	// Iniciar servidor HTTP (para frontend futuro)
	srv := server.New(cfg, mon)
	go srv.Start()

	// Executar primeiro ciclo imediatamente
	mon.RunCycle()

	// Agendar ciclos
	ticker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Millisecond)
	defer ticker.Stop()

	log.Printf("[INFO] Monitor ativo. Próxima verificação em %ds.", cfg.CheckInterval/1000)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			mon.RunCycle()
		case <-quit:
			log.Println("[INFO] Encerrando monitor...")
			srv.Stop()
			return
		}
	}
}

func printBanner(cfg *config.Config) {
	isExternal := cfg.NotificationAPIURL != "" && cfg.NotificationAPIURL != cfg.EvolutionAPIURL
	notifVia := "Mesma API monitorada"
	if isExternal {
		notifVia = cfg.NotificationAPIURL + " (EXTERNA)"
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║   MONITOR DE INSTÂNCIAS - EVOLUTION API v2.4.x              ║")
	fmt.Printf("║   Versão %s (Go)                                         ║\n", version)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ API Monitorada: %-44s║\n", cfg.EvolutionAPIURL)
	fmt.Printf("║ Intervalo:     %-44s║\n", fmt.Sprintf("%ds", cfg.CheckInterval/1000))
	fmt.Printf("║ Max Retry:     %-44s║\n", fmt.Sprintf("%d tentativas", cfg.MaxRestartAttempts))
	fmt.Printf("║ Notificar:     %-44s║\n", fmt.Sprintf("%v", cfg.NotificationEnabled))
	fmt.Printf("║ Admin:         %-44s║\n", cfg.NotificationAdminNumber)
	fmt.Printf("║ Notif. via:    %-44s║\n", notifVia)
	fmt.Printf("║ Instância TX:  %-44s║\n", cfg.NotificationSenderInstance)
	fmt.Printf("║ API Server:    %-44s║\n", fmt.Sprintf(":%d", cfg.ServerPort))
	fmt.Printf("║ Ignoradas:     %-44s║\n", cfg.IgnoreInstancesStr())
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
