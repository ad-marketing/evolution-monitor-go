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

const version = "2.2.2"

func main() {
	// Carregar configurações
	cfg := config.Load()

	// Banner
	printBanner(cfg)

	// Criar cliente da API monitorada
	monitoredClient := api.NewClient(cfg.EvolutionAPIURL, cfg.EvolutionAPIKey)

	// Criar o monitor (usa Telegram para notificações e inclui Chatwoot Reconnector)
	mon := monitor.New(cfg, monitoredClient)

	// Iniciar servidor HTTP
	srv := server.New(cfg, mon)
	go srv.Start()

	// Executar primeiro ciclo imediatamente
	mon.RunCycle()

	// Agendar ciclos de monitoramento
	ticker := time.NewTicker(time.Duration(cfg.CheckInterval) * time.Millisecond)
	defer ticker.Stop()

	log.Printf("[INFO] Monitor ativo. Próxima verificação em %ds.", cfg.CheckInterval/1000)

	// Ticker do Chatwoot Reconnector (modo A — periódico)
	// O intervalo é recalculado dinamicamente a cada disparo para refletir
	// alterações feitas pelo dashboard sem necessidade de reiniciar o serviço.
	chatwootCfg := cfg.GetChatwoot()
	chatwootInterval := chatwootIntervalDuration(chatwootCfg.IntervalMinutes)
	chatwootTicker := time.NewTicker(chatwootInterval)
	defer chatwootTicker.Stop()

	if chatwootCfg.Enabled {
		log.Printf("[INFO] Chatwoot Reconnector ativo (modo periódico: %dmin, on-reconnect: %v).",
			chatwootCfg.IntervalMinutes, chatwootCfg.OnReconnect)
		// Re-sincronização inicial logo após o boot
		go mon.RunChatwootResyncCycle()
	} else {
		log.Println("[INFO] Chatwoot Reconnector desabilitado.")
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			mon.RunCycle()
		case <-chatwootTicker.C:
			cw := cfg.GetChatwoot()
			if cw.Enabled {
				mon.RunChatwootResyncCycle()
			}
			// Reajustar o intervalo dinamicamente
			newInterval := chatwootIntervalDuration(cw.IntervalMinutes)
			if newInterval != chatwootInterval {
				chatwootInterval = newInterval
				chatwootTicker.Reset(chatwootInterval)
				log.Printf("[INFO] Intervalo do Chatwoot Reconnector ajustado para %dmin.", cw.IntervalMinutes)
			}
		case <-quit:
			log.Println("[INFO] Encerrando monitor...")
			srv.Stop()
			return
		}
	}
}

// chatwootIntervalDuration converte minutos em duração, aplicando um piso seguro
func chatwootIntervalDuration(minutes int) time.Duration {
	if minutes < 1 {
		minutes = 30
	}
	return time.Duration(minutes) * time.Minute
}

func printBanner(cfg *config.Config) {
	telegramCfg := cfg.GetTelegram()
	telegramStatus := "Desabilitado"
	if telegramCfg.Enabled && telegramCfg.BotToken != "" {
		telegramStatus = "Ativo (Chat: " + telegramCfg.ChatID + ")"
	} else if telegramCfg.Enabled {
		telegramStatus = "Ativo (Token não configurado)"
	}

	chatwootCfg := cfg.GetChatwoot()
	chatwootStatus := "Desabilitado"
	if chatwootCfg.Enabled {
		chatwootStatus = fmt.Sprintf("Ativo (%dmin)", chatwootCfg.IntervalMinutes)
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
	fmt.Printf("║ Chatwoot:      %-44s║\n", chatwootStatus)
	fmt.Printf("║ API Server:    %-44s║\n", fmt.Sprintf(":%d", cfg.ServerPort))
	fmt.Printf("║ Ignoradas:     %-44s║\n", cfg.IgnoreInstancesStr())
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
