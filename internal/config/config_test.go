package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"SERVER_PORT", "DB_DSN", "REDIS_ADDR", "OLLAMA_URL", "MAX_REQUESTS"} {
		t.Setenv(k, "")
	}

	cfg := Load()

	if cfg.ServerPort != ":8080" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, ":8080")
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "localhost:6379")
	}
	if cfg.OllamaURL != "http://localhost:11434" {
		t.Errorf("OllamaURL = %q, want %q", cfg.OllamaURL, "http://localhost:11434")
	}
	if cfg.MaxRequests != 5 {
		t.Errorf("MaxRequests = %d, want 5", cfg.MaxRequests)
	}
	if cfg.DBDSN == "" {
		t.Error("DBDSN must not be empty")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("SERVER_PORT", ":9999")
	t.Setenv("REDIS_ADDR", "redis.example:6379")
	t.Setenv("OLLAMA_URL", "http://ollama.example:11434")
	t.Setenv("MAX_REQUESTS", "10")

	cfg := Load()

	if cfg.ServerPort != ":9999" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, ":9999")
	}
	if cfg.RedisAddr != "redis.example:6379" {
		t.Errorf("RedisAddr = %q, want %q", cfg.RedisAddr, "redis.example:6379")
	}
	if cfg.OllamaURL != "http://ollama.example:11434" {
		t.Errorf("OllamaURL = %q, want %q", cfg.OllamaURL, "http://ollama.example:11434")
	}
	if cfg.MaxRequests != 10 {
		t.Errorf("MaxRequests = %d, want 10", cfg.MaxRequests)
	}
}

func TestLoadInvalidMaxRequests(t *testing.T) {
	t.Setenv("MAX_REQUESTS", "not-a-number")

	cfg := Load()

	if cfg.MaxRequests != 5 {
		t.Errorf("MaxRequests = %d, want default 5 for invalid value", cfg.MaxRequests)
	}
}
