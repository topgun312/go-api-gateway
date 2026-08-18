// Package proxy проксирует HTTP-запросы клиента в Ollama.
package proxy

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// OllamaProxy — HTTP-клиент, пересылающий запросы в Ollama.
type OllamaProxy struct {
	targetURL string
	client    *http.Client
}

// NewOllamaProxy создаёт прокси до targetURL (например,
// http://localhost:11434/api/generate) с таймаутом запроса 60 секунд.
func NewOllamaProxy(targetURL string) *OllamaProxy {
	return &OllamaProxy{
		targetURL: targetURL,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Proxy читает тело запроса r и отправляет POST на targetURL,
// пробрасывая статус и тело ответа Ollama клиенту как есть.
func (p *OllamaProxy) Proxy(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	outReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		p.targetURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, `{"error": "bad request"}`, http.StatusBadRequest)
		return
	}
	outReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(outReq)
	if err != nil {
		http.Error(w, `{"error":"upstream unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
