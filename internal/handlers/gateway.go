// Package handlers содержит HTTP-обработчик API-шлюза.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"go-api-gateway/internal/models"
	"go-api-gateway/internal/ratelimit"
)

// UserStore — источник пользователей по API-ключу.
// *storage.Store реализует этот интерфейс.
type UserStore interface {
	GetUserByAPIKey(ctx context.Context, apiKey string) (*models.User, error)
}

// RequestProxy — отправка запроса в апстрим (Ollama).
// *proxy.OllamaProxy реализует этот интерфейс.
type RequestProxy interface {
	Proxy(w http.ResponseWriter, r *http.Request)
}

// Gateway — основной HTTP-обработчик API-шлюза.
// Цепочка обработки запроса: auth → rate limit → proxy.
type Gateway struct {
	store   UserStore
	limiter ratelimit.Limiter
	proxy   RequestProxy
}

// NewGateway собирает шлюз из хранилища пользователей, лимитера и прокси.
func NewGateway(store UserStore, limiter ratelimit.Limiter, p RequestProxy) *Gateway {
	return &Gateway{store: store, limiter: limiter, proxy: p}
}

// ServeHTTP реализует http.Handler: проверяет API-ключ, применяет
// rate limit и проксирует запрос в Ollama.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	apiKey := extractAPIKey(r)
	if apiKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "API key required"})
		return
	}

	user, err := g.store.GetUserByAPIKey(ctx, apiKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Internal error"})
		return
	}

	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Invalid API key"})
		return
	}

	allowed, count, err := g.limiter.Allow(ctx, apiKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Internal error"})
		return
	}

	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(g.limiter.Limit()))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, g.limiter.Limit()-count)))
	if !allowed {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "rate limit exceeded"})
		return
	}

	g.proxy.Proxy(w, r)
}

func extractAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("X-API-Key")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
