package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config contém todas as configurações do monitor
type Config struct {
	// API Monitorada
	EvolutionAPIURL string `json:"evolution_api_url"`
	EvolutionAPIKey string `json:"evolution_api_key"`

	// Intervalo e tentativas
	CheckInterval      int `json:"check_interval"`       // em milissegundos
	MaxRestartAttempts int `json:"max_restart_attempts"`
	WaitAfterRestart   int `json:"wait_after_restart"`   // em milissegundos

	// Telegram
	Telegram TelegramConfig `json:"telegram"`

	// Template de mensagem
	MessageTemplate MessageTemplateConfig `json:"message_template"`

	// Avançado
	IgnoreInstances []string `json:"ignore_instances"`
	Verbose         bool     `json:"verbose"`

	// Servidor HTTP
	ServerPort int `json:"server_port"`

	// Mutex para acesso concorrente
	mu sync.RWMutex
}

// TelegramConfig configurações do bot Telegram
type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
	Enabled  bool   `json:"enabled"`
}

// MessageTemplateConfig template da mensagem de notificação
type MessageTemplateConfig struct {
	Template string `json:"template"`
}

const DefaultTemplate = `🚨 *Instância Desconectada*

📛 *Instância:* {{instance_name}}
📊 *Status:* {{status}}
🔄 *Tentativas:* {{attempts}}/{{max_attempts}}
🕐 *Horário:* {{timestamp}}
🖥️ *Servidor:* {{server_url}}

⚠️ A reconexão automática falhou. Verifique manualmente.`

const settingsFile = "/data/settings.json"

// Load carrega as configurações a partir das variáveis de ambiente
func Load() *Config {
	cfg := &Config{
		EvolutionAPIURL: getEnv("EVOLUTION_API_URL", "http://localhost:8080"),
		EvolutionAPIKey: getEnv("EVOLUTION_API_KEY", ""),

		CheckInterval:      getEnvInt("CHECK_INTERVAL", 60000),
		MaxRestartAttempts: getEnvInt("MAX_RESTART_ATTEMPTS", 3),
		WaitAfterRestart:   getEnvInt("WAIT_AFTER_RESTART", 10000),

		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
			ChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
			Enabled:  getEnvBool("TELEGRAM_ENABLED", true),
		},

		MessageTemplate: MessageTemplateConfig{
			Template: getEnv("NOTIFICATION_TEMPLATE", DefaultTemplate),
		},

		Verbose:    getEnvBool("VERBOSE", false),
		ServerPort: getEnvInt("SERVER_PORT", 3500),
	}

	// Parse ignore instances
	ignoreStr := getEnv("IGNORE_INSTANCES", "")
	if ignoreStr != "" {
		cfg.IgnoreInstances = strings.Split(ignoreStr, ",")
		for i := range cfg.IgnoreInstances {
			cfg.IgnoreInstances[i] = strings.TrimSpace(cfg.IgnoreInstances[i])
		}
	}

	// Tentar carregar configurações salvas (sobrescreve env vars para telegram/template)
	cfg.loadFromFile()

	return cfg
}

// GetTelegram retorna a config do Telegram de forma thread-safe
func (c *Config) GetTelegram() TelegramConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Telegram
}

// GetMessageTemplate retorna o template de forma thread-safe
func (c *Config) GetMessageTemplate() MessageTemplateConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.MessageTemplate
}

// UpdateSettings atualiza as configurações de Telegram e template
func (c *Config) UpdateSettings(telegram TelegramConfig, template MessageTemplateConfig) error {
	c.mu.Lock()
	c.Telegram = telegram
	c.MessageTemplate = template
	c.mu.Unlock()

	return c.saveToFile()
}

// IgnoreInstancesStr retorna as instâncias ignoradas como string
func (c *Config) IgnoreInstancesStr() string {
	if len(c.IgnoreInstances) == 0 {
		return "Nenhuma"
	}
	return strings.Join(c.IgnoreInstances, ", ")
}

// IsIgnored verifica se uma instância está na lista de ignorados
func (c *Config) IsIgnored(name string) bool {
	for _, ignored := range c.IgnoreInstances {
		if ignored == name {
			return true
		}
	}
	return false
}

// Estrutura para persistência em arquivo
type savedSettings struct {
	Telegram        TelegramConfig        `json:"telegram"`
	MessageTemplate MessageTemplateConfig `json:"message_template"`
}

func (c *Config) loadFromFile() {
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return // Arquivo não existe ainda, usa env vars
	}

	var saved savedSettings
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}

	// Sobrescreve apenas se o arquivo tiver valores
	if saved.Telegram.BotToken != "" {
		c.Telegram = saved.Telegram
	}
	if saved.MessageTemplate.Template != "" {
		c.MessageTemplate = saved.MessageTemplate
	}
}

func (c *Config) saveToFile() error {
	// Criar diretório se não existir
	os.MkdirAll("/data", 0755)

	c.mu.RLock()
	saved := savedSettings{
		Telegram:        c.Telegram,
		MessageTemplate: c.MessageTemplate,
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, data, 0644)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}
