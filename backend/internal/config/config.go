package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DBPath         string
	JWTSecret      string
	EncryptionKey  string // 32-byte key (base64 or raw) used to encrypt provider credentials at rest
	SyncIntervalMi int    // minutes between scheduled cost syncs
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8080"),
		DBPath:         getEnv("DB_PATH", "data/cloudcost.db"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret-change-me"),
		EncryptionKey:  getEnv("ENCRYPTION_KEY", "dev-encryption-key-32-bytes-long"),
		SyncIntervalMi: getEnvInt("SYNC_INTERVAL_MINUTES", 60),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
