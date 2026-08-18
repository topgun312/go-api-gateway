package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxySuccess(t *testing.T) {
	var gotPath, gotContentType, gotBody string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	p := NewOllamaProxy(upstream.URL + "/api/generate")

	req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(`{"model":"x"}`))
	rec := httptest.NewRecorder()
	p.Proxy(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != `{"ok":true}` {
		t.Errorf("body = %q, want %q", got, `{"ok":true}`)
	}
	if gotContentType != "application/json" {
		t.Errorf("upstream Content-Type = %q, want %q", gotContentType, "application/json")
	}
	if gotPath != "/api/generate" {
		t.Errorf("upstream path = %q, want %q", gotPath, "/api/generate")
	}
	if !strings.Contains(gotBody, `"model":"x"`) {
		t.Errorf("upstream body = %q, want it to contain %q", gotBody, `"model":"x"`)
	}
}

func TestProxyUpstreamStatusPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer upstream.Close()

	p := NewOllamaProxy(upstream.URL)

	req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	p.Proxy(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if got := rec.Body.String(); got != `{"error":"upstream"}` {
		t.Errorf("body = %q, want %q", got, `{"error":"upstream"}`)
	}
}

func TestProxyUpstreamUnavailable(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	url := upstream.URL
	upstream.Close() // адрес больше никто не слушает

	p := NewOllamaProxy(url)

	req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	p.Proxy(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	if !strings.Contains(rec.Body.String(), "upstream unavailable") {
		t.Errorf("body = %q, want it to contain %q", rec.Body.String(), "upstream unavailable")
	}
}
