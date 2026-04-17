package main

import (
	"crypto/rand"
	"encoding/json"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
)

var (
	store = map[string]string{}
	hits  = map[string]int{}
	mu    sync.RWMutex
)

func generateCode() (string, error) {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	code, err := generateCode()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	store[code] = body.URL
	hits[code] = 0
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"code":      code,
		"short_url": fmt.Sprintf("http://%s/%s", r.Host, code),
	})
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Path[1:]
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	mu.Lock()
	url, ok := store[code]
	if ok {
		hits[code]++
	}
	mu.Unlock()

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

func analyticsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Path[len("/analytics/"):]

	mu.RLock()
	defer mu.RUnlock()

	if code != "" {
		count, ok := hits[code]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": code, "hits": count})
		return
	}

	snapshot := make(map[string]int, len(hits))
	for k, v := range hits {
		snapshot[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

func main() {
	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/analytics/", analyticsHandler)
	http.HandleFunc("/analytics", analyticsHandler)
	http.HandleFunc("/", redirectHandler)

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
