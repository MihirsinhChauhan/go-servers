// internal/handlers/auth.go
package handlers

import (
	"chirpy/internal/api"
	"chirpy/internal/auth"
	"chirpy/internal/logger"
	"chirpy/internal/models"
	"chirpy/internal/utils"
	"context"
	"encoding/json"
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

		// Determine expiration
		var expiresIn int64 = 3600 // default 1 hour
		if req.ExpiresInSeconds != nil {
			if *req.ExpiresInSeconds > 3600 {
				expiresIn = 3600
			} else if *req.ExpiresInSeconds > 0 {
				expiresIn = int64(*req.ExpiresInSeconds)
			}
		}

		// Generate JWT
		tokenString, err := auth.MakeJWT(user.ID, cfg.JWTSecret, time.Duration(expiresIn)*time.Second)
		if err != nil {
			logger.Logger.Errorw("Token Creation failed","error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create token")
			return
		}

		resp := models.LoginResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
			Token:     tokenString,
		}

		logger.Logger.Infow("Login successful", "user_id", user.ID, "email", user.Email)
		utils.RespondWithJSON(w, http.StatusOK, resp)
	}
}