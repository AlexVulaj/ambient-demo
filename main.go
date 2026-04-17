package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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

	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(indexHTML))
		return
	}

	code := r.URL.Path[1:]

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

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>URL Shortener</title>
  <style>
    body { font-family: sans-serif; max-width: 480px; margin: 80px auto; padding: 0 16px; }
    h1 { font-size: 1.4rem; margin-bottom: 24px; }
    input { width: 100%; box-sizing: border-box; padding: 10px; font-size: 1rem; border: 1px solid #ccc; border-radius: 4px; }
    button { margin-top: 10px; width: 100%; padding: 10px; font-size: 1rem; background: #0070f3; color: #fff; border: none; border-radius: 4px; cursor: pointer; }
    button:hover { background: #005ed4; }
    #result { margin-top: 20px; }
    #result a { color: #0070f3; word-break: break-all; }
    .error { color: #c00; }
  </style>
</head>
<body>
  <h1>URL Shortener</h1>
  <input id="url" type="url" placeholder="https://example.com" />
  <button onclick="shorten()">Shorten</button>
  <div id="result"></div>

  <script>
    async function shorten() {
      const url = document.getElementById('url').value.trim();
      const result = document.getElementById('result');
      result.innerHTML = '';

      if (!url) {
        result.innerHTML = '<p class="error">Please enter a URL.</p>';
        return;
      }

      const resp = await fetch('/shorten', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
      });

      if (!resp.ok) {
        result.innerHTML = '<p class="error">Failed to shorten URL.</p>';
        return;
      }

      const data = await resp.json();
      result.innerHTML = '<p>Short URL: <a href="' + data.short_url + '">' + data.short_url + '</a></p>';
    }

    document.getElementById('url').addEventListener('keydown', e => {
      if (e.key === 'Enter') shorten();
    });
  </script>
</body>
</html>`

func main() {
	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/analytics/", analyticsHandler)
	http.HandleFunc("/analytics", analyticsHandler)
	http.HandleFunc("/", redirectHandler)

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
