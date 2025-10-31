package main

import (
	"chirpy/internal/api"
	"chirpy/internal/database"
	"chirpy/internal/handlers"
	"chirpy/internal/logger"
	"chirpy/internal/middleware"
	"database/sql"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		logger.Logger.Fatalw("Error loading .env file", "error", err)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		logger.Logger.Fatal("DB_URL not set in environment")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Logger.Fatal("JWT_SECRET is required in .env")
	}

	platform := os.Getenv("PLATFORM")

	// Open DB
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Logger.Fatalw("Failed to open database", "error", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		logger.Logger.Fatalw("Database ping failed", "error", err)
	}

	dbQueries := database.New(db)

	// Initialize config
	cfg := &api.Config{
		DB:       dbQueries,
		Platform: platform,
		JWTSecret: jwtSecret,
	}

	// Setup router
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Health check", "path", r.URL.Path, "method", r.Method)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Static files with logging middleware
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", middleware.Logging(middleware.MetricsInc(cfg, http.StripPrefix("/app", fileServer))))

	// Admin
	mux.HandleFunc("GET /admin/metrics", handlers.HandleMetrics(cfg))
	mux.HandleFunc("POST /admin/reset", handlers.HandleReset(cfg))

	// API
	mux.HandleFunc("POST /api/login", handlers.HandleLogin(cfg))
	mux.HandleFunc("POST /api/users", handlers.HandleCreateUser(cfg))
	mux.HandleFunc("POST /api/chirps", handlers.HandleCreateChirp(cfg))
	mux.HandleFunc("GET /api/chirps", handlers.HandleGetAllChirps(cfg))
	mux.HandleFunc("GET /api/chirps/{chirpID}", handlers.HandleGetChirpByID(cfg))

	// Start server
	logger.Logger.Infow("Server starting", "port", 8080, "platform", platform)
	logger.Logger.Fatal(http.ListenAndServe(":8080", mux))
}