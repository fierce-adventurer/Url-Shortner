package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fierceadventurer/Url-Shortner/internal/shortner"
	"github.com/fierceadventurer/Url-Shortner/internal/store"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	_ "github.com/fierceadventurer/Url-Shortner/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	counter uint64
	db      *store.CloudStore
	baseURL string
)

type ShortenRequest struct {
	URL string `json:"url" example:"https://go.dev"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url" example:"http://localhost:8080/abc123"`
}

var visitors = make(map[string]*rate.Limiter)
var mu sync.Mutex

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(1, 5) // 1 request per second with a burst of 5
		visitors[ip] = limiter
	}

	return limiter
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		limiter := getVisitor(ip)
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// @title URL Shortener API
// @version 1.0
// @description A fast, vanilla Go URL shortener using Postgres and Redis.
// @host localhost:8080
// @BasePath /
func main() {
	counter = uint64(time.Now().Unix())
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, relying on system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	baseURL = os.Getenv("BASE_URL")

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

	mux.HandleFunc("POST /shorten", rateLimitMiddleware(handleShorten))
	mux.HandleFunc("GET /{code}", handleRedirect)
	mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: enableCORS(mux),
	}

	go func() {
		fmt.Println("Server listening on :8080")
		fmt.Println("Swagger UI available at: http://localhost:8080/swagger/index.html")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Server exiting")
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

	go func() {
		if err := db.IncrementClick(code); err != nil {
			log.Printf("Failed to increment click count for code %s: %v", code, err)
		}
	}()

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
