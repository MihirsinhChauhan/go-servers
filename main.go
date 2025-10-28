package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

type ValidateChirpRequest struct {
	Body string `json:"body"`
}

type ValidateChirpValidResponse struct {
	Valid bool `json:"valid"`
}

type ValidateChirpErrorResponse struct {
	Error string `json:"error"`
}

// CleanedChirpResponse is the shape we now return.
type CleanedChirpResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

// New POST /api/validate_chirp
func (cfg *apiConfig) handlerValidatecChirp(w http.ResponseWriter, r *http.Request) {
	// only Post
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	// decode json body
	var req ValidateChirpRequest
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

	cleaned := cleanProfanity(req.Body)
	// valid chirp
	respondWithJSON(w,http.StatusOK, CleanedChirpResponse{CleanedBody: cleaned}) 

}
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
	res := ValidateChirpErrorResponse{Error:msg}
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
	// Reset the counter to 0
	cfg.fileserverHits.Store(0)
	// Set Content-Type header
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// Write 200 OK status
	w.WriteHeader(http.StatusOK)
	// Write a simple confirmation
	w.Write([]byte("Counter reset"))
}

func main() {
	// Initialize apiConfig
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
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

	// Register /validate_chirp endpoint
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.handlerValidatecChirp)

	// Create a new Server with the mux as handler and address set to :8080
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start the server
	server.ListenAndServe()
}