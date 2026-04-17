package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setup() {
	mu.Lock()
	store = map[string]string{}
	mu.Unlock()
}

// POST /shorten

func TestShortenReturnsCodeAndShortURL(t *testing.T) {
	setup()
	body := bytes.NewBufferString(`{"url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	w := httptest.NewRecorder()

	shortenHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("response is not valid JSON")
	}
	if resp["code"] == "" {
		t.Error("expected non-empty code")
	}
	if resp["short_url"] == "" {
		t.Error("expected non-empty short_url")
	}
}

func TestShortenStoresURL(t *testing.T) {
	setup()
	body := bytes.NewBufferString(`{"url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	w := httptest.NewRecorder()

	shortenHandler(w, req)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	mu.RLock()
	stored, ok := store[resp["code"]]
	mu.RUnlock()

	if !ok {
		t.Fatal("code not found in store")
	}
	if stored != "https://example.com" {
		t.Errorf("expected stored URL to be https://example.com, got %s", stored)
	}
}

func TestShortenRejectsMissingURL(t *testing.T) {
	setup()
	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	w := httptest.NewRecorder()

	shortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestShortenRejectsInvalidJSON(t *testing.T) {
	setup()
	body := bytes.NewBufferString(`not-json`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	w := httptest.NewRecorder()

	shortenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestShortenRejectsNonPost(t *testing.T) {
	setup()
	req := httptest.NewRequest(http.MethodGet, "/shorten", nil)
	w := httptest.NewRecorder()

	shortenHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// GET /{code}

func TestRedirectFound(t *testing.T) {
	setup()
	mu.Lock()
	store["abc123"] = "https://example.com"
	mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	redirectHandler(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com" {
		t.Errorf("expected Location https://example.com, got %s", loc)
	}
}

func TestRedirectNotFound(t *testing.T) {
	setup()
	req := httptest.NewRequest(http.MethodGet, "/doesnotexist", nil)
	w := httptest.NewRecorder()

	redirectHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRootServesUI(t *testing.T) {
	setup()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	redirectHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", ct)
	}
}

func TestRedirectRejectsNonGet(t *testing.T) {
	setup()
	req := httptest.NewRequest(http.MethodPost, "/abc123", nil)
	w := httptest.NewRecorder()

	redirectHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// Round-trip

func TestShortenThenRedirect(t *testing.T) {
	setup()

	// Shorten
	body := bytes.NewBufferString(`{"url":"https://go.dev"}`)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	w := httptest.NewRecorder()
	shortenHandler(w, req)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	code := resp["code"]

	// Redirect
	req2 := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	w2 := httptest.NewRecorder()
	redirectHandler(w2, req2)

	if w2.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w2.Code)
	}
	if loc := w2.Header().Get("Location"); loc != "https://go.dev" {
		t.Errorf("expected Location https://go.dev, got %s", loc)
	}
}
