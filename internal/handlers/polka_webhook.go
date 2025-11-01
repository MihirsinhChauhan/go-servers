package handlers

import (
	"chirpy/internal/api"
	"chirpy/internal/logger"
	"chirpy/internal/utils"
	"chirpy/internal/models"
	"chirpy/internal/auth"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func HandlePolkaWebhook(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Polka webhook received",
			"method", r.Method,
			"path", r.URL.Path,
		)
		key, err := auth.GetAPIKey(r.Header)
		if err != nil {
			logger.Logger.Warnw("Missing or invalid API key", "error", err)
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		if key != cfg.PolkaKey {
			logger.Logger.Warnw("Invalid Polka API key", "provided_key", key)
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		var payload models.PolkaPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		// 1. Ignore everything except user.upgraded
		if payload.Event != "user.upgraded" {
			utils.RespondWithJSON(w, http.StatusNoContent, nil)
			return
		}

		// 2. Parse UUID
		userID, err := uuid.Parse(payload.Data.UserID)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid user_id")
			return
		}

		// 3. Upgrade in DB
		if err := cfg.DB.UpgradeToChirpyRed(r.Context(), userID); err != nil {
			// sqlc returns sql.ErrNoRows when the UPDATE affects 0 rows
			if err.Error() == "sql: no rows in result set" {
				utils.RespondWithError(w, http.StatusNotFound, "User not found")
				return
			}
			logger.Logger.Errorw("Failed to upgrade user to Chirpy Red",
				"error", err,
				"user_id", userID,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to upgrade")
			return
		}

		// 4. Success → 204
		utils.RespondWithJSON(w, http.StatusNoContent, nil)
	}
}