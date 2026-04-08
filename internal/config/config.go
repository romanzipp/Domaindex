package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppHost  string
	AppPort  string
	AppSecret string

	DBDriver string
	DBDSN    string

	AppriseURL string
	AppriseKey string

	WhoisRefreshInterval time.Duration
}

func Load() *Config {
	whoisInterval := 6 * time.Hour
	if v := os.Getenv("WHOIS_REFRESH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			whoisInterval = d
		}
	}

	return &Config{
		AppHost:   getEnv("APP_HOST", "0.0.0.0"),
		AppPort:   getEnv("APP_PORT", "8080"),
		AppSecret: getEnv("APP_SECRET", "change-me-in-production"),

		DBDriver: getEnv("DB_DRIVER", "sqlite"),
		DBDSN:    getEnv("DB_DSN", "data/domain-manager.db"),

		AppriseURL: os.Getenv("APPRISE_URL"),
		AppriseKey: os.Getenv("APPRISE_KEY"),

		WhoisRefreshInterval: whoisInterval,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
