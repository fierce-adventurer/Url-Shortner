package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/fierceadventurer/Url-Shortner/internal/shortner"
	"github.com/fierceadventurer/Url-Shortner/internal/store"
)

var counter uint64 = 10000

func main() {
	mux := http.NewServeMux()
	db := store.NewInMemoryStore()

	mux.HandleFunc("POST /shorten", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			URL string `json:"url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		id := atomic.AddUint64(&counter, 1)

		code := shortner.Encode(id)

		db.Save(code, payload.URL)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"short_url": fmt.Sprintf("http://localhost:8080/%s", code),
		})
	})

	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")

		originalURL, _ := db.Get(code)
		if originalURL == "" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusMovedPermanently)

	})

	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", mux)
}
