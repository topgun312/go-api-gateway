// Package config загружает настройки приложения из env-переменных и файла .env.
package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config содержит настройки приложения, загружаемые из env-переменных.
type Config struct {
	ServerPort  string
	DBDSN       string
	RedisAddr   string
	OllamaURL   string
	MaxRequests int
}

// Load читает конфигурацию из файла .env (если он есть) и env-переменных.
// Для неуказанных переменных используются значения по умолчанию.
func Load() *Config {
	// Подгружаем .env (если он есть) — os.Getenv по умолчанию не читает файлы.
	// Ошибку игнорируем: .env необязателен (в Docker значения приходят из compose).
	_ = godotenv.Load()
	return &Config{
		ServerPort:  getenv("SERVER_PORT", ":8080"),
		DBDSN:       getenv("DB_DSN", "postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable"),
		RedisAddr:   getenv("REDIS_ADDR", "localhost:6379"),
		OllamaURL:   getenv("OLLAMA_URL", "http://localhost:11434"),
		MaxRequests: getenvInt("MAX_REQUESTS", 5),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
