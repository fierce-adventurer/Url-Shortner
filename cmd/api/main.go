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

	_ "github.com/fierceadventurer/Url-Shortner/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	counter uint64 = 10000
	db      *store.CloudStore
	baseURL string
)

type ShortenRequest struct {
	URL string `json:"url" example:"https://go.dev"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url" example:"http://localhost:8080/abc123"`
}

// @title URL Shortener API
// @version 1.0
// @description A fast, vanilla Go URL shortener using Postgres and Redis.
// @host localhost:8080
// @BasePath /
func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, relying on system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	baseURL := os.Getenv("BASE_URL")

	if dbURL == "" || redisURL == "" {
		log.Fatal("DATABASE_URL and REDIS_URL must be set")
	}

	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	var err error

	db, err = store.NewCloudStore(dbURL, redisURL)
	if err != nil {
		log.Fatalf("Failed to initialize cloud storage: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /shorten", handleShorten)
	mux.HandleFunc("GET /{code}", handleRedirect)
	mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)

	fmt.Println("Server listening on :8080")
	fmt.Println("Swagger UI available at: http://localhost:8080/swagger/index.html")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

// handleShorten shortens a given URL
// @Summary Shorten a long URL
// @Description Takes a long URL, saves it to the database, and returns a short Base62 URL.
// @Accept json
// @Produce json
// @Param request body ShortenRequest true "The long URL to shorten"
// @Success 200 {object} ShortenResponse
// @Failure 400 {string} string "Invalid request payload"
// @Failure 500 {string} string "Internal server error"
// @Router /shorten [post]
func handleShorten(w http.ResponseWriter, r *http.Request) {
	var payload ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	id := atomic.AddUint64(&counter, 1)
	code := shortner.Encode(id)

	if err := db.Save(code, payload.URL); err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL: fmt.Sprintf("%s/%s", baseURL, code),
	})
}

// handleRedirect redirects a short code to its original URL
// @Summary Redirect to original URL
// @Description Takes a short code in the path and issues a 301 redirect to the original URL.
// @Param code path string true "The 7-character short code"
// @Success 301
// @Failure 404 {string} string "Not Found"
// @Router /{code} [get]
func handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	originalURL, err := db.Get(code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}
