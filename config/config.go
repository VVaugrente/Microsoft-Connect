package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	Port         string
	GeminiAPIKey string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	return &Config{
		ClientID:     getEnv("CLIENT_ID", ""),
		ClientSecret: getEnv("CLIENT_SECRET", ""),
		TenantID:     getEnv("TENANT_ID", ""),
		Port:         getEnv("PORT", "10000"),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
