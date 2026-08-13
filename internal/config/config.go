package config

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds all server configuration loaded from environment variables.
type Config struct {
	Profile   string
	LogLevel  string
	LogFormat string
	Port      string

	// MinuteMail api-gateway base URL (in-cluster service DNS).
	APIBase string
}

// Load reads environment variables and constructs the runtime configuration.
func Load() *Config {
	profile := strings.ToLower(strings.TrimSpace(getEnv("PROFILE", "dev")))

	cfg := &Config{
		Profile:   profile,
		LogLevel:  getEnv("LOG_LEVEL", "warn"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
		Port:      getEnv("PORT", "8080"),
		APIBase:   strings.TrimRight(getEnv("API_BASE", "http://mm-api-gateway:80"), "/"),
	}

	slog.Default().Debug("loaded config",
		"api_base", cfg.APIBase,
		"port", cfg.Port,
		"profile", cfg.Profile,
	)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
