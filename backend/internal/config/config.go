package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL                 string
	JWTSecret                   string
	BridgeKey                   string
	N8NPendingActionWebhookURL  string
	AppUsers                    string
	Port                        string
	CaseExpiryDays              int
	VAPIDPublicKey              string
	VAPIDPrivateKey             string
	VAPIDSubject                string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:                getEnv("DATABASE_URL", ""),
		JWTSecret:                  getEnv("JWT_SECRET", "dev-secret-change-me"),
		BridgeKey:                  getEnv("BRIDGE_KEY", "bridge-local-dev-key-2026"),
		N8NPendingActionWebhookURL: getEnv("N8N_PENDING_ACTION_WEBHOOK_URL", ""),
		AppUsers:                   getEnv("APP_USERS", ""),
		Port:                       getEnv("PORT", "8080"),
		CaseExpiryDays:             getEnvInt("CASE_EXPIRY_DAYS", 1),
		VAPIDPublicKey:             getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:            getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:               getEnv("VAPID_SUBJECT", "mailto:admin@example.com"),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}
