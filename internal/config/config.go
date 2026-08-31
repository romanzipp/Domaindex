package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppHost             string
	AppPort             string
	AppSecret           string
	RegistrationEnabled bool

	DBDriver string
	DBDSN    string

	NotificationURLs []string

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
		AppHost:             getEnv("APP_HOST", "0.0.0.0"),
		AppPort:             getEnv("APP_PORT", "8080"),
		AppSecret:           getEnv("APP_SECRET", "change-me-in-production"),
		RegistrationEnabled: getEnvBool("REGISTRATION_ENABLED", true),

		DBDriver: getEnv("DB_DRIVER", "sqlite"),
		DBDSN:    getEnv("DB_DSN", "data/domaindex.db"),

		NotificationURLs: getEnvList("NOTIFICATION_URLS"),

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

func getEnvList(key string) []string {
	var out []string

	for _, v := range strings.Split(os.Getenv(key), ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}

	return out
}
