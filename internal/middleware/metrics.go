package middleware

import (
	"net/http"
	"chirpy/internal/api"
)
func MetricsInc(cfg *api.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}