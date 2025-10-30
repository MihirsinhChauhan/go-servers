package handlers

import (
	"chirpy/internal/api"
	"chirpy/internal/logger"
	"chirpy/internal/models"
	"chirpy/internal/utils"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func HandleCreateUser(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Create user request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		if r.Method != http.MethodPost {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		var req models.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warnw("Invalid JSON for user creation",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		if req.Email == "" {
			logger.Logger.Warnw("Empty email supplied")
			utils.RespondWithError(w, http.StatusBadRequest, "Email is required")
			return
		}

		ctx := context.Background()
		user, err := cfg.DB.CreateUser(ctx, req.Email)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				logger.Logger.Warnw("Duplicate email attempt",
					"email", req.Email,
				)
				utils.RespondWithError(w, http.StatusConflict, "Email already exists")
				return
			}
			logger.Logger.Errorw("Failed to insert user into DB",
				"email", req.Email,
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		resp := models.CreateUserResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		}

		logger.Logger.Infow("User created successfully",
			"user_id", user.ID,
			"email", user.Email,
		)

		utils.RespondWithJSON(w, http.StatusCreated, resp)
	}
}