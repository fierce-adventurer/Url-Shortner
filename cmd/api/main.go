package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/fierceadventurer/Url-Shortner/internal/shortner"
	"github.com/fierceadventurer/Url-Shortner/internal/store"
	"github.com/joho/godotenv"
)

var counter uint64 = 10000

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, relying on system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URl")
	redisURL := os.Getenv("REDIS_URL")
	baseURL := os.Getenv("BASE_URL")

	if dbURL == "" || redisURL == "" {
		log.Fatal("DATABASE_URL and REDIS_URL must be set")
	}

	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	// initialize cloud storage
	db, err := store.NewCloudStore(dbURL, redisURL)
	if err != nil {
		log.Fatalf("Failed to initialize cloud storage: %v", err)
	}

	mux := http.NewServeMux()

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

		// saving into cloud storage
		if err := db.Save(code, payload.URL); err != nil {
			log.Printf("database error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"short_url": fmt.Sprintf("%s/%s", baseURL, code),
		})
	})

	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")

		originalURL, err := db.Get(code)

		if err != nil {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusMovedPermanently)

	})

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
