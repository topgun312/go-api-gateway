package proxy

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type OllamaProxy struct {
	targetURL string
	client    *http.Client
}

func NewOllamaProxy(targetURL string) *OllamaProxy {
	return &OllamaProxy{
		targetURL: targetURL,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

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
