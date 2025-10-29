package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"net/http"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver (side-effect import)

	"chirpy/internal/database" // <-- replace with your module name
)

// config
type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	Platform		string
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// CleanedChirpResponse is the shape we now return.
type CleanedChirpResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

type CreateUserRequest struct {
	Email string `json:"email"`
}

type CreateUserResponse struct {
	ID uuid.UUID `json:"id"`
	Email string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

}

type ChirpsRequest struct {
	Body string `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type ChirpsResponse struct {
	ID uuid.UUID `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Body string `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

// Post /api/user - creat user via sqlc
func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err !=nil {
		respondWithError(w, http.StatusBadRequest,"Invalid JSON")
		return
	}

	if req.Email == "" {
		respondWithError(w,http.StatusBadRequest, "Email is required")
		return 
	}

	ctx := context.Background()
	user, err := cfg.DB.CreateUser(ctx, req.Email)
	if err != nil {
		// Handle unique constraint violation
		if strings.Contains(err.Error(), "duplicate key") {
			respondWithError(w, http.StatusConflict, "Email already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}
	resp := CreateUserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}

	respondWithJSON(w, http.StatusCreated, resp)
}


// POST /api/chirps
func (cfg *apiConfig) handlerCreateChirps(w http.ResponseWriter, r *http.Request) {
	// only Post
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	// decode json body
	var req ChirpsRequest
	decoder := 	json.NewDecoder(r.Body)
	

	if err:= decoder.Decode(&req); err !=nil {
		respondWithError(w,http.StatusBadRequest, "Something went wrong")
		return
	} 

	// check bool
	if len(req.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanedBody := cleanProfanity(req.Body)

	ctx:= context.Background()

	chirps, err:= cfg.DB.CreateChirps(ctx, database.CreateChirpsParams{
		Body: cleanedBody,
		UserID: req.UserID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "foregin key") {
			respondWithError(w, http.StatusBadRequest, "Invalid user_id")
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to create Chirp")
	}

	resp := ChirpsResponse{
		ID: chirps.ID,
		CreatedAt: chirps.CreatedAt.Format(time.RFC3339),
		UpdatedAt: chirps.UpdatedAt.Format(time.RFC3339),
		Body: chirps.Body,
		UserID: chirps.UserID,
	}
	respondWithJSON(w, http.StatusCreated, resp)
	


}

// GET /api/chirps

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
        return
	}

	ctx:= context.Background()

	dbChirps, err:= cfg.DB.GetAllChirps(ctx)
	if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to retrieve chirps")
        return
    }
	chirps := make([]ChirpsResponse, len(dbChirps))
	for i, c := range dbChirps {
        chirps[i] = ChirpsResponse{
            ID:        c.ID,
            CreatedAt: c.CreatedAt.Format(time.RFC3339),
            UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
            Body:      c.Body,
            UserID:    c.UserID,
        }
    }

    respondWithJSON(w, http.StatusOK, chirps)


}

// helper
func cleanProfanity(s string) string {
	profane := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	// Split on whitespace while preserving it
	re := regexp.MustCompile(`(\S+|\s+)`)
	parts := re.FindAllString(s, -1)

	var result strings.Builder
	for _, part := range parts {
		// If part is purely whitespace, write it unchanged
		if len(strings.TrimSpace(part)) == 0 {
			result.WriteString(part)
			continue
		}

		// Check if this token is purely alphanumeric (no punctuation)
		isPureWord := true
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				isPureWord = false
				break
			}
		}

		// Only replace if it's a pure word (no punctuation) and matches profane list
		if isPureWord && profane[strings.ToLower(part)] {
			result.WriteString("****")
		} else {
			result.WriteString(part)
		}
	}
	return result.String()
}

// helper
func respondWithError(w http.ResponseWriter,code int, msg string) {
	res := ErrorResponse{Error:msg}
	respondWithJSON(w, code, res)
}

// helper 
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w,r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// fmt.Fprintf(w, "Hits: %d", cfg.fileserverHits.Load())
	// Use fmt.Sprintf to render the HTML template
	html := fmt.Sprintf(`
		<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		</html>`, cfg.fileserverHits.Load())

			w.Write([]byte(html))
		
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	// Only allow in dev
	if cfg.Platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Forbidden: reset only allowed in dev")
		return
	}

	ctx := context.Background()
	if err := cfg.DB.DeleteAllChirps(ctx); err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to delete chirps")
        return
    }
	if err := cfg.DB.DeleteAllUsers(ctx); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to reset users")
		return
	}

	// Reset metrics too
	cfg.fileserverHits.Store(0)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Users and hits reset"))
}

func main() {
	if err:= godotenv.Load() ; err!=nil {
		log.Fatal("error loading .env ")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URl is not in env")
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM not set in .env")
	}

	// Open DB
	db, err := sql.Open("postgres",dbURL)
	if err != nil {
		log.Fatalf("sql.Open: %v" , err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("db.Ping: %v", err)
	}

	// SQLC Queries
	dbQueries := database.New(db)

	// Initialize apiConfig
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
		DB: dbQueries,
		Platform: platform,
	}

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Register the /healthz endpoint
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create a FileServer to serve files from the current directory (.)
	fileServer := http.FileServer(http.Dir("."))

	// Register the FileServer for /app/ path with prefix stripped and metrics middleware
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	// Register the /metrics endpoint
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)

	// Register the /reset endpoint
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	// Register /chirps endpoint
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirps)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)

	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	// Create a new Server with the mux as handler and address set to :8080
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start the server
	log.Println("Server starting on :8080")
	log.Fatal(server.ListenAndServe())
}