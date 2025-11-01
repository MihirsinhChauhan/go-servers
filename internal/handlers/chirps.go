package handlers

import (
	"chirpy/internal/api"
	"chirpy/internal/auth"
	"chirpy/internal/database"
	"chirpy/internal/logger"
	"chirpy/internal/models"
	"chirpy/internal/utils"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"github.com/google/uuid"
)

func HandleCreateChirp(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Create chirp request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		if r.Method != http.MethodPost {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		tokenStr, err := auth.GetBearerToken(r.Header)
		if err != nil {
			logger.Logger.Warnw("Missing or malformed Authorization header",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}
		userID, err := auth.ValidateJWT(tokenStr, cfg.JWTSecret)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		var req models.ChirpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warnw("Invalid JSON payload for chirp creation",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if len(req.Body) > 140 {
			logger.Logger.Infow("Chirp rejected – too long",
				"length", len(req.Body),
				"user_id", req.UserID,
			)
			utils.RespondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}

		cleaned := utils.CleanProfanity(req.Body)
		if cleaned != req.Body {
			logger.Logger.Infow("Profanity filtered",
				"original", req.Body,
				"cleaned", cleaned,
				"user_id", req.UserID,
			)
		}

		ctx := context.Background()
		chirp, err := cfg.DB.CreateChirps(ctx, database.CreateChirpsParams{
			Body:   cleaned,
			UserID: userID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "foreign key") {
				logger.Logger.Warnw("Invalid user_id supplied",
					"user_id", req.UserID,
					"error", err,
				)
				utils.RespondWithError(w, http.StatusBadRequest, "Invalid user_id")
				return
			}
			logger.Logger.Errorw("Failed to create chirp in DB",
				"error", err,
				"user_id", req.UserID,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
			return
		}

		resp := models.ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt.Format(time.RFC3339),
			UpdatedAt: chirp.UpdatedAt.Format(time.RFC3339),
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

		logger.Logger.Infow("Chirp created successfully",
			"chirp_id", chirp.ID,
			"user_id", chirp.UserID,
		)

		utils.RespondWithJSON(w, http.StatusCreated, resp)
	}
}

func HandleGetAllChirps(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Get all chirps request",
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		ctx := context.Background()
		dbChirps, err := cfg.DB.GetAllChirps(ctx)
		if err != nil {
			logger.Logger.Errorw("Failed to fetch chirps from DB",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve chirps")
			return
		}

		chirps := make([]models.ChirpResponse, len(dbChirps))
		for i, c := range dbChirps {
			chirps[i] = models.ChirpResponse{
				ID:        c.ID,
				CreatedAt: c.CreatedAt.Format(time.RFC3339),
				UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
				Body:      c.Body,
				UserID:    c.UserID,
			}
		}

		logger.Logger.Infow("Returned all chirps",
			"count", len(chirps),
		)

		utils.RespondWithJSON(w, http.StatusOK, chirps)
	}
}

func HandleGetChirpByID(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Get chirp by ID request",
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		if r.Method != http.MethodGet {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		idStr := r.PathValue("chirpID")
		if idStr == "" {
			logger.Logger.Warnw("Missing chirp ID in path")
			utils.RespondWithError(w, http.StatusBadRequest, "Missing chirp ID")
			return
		}

		chirpID, err := uuid.Parse(idStr)
		if err != nil {
			logger.Logger.Warnw("Invalid chirp ID format",
				"id", idStr,
				"error", err,
			)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
			return
		}

		ctx := context.Background()
		dbChirp, err := cfg.DB.GetChirpByID(ctx, chirpID)
		if err != nil {
			if err == sql.ErrNoRows {
				logger.Logger.Infow("Chirp not found",
					"chirp_id", chirpID,
				)
				utils.RespondWithError(w, http.StatusNotFound, "Chirp not found")
				return
			}
			logger.Logger.Errorw("DB error while fetching chirp",
				"chirp_id", chirpID,
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve chirp")
			return
		}

		resp := models.ChirpResponse{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt.Format(time.RFC3339),
			UpdatedAt: dbChirp.UpdatedAt.Format(time.RFC3339),
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		}

		logger.Logger.Infow("Chirp retrieved",
			"chirp_id", dbChirp.ID,
			"user_id", dbChirp.UserID,
		)

		utils.RespondWithJSON(w, http.StatusOK, resp)
	}
}

func HandleDeleteChirp(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// === 1. Extract chirp ID from path ===
		chirpIDStr := r.PathValue("chirpID")
		if chirpIDStr == "" {
			logger.Logger.Warnw("Missing chirp ID in path")
			utils.RespondWithError(w, http.StatusBadRequest, "Missing chirp ID")
			return
		}

		chirpID, err := uuid.Parse(chirpIDStr)
		if err != nil {
			logger.Logger.Warnw("Invalid chirp ID format",
				"chirp_id", chirpIDStr,
				"error", err,
			)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
			return
		}

		// === 2. Authenticate user via JWT ===
		tokenStr, err := auth.GetBearerToken(r.Header)
		if err != nil {
			logger.Logger.Warnw("Missing or malformed Authorization header",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		userID, err := auth.ValidateJWT(tokenStr, cfg.JWTSecret)
		if err != nil {
			logger.Logger.Infow("Invalid or expired access token",
				"error", err,
				"token_preview", auth.TruncateToken(tokenStr),
			)
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// === 3. Fetch chirp with author info ===
		ctx := context.Background()
		chirp, err := cfg.DB.GetChirpByID(ctx, chirpID)
		if err != nil {
			if err == sql.ErrNoRows {
				logger.Logger.Infow("Chirp not found for deletion",
					"chirp_id", chirpID,
					"user_id", userID,
				)
				utils.RespondWithError(w, http.StatusNotFound, "Chirp not found")
				return
			}
			logger.Logger.Errorw("Database error fetching chirp",
				"chirp_id", chirpID,
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
			return
		}

		// === 4. Authorization: Only author can delete ===
		if chirp.UserID != userID {
			logger.Logger.Warnw("User attempted to delete another user's chirp",
				"chirp_id", chirp.ID,
				"requesting_user_id", userID,
				"chirp_owner_id", chirp.UserID,
			)
			utils.RespondWithError(w, http.StatusForbidden, "You are not the author of this chirp")
			return
		}

		// === 5. Delete chirp ===
		err = cfg.DB.DeleteChirp(ctx, chirpID)
		if err != nil {
			logger.Logger.Errorw("Failed to delete chirp from database",
				"chirp_id", chirpID,
				"user_id", userID,
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
			return
		}

		// === 6. Success: 204 No Content ===
		logger.Logger.Infow("Chirp deleted successfully",
			"chirp_id", chirpID,
			"user_id", userID,
		)

		w.WriteHeader(http.StatusNoContent)
	}
}