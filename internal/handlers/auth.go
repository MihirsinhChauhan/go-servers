// internal/handlers/auth.go
package handlers

import (
	"chirpy/internal/api"
	"chirpy/internal/auth"
	"chirpy/internal/logger"
	"chirpy/internal/models"
	"chirpy/internal/utils"
	"chirpy/internal/database"
	"context"
	"encoding/json"
	"database/sql"
	"net/http"
	"time"
)


func HandleLogin(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Login attempt", "path", r.URL.Path)

		var req models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warnw("Invalid login JSON", "error", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		if req.Email == "" || req.Password == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Email and password required")
			return
		}

		ctx := context.Background()
		user, err := cfg.DB.GetUserByEmail(ctx, req.Email)
		if err != nil {
			logger.Logger.Infow("Login failed: user not found or DB error", "email", req.Email)
			utils.RespondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}

		match, err := auth.CheckPasswordHash(req.Password, user.HashedPassword)
		if err != nil {
			logger.Logger.Errorw("Password check error", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Authentication error")
			return
		}
		if !match {
			logger.Logger.Infow("Login failed: wrong password", "email", req.Email)
			utils.RespondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
			return
		}
		
		// Generate JWT
		accessToken, err := auth.MakeJWT(user.ID, cfg.JWTSecret, time.Hour)
		if err != nil {
			logger.Logger.Errorw("Token Creation failed","error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create token")
			return
		}

		refreshToken, err := auth.MakeRefreshToken()
		if err != nil {
			logger.Logger.Errorw("Failed to generate refresh token", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create refresh token")
			return
		}

		expiresAt := time.Now().Add(60 * 24 * time.Hour)
		err = cfg.DB.CreateRefreshToken(ctx, database.CreateRefreshTokenParams{
			Token:     refreshToken,
			UserID:    user.ID,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			logger.Logger.Errorw("Failed to save refresh token", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create session")
			return
		}

		resp := models.LoginResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
			Token:     accessToken,
			RefreshToken: refreshToken,
			IsChirpyRed: user.IsChirpyRed,
		}

		logger.Logger.Infow("Login successful", "user_id", user.ID, "email", user.Email)
		utils.RespondWithJSON(w, http.StatusOK, resp)
	}
}

func HandleTokenRefresh(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr, err := auth.GetBearerToken(r.Header)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
			return
		}

		ctx := context.Background()
		rt, err := cfg.DB.GetRefreshToken(ctx, tokenStr)
		if err != nil {
			if err == sql.ErrNoRows {
				logger.Logger.Infow("Refresh token not found", "token_preview", auth.TruncateToken(tokenStr))
				utils.RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
				return
			}
			logger.Logger.Errorw("DB error looking up refresh token", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Server error")
			return
		}
		// Check if revoked
		if rt.RevokedAt.Valid {
			logger.Logger.Infow("Refresh token revoked", "token_preview", auth.TruncateToken(tokenStr))
			utils.RespondWithError(w, http.StatusUnauthorized, "Token revoked")
			return
		}

			// Check expiration
		if time.Now().After(rt.ExpiresAt) {
			logger.Logger.Infow("Refresh token expired", "expires_at", rt.ExpiresAt)
			utils.RespondWithError(w, http.StatusUnauthorized, "Token expired")
			return
		}
		// Generate new access token
		accessToken, err := auth.MakeJWT(rt.UserID, cfg.JWTSecret, time.Hour)
		if err != nil {
			logger.Logger.Errorw("Failed to create access token", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create token")
			return
		}

		resp := struct {
			Token string `json:"token"`
		}{
			Token: accessToken,
		}

		logger.Logger.Infow("Access token refreshed",
			"user_id", rt.UserID,
		)

		utils.RespondWithJSON(w, http.StatusOK, resp)
	}
}

func HandleTokenRevoke(cfg *api.Config) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}

	ctx := context.Background()
	_, err = cfg.DB.GetRefreshToken(ctx, tokenStr)
	if err != nil {
		if err == sql.ErrNoRows {
			// Still respond 204 — idempotent
			logger.Logger.Infow("Attempt to revoke non-existent token", "token_preview", auth.TruncateToken(tokenStr))
		} else {
			logger.Logger.Errorw("DB error checking token", "error", err)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Revoke it
	err = cfg.DB.RevokeRefreshToken(ctx, database.RevokeRefreshTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Token:     tokenStr,

	})
	if err != nil {
		logger.Logger.Errorw("Failed to revoke token", "error", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to revoke")
		return
	}

	logger.Logger.Infow("Refresh token revoked", "token_preview", auth.TruncateToken(tokenStr))

	w.WriteHeader(http.StatusNoContent)
	} 
}