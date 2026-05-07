package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	APIID       int
	APIHash     string
	SessionFile string
	BotToken    string
	Proxy       string
	HTTPPort    string
}

func Load() *Config {
	_ = godotenv.Load()

	apiIDStr := getEnv("TELEGRAM_API_ID", "")
	if apiIDStr == "" {
		log.Fatal("TELEGRAM_API_ID is required")
	}
	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		log.Fatalf("TELEGRAM_API_ID must be a number, got '%s'", apiIDStr)
	}

	apiHash := getEnv("TELEGRAM_API_HASH", "")
	if apiHash == "" {
		log.Fatal("TELEGRAM_API_HASH is required")
	}

	return &Config{
		APIID:       apiID,
		APIHash:     apiHash,
		SessionFile: getEnv("TELEGRAM_SESSION_FILE", "telegram.session"),
		BotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		Proxy:       getEnv("TELEGRAM_PROXY", ""),
		HTTPPort:    getEnv("HTTP_PORT", "8765"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}