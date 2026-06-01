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

const version = "2.0.0"

func main() {
	// Carregar configurações
	cfg := config.Load()

	// Banner
	printBanner(cfg)

	// Criar cliente da API monitorada
	monitoredClient := api.NewClient(cfg.EvolutionAPIURL, cfg.EvolutionAPIKey)

	// Criar o monitor (agora usa Telegram para notificações)
	mon := monitor.New(cfg, monitoredClient)

	// Iniciar servidor HTTP
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
	telegramCfg := cfg.GetTelegram()
	telegramStatus := "Desabilitado"
	if telegramCfg.Enabled && telegramCfg.BotToken != "" {
		telegramStatus = "Ativo (Chat: " + telegramCfg.ChatID + ")"
	} else if telegramCfg.Enabled {
		telegramStatus = "Ativo (Token não configurado)"
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║   MONITOR DE INSTÂNCIAS - EVOLUTION API v2.4.x              ║")
	fmt.Printf("║   Versão %s (Go) — Notificação via Telegram            ║\n", version)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ API Monitorada: %-44s║\n", cfg.EvolutionAPIURL)
	fmt.Printf("║ Intervalo:     %-44s║\n", fmt.Sprintf("%ds", cfg.CheckInterval/1000))
	fmt.Printf("║ Max Retry:     %-44s║\n", fmt.Sprintf("%d tentativas", cfg.MaxRestartAttempts))
	fmt.Printf("║ Telegram:      %-44s║\n", telegramStatus)
	fmt.Printf("║ API Server:    %-44s║\n", fmt.Sprintf(":%d", cfg.ServerPort))
	fmt.Printf("║ Ignoradas:     %-44s║\n", cfg.IgnoreInstancesStr())
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
