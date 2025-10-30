package middleware

import (
	"chirpy/internal/logger"
	"net/http"
	"time"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response to capture status
		rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)

		duration := time.Since(start)

		logger.Logger.Infow(
			"HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rr.status,
			"duration", duration,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

// responseRecorder captures status code
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}