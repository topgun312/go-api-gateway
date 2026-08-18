package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-api-gateway/internal/models"
)

type fakeStore struct {
	user *models.User
	err  error
}

func (f *fakeStore) GetUserByAPIKey(_ context.Context, _ string) (*models.User, error) {
	return f.user, f.err
}

type fakeLimiter struct {
	allowed bool
	count   int
	limit   int
	err     error
}

func (f *fakeLimiter) Allow(_ context.Context, _ string) (bool, int, error) {
	return f.allowed, f.count, f.err
}

func (f *fakeLimiter) Limit() int { return f.limit }

type fakeProxy struct {
	called bool
	status int
	body   string
}

func (f *fakeProxy) Proxy(w http.ResponseWriter, _ *http.Request) {
	f.called = true
	w.WriteHeader(f.status)
	_, _ = w.Write([]byte(f.body))
}

func doRequest(g *Gateway, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(`{"model":"x"}`))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	g.ServeHTTP(rec, req)
	return rec
}

func validKeyHeaders() map[string]string {
	return map[string]string{"Authorization": "Bearer test-key-123"}
}

func TestGatewayMissingAPIKey(t *testing.T) {
	g := NewGateway(&fakeStore{}, &fakeLimiter{}, &fakeProxy{})

	rec := doRequest(g, nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rec.Body.String(), "API key required") {
		t.Errorf("body = %q, want it to contain %q", rec.Body.String(), "API key required")
	}
}

func TestGatewayInvalidAPIKey(t *testing.T) {
	g := NewGateway(&fakeStore{user: nil}, &fakeLimiter{}, &fakeProxy{})

	rec := doRequest(g, validKeyHeaders())

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rec.Body.String(), "Invalid API key") {
		t.Errorf("body = %q, want it to contain %q", rec.Body.String(), "Invalid API key")
	}
}

func TestGatewayStoreError(t *testing.T) {
	g := NewGateway(&fakeStore{err: errors.New("db down")}, &fakeLimiter{}, &fakeProxy{})

	rec := doRequest(g, validKeyHeaders())

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestGatewayRateLimitExceeded(t *testing.T) {
	limiter := &fakeLimiter{allowed: false, count: 5, limit: 5}
	g := NewGateway(&fakeStore{user: &models.User{APIKey: "test-key-123"}}, limiter, &fakeProxy{})

	rec := doRequest(g, validKeyHeaders())

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if got := rec.Header().Get("X-RateLimit-Limit"); got != "5" {
		t.Errorf("X-RateLimit-Limit = %q, want %q", got, "5")
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("X-RateLimit-Remaining = %q, want %q", got, "0")
	}
	if got := rec.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, want %q", got, "60")
	}
}

func TestGatewayRateLimitError(t *testing.T) {
	g := NewGateway(&fakeStore{user: &models.User{APIKey: "test-key-123"}},
		&fakeLimiter{err: errors.New("redis down")}, &fakeProxy{})

	rec := doRequest(g, validKeyHeaders())

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestGatewaySuccess(t *testing.T) {
	p := &fakeProxy{status: http.StatusOK, body: `{"ok":true}`}
	limiter := &fakeLimiter{allowed: true, count: 2, limit: 5}
	g := NewGateway(&fakeStore{user: &models.User{APIKey: "test-key-123"}}, limiter, p)

	rec := doRequest(g, validKeyHeaders())

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !p.called {
		t.Error("proxy must be called on success path")
	}
	if got := rec.Body.String(); got != `{"ok":true}` {
		t.Errorf("body = %q, want %q", got, `{"ok":true}`)
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "3" {
		t.Errorf("X-RateLimit-Remaining = %q, want %q", got, "3")
	}
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"bearer", map[string]string{"Authorization": "Bearer abc"}, "abc"},
		{"lowercase bearer", map[string]string{"Authorization": "bearer abc"}, ""},
		{"x-api-key", map[string]string{"X-API-Key": "xyz"}, "xyz"},
		{"none", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if got := extractAPIKey(req); got != tt.want {
				t.Errorf("extractAPIKey = %q, want %q", got, tt.want)
			}
		})
	}
}
