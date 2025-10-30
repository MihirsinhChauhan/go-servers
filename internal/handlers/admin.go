package handlers

import (
	"chirpy/internal/api"
	"chirpy/internal/logger"
	"chirpy/internal/utils"
	"context"
	"fmt"
	"net/http"
)

func HandleMetrics(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Admin metrics requested",
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		hits := cfg.FileserverHits.Load()
		html := fmt.Sprintf(`
		<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		</html>`, hits)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))

		logger.Logger.Infow("Admin metrics served", "hits", hits)
	}
}

func HandleReset(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Reset request received",
			"platform", cfg.Platform,
			"remote_addr", r.RemoteAddr,
		)

		if cfg.Platform != "dev" {
			logger.Logger.Warnw("Reset attempted in non-dev environment",
				"platform", cfg.Platform,
			)
			utils.RespondWithError(w, http.StatusForbidden, "Reset only allowed in dev")
			return
		}

		ctx := context.Background()

		if err := cfg.DB.DeleteAllChirps(ctx); err != nil {
			logger.Logger.Errorw("Failed to delete chirps during reset",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete chirps")
			return
		}

		if err := cfg.DB.DeleteAllUsers(ctx); err != nil {
			logger.Logger.Errorw("Failed to delete users during reset",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete users")
			return
		}

		cfg.FileserverHits.Store(0)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Users and hits reset"))

		logger.Logger.Infow("Reset completed successfully")
	}
}