package storage

import (
	"context"
	"os"
	"testing"
)

// Integration-тест: требует живую PostgreSQL.
// Запуск: TEST_DB_DSN="postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable" go test ./internal/storage/
func TestGetUserByAPIKey(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set, skipping postgres integration test")
	}

	s, err := NewStore(dsn)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	const apiKey = "integration-test-key"

	// самодостаточность: кладём тестовую строку, потом подчищаем
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO users (api_key, name) VALUES ($1, $2) ON CONFLICT (api_key) DO NOTHING`,
		apiKey, "integration test"); err != nil {
		t.Fatalf("insert test row: %v", err)
	}
	t.Cleanup(func() {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM users WHERE api_key = $1`, apiKey)
	})

	t.Run("existing key", func(t *testing.T) {
		u, err := s.GetUserByAPIKey(ctx, apiKey)
		if err != nil {
			t.Fatal(err)
		}
		if u == nil {
			t.Fatal("want user, got nil")
		}
		if u.APIKey != apiKey {
			t.Errorf("APIKey = %q, want %q", u.APIKey, apiKey)
		}
		if u.Name != "integration test" {
			t.Errorf("Name = %q, want %q", u.Name, "integration test")
		}
		if u.ID == 0 {
			t.Error("ID must be filled from database")
		}
		if u.CreatedAt.IsZero() {
			t.Error("CreatedAt must be filled from database")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		u, err := s.GetUserByAPIKey(ctx, "definitely-not-a-key")
		if err != nil {
			t.Fatal(err)
		}
		if u != nil {
			t.Errorf("want nil for missing key, got %+v", u)
		}
	})
}

func TestNewStoreBadDSN(t *testing.T) {
	if _, err := NewStore("not-a-dsn"); err == nil {
		t.Error("NewStore must fail on invalid DSN")
	}
}
