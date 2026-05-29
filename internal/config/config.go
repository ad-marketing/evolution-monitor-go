package config

import (
	"os"
	"strconv"
	"strings"
)

// Config contém todas as configurações do monitor
type Config struct {
	// API Monitorada
	EvolutionAPIURL string
	EvolutionAPIKey string

	// Intervalo e tentativas
	CheckInterval      int // em milissegundos
	MaxRestartAttempts int
	WaitAfterRestart   int // em milissegundos

	// Notificação
	NotificationAPIURL          string
	NotificationAPIKey          string
	NotificationSenderInstance  string
	NotificationAdminNumber     string
	NotificationEnabled         bool

	// Avançado
	IgnoreInstances []string
	Verbose         bool

	// Servidor HTTP (para frontend futuro)
	ServerPort int
}

// Load carrega as configurações a partir das variáveis de ambiente
func Load() *Config {
	cfg := &Config{
		EvolutionAPIURL: getEnv("EVOLUTION_API_URL", "http://localhost:8080"),
		EvolutionAPIKey: getEnv("EVOLUTION_API_KEY", ""),

		CheckInterval:      getEnvInt("CHECK_INTERVAL", 60000),
		MaxRestartAttempts: getEnvInt("MAX_RESTART_ATTEMPTS", 3),
		WaitAfterRestart:   getEnvInt("WAIT_AFTER_RESTART", 10000),

		NotificationAPIURL:         getEnv("NOTIFICATION_API_URL", ""),
		NotificationAPIKey:         getEnv("NOTIFICATION_API_KEY", ""),
		NotificationSenderInstance: getEnv("NOTIFICATION_SENDER_INSTANCE", ""),
		NotificationAdminNumber:    getEnv("NOTIFICATION_ADMIN_NUMBER", ""),
		NotificationEnabled:        getEnvBool("NOTIFICATION_ENABLED", true),

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

	// Se não definiu API de notificação, usa a mesma monitorada
	if cfg.NotificationAPIURL == "" {
		cfg.NotificationAPIURL = cfg.EvolutionAPIURL
	}
	if cfg.NotificationAPIKey == "" {
		cfg.NotificationAPIKey = cfg.EvolutionAPIKey
	}

	return cfg
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
